package cachestub_test

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/stretchr/testify/require"
)

func TestTxStub(t *testing.T) {
	stub := newMockStub()
	_ = stub.PutState("KEY1", []byte("key1_value_1"))
	_ = stub.PutState("KEY2", []byte("key2_value_1"))
	_ = stub.PutState("KEY3", []byte("key3_value_1"))

	btchStub := cachestub.NewBatchCacheStub(stub)

	// transaction 1 changes value of key1 and deletes key2
	t.Run("tx1", func(t *testing.T) {
		txStub := btchStub.NewTxCacheStub("tx1")
		val, _ := txStub.GetState("KEY2")
		require.Equal(t, "key2_value_1", string(val))

		_ = txStub.PutState("KEY1", []byte("key1_value_2"))
		_ = txStub.DelState("KEY2")
		txStub.Commit()
	})

	// checking first transaction results were properly committed
	val1, _ := btchStub.GetState("KEY2")
	require.Equal(t, "", string(val1))

	val2, _ := btchStub.GetState("KEY1")
	require.Equal(t, "key1_value_2", string(val2))

	// transaction 2 changes value of the key2 and deletes key3
	t.Run("tx2", func(t *testing.T) {
		txStub := btchStub.NewTxCacheStub("tx1")
		val11, _ := txStub.GetState("KEY2")
		require.Equal(t, "", string(val11))

		val22, _ := txStub.GetState("KEY1")
		require.Equal(t, "key1_value_2", string(val22))

		_ = txStub.PutState("KEY2", []byte("key2_value_2"))
		_ = txStub.DelState("KEY3")
		txStub.Commit()
	})

	_ = btchStub.Commit()

	// checking state after batch commit
	require.Equal(t, 2, len(stub.state))
	require.Equal(t, "key1_value_2", string(stub.state["KEY1"]))
	require.Equal(t, "key2_value_2", string(stub.state["KEY2"]))

	// transaction 3 adds and deletes value for key 4
	t.Run("tx3", func(t *testing.T) {
		txStub := btchStub.NewTxCacheStub("tx2")
		_ = txStub.PutState("KEY4", []byte("key4_value_1"))
		_ = txStub.DelState("KEY4")
		txStub.Commit()
	})

	// btchStub checks if key 4 was deleted and changes its value
	val4, _ := btchStub.GetState("KEY4")
	require.Equal(t, "", string(val4))
	_ = btchStub.PutState("KEY4", []byte("key4_value_2"))

	_ = btchStub.Commit()

	require.Equal(t, "key4_value_2", string(stub.state["KEY4"]))

	// transaction 4 will not be committed, because value of key 4 was changed in batch state
	t.Run("tx4", func(t *testing.T) {
		txStub := btchStub.NewTxCacheStub("tx3")

		val, _ := txStub.GetState("KEY4")
		if string(val) == "" {
			_ = txStub.PutState("KEY4", []byte("key4_value_3"))
			txStub.Commit()
		}
	})

	// checking key 4 value was not changed, deleting key 4
	val5, _ := btchStub.GetState("KEY4")
	require.Equal(t, "key4_value_2", string(val5))

	_ = btchStub.DelState("KEY4")
	_ = btchStub.Commit()

	// checking state for key 4 was deleted
	require.Equal(t, 2, len(stub.state))
	_, ok := stub.state["KEY4"]
	require.Equal(t, false, ok)
}
