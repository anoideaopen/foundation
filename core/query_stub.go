package core

import (
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
)

const (
	// ErrMethodNotImplemented is the error message for not implemented methods
	ErrMethodNotImplemented = "method is not implemented for query"
	errFuncNotImplemented   = ErrMethodNotImplemented + ": %s"
)

type queryStub struct {
	shim.ChaincodeStubInterface
}

func newQueryStub(stub shim.ChaincodeStubInterface) *queryStub {
	return &queryStub{
		ChaincodeStubInterface: stub,
	}
}

func (qs *queryStub) PutState(_ string, _ []byte) error {
	return nil
}

func (qs *queryStub) DelState(_ string) error {
	return nil
}

func (qs *queryStub) SetStateValidationParameter(_ string, _ []byte) error {
	return nil
}

func (qs *queryStub) PutPrivateData(_ string, _ string, _ []byte) error {
	return nil
}

func (qs *queryStub) DelPrivateData(_, _ string) error {
	return nil
}

func (qs *queryStub) PurgePrivateData(_, _ string) error {
	return nil
}

func (qs *queryStub) SetPrivateDataValidationParameter(_, _ string, _ []byte) error {
	return nil
}

func (qs *queryStub) SetEvent(_ string, _ []byte) error {
	return nil
}
