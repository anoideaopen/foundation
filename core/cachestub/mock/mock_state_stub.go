package mock

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type StateStub struct {
	shim.ChaincodeStubInterface
	State map[string][]byte
}

func NewStateStub() *StateStub {
	return &StateStub{State: make(map[string][]byte)}
}

func (stub *StateStub) GetState(key string) ([]byte, error) {
	return stub.State[key], nil
}

func (stub *StateStub) PutState(key string, value []byte) error {
	stub.State[key] = value
	return nil
}

func (stub *StateStub) DelState(key string) error {
	delete(stub.State, key)
	return nil
}
