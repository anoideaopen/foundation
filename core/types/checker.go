package types

import "github.com/hyperledger/fabric-chaincode-go/v2/shim"

// Checker is an interface that can be implemented by types that can check themselves.
type Checker interface {
	Check() error
}

// CheckerWithStub is an interface that can be implemented by types that can check themselves.
type CheckerWithStub interface {
	CheckWithStub(stub shim.ChaincodeStubInterface) error
}
