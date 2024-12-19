package mockstub

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint: staticcheck
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

// MockStub represents mock stub structure
type MockStub struct {
	*mocks.ChaincodeStub
	GetStateCallsMap map[string][]byte
	InvokeACLMap     map[string]func(mockStub *MockStub, parameters ...string) peer.Response
}

// NewMockStub returns new mock stub
func NewMockStub(t *testing.T) *MockStub {
	mockStub := &MockStub{
		ChaincodeStub: new(mocks.ChaincodeStub),
	}

	// Important! Returns constant txID. Maybe needed to define another GetTxIDReturns for more than one transaction
	txID := [16]byte(uuid.New())
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))

	mockStub.GetSignedProposalReturns(&peer.SignedProposal{}, nil)

	err := mocks.SetCreator(mockStub.ChaincodeStub, mocks.AdminHexCert)
	require.NoError(t, err)

	mockStub.CreateCompositeKeyCalls(shim.CreateCompositeKey)
	mockStub.SplitCompositeKeyCalls(func(s string) (string, []string, error) {
		componentIndex := 1
		var components []string
		for i := 1; i < len(s); i++ {
			if s[i] == 0 {
				components = append(components, s[componentIndex:i])
				componentIndex = i + 1
			}
		}
		return components[0], components[1:], nil
	})

	mockStub.GetStateCallsMap = make(map[string][]byte)

	mockStub.GetStateCalls(func(key string) ([]byte, error) {
		value, ok := mockStub.GetStateCallsMap[key]
		if ok {
			return value, nil
		}

		return nil, nil
	})

	mockStub.InvokeACLMap = map[string]func(mockStub *MockStub, parameters ...string) peer.Response{
		FnCheckAddress:    MockACLCheckAddress,
		FnCheckKeys:       MockACLCheckKeys,
		FnGetAccountInfo:  MockACLGetAccountInfo,
		FnGetAccountsInfo: MockACLGetAccountsInfo,
	}

	mockStub.InvokeChaincodeCalls(func(chaincodeName string, args [][]byte, channelName string) peer.Response {
		if chaincodeName != "acl" && channelName != "acl" {
			return shim.Error("mock stub does not support chaincode " + chaincodeName + " and channel " + channelName + " calls")
		}
		functionName := string(args[0])

		parameters := make([]string, 0, len(args[1:]))
		for _, arg := range args[1:] {
			parameters = append(parameters, string(arg))
		}

		if function, ok := mockStub.InvokeACLMap[functionName]; ok {
			return function(mockStub, parameters...)
		}

		return shim.Error("mock stub does not support " + functionName + "function")
	})

	return mockStub
}

// SetConfig sets config to MockStub state
func (ms *MockStub) SetConfig(config string) {
	ms.GetStateCallsMap["__config"] = []byte(config)
}

// invokeChaincode invokes chaincode
func (ms *MockStub) invokeChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) peer.Response {
	ms.GetFunctionAndParametersReturns(functionName, parameters)

	// Artificial delay to update the nonce value
	time.Sleep(time.Millisecond * 5)

	return chaincode.Invoke(ms)
}

// QueryChaincode returns query result
func (ms *MockStub) QueryChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) peer.Response {
	return ms.invokeChaincode(chaincode, functionName, parameters...)
}

// NbTxInvokeChaincode returns non batched transaction result
func (ms *MockStub) NbTxInvokeChaincode(
	chaincode *core.Chaincode,
	functionName string,
	parameters ...string,
) peer.Response {
	return ms.invokeChaincode(chaincode, functionName, parameters...)
}

// NbTxInvokeChaincodeSigned returns non-batched transaction result with signed arguments
func (ms *MockStub) NbTxInvokeChaincodeSigned(
	chaincode *core.Chaincode,
	functionName string,
	user *mocks.UserFoundation,
	requestID string,
	chaincodeName string,
	channelName string,
	parameters ...string,
) peer.Response {
	params, err := getParametersSigned(functionName, user, requestID, chaincodeName, channelName, parameters...)
	if err != nil {
		return shim.Error(err.Error())
	}

	return ms.invokeChaincode(chaincode, functionName, params...)
}

// TxInvokeChaincode returns result of batchExecute transaction
func (ms *MockStub) TxInvokeChaincode(
	chaincode *core.Chaincode,
	functionName string,
	parameters ...string,
) (string, peer.Response) {
	resp := ms.invokeChaincode(chaincode, functionName, parameters...)
	if resp.GetStatus() != int32(shim.OK) || resp.GetMessage() != "" {
		return "", resp
	}
	txID := ms.GetTxID()

	key, err := ms.CreateCompositeKey("batchTransactions", []string{txID})
	if err != nil {
		return "", shim.Error(err.Error())
	}

	for i := 0; i < ms.PutStateCallCount(); i++ {
		putStateKey, rawValue := ms.PutStateArgsForCall(i)
		if putStateKey == key { //nolint:nestif
			pending := &pbfound.PendingTx{}
			if err := proto.Unmarshal(rawValue, pending); err != nil {
				return "", shim.Error(err.Error())
			}

			if pending.GetMethod() == functionName {
				ms.GetStateCallsMap[key] = rawValue

				hexTxID, err := hex.DecodeString(txID)
				if err != nil {
					return "", shim.Error(err.Error())
				}
				dataIn, err := proto.Marshal(&pbfound.Batch{TxIDs: [][]byte{hexTxID}})
				if err != nil {
					return "", shim.Error(err.Error())
				}

				err = mocks.SetCreator(ms.ChaincodeStub, mocks.BatchRobotCert)
				if err != nil {
					return "", shim.Error(err.Error())
				}

				resp = ms.invokeChaincode(chaincode, "batchExecute", []string{string(dataIn)}...)

				err = mocks.SetCreator(ms.ChaincodeStub, mocks.AdminHexCert)
				if err != nil {
					return "", shim.Error(err.Error())
				}

				delete(ms.GetStateCallsMap, key)

				break
			}
		}
	}

	return txID, resp
}

// TxInvokeChaincodeSigned returns result of batchExecute transaction with signed arguments
func (ms *MockStub) TxInvokeChaincodeSigned(
	chaincode *core.Chaincode,
	functionName string,
	user *mocks.UserFoundation,
	requestID string,
	chaincodeName string,
	channelName string,
	parameters ...string,
) (string, peer.Response) {
	params, err := getParametersSigned(functionName, user, requestID, chaincodeName, channelName, parameters...)
	if err != nil {
		return "", shim.Error(err.Error())
	}

	return ms.TxInvokeChaincode(chaincode, functionName, params...)
}

// getParametersSigned returns parameters string with specified user's signification
func getParametersSigned(
	functionName string,
	user *mocks.UserFoundation,
	requestID string,
	chaincodeName string,
	channelName string,
	parameters ...string,
) ([]string, error) {
	ctorArgs := append(append([]string{functionName, requestID, channelName, chaincodeName}, parameters...), mocks.GetNewStringNonce())

	pubKey, sMsg, err := user.Sign(ctorArgs...)
	if err != nil {
		return []string{}, err
	}

	return append(ctorArgs[1:], pubKey, base58.Encode(sMsg)), nil
}
