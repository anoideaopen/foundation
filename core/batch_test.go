package core

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	pb "google.golang.org/protobuf/proto"
)

const (
	testFnWithFiveArgsMethod = "testFnWithFiveArgsMethod"
	testFnWithSignedTwoArgs  = "testFnWithSignedTwoArgs"
)

var (
	testChaincodeName = "chaincode"

	argsForTestFnWithFive          = []string{"4aap@*", "hyexc566", "kiubfvr$", ";3vkpp", "g?otov;", "!djski", "gfgt^"}
	argsForTestFnWithSignedTwoArgs = []string{"1", "arg1"}

	sender = &proto.Address{
		UserID:  "UserId",
		Address: []byte("Address"),
	}

	txID            = "TestTxID"
	txIDBytes       = []byte(txID)
	testEncodedTxID = hex.EncodeToString(txIDBytes)
)

// chaincode for test batch method with signature and without signature
type testBatchContract struct {
	BaseContract
}

func (*testBatchContract) GetID() string {
	return "TEST"
}

func (*testBatchContract) TxTestFnWithFiveArgsMethod(_ string, _ string, _ string, _ string, _ string) error {
	return nil
}

// TxTestSignedFnWithArgs example function with a sender to check that the sender field will be omitted, and the argument setting starts with the 'val' parameter
// through this method we validate that arguments defined in method with sender *types.Sender validate in 'saveBatch' method correctly
func (*testBatchContract) TxTestFnWithSignedTwoArgs(_ *types.Sender, _ int64, _ string) error {
	return nil
}

type serieBatcheExecute struct {
	testIDBytes   []byte
	paramsWrongON bool
}

type serieBatches struct {
	FnName    string
	testID    string
	errorMsg  string
	timestamp *timestamp.Timestamp
}

// TestSaveToBatchWithWrongArgs - negative test with wrong Args in saveToBatch
func TestSaveToBatchWithWrongArgs(t *testing.T) {
	t.Parallel()

	s := &serieBatches{
		FnName:    testFnWithFiveArgsMethod,
		testID:    testEncodedTxID,
		errorMsg:  "",
		timestamp: createUtcTimestamp(),
	}

	wrongArgs := []string{"arg0", "arg1"}
	chainCode, errChainCode := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	errSave := chainCode.saveToBatch(mockStub, s.FnName, sender, wrongArgs, uint64(batchTimestamp.Seconds))
	assert.ErrorContains(t, errSave, "incorrect number of arguments, found 2 but expected more than 5")
}

// TestSaveToBatchWithSignedArgs - negative test with wrong Args in saveToBatch
func TestSaveToBatchWithSignedArgs(t *testing.T) {
	t.Parallel()
	s := &serieBatches{
		FnName:    testFnWithSignedTwoArgs,
		testID:    testEncodedTxID,
		errorMsg:  "",
		timestamp: createUtcTimestamp(),
	}

	chainCode, errChainCode := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, s.FnName, sender, argsForTestFnWithSignedTwoArgs, uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
}

// TestSaveToBatchWithWrongSignedArgs - negative test with wrong Args in saveToBatch
func TestSaveToBatchWithWrongSignedArgs(t *testing.T) {
	t.Parallel()

	s := &serieBatches{
		FnName:    testFnWithSignedTwoArgs,
		testID:    testEncodedTxID,
		errorMsg:  "",
		timestamp: createUtcTimestamp(),
	}

	wrongArgs := []string{"arg0", "arg1"}
	chainCode, errChainCode := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, s.FnName, sender, wrongArgs, uint64(batchTimestamp.Seconds))
	assert.EqualError(t, err, "validate arguments. strconv.ParseInt: parsing \"arg0\": invalid syntax")
}

