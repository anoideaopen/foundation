package contract

// MethodType represents the type of a method in the contract.
type MethodType int

// Constants representing the different types of methods.
const (
	MethodTypeTransaction MethodType = iota // Tx-prefixed transaction when using reflectx.Router.
	MethodTypeInvoke                        // NBTx-prefixed transaction when using reflectx.Router.
	MethodTypeQuery                         // Query-prefixed transaction when using reflectx.Router.
)

// Function represents the name of a chaincode function.
type Function = string

// Method represents an endpoint of a contract.
type Method struct {
	Type          MethodType // The type of the method.
	ChaincodeFunc Function   // The name of the chaincode function being called.
	MethodName    string     // The actual method name to be invoked.
	RequiresAuth  bool       // Indicates if the method requires authentication.
	ReturnsError  bool       // Indicates if the method returns an error.
	NumArgs       int        // Number of arguments the method takes (excluding the receiver).
	NumReturns    int        // Number of return values the method has.
}

// Router defines the interface for managing contract methods and routing calls.
type Router interface {
	// Check validates the provided arguments for the specified method.
	// It returns an error if the validation fails.
	Check(method string, args ...string) error

	// Invoke calls the specified method with the provided arguments.
	// It returns a slice of return values and an error if the invocation fails.
	Invoke(method string, args ...string) ([]any, error)

	// Methods retrieves a map of all available methods, keyed by their chaincode function names.
	Methods() map[Function]Method
}
