package reflectx

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/routing"
	"github.com/anoideaopen/foundation/core/stringsx"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

var (
	// ErrMethodAlreadyDefined is returned when a method has already been defined in the router.
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")

	// ErrUnsupportedMethod is returned when a method is not supported by the router.
	ErrUnsupportedMethod = errors.New("unsupported method")

	// ErrInvalidMethodName is returned when a method has an invalid name.
	ErrInvalidMethodName = errors.New("invalid method name")
)

// Router routes method calls to contract methods based on reflection.
type Router struct {
	contract any
	methods  map[routing.Function]routing.Method
}

// NewRouter creates a new Router instance with the given contract.
// It reflects on the methods of the provided contract and sets up routing for them.
//
// Parameters:
//   - baseContract: The contract instance to route methods for.
//
// Returns:
//   - *Router: A new Router instance.
//   - error: An error if the router setup fails.
func NewRouter(contract any) (*Router, error) {
	r := &Router{
		contract: contract,
		methods:  make(map[routing.Function]routing.Method),
	}

	for _, method := range Methods(contract) {
		ep, err := newReflectEndpoint(method, contract)
		if err != nil {
			if errors.Is(err, ErrUnsupportedMethod) {
				continue
			}

			return nil, err
		}

		if _, ok := r.methods[ep.ChaincodeFunc]; ok {
			return nil, fmt.Errorf("%w, method: '%s'", ErrMethodAlreadyDefined, ep.ChaincodeFunc)
		}

		r.methods[ep.ChaincodeFunc] = *ep
	}

	return r, nil
}

// Check validates the provided arguments for the specified method.
// It returns an error if the validation fails.
//
// Parameters:
//   - stub: The ChaincodeStubInterface instance to use for the validation.
//   - method: The name of the method to validate arguments for.
//   - args: The arguments to validate.
//
// Returns:
//   - error: An error if the validation fails.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	return ValidateArguments(r.contract, method, stub, args...)
}

// Invoke calls the specified method with the provided arguments.
// It returns a slice of return values and an error if the invocation fails.
//
// Parameters:
//   - stub: The ChaincodeStubInterface instance to use for the invocation.
//   - method: The name of the method to invoke.
//   - args: The arguments to pass to the method.
//
// Returns:
//   - []byte: A slice of bytes (JSON) representing the return values.
//   - error: An error if the invocation fails.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	result, err := Call(r.contract, method, stub, args...)
	if err != nil {
		return nil, err
	}

	if MethodReturnsError(r.contract, method) {
		if errorValue := result[len(result)-1]; errorValue != nil {
			return nil, errorValue.(error) //nolint:forcetypeassert
		}

		result = result[:len(result)-1]
	}

	switch len(result) {
	case 0:
		return json.Marshal(nil)
	case 1:
		if encoder, ok := result[0].(types.BytesEncoder); ok {
			return encoder.EncodeToBytes()
		}
		if encoder, ok := result[0].(types.StubBytesEncoder); ok {
			return encoder.EncodeToBytesWithStub(stub)
		}
		return json.Marshal(result[0])
	default:
		return json.Marshal(result)
	}
}

// Methods retrieves a map of all available methods, keyed by their chaincode function names.
//
// Returns:
//   - map[routing.Function]routing.Method: A map of all available methods.
func (r *Router) Methods() map[routing.Function]routing.Method {
	return r.methods
}

// newReflectEndpoint creates a new Method instance for the given method name and contract.
// It infers the method type, chaincode function name, and other attributes based on the method name and contract.
//
// Parameters:
//   - name: The name of the method.
//   - of: The contract instance.
//
// Returns:
//   - *routing.Method: A new Method instance.
//   - error: An error if the method creation fails.
func newReflectEndpoint(name string, of any) (*routing.Method, error) {
	const (
		batchedTransactionPrefix      = "Tx"
		transactionWithoutBatchPrefix = "NBTx"
		queryTransactionPrefix        = "Query"
	)

	method := &routing.Method{
		Type:          0,
		ChaincodeFunc: "",
		MethodName:    name,
		NumArgs:       0,
		RequiresAuth:  false,
	}

	switch {
	case strings.HasPrefix(method.MethodName, batchedTransactionPrefix):
		method.Type = routing.MethodTypeTransaction
		method.ChaincodeFunc = strings.TrimPrefix(method.MethodName, batchedTransactionPrefix)

	case strings.HasPrefix(method.MethodName, transactionWithoutBatchPrefix):
		method.Type = routing.MethodTypeInvoke
		method.ChaincodeFunc = strings.TrimPrefix(method.MethodName, transactionWithoutBatchPrefix)

	case strings.HasPrefix(method.MethodName, queryTransactionPrefix):
		method.Type = routing.MethodTypeQuery
		method.ChaincodeFunc = strings.TrimPrefix(method.MethodName, queryTransactionPrefix)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMethod, method.MethodName)
	}

	if len(method.ChaincodeFunc) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMethodName, method.MethodName)
	}

	method.ChaincodeFunc = stringsx.LowerFirstChar(method.ChaincodeFunc)
	method.NumArgs, _ = MethodParamCounts(of, method.MethodName)
	method.RequiresAuth = IsArgOfType(of, method.MethodName, 0, &types.Sender{})

	return method, nil
}
