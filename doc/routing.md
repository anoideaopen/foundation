# `routing` Package Documentation

The `routing` package provides an interface and various implementations of routers for managing method calls in smart contracts within the Hyperledger Fabric framework. The package includes several subsystems like `reflect`, `mux`, and `grpc`, each of which implements the `Router` interface to ensure flexibility and extensibility in handling contract invocations.

## Overview

The `Router` interface is central to this package, defining a set of methods required for routing calls, checking arguments, and managing method metadata. Depending on the use case, different router implementations can be employed to handle method routing dynamically or based on predefined rules.

### Main Components

1. **Router Interface**:
   - The core interface that all routers in this package must implement.
   - **Methods**:
     - `Check`: Validates the arguments for a specified contract method.
     - `Invoke`: Invokes the specified contract method with the given arguments.
     - `Handlers`: Returns a map linking method names to their corresponding contract functions.
     - `Method`: Retrieves the method name associated with a specified contract function.
     - `Function`: Retrieves the contract function name associated with a specified method.
     - `AuthRequired`: Indicates whether the method requires authentication.
     - `ArgCount`: Returns the number of arguments the specified method takes.
     - `IsTransaction`, `IsInvoke`, `IsQuery`: Determine the type of the method (transaction, invoke, or query).

2. **Subsystems**:
   - **Reflect Router**:
     - Uses reflection to dynamically route method calls based on method names and signatures.
     - Methods should be named with specific prefixes to indicate their type:
       - `Tx` for transactional methods that modify state.
       - `NBTx` for transactional methods that modify state without batching.
       - `Query` for read-only methods that do not modify state.
     - Supports dynamic type conversion and authorization based on method signatures.
   - **gRPC Router**:
     - Integrates with gRPC to route method calls defined in gRPC services.
     - Leverages gRPC service descriptors and protobuf extensions to manage method metadata and routing.
     - Supports context management for passing user and transaction data.
   - **Mux Router**:
     - A multiplexer that combines multiple routers, allowing the integration of different routing strategies.
     - Useful for combining reflection-based routing with gRPC or other custom routing mechanisms.

## Detailed Component Descriptions

### Router Interface

The `Router` interface defines the required methods for any router implementation. This interface is essential for routing method calls, validating inputs, and managing contract method metadata.

```go
type Router interface {
    Check(stub shim.ChaincodeStubInterface, method string, args ...string) error
    Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error)
    Handlers() map[string]string
    Method(function string) string
    Function(method string) string
    AuthRequired(method string) bool
    ArgCount(method string) int
    IsTransaction(method string) bool
    IsInvoke(method string) bool
    IsQuery(method string) bool
}
```

### Reflect Router

The `reflect` subsystem provides a dynamic router that uses Go's reflection capabilities to route method calls based on their names and signatures. 

- **Method Naming Conventions**:
  - **Transactional Methods** (`Tx` prefix): These methods modify the state of the ledger.
  - **Query Methods** (`Query` prefix): These methods are read-only and do not modify the state.
  - **Invoke Methods** (`NBInvoke` prefix): These methods are executed directly without batching.

- **Example**:
  ```go
  type FiatToken struct {
      // BaseToken provides common token functionality
      token.BaseToken
  }

  // TxEmit emits new tokens to the specified address.
  func (ft *FiatToken) TxEmit(sender *types.Sender, address *types.Address, amount *big.Int) error {
      // Method implementation
  }

  // QueryAllowedBalanceAdd queries if a balance adjustment is allowed.
  func (ft *FiatToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
      // Method implementation
  }
  ```

  - **Authorization**: If a method requires authorization, the first argument should be a `Sender` object.
  - **Dynamic Argument Parsing**: Arguments are parsed and validated based on the method signature.

### gRPC Router

The `grpc` subsystem integrates gRPC service definitions with the Hyperledger Fabric chaincode. It routes method calls based on gRPC service descriptors and manages additional metadata using protobuf extensions.

- **Protobuf Method Options**:
  - Methods are annotated using custom protobuf extensions to indicate the type (`transaction`, `invoke`, `query`) and whether authentication is required.

- **Example**:
  ```proto
  service FiatService {
      rpc AddBalanceByAdmin(BalanceAdjustmentRequest) returns (google.protobuf.Empty) {
          option (foundation.method_type) = METHOD_TYPE_TRANSACTION;
      }
  }
  ```

- **Go Implementation**:
  ```go
  func (ft *FiatToken) AddBalanceByAdmin(ctx context.Context, req *service.BalanceAdjustmentRequest) (*emptypb.Empty, error) {
      // Method implementation
  }
  ```

  - **Context Management**: The router manages contexts to pass information such as the sender and chaincode stub to the method.

### Mux Router

The `mux` subsystem allows combining multiple routers into a single unified router. This is particularly useful when different routing strategies are needed in the same chaincode.

- **Usage**:
  - Multiple routers (e.g., reflect and gRPC) can be combined using the `mux` router.
  - The combined router can then be passed to the chaincode for handling method calls.

- **Example**:
  ```go
  func main() {
      token := NewFiatToken()
      grpcRouter := grpc.NewRouter()
      reflectRouter := reflect.MustNewRouter(token)

      cc, err := core.NewCC(
          token,
          core.WithRouters(grpcRouter, reflectRouter),
      )
      if err != nil {
          log.Fatal(err)
      }

      if err = cc.Start(); err != nil {
          log.Fatal(err)
      }
  }
  ```

  - **Combining Routers**: The `core.WithRouters` function is used to combine multiple routers, which are then used to manage method routing in the chaincode.

## Chaincode Initialization and Usage

When initializing a chaincode with the `routing` package, the default router used is the `reflect` router, which dynamically routes method calls based on their names and signatures. This default behavior can be modified by explicitly specifying routers using the `WithRouters` option during chaincode initialization. When multiple routers are provided via `WithRouters`, the `mux` router is automatically engaged to handle the multiplexing of these routers.

### Default Initialization

If no routers are specified, the chaincode will default to using the `reflect` router:

```go
cc, err := core.NewCC(token)
if err != nil {
    log.Fatal(err)
}
```

### Custom Routers with `WithRouters`

To combine different routing strategies, such as `grpc` and `reflect`, you can initialize the chaincode with multiple routers using the `WithRouters` option:

```go
grpcRouter := grpc.NewRouter()
reflectRouter := reflect.MustNewRouter(token)

cc, err := core.NewCC(
    token,
    core.WithRouters(grpcRouter, reflectRouter),
)
if err != nil {
    log.Fatal(err)
}
```

In this setup, the `mux` router will manage routing between the provided `grpc` and `reflect` routers.

### Custom Router Implementation

Developers are also free to implement their own custom router by adhering to the `routing.Router` interface. Once implemented, this custom router can be integrated into the chaincode just like the built-in routers:

```go
type CustomRouter struct{}

// Implement all the methods of the routing.Router interface for CustomRouter

cc, err := core.NewCC(
    token,
    core.WithRouters(&CustomRouter{}),
)
if err != nil {
    log.Fatal(err)
}
```

This flexibility allows developers to tailor the routing behavior to the specific needs of their application.

### Examples and References

For real-world examples of how to set up and use different routers, you can refer to the following packages:

- **Reflect Router**: `github.com/anoideaopen/foundation/core/routing/reflect`
- **gRPC Router**: `github.com/anoideaopen/foundation/core/routing/grpc`
- **Mux Router**: `github.com/anoideaopen/foundation/core/routing/mux`
- **Chaincode Examples**: Look into the `foundation/test/chaincode/fiat` directory for various chaincode examples that illustrate the use of these routers in practice.
