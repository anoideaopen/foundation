package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type serieBatchExecute struct {
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
	chainCode, errChainCode := NewCC(&testBatchContract{})
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	config := fmt.Sprintf(
		`
{
	"contract": {
		"robotSKI":"%s",
		"symbol": "CC"
	}
}`,
		fixtures_test.RobotHashedCert,
	)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{[]byte(config)})

	err := applyConfig(&chainCode.contract, mockStub, []byte(config))
	assert.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(s.FnName)
	assert.NoError(t, err)

	errSave := chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		s.FnName,
		fn,
		sender,
		wrongArgs,
		uint64(batchTimestamp.Seconds),
	)
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

	chainCode, errChainCode := NewCC(&testBatchContract{})
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	config := fmt.Sprintf(
		`
{
	"contract": {
		"robotSKI":"%s",
		"symbol": "CC"
	}
}`,
		fixtures_test.RobotHashedCert,
	)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{[]byte(config)})

	err := applyConfig(&chainCode.contract, mockStub, []byte(config))
	assert.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(s.FnName)
	assert.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		s.FnName,
		fn,
		sender,
		argsForTestFnWithSignedTwoArgs,
		uint64(batchTimestamp.Seconds),
	)
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
	chainCode, errChainCode := NewCC(&testBatchContract{})
	assert.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	config := fmt.Sprintf(
		`
{
	"contract": {
		"robotSKI":"%s",
		"symbol": "CC"
	}
}`,
		fixtures_test.RobotHashedCert,
	)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{[]byte(config)})

	err := applyConfig(&chainCode.contract, mockStub, []byte(config))
	assert.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(s.FnName)
	assert.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		s.FnName,
		fn,
		sender,
		wrongArgs,
		uint64(batchTimestamp.Seconds),
	)
	assert.EqualError(t, err, "validate arguments. failed to convert arg value 'arg0' "+
		"to type '<int64 Value>' on index '0': strconv.ParseInt: parsing \"arg0\": invalid syntax")
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

	chainCode, errChainCode := NewCC(&testBatchContract{})
	assert.NoError(t, errChainCode)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)
	ms.TxTimestamp = s.timestamp

	_, err := ms.GetTxTimestamp()
	assert.NoError(t, err)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
	}
	cfgBytes, _ := json.Marshal(cfg)

	err = applyConfig(&chainCode.contract, ms, cfgBytes)
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(s.FnName)
	assert.ErrorContains(t, err, "method 'unknownFunctionName' not found")
	assert.Nil(t, fn)
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
	chainCode, errChainCode := NewCC(&testBatchContract{})
	assert.NoError(t, errChainCode)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
	}
	cfgBytes, _ := json.Marshal(cfg)

	err := ms.SetAdminCreatorCert("platformMSP")
	assert.NoError(t, err)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	require.Equal(t, int32(shim.OK), rsp.GetStatus(), rsp.GetMessage())

	err = applyConfig(&chainCode.contract, ms, cfgBytes)
	assert.NoError(t, err)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)
	if ser.timestamp != nil {
		ms.TxTimestamp = ser.timestamp
	}

	batchTimestamp, err := ms.GetTxTimestamp()
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(ser.FnName)
	assert.NoError(t, err)

	errSave := chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		ser.FnName,
		fn,
		sender,
		args,
		uint64(batchTimestamp.Seconds),
	)
	assert.NoError(t, errSave)
	ms.MockTransactionEnd(testEncodedTxID)

	state, err := ms.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
	assert.NotNil(t, state)
	assert.NoError(t, err)

	pending := new(proto.PendingTx)
	err = pb.Unmarshal(state, pending)
	assert.NoError(t, err)

	assert.Equal(t, pending.Args, args)

	pending, _, err = chainCode.loadFromBatch(ms, ser.testID)
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

	s := &serieBatchExecute{
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
	s := &serieBatchExecute{
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
func BatchExecuteTest(t *testing.T, ser *serieBatchExecute, args []string) peer.Response {
	chainCode, err := NewCC(&testBatchContract{})
	assert.NoError(t, err)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "TT",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
	}
	cfgBytes, _ := json.Marshal(cfg)

	err = ms.SetAdminCreatorCert("platformMSP")
	require.NoError(t, err)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	assert.Equal(t, int32(shim.OK), rsp.GetStatus())

	err = applyConfig(&chainCode.contract, ms, cfgBytes)
	assert.NoError(t, err)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := ms.GetTxTimestamp()
	assert.NoError(t, err)

	methods, err := parseContractMethods(chainCode.contract)
	assert.NoError(t, err)
	chainCode.methods = methods
	assert.NoError(t, err)

	method, err := chainCode.methods.Method(testFnWithFiveArgsMethod)
	assert.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		testFnWithFiveArgsMethod,
		method,
		nil,
		args,
		uint64(batchTimestamp.Seconds),
	)
	assert.NoError(t, err)
	ms.MockTransactionEnd(testEncodedTxID)
	state, err := ms.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
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

	return chainCode.batchExecute(telemetry.TraceContext{}, ms, string(dataIn), nil)
}

// TestBatchedTxExecute tests positive test for batchedTxExecute
func TestBatchedTxExecute(t *testing.T) {
	chainCode, err := NewCC(&testBatchContract{})
	assert.NoError(t, err)

	ms := stub.NewMockStub(testChaincodeName, chainCode)
	require.NotNil(t, ms)

	err = ms.SetAdminCreatorCert("platformMSP")
	require.NoError(t, err)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			Options:  &proto.ChaincodeOptions{},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
	}

	cfgBytes, _ := json.Marshal(cfg)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	require.Equal(t, int32(shim.OK), rsp.GetStatus())

	err = applyConfig(&chainCode.contract, ms, cfgBytes)
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	require.NoError(t, err)

	ms.TxID = testEncodedTxID

	btchStub := newBatchStub(ms)

	ms.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := ms.GetTxTimestamp()
	assert.NoError(t, err)

	chainCode.methods, err = parseContractMethods(chainCode.contract)
	assert.NoError(t, err)

	fn, err := chainCode.methods.Method(testFnWithFiveArgsMethod)
	assert.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		testFnWithFiveArgsMethod,
		fn,
		nil,
		argsForTestFnWithFive,
		uint64(batchTimestamp.Seconds))
	assert.NoError(t, err)
	ms.MockTransactionEnd(testEncodedTxID)

	resp, event := chainCode.batchedTxExecute(
		telemetry.TraceContext{},
		btchStub,
		txIDBytes,
		nil,
	)
	assert.NotNil(t, resp)
	assert.NotNil(t, event)
	assert.Nil(t, resp.Error)
	assert.Nil(t, event.Error)
}

// CreateUtcTimestamp returns a Google/protobuf/Timestamp in UTC
func createUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}
