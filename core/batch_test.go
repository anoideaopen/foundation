package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/cachestub"
	"github.com/anoideaopen/foundation/core/contract"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock/stub"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

const (
	testFnWithFiveArgsMethod = "testFnWithFiveArgsMethod"
	testFnWithSignedTwoArgs  = "testFnWithSignedTwoArgs"
)

var (
	testChaincodeName = "chaincode"

	argsForTestFnWithFive          = []string{"4aap@*", "hyexc566", "kiubfvr$", ";3vkpp", "g?otov;"}
	argsForTestFnWithSignedTwoArgs = []string{"1", "arg1"}

	sender = &proto.Address{
		UserID:  "UserID",
		Address: bytes.Repeat([]byte{0xF0}, 32),
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
	require.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
		},
	}
	config, _ := protojson.Marshal(cfgEtl)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{config})

	err := contract.Configure(chainCode.contract, mockStub, config)
	require.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	require.NoError(t, err)

	ep, err := chainCode.Method(s.FnName)
	require.NoError(t, err)

	errSave := chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		ep,
		sender,
		wrongArgs,
		uint64(batchTimestamp.Seconds),
	)
	require.EqualError(t, errSave, "incorrect number of arguments: found 2 but expected 5: validate TxTestFnWithFiveArgsMethod")
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
	require.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
		},
	}
	config, _ := protojson.Marshal(cfgEtl)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{config})

	err := contract.Configure(chainCode.contract, mockStub, config)
	require.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ep, err := chainCode.Method(s.FnName)
	require.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		ep,
		sender,
		argsForTestFnWithSignedTwoArgs,
		uint64(batchTimestamp.Seconds),
	)
	require.NoError(t, err)
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
	require.NoError(t, errChainCode)

	mockStub := stub.NewMockStub(testChaincodeName, chainCode)

	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
		},
	}
	config, _ := protojson.Marshal(cfgEtl)

	idBytes := [16]byte(uuid.New())
	mockStub.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{[]byte(config)})

	err := contract.Configure(chainCode.contract, mockStub, []byte(config))
	require.NoError(t, err)

	mockStub.TxID = testEncodedTxID
	mockStub.MockTransactionStart(testEncodedTxID)
	mockStub.TxTimestamp = s.timestamp

	batchTimestamp, err := mockStub.GetTxTimestamp()
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ep, err := chainCode.Method(s.FnName)
	require.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		mockStub,
		ep,
		sender,
		wrongArgs,
		uint64(batchTimestamp.Seconds),
	)
	require.EqualError(t, err, "invalid argument value: 'arg0': for type 'int64': validate TxTestFnWithSignedTwoArgs, argument 1")
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
	require.NoError(t, errChainCode)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)
	ms.TxTimestamp = s.timestamp

	_, err := ms.GetTxTimestamp()
	require.NoError(t, err)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
	}
	cfgBytes, _ := protojson.Marshal(cfg)

	err = contract.Configure(chainCode.contract, ms, cfgBytes)
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	_, err = chainCode.Method(s.FnName)
	require.ErrorContains(t, err, "method 'unknownFunctionName' not found")
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
	require.NoError(t, errChainCode)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
	}
	cfgBytes, _ := protojson.Marshal(cfg)

	err := ms.SetAdminCreatorCert("platformMSP")
	require.NoError(t, err)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	require.Equal(t, int32(shim.OK), rsp.GetStatus(), rsp.GetMessage())

	err = contract.Configure(chainCode.contract, ms, cfgBytes)
	require.NoError(t, err)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)
	if ser.timestamp != nil {
		ms.TxTimestamp = ser.timestamp
	}

	batchTimestamp, err := ms.GetTxTimestamp()
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ep, err := chainCode.Method(ser.FnName)
	require.NoError(t, err)

	errSave := chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		ep,
		sender,
		args,
		uint64(batchTimestamp.Seconds),
	)
	require.NoError(t, errSave)
	ms.MockTransactionEnd(testEncodedTxID)

	state, err := ms.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
	require.NotNil(t, state)
	require.NoError(t, err)

	pending := new(proto.PendingTx)
	err = pb.Unmarshal(state, pending)
	require.NoError(t, err)

	require.Equal(t, pending.Args, args)

	pending, _, err = chainCode.loadFromBatch(ms, ser.testID)
	if err != nil {
		require.Equal(t, ser.errorMsg, err.Error())
	} else {
		require.NoError(t, err)
		require.Equal(t, pending.Method, ser.FnName)
		require.Equal(t, pending.Args, args)
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
	require.NotNil(t, resp)
	require.Equal(t, resp.GetStatus(), int32(200))

	response := &proto.BatchResponse{}
	err := pb.Unmarshal(resp.GetPayload(), response)
	require.NoError(t, err)

	require.Len(t, response.TxResponses, 1)

	txResponse := response.TxResponses[0]
	require.Equal(t, txResponse.Id, txIDBytes)
	require.Equal(t, txResponse.Method, testFnWithFiveArgsMethod)
	require.Nil(t, txResponse.Error)
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
	require.NotNil(t, resp)
	require.Equal(t, resp.GetStatus(), int32(200))

	response := &proto.BatchResponse{}
	err := pb.Unmarshal(resp.GetPayload(), response)
	require.NoError(t, err)

	require.Len(t, response.TxResponses, 1)

	txResponse := response.TxResponses[0]
	require.Equal(t, txResponse.Id, testIDBytes)
	require.Equal(t, txResponse.Method, "")
	require.Equal(t, txResponse.Error.Error, "function and args loading error: transaction 776f6e646572 not found")
}

