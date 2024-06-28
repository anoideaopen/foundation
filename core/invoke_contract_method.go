package core

import (
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
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
//  3. Sets trace attributes for the arguments.
//  4. Checks the number of arguments, ensuring it matches the expected count.
//  5. Applies the configuration data to the contract.
//  6. Calls the contract method via the router.
//  7. Processes the return error if the method returns an error.
//  8. Sets the trace status to Ok if no error occurs and returns the result.
func (cc *Chaincode) InvokeContractMethod(
	traceCtx telemetry.TraceContext,
	stub shim.ChaincodeStubInterface,
	method contract.Method,
	sender *proto.Address,
	args []string,
) ([]byte, error) {
	_, span := cc.contract.TracingHandler().StartNewSpan(traceCtx, "chaincode.CallMethod")
	defer span.End()

	cc.contract.SetStub(stub)

	span.AddEvent("call")
	result, err := cc.Router().Invoke(method.MethodName, cc.PrependSender(method, sender, args)...)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}
