// Package routing defines the Router interface for managing smart contract method calls.
//
// This interface is essential for processing transaction method calls within the
// [github.com/anoideaopen/foundation/core] package. It provides mechanisms for validating
// method arguments via [Router.Check], executing methods via [Router.Invoke], and managing
// routing metadata.
//
// Router interface implementations include:
//   - [github.com/anoideaopen/foundation/core/routing/reflect.Router]: The default implementation
//     using reflection for dynamic method invocation.
//   - [github.com/anoideaopen/foundation/core/routing/mux.Router]: Combines multiple routers,
//     allowing flexible routing based on method names.
//   - [github.com/anoideaopen/foundation/core/routing/grpc.Router]: Routes method calls based on
//     GRPC service descriptions and protobuf extensions.
//
// In the [github.com/anoideaopen/foundation/core] package, the Router interface is used during
// Chaincode initialization. When creating a new Chaincode instance with
// [github.com/anoideaopen/foundation/core.NewCC], a router is configured to handle method routing.
// If no custom routers are provided, the default reflection-based router is used.
//
// The Router interface ensures that all method calls are properly validated, executed, and routed
// within the Chaincode environment.
//
// # Example
//
// Below is an example of initializing a GRPC router alongside a reflection-based router:
// See: [github.com/anoideaopen/foundation/test/chaincode/fiat/main.go].
//
//	package main
//
//	import (
//	    "log"
//
//	    "github.com/anoideaopen/foundation/core"
//	    "github.com/anoideaopen/foundation/core/routing/grpc"
//	    "github.com/anoideaopen/foundation/core/routing/reflect"
//	    "github.com/anoideaopen/foundation/test/chaincode/fiat/service"
//	)
//
//	func main() {
//	    // Create a new instance of the contract (e.g., FiatToken).
//	    token := NewFiatToken()
//
//	    // Initialize a GRPC router for handling method calls based on GRPC service descriptions.
//	    grpcRouter := grpc.NewRouter()
//
//	    // Initialize a reflection-based router for dynamic method invocation.
//	    reflectRouter := reflect.MustNewRouter(token)
//
//	    // Register the GRPC service server with the GRPC router.
//	    service.RegisterFiatServiceServer(grpcRouter, token)
//
//	    // Create a new Chaincode instance with the GRPC and reflection-based routers.
//	    cc, err := core.NewCC(
//	        token,
//	        core.WithRouters(grpcRouter, reflectRouter),
//	    )
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Start the Chaincode instance.
//	    if err = cc.Start(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
package routing

import "github.com/hyperledger/fabric-chaincode-go/shim"

// Router defines the interface for managing smart contract methods and routing calls.
// It is used in the core package to manage method calls, perform validation, and ensure proper
// routing of requests based on the type of call (transaction, invoke, query).
type Router interface {
	// Check validates the provided arguments for the specified method.
	// Validates the arguments for the specified contract method. Returns an error if the arguments are invalid.
	Check(stub shim.ChaincodeStubInterface, method string, args ...string) error

	// Invoke calls the specified method with the provided arguments.
	// Invokes the specified contract method and returns the execution result. Returns the result as a byte
	// slice ([]byte) or an error if invocation fails.
	Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error)

	// Handlers returns a map of method names to chaincode functions.
	// Returns a map linking method names to their corresponding contract functions.
	Handlers() map[string]string // map[method]function

	// Method retrieves the method associated with the specified chaincode function.
	// Returns the method name linked to the specified contract function.
	Method(function string) (method string)

	// Function returns the name of the chaincode function by the specified method.
	// Returns the contract function name associated with the specified method.
	Function(method string) (function string)

	// AuthRequired indicates if the method requires authentication.
	// Returns true if the method requires authentication, otherwise false.
	AuthRequired(method string) bool

	// ArgCount returns the number of arguments the method takes.
	// Returns the number of arguments expected by the specified method, excluding the receiver.
	ArgCount(method string) int

	// IsTransaction checks if the method is a transaction type.
	// Returns true if the method is a transaction, otherwise false.
	IsTransaction(method string) bool

	// IsInvoke checks if the method is an invoke type.
	// Returns true if the method is an invoke operation, otherwise false.
	IsInvoke(method string) bool

	// IsQuery checks if the method is a query type.
	// Returns true if the method is a read-only query, otherwise false.
	IsQuery(method string) bool
}
