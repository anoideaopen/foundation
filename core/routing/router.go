package routing

import "github.com/hyperledger/fabric-chaincode-go/shim"

// MethodType represents the type of a method in the contract.
type MethodType int

// Constants representing the different types of methods.
const (
	MethodTypeTransaction MethodType = iota // Tx-prefixed transaction when using reflectx.Router.
	MethodTypeInvoke                        // NBTx-prefixed transaction when using reflectx.Router.
	MethodTypeQuery                         // Query-prefixed transaction when using reflectx.Router.
)

// Method represents an endpoint of a contract.
type Method struct {
	Method       string     // The actual method name to be invoked.
	Function     string     // The name of the chaincode function being called.
	ArgCount     int        // Number of arguments the method takes (excluding the receiver).
	AuthRequired bool       // Indicates if the method requires authentication.
	Type         MethodType // The type of the method.
}

// IsInvoke checks if the method is an invoke type.
func (m Method) IsInvoke() bool {
	return m.Type == MethodTypeInvoke
}

// IsQuery checks if the method is a query type.
func (m Method) IsQuery() bool {
	return m.Type == MethodTypeQuery
}

// IsTransaction checks if the method is a transaction type.
func (m Method) IsTransaction() bool {
	return m.Type == MethodTypeTransaction
}

// Router defines the interface for managing contract methods and routing calls.
type Router interface {
	// Check validates the provided arguments for the specified method.
	// It returns an error if the validation fails.
	Check(stub shim.ChaincodeStubInterface, method string, args ...string) error

	// Invoke calls the specified method with the provided arguments.
	// It returns a byte slice of response and an error if the invocation fails.
	Invoke(stub shim.ChaincodeStubInterface, method string, args ...string) ([]byte, error)

	// Methods retrieves a map of all available methods, keyed by their chaincode function names.
	Methods() map[string]Method
}
