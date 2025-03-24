package cachestub

import (
	"fmt"
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
	valKey1Value2 = "key1_value2"
	valKey2Value1 = "key2_value1"
	valKey4Value1 = "key4_value1"
)

var testError = fmt.Errorf("test error")

func TestBatchStub(t *testing.T) {
	t.Run("GetState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		stateStub.GetStateReturns([]byte(valKey1Value1), nil)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// requesting data from state
		result, err := batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value1), result)

		// checking mock stub calls
		require.Equal(t, 1, stateStub.GetStateCallCount())
	})

	t.Run("[negative] GetState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		stateStub.GetStateReturns(nil, testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// requesting data from state
		_, err := batchStub.GetState(valKey1)
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.GetStateCallCount())
	})

	t.Run("PutState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// sending data to state
		err := batchStub.PutState(valKey1, []byte(valKey1Value1))
		require.NoError(t, err)

		err = batchStub.PutState(valKey1, []byte(valKey1Value2))
		require.NoError(t, err)

		err = batchStub.PutState(valKey2, []byte(valKey2Value1))
		require.NoError(t, err)

		// checking added data
		result, err := batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value2), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		// checking mock stub calls
		require.Equal(t, 0, stateStub.PutStateCallCount())

		// committing batch stub data
		err = batchStub.Commit()
		require.NoError(t, err)

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 2, stateStub.PutStateCallCount())
	})

	t.Run("[negative] PutState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		stateStub.PutStateReturns(testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		err := batchStub.PutState(valKey1, []byte(valKey1Value1))
		require.NoError(t, err)

		err = batchStub.Commit()
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.PutStateCallCount())
	})

	t.Run("DelState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// deleting data from state
		err := batchStub.DelState(valKey1)
		require.NoError(t, err)

		err = batchStub.DelState(valKey1)
		require.NoError(t, err)

		err = batchStub.DelState(valKey2)
		require.NoError(t, err)

		// checking data deleted
		result, err := batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// checking mock stub calls
		require.Equal(t, 0, stateStub.DelStateCallCount())

		// committing batch stub data
		err = batchStub.Commit()
		require.NoError(t, err)

		// checking mock stub calls
		require.Equal(t, 0, stateStub.GetStateCallCount())
		require.Equal(t, 2, stateStub.DelStateCallCount())
	})

	t.Run("[negative] DelState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		stateStub.DelStateReturns(testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// deleting data from state
		err := batchStub.DelState(valKey1)
		require.NoError(t, err)

		err = batchStub.Commit()
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.DelStateCallCount())
	})

	t.Run("Mixed test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}

		// preparing state stub
		stateStub.GetStateReturnsOnCall(0, []byte(valKey1Value1), nil)
		stateStub.GetStateReturnsOnCall(1, []byte(valKey2Value1), nil)
		stateStub.GetStateReturnsOnCall(2, []byte(nil), nil)
		stateStub.GetStateReturnsOnCall(3, []byte(valKey4Value1), nil)

		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)

		// requesting data from state
		result, err := batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value1), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		result, err = batchStub.GetState(valKey3)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey4)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey4Value1), result)

		// changing data in key 1
		err = batchStub.PutState(valKey1, []byte(valKey1Value2))
		require.NoError(t, err)

		// deleting data in key 2
		err = batchStub.DelState(valKey2)
		require.NoError(t, err)

		// adding and deleting data in key 4
		err = batchStub.PutState(valKey4, []byte(valKey4Value2))
		require.NoError(t, err)

		err = batchStub.DelState(valKey4)
		require.NoError(t, err)

		// checking changed data before commit
		result, err = batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value2), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey3)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey4)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		err = batchStub.Commit()

		// checking data after commit
		result, err = batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value2), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey3)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey4)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// checking mock stub calls
		require.Equal(t, 4, stateStub.GetStateCallCount())
		require.Equal(t, 1, stateStub.PutStateCallCount())
		require.Equal(t, 2, stateStub.DelStateCallCount())
	})
}
