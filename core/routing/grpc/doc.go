// Package grpc provides a GRPC-based router for smart contract method invocation.
// This package uses protocol buffers (protobuf) and GRPC service descriptions
// to dynamically route and invoke smart contract methods based on predefined
// service and method definitions.
//
// # GRPC Routing
//
// The core functionality of this package revolves around the Router type, which implements
// the github.com/anoideaopen/foundation/core/routing.Router interface. The router dynamically routes
// incoming method calls based on GRPC service definitions and protobuf method options.
//
// Supported GRPC Method Types:
//   - Transaction: Methods that modify the blockchain state. These are identified by the
//     custom `method_type` option set to `METHOD_TYPE_TRANSACTION` in the protobuf definition.
//   - Invoke: Methods that are executed directly as Hyperledger Fabric Invoke transactions,
//     bypassing the batching mechanism.
//   - Query: Read-only methods that retrieve data without altering the blockchain state.
//
// The router also supports custom method authorization settings, defined using the
// `method_auth` option in the protobuf file. These settings allow developers to specify
// whether a method requires authentication.
//
// # Protobuf Example
//
// Here is an example of a protobuf definition that includes custom extensions for method types and authorization:
//
//	syntax = "proto3";
//
//	package foundationtoken;
//
//	option go_package = "github.com/anoideaopen/foundation/test/chaincode/fiat/service";
//
//	import "google/protobuf/empty.proto";
//	import "validate/validate.proto";
//	import "method_options.proto"; // Import custom options.
//
//	message Address {
//	    string base58check = 1 [(validate.rules).string = {pattern: "^[1-9A-HJ-NP-Za-km-z]+$"}];
//	}
//
//	message BigInt {
//	    string value = 1 [(validate.rules).string = {pattern: "^[0-9]+$"}];
//	}
//
//	message BalanceAdjustmentRequest {
//	    Address address = 1 [(validate.rules).message.required = true];
//	    BigInt amount   = 2 [(validate.rules).message.required = true];
//	    string reason   = 3 [(validate.rules).string = {min_len: 1, max_len: 200}];
//	}
//
//	service FiatService {
//	    rpc AddBalanceByAdmin(BalanceAdjustmentRequest) returns (google.protobuf.Empty) {
//	        option (foundation.method_type) = METHOD_TYPE_TRANSACTION;
//	    }
//	}
//
// # Usage Example
//
// To use the GRPC router in your chaincode, you can define a service in your Go code
// that implements the methods defined in your protobuf file, and then register this service
// with the GRPC router:
//
//	package main
//
//	import (
//	    "context"
//	    "errors"
//	    mbig "math/big"
//
//	    "github.com/anoideaopen/foundation/core/routing/grpc"
//	    "github.com/anoideaopen/foundation/test/chaincode/fiat/service"
//	    "google.golang.org/protobuf/types/known/emptypb"
//	)
//
//	// FiatToken represents a custom smart contract
//	type FiatToken struct {
//	    service.UnimplementedFiatServiceServer
//	}
//
//	// AddBalanceByAdmin implements a method to add balance by an admin
//	func (ft *FiatToken) AddBalanceByAdmin(ctx context.Context, req *service.BalanceAdjustmentRequest) (*emptypb.Empty, error) {
//	    if grpc.SenderFromContext(ctx) == "" {
//	        return nil, errors.New("unauthorized")
//	    }
//
//	    if grpc.StubFromContext(ctx) == nil {
//	        return nil, errors.New("stub is nil")
//	    }
//
//	    value, _ := mbig.NewInt(0).SetString(req.GetAmount().GetValue(), 10)
//	    return &emptypb.Empty{}, balance.Add(
//	        grpc.StubFromContext(ctx),
//	        balance.BalanceTypeToken,
//	        req.GetAddress().GetBase58Check(),
//	        "",
//	        value,
//	    )
//	}
//
//	func main() {
//	    contract := &FiatToken{}
//	    grpcRouter := grpc.NewRouter()
//
//	    // Register the service with the GRPC router
//	    service.RegisterFiatServiceServer(grpcRouter, contract)
//
//	    // Initialize chaincode with the GRPC router
//	    cc, err := core.NewCC(
//	        contract,
//	        core.WithRouters(grpcRouter),
//	    )
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Start chaincode
//	    if err = cc.Start(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
// # Method Invocation and Authorization
//
// When a method is invoked via the GRPC router, the router first checks the method
// options defined in the protobuf file to determine the type of method (transaction,
// invoke, or query) and whether authentication is required. If authentication is
// required, the router will validate the sender before invoking the method.
//
// Below is an example of initializing a GRPC router alongside a reflection-based router:
//
// See: [github.com/anoideaopen/foundation/test/chaincode/fiat]
package grpc
