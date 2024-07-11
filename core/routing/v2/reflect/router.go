package reflect

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/core/stringsx"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// Reflect router errors.
var (
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")
	ErrUnsupportedMethod    = errors.New("unsupported method")
	ErrInvalidMethodName    = errors.New("invalid method name")
)

// Router routes method calls to contract methods based on reflection.
type Router struct {
	service          any
	methodHandler    map[string]handler // map[method]handler
	methodToFunction map[string]string  // map[method]function
	functionToMethod map[string]string  // map[function]method
}

// NewRouter creates a new Router instance with the given contract.
func NewRouter(contract any) (*Router, error) {
	var (
		methodHandler    = make(map[string]handler)
		methodToFunction = make(map[string]string)
		functionToMethod = make(map[string]string)
	)
	for _, method := range Methods(contract) {
		h, err := newHandler(method, contract)
		if err != nil {
			if errors.Is(err, ErrUnsupportedMethod) {
				continue
			}

			return nil, err
		}

		if _, ok := methodToFunction[h.function]; ok {
			return nil, fmt.Errorf("%w, method: '%s'", ErrMethodAlreadyDefined, h.function)
		}

		methodHandler[h.function] = h
		methodToFunction[h.method] = h.function
		functionToMethod[h.function] = h.method
	}

	return &Router{
		service:          contract,
		methodHandler:    methodHandler,
		methodToFunction: methodToFunction,
		functionToMethod: functionToMethod,
	}, nil
}

// MustNewRouter creates a new Router instance with the given contract and panics if
// an error occurs.
func MustNewRouter(contract any) *Router {
	r, err := NewRouter(contract)
	if err != nil {
		panic(err)
	}

	return r
}

// Check validates the provided arguments for the specified method.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	return ValidateArguments(r.service, method, stub, args...)
}

// Invoke calls the specified method with the provided arguments.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	result, err := Call(r.service, method, stub, args...)
	if err != nil {
		return nil, err
	}

	if MethodReturnsError(r.service, method) {
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

// Handlers returns a map of method names to chaincode functions.
func (r *Router) Handlers() map[string]string { // map[method]function
	return r.methodToFunction
}

// Method retrieves the method associated with the specified chaincode function.
func (r *Router) Method(function string) (method string) {
	return r.functionToMethod[function]
}

// Function returns the name of the chaincode function by the specified method.
func (r *Router) Function(method string) (function string) {
	return r.methodToFunction[method]
}

// AuthRequired indicates if the method requires authentication.
func (r *Router) AuthRequired(method string) bool {
	return r.methodHandler[method].authRequired
}

// ArgCount returns the number of arguments the method takes (excluding the receiver).
func (r *Router) ArgCount(method string) int {
	return r.methodHandler[method].argCount
}

// IsTransaction checks if the method is a transaction type.
func (r *Router) IsTransaction(method string) bool {
	return r.methodHandler[method].isTransaction
}

// IsInvoke checks if the method is an invoke type.
func (r *Router) IsInvoke(method string) bool {
	return r.methodHandler[method].isInvoke
}

// IsQuery checks if the method is a query type.
func (r *Router) IsQuery(method string) bool {
	return r.methodHandler[method].isQuery
}

type handler struct {
	method        string
	function      string
	argCount      int
	authRequired  bool
	isTransaction bool
	isInvoke      bool
	isQuery       bool
}

func newHandler(name string, of any) (handler, error) {
	const (
		batchedTransactionPrefix      = "Tx"
		transactionWithoutBatchPrefix = "NBTx"
		queryTransactionPrefix        = "Query"
	)

	h := handler{
		method: name,
	}

	switch {
	case strings.HasPrefix(name, batchedTransactionPrefix):
		h.isTransaction = true
		h.function = strings.TrimPrefix(name, batchedTransactionPrefix)

	case strings.HasPrefix(name, transactionWithoutBatchPrefix):
		h.isInvoke = true
		h.function = strings.TrimPrefix(name, transactionWithoutBatchPrefix)

	case strings.HasPrefix(name, queryTransactionPrefix):
		h.isQuery = true
		h.function = strings.TrimPrefix(name, queryTransactionPrefix)

	default:
		return handler{}, fmt.Errorf("%w: %s", ErrUnsupportedMethod, name)
	}

	if len(h.function) == 0 {
		return handler{}, fmt.Errorf("%w: %s", ErrInvalidMethodName, name)
	}

	h.function = stringsx.LowerFirstChar(h.function)
	h.argCount = InputParamCounts(of, name)
	h.authRequired = IsArgOfType(of, name, 0, &types.Sender{})

	return h, nil
}
