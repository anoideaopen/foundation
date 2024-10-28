package cachestub

import (
	"github.com/anoideaopen/foundation/core/cachestub/mock"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	txID1 = "txID1"
	txID2 = "txID2"
	txID3 = "txID3"
)

func TestTxStub(t *testing.T) {
	stateStub := mock.NewStateStub()

	// preparing cacheStub values
	_ = stateStub.PutState(valKey1, []byte(valKey1Value1))
	_ = stateStub.PutState(valKey2, []byte(valKey2Value1))
	_ = stateStub.PutState(valKey3, []byte(valKey3Value1))

	batchStub := NewBatchCacheStub(stateStub)

	// transaction 1 changes value of key1 and deletes key2
	t.Run("tx1", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID1)
		val, _ := txStub.GetState(valKey2)
		require.Equal(t, valKey2Value1, string(val))

		_ = txStub.PutState(valKey1, []byte(valKey1Value2))
		_ = txStub.DelState(valKey2)
		txStub.Commit()
	})

	// checking first transaction results were properly committed
	val1, _ := batchStub.GetState(valKey2)
	require.Equal(t, "", string(val1))

	val2, _ := batchStub.GetState(valKey1)
	require.Equal(t, valKey1Value2, string(val2))

	// transaction 2 changes value of the key2 and deletes key3
	t.Run("tx2", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID1)
		val11, _ := txStub.GetState(valKey2)
		require.Equal(t, "", string(val11))

		val22, _ := txStub.GetState(valKey1)
		require.Equal(t, valKey1Value2, string(val22))

		_ = txStub.PutState(valKey2, []byte(valKey2Value2))
		_ = txStub.DelState(valKey3)
		txStub.Commit()
	})

	_ = batchStub.Commit()

	// checking state after batch commit
	require.Equal(t, 2, len(stateStub.State))
	require.Equal(t, valKey1Value2, string(stateStub.State[valKey1]))
	require.Equal(t, valKey2Value2, string(stateStub.State[valKey2]))

	// transaction 3 adds and deletes value for key 4
	t.Run("tx3", func(_ *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID2)
		_ = txStub.PutState(valKey4, []byte(valKey4Value1))
		_ = txStub.DelState(valKey4)
		txStub.Commit()
	})

	// batchStub checks if key 4 was deleted and changes its value
	val4, _ := batchStub.GetState(valKey4)
	require.Equal(t, "", string(val4))
	_ = batchStub.PutState(valKey4, []byte(valKey4Value2))

	_ = batchStub.Commit()

	require.Equal(t, valKey4Value2, string(stateStub.State[valKey4]))

	// transaction 4 will not be committed, because value of key 4 was changed in batch state
	t.Run("tx4", func(_ *testing.T) {
		txStub := batchStub.NewTxCacheStub(txID3)

		val, _ := txStub.GetState(valKey4)
		if string(val) == "" {
			_ = txStub.PutState(valKey4, []byte(valKey4Value3))
			txStub.Commit()
		}
	})

	// checking key 4 value was not changed, deleting key 4
	val5, _ := batchStub.GetState(valKey4)
	require.Equal(t, valKey4Value2, string(val5))

	_ = batchStub.DelState(valKey4)
	_ = batchStub.Commit()

	// checking state for key 4 was deleted
	require.Equal(t, 2, len(stateStub.State))
	_, ok := stateStub.State[valKey4]
	require.Equal(t, false, ok)
}
