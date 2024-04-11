package cachestub_test

import "github.com/hyperledger/fabric-chaincode-go/shimtest"

type mockStub struct {
	shimtest.MockStub
	state map[string][]byte
}

func newMockStub() *mockStub {
	return &mockStub{state: make(map[string][]byte)}
}

func (stub *mockStub) GetState(key string) ([]byte, error) {
	return stub.state[key], nil
}

func (stub *mockStub) PutState(key string, value []byte) error {
	stub.state[key] = value
	return nil
}

func (stub *mockStub) DelState(key string) error {
	delete(stub.state, key)
	return nil
}
