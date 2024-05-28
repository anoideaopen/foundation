// Package routing provides functionality for defining and handling endpoints
// in a blockchain application context.
package routing

import "fmt"

// EndpointType represents the type of an endpoint in the routing package.
type EndpointType int

// String returns the string representation of an EndpointType.
func (t EndpointType) String() string {
	switch t {
	case EndpointTypeTransaction:
		return "transaction"
	case EndpointTypeInvoke:
		return "invoke"
	case EndpointTypeQuery:
		return "query"
	default:
		return fmt.Sprintf("unknown (%d)", t)
	}
}

// Constants representing the different types of endpoints.
const (
	EndpointTypeTransaction EndpointType = iota
	EndpointTypeInvoke
	EndpointTypeQuery
)

// Endpoint represents an endpoint in the routing system.
type Endpoint struct {
	Type          EndpointType // The type of the endpoint.
	ChaincodeFunc string       // The name of the chaincode function being called.
	MethodName    string       // The actual method name to be invoked.
	NumArgs       int          // Number of arguments the method takes (excluding the receiver).
	NumReturns    int          // Number of return values the method has.
	RequiresAuth  bool         // Indicates if the method requires authentication.
}
