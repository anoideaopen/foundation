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
//
// The Router struct combines multiple [github.com/anoideaopen/foundation/core/routing.Router]
// instances, allowing them to be used together. Each method call is delegated to the appropriate
// router based on a method-to-function mapping. This enables complex routing logic where
// different types of method calls are handled by different routers.
//
// The multiplexer ensures that each method is routed to the correct handler according
// to the method-to-function mappings defined by the underlying routers.
type Router struct {
	methodRouter     map[string]routing.Router // method -> router
	methodToFunction map[string]string         // method -> function
	functionToMethod map[string]string         // function -> method
}

// NewRouter creates a new Router with the provided
// [github.com/anoideaopen/foundation/core/routing.Router] instances.
//
// It iterates over the provided routers and builds internal mappings between methods
// and functions. If a function is defined in more than one router, an error is returned
// to avoid conflicts. This function is used to initialize the multiplexer that manages
// multiple routers.
//
// If multiple routers are provided during the initialization of a chaincode using
// [github.com/anoideaopen/foundation/core.WithRouters], this function is automatically
// invoked to create a multiplexer.
//
// Returns an error if any chaincode function is defined more than once.
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
//
// The Check method delegates the validation of arguments to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
// If the method is not supported by any of the provided routers, it returns ErrUnsupportedMethod.
func (r *Router) Check(stub shim.ChaincodeStubInterface, method string, args ...string) error {
	if router, ok := r.methodRouter[method]; ok {
		return router.Check(stub, method, args...)
	}

	return ErrUnsupportedMethod
}

// Invoke calls the specified method with the provided arguments.
//
// The Invoke method routes the call to the correct
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
// If the method is not supported by any of the provided routers, it returns ErrUnsupportedMethod.
func (r *Router) Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error) {
	if router, ok := r.methodRouter[method]; ok {
		return router.Invoke(stub, method, args...)
	}

	return nil, ErrUnsupportedMethod
}

// Handlers returns a map of method names to chaincode functions.
//
// This map is built based on the underlying routers and their mappings.
// It provides a comprehensive view of which methods are handled by which functions
// across all combined [github.com/anoideaopen/foundation/core/routing.Router] instances.
func (r *Router) Handlers() map[string]string { // map[method]function
	return r.methodToFunction
}

// Method retrieves the method associated with the specified chaincode function.
//
// This method returns the method name linked to the specified contract function,
// based on the internal mappings of the multiplexer.
func (r *Router) Method(function string) string {
	return r.functionToMethod[function]
}

// Function returns the name of the chaincode function by the specified method.
//
// This method returns the contract function name associated with the specified method,
// based on the internal mappings of the multiplexer.
func (r *Router) Function(method string) string {
	return r.methodToFunction[method]
}

// AuthRequired indicates if the method requires authentication.
//
// This method delegates the check to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
func (r *Router) AuthRequired(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.AuthRequired(method)
	}

	return false
}

// ArgCount returns the number of arguments the method takes (excluding the receiver).
//
// This method delegates the check to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
func (r *Router) ArgCount(method string) int {
	if router, ok := r.methodRouter[method]; ok {
		return router.ArgCount(method)
	}

	return 0
}

// IsTransaction checks if the method is a transaction type.
//
// This method delegates the check to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
func (r *Router) IsTransaction(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsTransaction(method)
	}

	return false
}

// IsInvoke checks if the method is an invoke type.
//
// This method delegates the check to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
func (r *Router) IsInvoke(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsInvoke(method)
	}

	return false
}

// IsQuery checks if the method is a query type.
//
// This method delegates the check to the appropriate
// [github.com/anoideaopen/foundation/core/routing.Router] based on the method name.
func (r *Router) IsQuery(method string) bool {
	if router, ok := r.methodRouter[method]; ok {
		return router.IsQuery(method)
	}

	return false
}
