package routing

import "github.com/hyperledger/fabric-chaincode-go/shim"

// Router defines the interface for managing contract methods and routing calls.
type Router interface {
	// Check validates the provided arguments for the specified method.
	Check(stub shim.ChaincodeStubInterface, method string, args ...string) error

	// Invoke calls the specified method with the provided arguments.
	Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error)

	// Handlers returns a map of method names to chaincode functions.
	Handlers() map[string]string // map[method]function

	// Method retrieves the method associated with the specified chaincode function.
	Method(function string) (method string)

	// Function returns the name of the chaincode function by the specified method.
	Function(method string) (function string)

	// AuthRequired indicates if the method requires authentication.
	AuthRequired(method string) bool

	// ArgCount returns the number of arguments the method takes.
	ArgCount(method string) int

	// IsTransaction checks if the method is a transaction type.
	IsTransaction(method string) bool

	// IsInvoke checks if the method is an invoke type.
	IsInvoke(method string) bool

	// IsQuery checks if the method is a query type.
	IsQuery(method string) bool
}
