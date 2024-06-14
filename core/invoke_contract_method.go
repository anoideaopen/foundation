package core

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/codec"
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// InvokeContractMethod calls a Chaincode contract method, processes the arguments, and returns the result as bytes.
//
// Parameters:
//   - traceCtx: The telemetry trace context for tracing the method invocation.
//   - stub: The ChaincodeStubInterface instance used for invoking the method.
//   - method: The contract.Method instance representing the method to be invoked.
//   - sender: The sender's address, if the method requires authentication.
//   - args: A slice of strings representing the arguments to be passed to the method.
//   - cfgBytes: A byte slice containing the configuration data for the contract.
//
// Returns:
//   - A byte slice containing the serialized return value of the method, or an error if an issue occurs.
//
// The function performs the following steps:
//  1. Initializes a new span for tracing.
//  2. Adds the sender's address to the arguments if provided.
//  3. Checks the number of arguments, ensuring it matches the expected count.
//  4. Applies the configuration data to the contract.
//  5. Calls the contract method via the router.
//  6. Checks the number of return values, ensuring it matches the expected count.
//  7. Processes the return error if the method returns an error.
//  8. If the return value implements the codec.BytesEncoder interface, calls the EncodeToBytes() method to
//     get the byte slice.
//  9. Otherwise, serializes the return value(s) to JSON. If there is one return value, it is serialized directly.
//     If there are multiple return values, they are serialized as a JSON array.
func (cc *Chaincode) InvokeContractMethod(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	method contract.Method,
	sender *proto.Address,
	args []string,
	cfgBytes []byte,
) ([]byte, error) {
	_, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.CallMethod")
	defer span.End()

	args = cc.PrependSender(method, sender, args)

	span.SetAttributes(attribute.StringSlice("args", args))
	if method.NumArgs != len(args) {
		err := fmt.Errorf("expected %d arguments, got %d", method.NumArgs, len(args))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.AddEvent("applying config")
	if err := contract.Configure(cc.contract, stub, cfgBytes); err != nil {
		return nil, err
	}

	span.AddEvent("call")
	result, err := cc.Router().Invoke(method.MethodName, args...)
	if err != nil {
		return nil, err
	}

	if len(result) != method.NumReturns {
		err := fmt.Errorf("expected %d return values, got %d", method.NumReturns, len(result))
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if method.ReturnsError {
		if errorValue := result[len(result)-1]; errorValue != nil {
			if err, ok := errorValue.(error); ok {
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			span.SetStatus(codes.Error, requireInterfaceErrMsg)
			return nil, errors.New(requireInterfaceErrMsg)
		}

		result = result[:len(result)-1]
	}

	span.SetStatus(codes.Ok, "")
	switch len(result) {
	case 0:
		return json.Marshal(nil)
	case 1:
		if be, ok := result[0].(codec.BytesEncoder); ok {
			return be.EncodeToBytes()
		}
		return json.Marshal(result[0])
	default:
		return json.Marshal(result)
	}
}
