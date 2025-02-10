package cachestub

import (
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
)

const (
	txID1 = "txID1"
	txID2 = "txID2"
	txID3 = "txID3"

	valKey1Value3 = "key1_value3"
	valKey3Value1 = "key3_value1"
	valKey4Value2 = "key4_value2"
	valKey4Value3 = "key4_value3"
)

func TestTxStub(t *testing.T) {

	t.Run("GetState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing state stub values
		stateStub.GetStateReturns([]byte(valKey1Value1), nil)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// requesting data from state
		result, err := txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value1, string(result))

		// checking mock stub calls
		require.Equal(t, 1, stateStub.GetStateCallCount())
	})

	t.Run("[negative] GetState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing error response
		stateStub.GetStateReturns(nil, testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// requesting data from state
		_, err := txStub.GetState(valKey1)
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.GetStateCallCount())
	})

	t.Run("PutState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing state data
		stateStub.GetStateReturnsOnCall(0, []byte(valKey1Value1), nil)
		stateStub.GetStateReturnsOnCall(1, []byte(nil), nil)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// checking previously saved data
		result, err := txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value1, string(result))

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// checking previously saved data again
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value1, string(result))

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// sending data to state
		err = txStub.PutState(valKey1, []byte(valKey1Value2))
		require.NoError(t, err)

		err = txStub.PutState(valKey1, []byte(valKey1Value3))
		require.NoError(t, err)

		err = txStub.PutState(valKey2, []byte(valKey2Value1))
		require.NoError(t, err)

		// checking tx stub result before commit
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value3, string(result))

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, valKey2Value1, string(result))

		txStub.Commit()

		// checking tx stub result after commit
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value3, string(result))

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, valKey2Value1, string(result))

		// checking batch stub before commit
		result, err = batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value3), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		// checking mock stub calls
		require.Equal(t, 0, stateStub.PutStateCallCount())

		// committing batch stub data
		err = batchStub.Commit()
		require.NoError(t, err)

		// checking batch stub after commit
		result, err = batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value3), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		// checking mock stub calls
		require.Equal(t, 2, stateStub.GetStateCallCount())
		require.Equal(t, 2, stateStub.PutStateCallCount())
	})

	t.Run("[negative] PutState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing error response
		stateStub.PutStateReturns(testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		err := txStub.PutState(valKey1, []byte(valKey1Value1))
		require.NoError(t, err)

		txStub.Commit()

		err = batchStub.Commit()
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.PutStateCallCount())
	})

	t.Run("DelState test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing data for deletion
		stateStub.GetStateReturnsOnCall(0, []byte(valKey1Value1), nil)
		stateStub.GetStateReturnsOnCall(1, []byte(valKey2Value1), nil)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// checking data before deletion
		result, err := txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value1), result)

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		// checking data one more time
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey1Value1), result)

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		// deleting data from state
		err = txStub.DelState(valKey1)
		require.NoError(t, err)

		err = txStub.DelState(valKey1)
		require.NoError(t, err)

		err = txStub.DelState(valKey2)
		require.NoError(t, err)

		// checking tx stub data before commit
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		txStub.Commit()

		// checking tx stub data after commit
		result, err = txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// checking batch stub before commit
		result, err = batchStub.GetState(valKey1)
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

		// checking batch stub after commit
		result, err = batchStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		result, err = batchStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		// checking mock stub calls
		require.Equal(t, 2, stateStub.GetStateCallCount())
		require.Equal(t, 2, stateStub.DelStateCallCount())
	})

	t.Run("[negative] DelState error test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		stateStub.DelStateReturns(testError)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// deleting data from tx stub
		err := txStub.DelState(valKey1)
		require.NoError(t, err)

		txStub.Commit()

		// committing changes
		err = batchStub.Commit()
		require.Errorf(t, err, testError.Error())

		// checking mock stub calls
		require.Equal(t, 1, stateStub.DelStateCallCount())
	})

	t.Run("Mixed test", func(t *testing.T) {
		stateStub := &mocks.ChaincodeStub{}
		// preparing test data
		stateStub.GetStateReturnsOnCall(0, []byte(valKey1Value1), nil)
		stateStub.GetStateReturnsOnCall(1, []byte(valKey2Value1), nil)
		stateStub.GetStateReturnsOnCall(2, []byte(valKey3Value1), nil)
		stateStub.GetStateReturnsOnCall(3, []byte(nil), nil)
		// creating batch cache stub
		batchStub := NewBatchCacheStub(stateStub)
		// creating tx cache stub
		txTime := createUtcTimestamp()
		txStub := batchStub.NewTxCacheStub(txID1, txTime)

		// checking test data
		result, err := txStub.GetState(valKey1)
		require.NoError(t, err)
		require.Equal(t, valKey1Value1, string(result))

		result, err = txStub.GetState(valKey2)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey2Value1), result)

		result, err = txStub.GetState(valKey3)
		require.NoError(t, err)
		require.Equal(t, []byte(valKey3Value1), result)

		result, err = batchStub.GetState(valKey4)
		require.NoError(t, err)
		require.Equal(t, []byte(nil), result)

		_ = txStub.PutState(valKey1, []byte(valKey1Value2))
		_ = txStub.DelState(valKey2)
		txStub.Commit()

		// checking first transaction results were properly committed
		val, _ := batchStub.GetState(valKey2)
		require.Equal(t, "", string(val))

		val, _ = batchStub.GetState(valKey1)
		require.Equal(t, valKey1Value2, string(val))

		// 2nd iteration of 1st tx stub changes
		_ = txStub.PutState(valKey2, []byte(valKey2Value1))
		_ = txStub.DelState(valKey3)
		txStub.Commit()

		// creating second transaction in batch
		txStub = batchStub.NewTxCacheStub(txID2, txTime)
		_ = txStub.PutState(valKey4, []byte(valKey4Value1))
		_ = txStub.DelState(valKey4)
		txStub.Commit()

		// batchStub checks if key 4 was deleted and changes its value
		val, _ = batchStub.GetState(valKey4)
		require.Equal(t, "", string(val))
		_ = batchStub.PutState(valKey4, []byte(valKey4Value2))

		_ = batchStub.Commit()

		// creating third transaction in batch
		txStub = batchStub.NewTxCacheStub(txID3, txTime)

		val, _ = txStub.GetState(valKey4)
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
		require.Equal(t, 4, stateStub.GetStateCallCount())
		require.Equal(t, 5, stateStub.PutStateCallCount())
		require.Equal(t, 3, stateStub.DelStateCallCount())
	})
}

func TestAccountingRecord(t *testing.T) {
	cases := []struct {
		name            string
		amount          *big.Int
		expectedRecords int
	}{
		{
			name:            "accounting record with amount",
			amount:          big.NewInt(1),
			expectedRecords: 1,
		},
		{
			name:            "accounting record with zero amount",
			amount:          big.NewInt(0),
			expectedRecords: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			const (
				tokenName = "testToken"
				reason    = "testReason"
			)

			stateStub := &mocks.ChaincodeStub{}
			// creating batch cache stub
			batchStub := NewBatchCacheStub(stateStub)
			// creating tx cache stub
			txTime := createUtcTimestamp()
			txStub := batchStub.NewTxCacheStub(txID1, txTime)

			txStub.AddAccountingRecord(
				tokenName,
				&types.Address{},
				&types.Address{},
				c.amount,
				balance.BalanceTypeToken,
				balance.BalanceTypeToken,
				reason,
			)

			require.Equal(t, c.expectedRecords, len(txStub.Accounting))
		})
	}
}

// CreateUtcTimestamp returns a Google/protobuf/Timestamp in UTC
func createUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}
