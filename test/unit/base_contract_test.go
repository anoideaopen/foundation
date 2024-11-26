package unit

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	pb "google.golang.org/protobuf/proto"
)

const (
	testTokenName      = "Testing Token"
	testTokenSymbol    = "TT"
	testTokenCCName    = "tt"
	testTokenWithGroup = "tt_testGroup"
	testGroup          = "testGroup"

	testMessageEmptyNonce = "\"0\""

	testGetNonceFnName      = "getNonce"
	testHelloWorldFnName    = "helloWorld"
	testHelloWorldSetFnName = "helloWorldSet"
)

type TestToken struct {
	token.BaseToken
}

type TestStruct struct {
	hello string
}

func (s *TestStruct) EncodeToBytes() ([]byte, error) {
	if s.hello == "" {
		s.hello = "Hello World"
	}

	return []byte(s.hello), nil
}

func (s *TestStruct) DecodeFromBytes(in []byte) error {
	if string(in) == "" {
		return errors.New("hello world should not be empty")
	}

	s.hello = string(in)
	return nil
}

func (tt *TestToken) QueryHelloWorld() (*TestStruct, error) {
	return &TestStruct{}, nil
}

func (tt *TestToken) TxHelloWorldSet(in *TestStruct) error {
	return nil
}

func (tt *TestToken) TxTestCall() error {
	traceCtx := tt.GetTraceContext()
	_, span := tt.TracingHandler().StartNewSpan(traceCtx, "TxTestCall()")
	defer span.End()

	return nil
}

func (tt *TestToken) TxFailedTestCall() error {
	traceCtx := tt.GetTraceContext()
	_, span := tt.TracingHandler().StartNewSpan(traceCtx, "TxTestCall()")
	defer span.End()

	return errors.New("ALARM")
}

func (tt *TestToken) TxEmissionAdd(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}
	if err := tt.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return tt.EmissionAdd(amount)
}

func TestBytesEncoder(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("Get bytes encoded response", func(t *testing.T) {
		mockStub.GetStateReturns([]byte(config), nil)
		mockStub.GetFunctionAndParametersReturns(testHelloWorldFnName, []string{})
		resp := cc.Invoke(mockStub)
		require.Equal(t, resp.GetPayload(), []byte("Hello World"))
	})
}

func TestBytesDecoder(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("Set bytes encoded response", func(t *testing.T) {
		mockStub.GetStateReturns([]byte(config), nil)
		mockStub.GetFunctionAndParametersReturns(testHelloWorldSetFnName, []string{"Hi!"})
		resp := cc.Invoke(mockStub)
		require.Empty(t, resp.GetMessage())
	})

	t.Run("Set bytes encoded response", func(t *testing.T) {
		mockStub.GetStateReturns([]byte(config), nil)
		mockStub.GetFunctionAndParametersReturns(testHelloWorldSetFnName, []string{""})
		resp := cc.Invoke(mockStub)
		require.NotEmpty(t, resp.GetMessage())
	})
}

// TestGetEmptyNonce - Checking that new wallet have empty nonce
func TestGetEmptyNonce(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("Get nonce with new wallet", func(t *testing.T) {
		accInfo := &pbfound.AccountInfo{
			GrayListed:  false,
			BlackListed: false,
		}

		rawAccInfo, err := json.Marshal(accInfo)
		require.NoError(t, err)

		mockStub.GetStateReturnsOnCall(0, []byte(config), nil)
		mockStub.GetStateReturnsOnCall(1, nil, nil) // nonce not exists in the chaincode state
		mockStub.GetFunctionAndParametersReturns(testGetNonceFnName, []string{owner.AddressBase58Check})
		// mock acl response
		mockStub.InvokeChaincodeReturns(peer.Response{
			Status:  http.StatusOK,
			Message: "",
			Payload: rawAccInfo,
		})
		resp := cc.Invoke(mockStub)
		require.Equal(t, testMessageEmptyNonce, string(resp.GetPayload()))
	})
}

// TestGetNonce - Checking that the nonce after some operation is not null
func TestGetNonce(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("Get nonce with new wallet", func(t *testing.T) {
		nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

		accInfo := &pbfound.AccountInfo{
			GrayListed:  false,
			BlackListed: false,
		}

		rawAccInfo, err := json.Marshal(accInfo)
		require.NoError(t, err)

		mockStub.GetStateReturnsOnCall(0, []byte(config), nil)
		mockStub.GetStateReturnsOnCall(1, []byte(nonce), nil) // nonce exists in the chaincode state
		mockStub.GetFunctionAndParametersReturns(testGetNonceFnName, []string{owner.AddressBase58Check})
		// mock acl response
		mockStub.InvokeChaincodeReturns(peer.Response{
			Status:  http.StatusOK,
			Message: "",
			Payload: rawAccInfo,
		})
		resp := cc.Invoke(mockStub)
		require.NotEqual(t, testMessageEmptyNonce, string(resp.GetPayload()))
	})
}

// TestInit - Checking that init with right mspId working
func TestInit(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("[negative] Init with wrong cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})
		err = mocks.SetCreator(mockStub, BatchRobotCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Equal(t, "init: validating admin creator: incorrect sender's OU, expected 'admin' but found 'client'", resp.GetMessage())
	})

	t.Run("Init with correct cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})

		err = mocks.SetCreatorCert(mockStub, mocks.TestCreatorMSP, mocks.AdminCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Empty(t, resp.GetMessage())

		require.Equal(t, 1, mockStub.PutStateCallCount())
		key, value := mockStub.PutStateArgsForCall(0)
		require.Equal(t, configKey, key)
		require.Equal(t, config, string(value))
	})
}

// TestTxHealthCheck - Checking healthcheck method.
func TestTxHealthCheck(t *testing.T) {
	mockStub := mocks.NewMockStub(t)
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("Healthcheck checking", func(t *testing.T) {
		fnHealthCheck := "healthCheck"

		ownerAddress := sha3.Sum256(owner.PublicKeyBytes)

		pending := &pbfound.PendingTx{
			Method: fnHealthCheck,
			Sender: &pbfound.Address{
				UserID:       owner.UserID,
				Address:      ownerAddress[:],
				IsIndustrial: false,
				IsMultisig:   false,
			},
			Args:  []string{},
			Nonce: uint64(time.Now().UnixNano() / 1000000),
		}
		pendingMarshalled, err := pb.Marshal(pending)
		require.NoError(t, err)

		dataIn, err := pb.Marshal(&pbfound.Batch{TxIDs: [][]byte{[]byte("testTxID")}})
		require.NoError(t, err)

		err = mocks.SetCreator(mockStub, BatchRobotCert)
		require.NoError(t, err)

		mockStub.GetFunctionAndParametersReturns("batchExecute", []string{string(dataIn)})

		mockStub.GetStateReturnsOnCall(0, []byte(config), nil)
		mockStub.GetStateReturnsOnCall(1, pendingMarshalled, nil)

		resp := cc.Invoke(mockStub)
		require.Empty(t, resp.GetMessage())
		require.NotEmpty(t, resp.GetPayload())
	})
}
