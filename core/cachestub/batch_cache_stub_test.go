package cachestub

import (
	"github.com/anoideaopen/foundation/core/cachestub/mock"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	valKey1 = "KEY1"
	valKey2 = "KEY2"
	valKey3 = "KEY3"
	valKey4 = "KEY4"

	valKey1Value1 = "key1_value1"
	valKey2Value1 = "key2_value1"
	valKey3Value1 = "key3_value1"
	valKey4Value1 = "key4_value1"

	valKey1Value2 = "key1_value2"
	valKey2Value2 = "key2_value2"
	valKey4Value2 = "key4_value2"

	valKey4Value3 = "key4_value3"
)

func TestBatchStub(t *testing.T) {
	stateStub := mock.NewStateStub()

	// preparing cacheStub values
	_ = stateStub.PutState(valKey1, []byte(valKey1Value1))
	_ = stateStub.PutState(valKey2, []byte(valKey2Value1))
	_ = stateStub.PutState(valKey3, []byte(valKey3Value1))

	// creating batch cache stub
	batchStub := NewBatchCacheStub(stateStub)

	// changing key1 value
	_ = batchStub.PutState(valKey1, []byte(valKey1Value2))
	// deleting key2 value
	_ = batchStub.DelState(valKey2)
	// committing changes to mockStub
	_ = batchStub.Commit()

	// checking key2 value is deleted
	val1, _ := batchStub.GetState(valKey2)
	require.Equal(t, "", string(val1))

	// checking key1 value changed
	val2, _ := batchStub.GetState(valKey1)
	require.Equal(t, valKey1Value2, string(val2))

	// setting key2 value 2
	_ = batchStub.PutState(valKey2, []byte(valKey2Value2))
	// deleting key3 value
	_ = batchStub.DelState(valKey3)
	// committing changes to mock stub
	_ = batchStub.Commit()

	// checking mock stub state length
	require.Equal(t, 2, len(stateStub.State))
	// checking mock stub key1 value
	require.Equal(t, valKey1Value2, string(stateStub.State[valKey1]))
	// checking mock stub key2 value
	require.Equal(t, valKey2Value2, string(stateStub.State[valKey2]))

	// adding key4 value
	_ = batchStub.PutState(valKey4, []byte(valKey4Value1))
	// deleting key4 value
	_ = batchStub.DelState(valKey4)
	// committing changes to mock stub
	_ = batchStub.Commit()

	_, ok := stateStub.State[valKey4]
	require.Equal(t, false, ok)
}
