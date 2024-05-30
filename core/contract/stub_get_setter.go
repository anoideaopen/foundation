package contract

import "github.com/hyperledger/fabric-chaincode-go/shim"

// StubGetSetter defines methods for getting and setting the ChaincodeStubInterface.
type StubGetSetter interface {
	// GetStub retrieves the current ChaincodeStubInterface.
	GetStub() shim.ChaincodeStubInterface

	// SetStub sets the provided ChaincodeStubInterface.
	SetStub(shim.ChaincodeStubInterface)
}
