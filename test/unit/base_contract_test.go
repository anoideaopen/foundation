package unit

import (
	"encoding/hex"
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
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
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

func (tt *TestToken) TxHelloWorldSet(_ *TestStruct) error {
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
		functionName              string
		invokeFunction            func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response
		resultMessage             string
		preparePayloadEqual       func() []byte
		preparePayloadNotEqual    func() []byte
		prepareFunctionParameters func(owner *mocks.UserFoundation) []string
		prepareMockStubAdditional func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation)
	}{
		{
			name:         "bytes encoder test",
			functionName: testFnHelloWorld,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.NbTxInvokeChaincode(cc, functionName, parameters...)
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return []byte("Hello World")
			},
		},
		{
			name:         "bytes decoder test",
			functionName: testFnHelloWorldSet,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{"Hi!"}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.NbTxInvokeChaincode(cc, functionName, parameters...)
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return nil
			},
		},
		{
			name:         "[negative] bytes decoder test",
			functionName: testFnHelloWorldSet,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{""}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.NbTxInvokeChaincode(cc, functionName, parameters...)
			},
			resultMessage: testMessageDecodeWrongParameter,
			preparePayloadEqual: func() []byte {
				return nil
			},
		},
		{
			name:         "empty nonce in state",
			functionName: testFnGetNonce,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{owner.AddressBase58Check}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.NbTxInvokeChaincode(cc, functionName, parameters...)
			},
			resultMessage: "",
			preparePayloadEqual: func() []byte {
				return []byte(testMessageEmptyNonce)
			},
		},
		{
			name:         "existed nonce in state",
			functionName: testFnGetNonce,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{owner.AddressBase58Check}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.NbTxInvokeChaincode(cc, functionName, parameters...)
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation) {
				prefix := hex.EncodeToString([]byte{core.StateKeyNonce})
				key, err := mockStub.CreateCompositeKey(prefix, []string{owner.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = []byte(strconv.FormatInt(time.Now().UnixNano()/1000000, 10))
			},
			resultMessage: "",
			preparePayloadNotEqual: func() []byte {
				return []byte(testMessageEmptyNonce)
			},
		},
		{
			name:         "health check",
			functionName: testFnHealthCheck,
			prepareFunctionParameters: func(owner *mocks.UserFoundation) []string {
				return []string{}
			},
			invokeFunction: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			resultMessage: "",
			preparePayloadNotEqual: func() []byte {
				return nil
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			tt := &TestToken{}
			mockStub.CreateAndSetConfig(testTokenName, testTokenSymbol, 8,
				owner.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(tt)
			require.NoError(t, err)

			// preparing mockStub
			if test.prepareMockStubAdditional != nil {
				test.prepareMockStubAdditional(t, mockStub, owner)
			}

			resp := test.invokeFunction(cc, mockStub, test.functionName, owner, test.prepareFunctionParameters(owner)...)
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
	config := mockStub.CreateAndSetConfig(testTokenName, testTokenSymbol, 8,
		owner.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	t.Run("[negative] Init with wrong cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})
		err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Equal(t, "init: validating admin creator: incorrect sender's OU, expected 'admin' but found 'client'", resp.GetMessage())
	})

	t.Run("Init with correct cert", func(t *testing.T) {
		mockStub.GetStringArgsReturns([]string{config})

		err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.AdminHexCert)
		require.NoError(t, err)

		resp := cc.Init(mockStub)
		require.Empty(t, resp.GetMessage())

		require.Equal(t, 1, mockStub.PutStateCallCount())
		key, value := mockStub.PutStateArgsForCall(0)
		require.Equal(t, configKey, key)
		require.Equal(t, config, string(value))
	})
}
