package cachestub_test

import (
	"testing"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/stretchr/testify/require"
)

func TestBatchStub(t *testing.T) {
	mockStub := newMockStub()
	_ = mockStub.PutState("KEY1", []byte("key1_value_1"))
	_ = mockStub.PutState("KEY2", []byte("key2_value_1"))
	_ = mockStub.PutState("KEY3", []byte("key3_value_1"))

	batchStub := cachestub.NewBatchCacheStub(mockStub)

	_ = batchStub.PutState("KEY1", []byte("key1_value_2"))
	_ = batchStub.DelState("KEY2")
	_ = batchStub.Commit()

	val1, _ := batchStub.GetState("KEY2")
	require.Equal(t, "", string(val1))

	val2, _ := batchStub.GetState("KEY1")
	require.Equal(t, "key1_value_2", string(val2))

	_ = batchStub.PutState("KEY2", []byte("key2_value_2"))
	_ = batchStub.DelState("KEY3")
	_ = batchStub.Commit()

	require.Equal(t, 2, len(mockStub.state))
	require.Equal(t, "key1_value_2", string(mockStub.state["KEY1"]))
	require.Equal(t, "key2_value_2", string(mockStub.state["KEY2"]))

	_ = batchStub.PutState("KEY4", []byte("key4_value_1"))
	_ = batchStub.DelState("KEY4")

	_ = batchStub.Commit()

	_, ok := mockStub.state["KEY4"]
	require.Equal(t, false, ok)
}
