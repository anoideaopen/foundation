package core

import (
	"testing"

	"github.com/anoideaopen/foundation/mocks"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/stretchr/testify/require"
)

const (
	valKey        = "key"
	valValue1     = "value"
	valValue2     = "value 2"
	valValue3     = "value 3"
	valCollection = "collection"
	valNew        = "new"
	valEvent      = "event"
	valPayload    = "payload"
)

//go:generate counterfeiter -generate

//counterfeiter:generate -o ../mocks/chaincode_stub.go --fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

func TestQueryStub(t *testing.T) {
	mockStub := &mocks.ChaincodeStub{}

	// preparing mockStub
	mockStub.GetStateReturns([]byte(valValue1), nil)
	mockStub.GetPrivateDataReturns([]byte(valValue2), nil)
	qs := newQueryStub(mockStub)

	t.Run("GetState [positive]", func(t *testing.T) {
		val, err := qs.GetState(valKey)
		require.NoError(t, err)
		require.Equal(t, valValue1, string(val))
	})

	t.Run("PutState [negative]", func(t *testing.T) {
		_ = qs.PutState(valKey, []byte(""))
		// checking PutState was not called
		require.Equal(t, mockStub.PutStateCallCount(), 0)
	})

	t.Run("DelState [negative]", func(t *testing.T) {
		_ = qs.DelState(valKey)
		// checking DelState was not called
		require.Equal(t, mockStub.DelStateCallCount(), 0)
	})

	t.Run("SetStateValidationParameter [negative]", func(t *testing.T) {
		_ = qs.SetStateValidationParameter(valKey, []byte(valNew))
		// checking SetStateValidationParameter was not called
		require.Equal(t, mockStub.SetStateValidationParameterCallCount(), 0)
	})

	t.Run("GetPrivateData [positive]", func(t *testing.T) {
		val, err := qs.GetPrivateData(valKey, valValue2)
		require.NoError(t, err)
		require.Equal(t, valValue2, string(val))
	})

	t.Run("PutPrivateData [negative]", func(t *testing.T) {
		_ = qs.PutPrivateData(valCollection, valKey, []byte(valValue3))
		// checking PutPrivateData was not called
		require.Equal(t, mockStub.PutPrivateDataCallCount(), 0)
	})

	t.Run("DelPrivateData [negative]", func(t *testing.T) {
		_ = qs.DelPrivateData(valCollection, valKey)
		// checking DelPrivateData was not called
		require.Equal(t, mockStub.DelPrivateDataCallCount(), 0)
	})

	t.Run("PurgePrivateData [negative]", func(t *testing.T) {
		_ = qs.PurgePrivateData("collection", "key")
		// checking PurgePrivateData was not called
		require.Equal(t, mockStub.PurgePrivateDataCallCount(), 0)
	})

	t.Run("SetPrivateDataValidationParameter [negative]", func(t *testing.T) {
		_ = qs.SetPrivateDataValidationParameter(valCollection, valKey, []byte(valNew))
		// checking SetPrivateDataValidationParameter was not called
		require.Equal(t, mockStub.SetPrivateDataValidationParameterCallCount(), 0)
	})

	t.Run("SetEvent [negative]", func(t *testing.T) {
		_ = qs.SetEvent(valEvent, []byte(valPayload))
		// checking SetEvent was not called
		require.Equal(t, mockStub.SetEventCallCount(), 0)
	})
}
