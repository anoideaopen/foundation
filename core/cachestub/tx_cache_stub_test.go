package cachestub_test

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/stretchr/testify/require"
)

func TestTxStub(t *testing.T) {
	mockStub := newMockStub()
	_ = mockStub.PutState("KEY1", []byte("key1_value_1"))
	_ = mockStub.PutState("KEY2", []byte("key2_value_1"))
	_ = mockStub.PutState("KEY3", []byte("key3_value_1"))

	batchStub := cachestub.NewBatchCacheStub(mockStub)

	// transaction 1 changes value of key1 and deletes key2
	t.Run("tx1", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub("tx1")
		val, _ := txStub.GetState("KEY2")
		require.Equal(t, "key2_value_1", string(val))

		_ = txStub.PutState("KEY1", []byte("key1_value_2"))
		_ = txStub.DelState("KEY2")
		txStub.Commit()
	})

	// checking first transaction results were properly committed
	val1, _ := batchStub.GetState("KEY2")
	require.Equal(t, "", string(val1))

	val2, _ := batchStub.GetState("KEY1")
	require.Equal(t, "key1_value_2", string(val2))

	// transaction 2 changes value of the key2 and deletes key3
	t.Run("tx2", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub("tx1")
		val11, _ := txStub.GetState("KEY2")
		require.Equal(t, "", string(val11))

		val22, _ := txStub.GetState("KEY1")
		require.Equal(t, "key1_value_2", string(val22))

		_ = txStub.PutState("KEY2", []byte("key2_value_2"))
		_ = txStub.DelState("KEY3")
		txStub.Commit()
	})

	_ = batchStub.Commit()

	// checking state after batch commit
	require.Equal(t, 2, len(mockStub.state))
	require.Equal(t, "key1_value_2", string(mockStub.state["KEY1"]))
	require.Equal(t, "key2_value_2", string(mockStub.state["KEY2"]))

	// transaction 3 adds and deletes value for key 4
	t.Run("tx3", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub("tx2")
		_ = txStub.PutState("KEY4", []byte("key4_value_1"))
		_ = txStub.DelState("KEY4")
		txStub.Commit()
	})

	// batchStub checks if key 4 was deleted and changes its value
	val4, _ := batchStub.GetState("KEY4")
	require.Equal(t, "", string(val4))
	_ = batchStub.PutState("KEY4", []byte("key4_value_2"))

	_ = batchStub.Commit()

	require.Equal(t, "key4_value_2", string(mockStub.state["KEY4"]))

	// transaction 4 will not be committed, because value of key 4 was changed in batch state
	t.Run("tx4", func(t *testing.T) {
		txStub := batchStub.NewTxCacheStub("tx3")

		val, _ := txStub.GetState("KEY4")
		if string(val) == "" {
			_ = txStub.PutState("KEY4", []byte("key4_value_3"))
			txStub.Commit()
		}
	})

	// checking key 4 value was not changed, deleting key 4
	val5, _ := batchStub.GetState("KEY4")
	require.Equal(t, "key4_value_2", string(val5))

	_ = batchStub.DelState("KEY4")
	_ = batchStub.Commit()

	// checking state for key 4 was deleted
	require.Equal(t, 2, len(mockStub.state))
	_, ok := mockStub.state["KEY4"]
	require.Equal(t, false, ok)
}
