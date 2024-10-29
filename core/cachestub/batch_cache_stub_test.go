package cachestub

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"testing"

	"github.com/anoideaopen/foundation/mocks"
	"github.com/stretchr/testify/require"
)

const (
	valKey1 = "KEY1"
	valKey2 = "KEY2"
	valKey3 = "KEY3"
	valKey4 = "KEY4"

	valKey1Value1 = "key1_value1"
	valKey2Value1 = "key2_value1"
	valKey4Value1 = "key4_value1"
)

//go:generate counterfeiter -generate

//counterfeiter:generate -o ../mocks/chaincode_stub.go --fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

func TestBatchStub(t *testing.T) {
	stateStub := &mocks.ChaincodeStub{}

	// creating batch cache stub
	batchStub := NewBatchCacheStub(stateStub)

	// 1st transaction changes key 1 value and deletes from key 2
	t.Run("first batch transaction", func(t *testing.T) {
		// changing key1 value
		_ = batchStub.PutState(valKey1, []byte(valKey1Value1))
		// deleting key2 value
		_ = batchStub.DelState(valKey2)
		// committing changes to mockStub
		_ = batchStub.Commit()

		// checking key1 value added
		val, _ := batchStub.GetState(valKey1)
		require.Equal(t, valKey1Value1, string(val))

		// checking key2 value is deleted
		val, _ = batchStub.GetState(valKey2)
		require.Equal(t, "", string(val))

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 1, stateStub.PutStateCallCount())
		require.Equal(t, 1, stateStub.DelStateCallCount())
	})

	// 2nd batch transaction changes key 2  value and deletes key 3
	t.Run("second batch transaction", func(t *testing.T) {
		// adding key2 value
		_ = batchStub.PutState(valKey2, []byte(valKey2Value1))
		// deleting key3 value
		_ = batchStub.DelState(valKey3)
		// committing changes to mock stub
		_ = batchStub.Commit()

		// checking key2 value added
		val, _ := batchStub.GetState(valKey2)
		require.Equal(t, valKey2Value1, string(val))

		// checking key3 value is deleted
		val, _ = batchStub.GetState(valKey3)
		require.Equal(t, "", string(val))

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 3, stateStub.PutStateCallCount())
		require.Equal(t, 2, stateStub.DelStateCallCount())
	})

	// 3rd batch transaction adds and deletes key 4 value
	t.Run("third batch transaction", func(t *testing.T) {
		// adding key4 value
		_ = batchStub.PutState(valKey4, []byte(valKey4Value1))
		// deleting key4 value
		_ = batchStub.DelState(valKey4)
		// committing changes to mock stub
		_ = batchStub.Commit()

		val, _ := batchStub.GetState(valKey4)
		require.Equal(t, "", string(val))

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 5, stateStub.PutStateCallCount())
		require.Equal(t, 4, stateStub.DelStateCallCount())
	})
}
