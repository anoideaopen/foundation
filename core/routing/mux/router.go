package mux

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/routing"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

var (
	// ErrChaincodeFunction is returned when a chaincode function has already been defined in the router.
	ErrChaincodeFunction = errors.New("chaincode function already defined")

	// ErrUnsupportedMethod is returned when a method is not supported by the router.
	ErrUnsupportedMethod = errors.New("unsupported method")
)

// Router is a multiplexer that routes methods to the appropriate handler.
type Router struct {
	methodRouter     map[string]routing.Router // method -> router
	methodToFunction map[string]string         // method -> function
	functionToMethod map[string]string         // function -> method
}

// NewRouter creates a new Router with the provided routing.Router instances.
// It returns an error if any chaincode function is defined more than once.
func NewRouter(router ...routing.Router) (*Router, error) {
	var (
		methodRouter     = make(map[string]routing.Router)
		methodToFunction = make(map[string]string)
		functionToMethod = make(map[string]string)
	)
	for _, r := range router {
		for method, function := range r.Handlers() {
			if _, ok := functionToMethod[function]; ok {
				return nil, fmt.Errorf("%w, function: '%s'", ErrChaincodeFunction, function)
			}

			methodRouter[method] = r
			methodToFunction[method] = function
			functionToMethod[function] = method
		}
	}

	return &Router{
		methodRouter:     methodRouter,
		methodToFunction: methodToFunction,
		functionToMethod: functionToMethod,
	}, nil
}

// Check validates the provided arguments for the specified method.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	if router, ok := r.methodRouter[method]; ok {
		return router.Check(stub, method, args...)
	}

	return ErrUnsupportedMethod
}

// Invoke calls the specified method with the provided arguments.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	if router, ok := r.methodRouter[method]; ok {
		return router.Invoke(stub, method, args...)
	}

	return nil, ErrUnsupportedMethod
}

// Handlers returns a map of method names to chaincode functions.
func (r *Router) Handlers() map[string]string { // map[method]function
	return r.methodToFunction
}

// Method retrieves the method associated with the specified chaincode function.
func (r *Router) Method(function string) string {
	return r.functionToMethod[function]
}

// Function returns the name of the chaincode function by the specified method.
func (r *Router) Function(method string) string {
	return r.methodToFunction[method]
}

// AuthRequired indicates if the method requires authentication.
func (r *Router) AuthRequired(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.AuthRequired(method)
	}

	return false
}

// ArgCount returns the number of arguments the method takes (excluding the receiver).
func (r *Router) ArgCount(method string) int {
	if router, ok := r.methodRouter[method]; ok {
		return router.ArgCount(method)
	}

	return 0
}

// IsTransaction checks if the method is a transaction type.
func (r *Router) IsTransaction(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsTransaction(method)
	}

	return false
}

// IsInvoke checks if the method is an invoke type.
func (r *Router) IsInvoke(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsInvoke(method)
	}

	return false
}

// IsQuery checks if the method is a query type.
func (r *Router) IsQuery(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsQuery(method)
	}

	return false
}
