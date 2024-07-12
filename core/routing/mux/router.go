package mux

import (
	"errors"

	"github.com/anoideaopen/foundation/core/routing"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

var (
	// ErrMethodAlreadyDefined is returned when a method has already been defined in the router.
	ErrMethodAlreadyDefined = errors.New("pure method has already defined")

	// ErrUnsupportedMethod is returned when a method is not supported by the router.
	ErrUnsupportedMethod = errors.New("unsupported method")
)

// Router is a multiplexer that routes methods to the appropriate handler.
type Router struct {
	methodRouter map[string]routing.Router // Method -> Router
	routers      []routing.Router
}

// NewRouter creates a new Router with the provided routing.Router instances.
// It returns an error if any method is defined more than once.
func NewRouter(router ...routing.Router) (*Router, error) {
	methodRouter := make(map[string]routing.Router)

	for _, r := range router {
		for _, method := range r.Methods() {
			if _, ok := methodRouter[method.Method]; ok {
				return nil, ErrMethodAlreadyDefined
			}

			methodRouter[method.Method] = r
		}
	}

	return &Router{
		methodRouter: methodRouter,
		routers:      router,
	}, nil
}

// Check validates the provided arguments for the specified method.
// It returns an error if the validation fails.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	if m, ok := r.methodRouter[method]; ok {
		return m.Check(stub, method, args...)
	}

	return ErrUnsupportedMethod
}

// Invoke calls the specified method with the provided arguments.
// It returns a byte slice of response and an error if the invocation fails.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	if m, ok := r.methodRouter[method]; ok {
		return m.Invoke(stub, method, args...)
	}

	return nil, ErrUnsupportedMethod
}

// Methods retrieves a map of all available methods, keyed by their chaincode function names.
func (r *Router) Methods() map[string]routing.Method {
	methods := make(map[string]routing.Method, len(r.methodRouter))

	for _, r := range r.routers {
		for fn, m := range r.Methods() {
			methods[fn] = m
		}
	}

	return methods
}
