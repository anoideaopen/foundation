// Package mux provides a multiplexer (or router) that allows multiple
// [github.com/anoideaopen/foundation/core/routing.Router] instances to be used together.
// This is useful when different parts of a smart contract require distinct routing logic
// or when multiple routing strategies need to be combined. The `mux.Router` acts as a central hub
// that delegates method calls to the appropriate router based on the method name.
//
// When initializing a chaincode, if multiple routers are provided using
// [github.com/anoideaopen/foundation/core.WithRouters], this multiplexer will be
// automatically used to manage them. It ensures that each method is routed to the correct handler
// according to the method-to-function mappings defined by the underlying routers.
//
// Example usage:
//
//	// Initialize individual routers
//	reflectRouter := reflect.NewRouter(myContract)
//	grpcRouter := grpc.NewRouter()
//
//	// Create a multiplexer router
//	muxRouter, err := mux.NewRouter(reflectRouter, grpcRouter)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use the multiplexer router in your chaincode initialization
//	cc, err := core.NewCC(myContract, core.WithRouter(muxRouter))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start the chaincode
//	if err = cc.Start(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Error Handling
//
// If a method or function is defined more than once across the provided routers,
// the `NewRouter` function will return an `ErrChaincodeFunction` error to avoid
// ambiguity in routing.
//
// Additionally, if a method is not supported by any of the provided routers,
// the multiplexer will return an `ErrUnsupportedMethod` error.
package mux
