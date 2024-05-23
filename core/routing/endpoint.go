package routing

import "fmt"

type EndpointType int

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

const (
	EndpointTypeTransaction EndpointType = iota
	EndpointTypeInvoke
	EndpointTypeQuery
)

type Endpoint struct {
	Type          EndpointType // The type of the endpoint.
	ChaincodeFunc string       // The name of the chaincode function being called.
	MethodName    string       // The actual method name to be invoked.
	NumArgs       int          // Number of arguments the method takes (excluding the receiver).
	NumReturns    int          // Number of return values the method has.
}
