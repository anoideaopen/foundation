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

type MockStub struct {
	stub             *mocks.ChaincodeStub
	getStateCallsMap map[string][]byte
}

// NewMockStub returns new mock stub
func NewMockStub(t *testing.T) *MockStub {
	mockStub := new(mocks.ChaincodeStub)
	txID := [16]byte(uuid.New())
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetSignedProposalReturns(&peer.SignedProposal{}, nil)

	err := mocks.SetCreatorCert(mockStub, mocks.TestCreatorMSP, mocks.AdminCert)
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

	getStateCallsMap := make(map[string][]byte)

	mockStub.GetStateCalls(func(key string) ([]byte, error) {
		value, ok := getStateCallsMap[key]
		if ok {
			return value, nil
		}

		return nil, nil
	})

	mockStub.InvokeChaincodeCalls(func(chaincodeName string, args [][]byte, channelName string) peer.Response {
		if chaincodeName != "acl" && channelName != "acl" {
			return shim.Error("mock stub does not support chaincode " + chaincodeName + " and channel " + channelName + " calls")
		}
		functionName := string(args[0])

		parameters := make([]string, 0, len(args[1:]))
		for _, arg := range args[1:] {
			parameters = append(parameters, string(arg))
		}

		switch functionName {
		case FnCheckAddress:
			return MockACLCheckAddress(parameters[0])
		case FnCheckKeys:
			return MockACLCheckKeys(parameters[0])
		case FnGetAccountInfo:
			return MockACLGetAccountInfo()
		case FnGetAccountsInfo:
			return MockACLGetAccountsInfo()
		}

		return shim.Error("mock stub does not support " + functionName + "function")
	})

	return &MockStub{
		stub:             mockStub,
		getStateCallsMap: getStateCallsMap,
	}
}

func (ms *MockStub) GetStub() *mocks.ChaincodeStub {
	return ms.stub
}

func (ms *MockStub) GetState() map[string][]byte {
	return ms.getStateCallsMap
}

func (ms *MockStub) SetConfig(config string) {
	ms.getStateCallsMap["__config"] = []byte(config)
}

func (ms *MockStub) invokeChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) peer.Response {
	ms.stub.GetFunctionAndParametersReturns(functionName, parameters)
	return chaincode.Invoke(ms.stub)
}

func (ms *MockStub) QueryChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) peer.Response {
	return ms.invokeChaincode(chaincode, functionName, parameters...)
}

func (ms *MockStub) NbTxInvokeChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) peer.Response {
	return ms.invokeChaincode(chaincode, functionName, parameters...)
}

func (ms *MockStub) TxInvokeChaincode(chaincode *core.Chaincode, functionName string, parameters ...string) (string, peer.Response) {
	resp := ms.invokeChaincode(chaincode, functionName, parameters...)
	if resp.GetStatus() != int32(shim.OK) || resp.GetMessage() != "" {
		return "", resp
	}
	txID := ms.stub.GetTxID()

	key, err := ms.stub.CreateCompositeKey("batchTransactions", []string{txID})
	if err != nil {
		return "", shim.Error(err.Error())
	}

	for i := 0; i < ms.stub.PutStateCallCount(); i++ {
		putStateKey, rawValue := ms.stub.PutStateArgsForCall(i)
		if putStateKey == key {
			pending := &pbfound.PendingTx{}
			if err := proto.Unmarshal(rawValue, pending); err != nil {
				return "", shim.Error(err.Error())
			}

			if pending.GetMethod() == functionName {
				ms.getStateCallsMap[key] = rawValue

				hexTxID, err := hex.DecodeString(txID)
				if err != nil {
					return "", shim.Error(err.Error())
				}
				dataIn, err := proto.Marshal(&pbfound.Batch{TxIDs: [][]byte{hexTxID}})
				if err != nil {
					return "", shim.Error(err.Error())
				}

				err = mocks.SetCreator(ms.stub, mocks.BatchRobotCert)
				if err != nil {
					return "", shim.Error(err.Error())
				}

				resp = ms.invokeChaincode(chaincode, "batchExecute", []string{string(dataIn)}...)

				err = mocks.SetCreatorCert(ms.stub, mocks.TestCreatorMSP, mocks.AdminCert)
				if err != nil {
					return "", shim.Error(err.Error())
				}

				delete(ms.getStateCallsMap, key)

				break
			}
		}
	}

	return txID, resp
}

func (ms *MockStub) TxInvokeChaincodeSigned(
	chaincode *core.Chaincode,
	functionName string,
	user *mocks.UserFoundation,
	requestID string,
	chaincodeName string,
	channelName string,
	parameters ...string,
) (string, peer.Response) {
	// Artificial delay to update the nonce value
	time.Sleep(time.Millisecond * 5)

	ctorArgs := append(append([]string{functionName, requestID, channelName, chaincodeName}, parameters...), mocks.GetNewStringNonce())

	pubKey, sMsg, err := user.Sign(ctorArgs...)
	if err != nil {
		return "", shim.Error(err.Error())
	}

	return ms.TxInvokeChaincode(chaincode, functionName, append(ctorArgs[1:], pubKey, base58.Encode(sMsg))...)
}
