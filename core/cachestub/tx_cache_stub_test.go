package cachestub

import (
	"testing"

	"github.com/anoideaopen/foundation/mocks"
	"github.com/stretchr/testify/require"
)

const (
	txID1 = "txID1"
	txID2 = "txID2"
	txID3 = "txID3"

	valKey4Value2 = "key4_value2"
	valKey4Value3 = "key4_value3"
)

func TestTxStub(t *testing.T) {
	stateStub := &mocks.ChaincodeStub{}

	batchStub := NewBatchCacheStub(stateStub)

	// 1st batch transaction 1 adds value of key1, deletes key2 then adds key 2 value
	t.Run("batch transaction 1", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID1)

		_ = txStub.PutState(valKey1, []byte(valKey1Value1))
		_ = txStub.DelState(valKey2)
		txStub.Commit()

		// checking first transaction results were properly committed
		val, _ := batchStub.GetState(valKey2)
		require.Equal(t, "", string(val))

		val, _ = batchStub.GetState(valKey1)
		require.Equal(t, valKey1Value1, string(val))

		// 2nd iteration of 1st tx stub changes
		_ = txStub.PutState(valKey2, []byte(valKey2Value1))
		_ = txStub.DelState(valKey3)
		txStub.Commit()

		_ = batchStub.Commit()

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 2, stateStub.PutStateCallCount())
		require.Equal(t, 1, stateStub.DelStateCallCount())
	})

	// 2nd batch transaction adds and deletes value for key 4, then changes key4 value
	t.Run("batch transaction 2", func(_ *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID2)
		_ = txStub.PutState(valKey4, []byte(valKey4Value1))
		_ = txStub.DelState(valKey4)
		txStub.Commit()

		// batchStub checks if key 4 was deleted and changes its value
		val, _ := batchStub.GetState(valKey4)
		require.Equal(t, "", string(val))
		_ = batchStub.PutState(valKey4, []byte(valKey4Value2))

		_ = batchStub.Commit()

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 5, stateStub.PutStateCallCount())
		require.Equal(t, 2, stateStub.DelStateCallCount())
	})

	// 3rd tx transaction will not be committed, because value of key 4 was changed in batch state
	t.Run("batch transaction 3", func(_ *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID3)

		val, _ := txStub.GetState(valKey4)
		if string(val) == "" {
			_ = txStub.PutState(valKey4, []byte(valKey4Value3))
			txStub.Commit()
		}

		// checking key 4 value was not changed, deleting key 4
		val, _ = batchStub.GetState(valKey4)
		require.Equal(t, valKey4Value2, string(val))

		_ = batchStub.DelState(valKey4)
		_ = batchStub.Commit()

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 7, stateStub.PutStateCallCount())
		require.Equal(t, 4, stateStub.DelStateCallCount())
	})
}
