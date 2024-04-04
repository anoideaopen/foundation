package core

import (
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest" //nolint:staticcheck
	"github.com/stretchr/testify/assert"
)

func TestQueryStub(t *testing.T) {
	stub := shimtest.NewMockStub("query", nil)

	stub.MockTransactionStart("txID")
	err := stub.PutState("key", []byte("value"))
	assert.NoError(t, err)
	stub.MockTransactionEnd("txID")

	qs := newQueryStub(stub)

	t.Run("GetState [positive]", func(t *testing.T) {
		val, _ := qs.GetState("key")
		assert.Equal(t, "value", string(val))
	})

	t.Run("PutState [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.PutState("key", []byte(""))
		assert.EqualError(t, err, ErrMethodNotImplemented)
	})

	t.Run("DelState [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.DelState("key")
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetStateValidationParameter [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetStateValidationParameter("key", []byte("new"))
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	err = stub.PutPrivateData("collection", "key", []byte("value"))
	assert.NoError(t, err)

	t.Run("PutPrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err = qs.PutPrivateData("collection", "key", []byte("value2"))
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("DelPrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err = qs.DelPrivateData("collection", "key")
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("PurgePrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.PurgePrivateData("collection", "key")
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetPrivateDataValidationParameter [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetPrivateDataValidationParameter("collection", "key", []byte("new"))
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetEvent [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetEvent("event", []byte("payload"))
		assert.Errorf(t, err, ErrMethodNotImplemented)
	})
}