// BatchExecuteTest - basic test for SaveBatch, LoadBatch and batchExecute
func BatchExecuteTest(t *testing.T, ser *serieBatchExecute, args []string) peer.Response {
	chainCode, err := NewCC(&testBatchContract{})
	require.NoError(t, err)

	ms := stub.NewMockStub(testChaincodeName, chainCode)

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "TT",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
	}
	cfgBytes, _ := protojson.Marshal(cfg)

	err = ms.SetAdminCreatorCert("platformMSP")
	require.NoError(t, err)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	require.Equal(t, int32(shim.OK), rsp.GetStatus())

	err = contract.Configure(chainCode.contract, ms, cfgBytes)
	require.NoError(t, err)

	ms.TxID = testEncodedTxID
	ms.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := ms.GetTxTimestamp()
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ep, err := chainCode.Method(testFnWithFiveArgsMethod)
	require.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		ep,
		nil,
		args,
		uint64(batchTimestamp.Seconds),
	)
	require.NoError(t, err)
	ms.MockTransactionEnd(testEncodedTxID)
	state, err := ms.GetState(fmt.Sprintf("\u0000batchTransactions\u0000%s\u0000", testEncodedTxID))
	require.NotNil(t, state)
	require.NoError(t, err)

	pending := new(proto.PendingTx)
	err = pb.Unmarshal(state, pending)
	require.NoError(t, err)

	require.Equal(t, pending.Method, testFnWithFiveArgsMethod)
	require.Equal(t, pending.Timestamp, batchTimestamp.Seconds)
	require.Equal(t, pending.Args, args)

	dataIn, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{ser.testIDBytes}})
	require.NoError(t, err)

	return chainCode.batchExecute(telemetry.TraceContext{}, ms, string(dataIn))
}

// TestBatchedTxExecute tests positive test for batchedTxExecute
func TestBatchedTxExecute(t *testing.T) {
	chainCode, err := NewCC(&testBatchContract{})
	require.NoError(t, err)

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

	cfgBytes, _ := protojson.Marshal(cfg)

	idBytes := [16]byte(uuid.New())
	rsp := ms.MockInit(hex.EncodeToString(idBytes[:]), [][]byte{cfgBytes})
	require.Equal(t, int32(shim.OK), rsp.GetStatus())

	err = contract.Configure(chainCode.contract, ms, cfgBytes)
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ms.TxID = testEncodedTxID

	btchStub := cachestub.NewBatchCacheStub(ms)

	ms.MockTransactionStart(testEncodedTxID)

	batchTimestamp, err := ms.GetTxTimestamp()
	require.NoError(t, err)

	_, err = buildRouter(chainCode.contract)
	require.NoError(t, err)

	ep, err := chainCode.Method(testFnWithFiveArgsMethod)
	require.NoError(t, err)

	err = chainCode.saveToBatch(
		telemetry.TraceContext{},
		ms,
		ep,
		nil,
		argsForTestFnWithFive,
		uint64(batchTimestamp.Seconds))
	require.NoError(t, err)
	ms.MockTransactionEnd(testEncodedTxID)

	resp, event := chainCode.batchedTxExecute(
		telemetry.TraceContext{},
		btchStub,
		txIDBytes,
	)
	require.NotNil(t, resp)
	require.NotNil(t, event)
	require.Nil(t, resp.Error)
	require.Nil(t, event.Error)
}

// CreateUtcTimestamp returns a Google/protobuf/Timestamp in UTC
func createUtcTimestamp() *timestamp.Timestamp {
	now := time.Now().UTC()
	secs := now.Unix()
	nanos := int32(now.UnixNano() - (secs * 1000000000))
	return &(timestamp.Timestamp{Seconds: secs, Nanos: nanos})
}