// TestSaveAndLoadToBatchWithWrongFnParameter - negative test with wrong Fn Name in saveToBatch
func TestSaveToBatchWrongFnName(t *testing.T) {
	t.Parallel()

	s := &serieBatches{
		FnName:    "unknownFunctionName",
		testID:    testEncodedTxID,
		errorMsg:  "",
		timestamp: createUtcTimestamp(),
	}

	chainCode, errChainCode := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	errSave := chainCode.saveToBatch(mockStub, s.FnName, sender, argsForTestFnWithFive, uint64(batchTimestamp.Seconds))
	assert.ErrorContains(t, errSave, "method 'unknownFunctionName' not found")
}

// TestSaveAndLoadToBatchWithWrongID - negative test with wrong ID for loadToBatch
func TestSaveAndLoadToBatchWithWrongID(t *testing.T) {
	t.Parallel()

	s := &serieBatches{
		FnName:    testFnWithFiveArgsMethod,
		testID:    "wonder",
		errorMsg:  "transaction wonder not found",
		timestamp: createUtcTimestamp(),
	}

	SaveAndLoadToBatchTest(t, s, argsForTestFnWithFive)
}

// SaveAndLoadToBatchTest - basic test to check Args in saveToBatch and loadFromBatch
func SaveAndLoadToBatchTest(t *testing.T, ser *serieBatches, args []string) {
	chainCode, errChainCode := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	if ser.timestamp != nil {
		mockStub.TxTimestamp = ser.timestamp
	}

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	errSave := chainCode.saveToBatch(mockStub, ser.FnName, sender, args, uint64(batchTimestamp.Seconds))
	assert.NoError(t, errSave)
	mockStub.MockTransactionEnd(testEncodedTxID)
	state, err := mockStub.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
	assert.NotNil(t, state)
	assert.NoError(t, err)

	pending := new(proto.PendingTx)
	err = pb.Unmarshal(state, pending)
	assert.NoError(t, err)

	assert.Equal(t, pending.Args, args)

	pending, _, err = chainCode.loadFromBatch(mockStub, ser.testID, batchTimestamp.Seconds)
	if err != nil {
		assert.Equal(t, ser.errorMsg, err.Error())
	} else {
		assert.NoError(t, err)
		assert.Equal(t, pending.Method, ser.FnName)
		assert.Equal(t, pending.Args, args)
	}
}

// TestBatchExecuteWithRightParams - positive test for SaveBatch, LoadBatch and batchExecute
func TestBatchExecuteWithRightParams(t *testing.T) {
	t.Parallel()

	s := &serieBatcheExecute{
		testIDBytes:   txIDBytes,
		paramsWrongON: false,
	}

	resp := BatchExecuteTest(t, s, argsForTestFnWithFive)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.GetStatus(), int32(200))

	response := &proto.BatchResponse{}
	err := pb.Unmarshal(resp.GetPayload(), response)
	assert.NoError(t, err)

	assert.Len(t, response.TxResponses, 1)

	txResponse := response.TxResponses[0]
	assert.Equal(t, txResponse.Id, txIDBytes)
	assert.Equal(t, txResponse.Method, testFnWithFiveArgsMethod)
	assert.Nil(t, txResponse.Error)
}

// TestBatchExecuteWithWrongParams - negative test with wrong parameters in batchExecute
// Test must be failed, but it is passed
func TestBatchExecuteWithWrongParams(t *testing.T) {
	t.Parallel()

	testIDBytes := []byte("wonder")
	s := &serieBatcheExecute{
		testIDBytes:   testIDBytes,
		paramsWrongON: true,
	}

	resp := BatchExecuteTest(t, s, argsForTestFnWithFive)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.GetStatus(), int32(200))

	response := &proto.BatchResponse{}
	err := pb.Unmarshal(resp.GetPayload(), response)
	assert.NoError(t, err)

	assert.Len(t, response.TxResponses, 1)

	txResponse := response.TxResponses[0]
	assert.Equal(t, txResponse.Id, testIDBytes)
	assert.Equal(t, txResponse.Method, "")
	assert.Equal(t, txResponse.Error.Error, "function and args loading error: transaction 776f6e646572 not found")
}

