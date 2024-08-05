// Package routing defines the Router interface for managing smart contract method calls.
//
// This interface is essential for processing transaction method calls within the
// [github.com/anoideaopen/foundation/core] package. It provides mechanisms for validating
// method arguments via [Router.Check], executing methods via [Router.Invoke], and managing
// routing metadata.
//
// Router interface implementations include:
//   - [github.com/anoideaopen/foundation/core/routing/reflect]: The default implementation
//     using reflection for dynamic method invocation.
//   - [github.com/anoideaopen/foundation/core/routing/mux]: Combines multiple routers,
//     allowing flexible routing based on method names.
//   - [github.com/anoideaopen/foundation/core/routing/grpc]: Routes method calls based on
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
//
// See: [github.com/anoideaopen/foundation/test/chaincode/fiat]
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
