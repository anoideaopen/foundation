// Package reflect provides functionality for routing smart contract method calls
// using Go reflection. This package allows dynamic invocation of contract methods
// based on their names and types, enabling flexible and generic handling of
// blockchain transactions.
//
// Method Naming Conventions:
//
// To properly utilize the reflection-based routing, methods in the smart contract
// should follow specific naming conventions. These conventions help the router
// identify the type of method and route it accordingly:
//
//   - "TxMethod": Prefix "Tx" indicates that the method is a transaction and may
//     modify the blockchain state.
//   - "NBTxMethod": Prefix "NBTx" stands for "Non-Batched Transaction," indicating
//     a transaction that should not be batched with others.
//   - "QueryMethod": Prefix "Query" indicates that the method is a query, meaning
//     it only reads data without modifying the blockchain state.
//
// Authorization:
//
// If a method requires authorization, the first argument passed to the method
// is the [github.com/anoideaopen/foundation/core/types.Sender], representing the address
// of the user who invoked the contract.
//
// This is automatically handled by the router, which prepends the sender to the
// list of arguments before the method is invoked.
//
// Example:
//
//	type MyContract struct {}
//
//	// TxCreateUser is a transaction method that creates a new user.
//	// It requires authorization, so the first argument is the sender's address.
//	func (c *MyContract) TxCreateUser(sender *types.Sender, args []string) error {
//	    // Implementation
//	}
//
//	// QueryGetUser retrieves a user by ID without modifying the state.
//	// Since it's a query, it does not require authorization.
//	func (c *MyContract) QueryGetUser(args []string) (User, error) {
//	    // Implementation
//	}
//
// Method Parsing and Invocation:
//
// The core of this package revolves around the ability to dynamically parse and
// invoke methods on the contract using reflection. This is achieved through
// functions that inspect the contract's methods, validate arguments, and handle
// the execution.
//
// Method Inspection and Invocation:
//
// The package provides utilities for reflectively inspecting methods, checking
// argument types, and invoking methods. The Call function is a central part of
// this process:
//
//	func Call(v any, method string, stub shim.ChaincodeStubInterface, args ...string) ([]any, error) {
//	    // Implementation
//	}
//
// This function first checks whether the specified method exists on the given
// contract object. It then verifies the number and types of the provided arguments.
// If the method requires authorization, the Sender argument is automatically prepended.
// After all checks pass, it invokes the method and returns the result.
//
// Example:
//
//	contract := &MyContract{}
//	result, err := reflect.Call(contract, "TxCreateUser", stub, "senderAddress", []string{"arg1", "arg2"})
//	if err != nil {
//	    log.Fatalf("Method invocation failed: %v", err)
//	}
//
// Argument Parsing:
//
// Argument parsing is handled by the ParseValue function, which converts
// string arguments into their corresponding Go types. The function supports
// various data formats, including plain strings, JSON, and custom interfaces
// such as BytesDecoder and StubBytesDecoder.
//
// Example:
//
//	func ParseValue(s string, t reflect.Type, stub shim.ChaincodeStubInterface) (reflect.Value, error) {
//	    // Implementation
//	}
//
// This function first checks if the target type is a simple string or a pointer
// to a string. If not, it attempts to decode the string using various interfaces
// and JSON. If all decoding attempts fail, it returns a ValueError.
//
// Interface Types:
//
// The package also supports custom interface types for argument parsing and
// validation. Interfaces such as Checker and CheckerWithStub
// allow for additional validation and processing logic to be applied to arguments
// before invoking a method.
//
// Example:
//
//	type MyType struct {
//	    Value string
//	}
//
//	func (v *MyValidator) Check() error {
//	    if v.Value == "" {
//	        return errors.New("value cannot be empty")
//	    }
//	    return nil
//	}
//
//	func (c *MyContract) TxCreateValidatedUser(sender *proto.Address, v *MyValidator) error {
//	    // Implementation
//	}
//
// In this example, the Check method of the MyValidator struct is called to
// validate the argument before invoking TxCreateValidatedUser.
//
// Integration with Core:
//
// In the core package, the reflect.Router is often used as the default router
// for managing contract methods. When initializing a chaincode using
// github.com/anoideaopen/foundation/core.NewCC, a router can be provided via
// core.WithRouters, and if no custom routers are provided, the default
// reflection-based router is used.
//
// Example of chaincode initialization:
//
//	contract := &MyContract{}
//	router, err := reflect.NewRouter(contract)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	cc, err := core.NewCC(contract, core.WithRouters(router))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cc.Start()
//
// Error Handling:
//
// The package defines several custom errors to handle cases where reflection
// encounters issues, such as methods not being found, incorrect argument types,
// or validation failures. These errors ensure that any issues in method invocation
// are caught and handled appropriately.
package reflect
