package unit

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
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

	testMessageDecodeWrongParameter = "invalid argument value: '': for type '*unit.TestStruct': 'hello world should not be empty': validate TxHelloWorldSet, argument 0"
	testMessageEmptyNonce           = "\"0\""

	testFnGetNonce      = "getNonce"
	testFnHelloWorld    = "helloWorld"
	testFnHelloWorldSet = "helloWorldSet"
	testFnBatchExecute  = "batchExecute"
	testFnHealthCheck   = "healthCheck"
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

func TestContractMethods(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                      string
		owner                     *mocks.UserFoundation
		needACLAccess             bool
		functionName              string
		resultMessage             string
		preparePayloadEqual       func() []byte
		preparePayloadNotEqual    func() []byte
		prepareFunctionParameters func(owner *mocks.UserFoundation) []string
		prepareMockStubAdditional func(t *testing.T, mockStub *mocks.ChaincodeStub, owner *mocks.UserFoundation)
	}{
		{
			name:          "bytes encoder test",
			needACLAccess: false,
			functionName:  testFnHelloWorld,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{}
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return []byte("Hello World")
			},
		},
		{
			name:          "bytes decoder test",
			needACLAccess: false,
			functionName:  testFnHelloWorldSet,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{"Hi!"}
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return nil
			},
		},
		{
			name:          "[negative] bytes decoder test",
			needACLAccess: false,
			functionName:  testFnHelloWorldSet,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{""}
			},
			resultMessage: testMessageDecodeWrongParameter,
			preparePayloadEqual: func() []byte {
				return nil
			},
		},
		{
			name:          "empty nonce in state",
			needACLAccess: true,
			functionName:  testFnGetNonce,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{owner.AddressBase58Check}
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return []byte(testMessageEmptyNonce)
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, nil, nil)
			},
		},
		{
			name:          "existed nonce in state",
			needACLAccess: true,
			functionName:  testFnGetNonce,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{owner.AddressBase58Check}
			},
			resultMessage: "",
			preparePayloadNotEqual: func() []byte {
				return []byte(testMessageEmptyNonce)
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, []byte(strconv.FormatInt(time.Now().UnixNano()/1000000, 10)), nil)
			},
		},
		{
			name:         "health check",
			functionName: testFnBatchExecute,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				dataIn, err := pb.Marshal(&pbfound.Batch{TxIDs: [][]byte{[]byte("testTxID")}})
				require.NoError(t, err)

				return []string{string(dataIn)}
			},
			resultMessage: "",
			preparePayloadNotEqual: func() []byte {
				return nil
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner *mocks.UserFoundation) {
				ownerAddress := sha3.Sum256(owner.PublicKeyBytes)

				pending := &pbfound.PendingTx{
					Method: testFnHealthCheck,
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

				err = mocks.SetCreator(mockStub, BatchRobotCert)
				require.NoError(t, err)

				mockStub.GetStateReturnsOnCall(1, pendingMarshalled, nil)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			tt := &TestToken{}
			config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
				owner.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(tt)
			require.NoError(t, err)

			// preparing mockStub
			if test.needACLAccess {
				mocks.ACLGetAccountInfo(t, mockStub.ChaincodeStub, 0)
			}

			mockStub.GetStateReturnsOnCall(0, []byte(config), nil)

			if test.prepareMockStubAdditional != nil {
				test.prepareMockStubAdditional(t, mockStub.ChaincodeStub, owner)
			}

			mockStub.GetFunctionAndParametersReturns(test.functionName, test.prepareFunctionParameters(owner))
			resp := cc.Invoke(mockStub)
			if test.resultMessage != "" {
				require.Equal(t, test.resultMessage, resp.GetMessage())
			} else {
				require.Empty(t, resp.GetMessage())
				if test.preparePayloadEqual != nil {
					require.Equal(t, test.preparePayloadEqual(), resp.Payload)
				}
				if test.preparePayloadNotEqual != nil {
					require.NotEqual(t, test.preparePayloadNotEqual(), resp.Payload)
				}
			}
		})
	}
}

// TestInit - Checking that init with right mspId working
func TestInit(t *testing.T) {
	mockStub := mockstub.NewMockStub(t)

	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("[negative] Init with wrong cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})
		err = mocks.SetCreator(mockStub.ChaincodeStub, BatchRobotCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Equal(t, "init: validating admin creator: incorrect sender's OU, expected 'admin' but found 'client'", resp.GetMessage())
	})

	t.Run("Init with correct cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})

		err = mocks.SetCreatorCert(mockStub.ChaincodeStub, mocks.TestCreatorMSP, mocks.AdminCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Empty(t, resp.GetMessage())

		require.Equal(t, 1, mockStub.PutStateCallCount())
		key, value := mockStub.PutStateArgsForCall(0)
		require.Equal(t, configKey, key)
		require.Equal(t, config, string(value))
	})
}
