package types

import "github.com/hyperledger/fabric-chaincode-go/shim"

// Validator is an interface that can be implemented by types that can validate themselves.
type Validator interface {
	Validate() error
}

// ValidatorWithStub is an interface that can be implemented by types that can validate themselves.
type ValidatorWithStub interface {
	ValidateWithStub(stub shim.ChaincodeStubInterface) error
}
