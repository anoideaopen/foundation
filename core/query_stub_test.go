package core

import (
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest" //nolint:staticcheck
	"github.com/stretchr/testify/require"
)

func TestQueryStub(t *testing.T) {
	stub := shimtest.NewMockStub("query", nil)

	stub.MockTransactionStart("txID")
	err := stub.PutState("key", []byte("value"))
	require.NoError(t, err)
	stub.MockTransactionEnd("txID")

	qs := newQueryStub(stub)

	t.Run("GetState [positive]", func(t *testing.T) {
		val, _ := qs.GetState("key")
		require.Equal(t, "value", string(val))
	})

	t.Run("PutState [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.PutState("key", []byte(""))
		require.EqualError(t, err, ErrMethodNotImplemented)
	})

	t.Run("DelState [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.DelState("key")
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetStateValidationParameter [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetStateValidationParameter("key", []byte("new"))
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	err = stub.PutPrivateData("collection", "key", []byte("value"))
	require.NoError(t, err)

	t.Run("PutPrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err = qs.PutPrivateData("collection", "key", []byte("value2"))
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("DelPrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err = qs.DelPrivateData("collection", "key")
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("PurgePrivateData [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.PurgePrivateData("collection", "key")
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetPrivateDataValidationParameter [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetPrivateDataValidationParameter("collection", "key", []byte("new"))
		require.Errorf(t, err, ErrMethodNotImplemented)
	})

	t.Run("SetEvent [negative]", func(t *testing.T) {
		t.Skip()
		err := qs.SetEvent("event", []byte("payload"))
		require.Errorf(t, err, ErrMethodNotImplemented)
	})
}
