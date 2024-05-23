package core

import (
	"github.com/anoideaopen/foundation/core/routing"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type Router interface {
	Endpoints() []routing.Endpoint
	Endpoint(chaincodeFunc string) (routing.Endpoint, error)
	ValidateArguments(method string, stub shim.ChaincodeStubInterface, args ...string) error
	Call(method string, stub shim.ChaincodeStubInterface, args ...string) ([]any, error)
}
