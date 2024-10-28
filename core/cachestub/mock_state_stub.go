package cachestub

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type stateStub struct {
	shim.ChaincodeStubInterface
	state map[string][]byte
}

func newStateStub() *stateStub {
	return &stateStub{state: make(map[string][]byte)}
}

func (stub *stateStub) GetState(key string) ([]byte, error) {
	return stub.state[key], nil
}

func (stub *stateStub) PutState(key string, value []byte) error {
	stub.state[key] = value
	return nil
}

func (stub *stateStub) DelState(key string) error {
	delete(stub.state, key)
	return nil
}
