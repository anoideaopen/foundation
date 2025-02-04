package routing

import "github.com/hyperledger/fabric-chaincode-go/v2/shim"

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