// BatchExecuteTest - basic test for SaveBatch, LoadBatch and batchExecute
func BatchExecuteTest(t *testing.T, ser *serieBatcheExecute, args []string) peer.Response {
	chainCode, err := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, err)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, testFnWithFiveArgsMethod, nil, args, uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
	mockStub.MockTransactionEnd(testEncodedTxID)
	state, err := mockStub.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
	assert.NotNil(t, state)
	assert.NoError(t, err)

	pending := new(proto.PendingTx)
	err = pb.Unmarshal(state, pending)
	assert.NoError(t, err)

	assert.Equal(t, pending.Method, testFnWithFiveArgsMethod)
	assert.Equal(t, pending.Timestamp, batchTimestamp.Seconds)
	assert.Equal(t, pending.Args, args)

	dataIn, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{ser.testIDBytes}})
	assert.NoError(t, err)

	return chainCode.batchExecute(mockStub, string(dataIn), nil, nil)
}

// TestBatchedTxExecute - positive test for batchedTxExecute
func TestBatchedTxExecute(t *testing.T) {
	chainCode, err := NewCC(&testBatchContract{}, nil)
	assert.NoError(t, err)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	mockStub.TxID = testEncodedTxID

	btchStub := newBatchStub(mockStub)

	mockStub.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, testFnWithFiveArgsMethod, nil, argsForTestFnWithFive, uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
	mockStub.MockTransactionEnd(testEncodedTxID)

	resp, event := chainCode.batchedTxExecute(btchStub, txIDBytes, batchTimestamp.Seconds, nil, nil)
	assert.NotNil(t, resp)
	assert.NotNil(t, event)
	assert.Nil(t, resp.Error)
	assert.Nil(t, event.Error)
}

// TestOkTxExecuteWithTTL - positive test for batchedTxExecute whit ttl
func TestOkTxExecuteWithTTL(t *testing.T) {
	chainCode, err := NewCC(&testBatchContract{}, &ContractOptions{
		TxTTL: 5,
	})
	assert.NoError(t, err)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)
	mockStub.TxID = testEncodedTxID
	btchStub := newBatchStub(mockStub)
	mockStub.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, testFnWithFiveArgsMethod, nil, argsForTestFnWithFive, uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
	mockStub.MockTransactionEnd(testEncodedTxID)

	resp, event := chainCode.batchedTxExecute(btchStub, txIDBytes, batchTimestamp.Seconds, nil, nil)
	assert.NotNil(t, resp)
	assert.NotNil(t, event)
	assert.Nil(t, resp.Error)
	assert.Nil(t, event.Error)

	assert.Equal(t, resp.Id, txIDBytes)
	assert.Equal(t, resp.Method, testFnWithFiveArgsMethod)
	assert.Equal(t, event.Id, txIDBytes)
	assert.Equal(t, event.Method, testFnWithFiveArgsMethod)
}

// TestFalseTxExecuteWithTTL - negative test for batchedTxExecute whit ttl
func TestFailTxExecuteWithTTL(t *testing.T) {
	chainCode, err := NewCC(&testBatchContract{}, &ContractOptions{
		TxTTL: 5,
	})
	assert.NoError(t, err)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)
	mockStub.TxID = testEncodedTxID
	btchStub := newBatchStub(mockStub)
	mockStub.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	err = chainCode.saveToBatch(mockStub, testFnWithFiveArgsMethod, nil, argsForTestFnWithFive, uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
	mockStub.MockTransactionEnd(testEncodedTxID)

	resp, event := chainCode.batchedTxExecute(btchStub, txIDBytes, batchTimestamp.Seconds+6, nil, nil)
	assert.NotNil(t, resp)
	assert.NotNil(t, event)
	assert.NotNil(t, resp.Error)
	assert.NotNil(t, event.Error)
	assert.Equal(t, resp.Id, txIDBytes)
	assert.Equal(t, event.Id, txIDBytes)
	assert.Contains(t, resp.Error.Error, "function and args loading error: transaction expired")
	assert.Contains(t, event.Error.Error, "function and args loading error: transaction expired")
}

// CreateUtcTimestamp returns a google/protobuf/Timestamp in UTC
func createUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}
