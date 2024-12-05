package mocks

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

type Ledger struct {
	t          *testing.T
	configs    map[string]string
	stubs      map[string]*ChaincodeStub
	chaincodes map[string]*core.Chaincode
}

func NewLedger(t *testing.T) *Ledger {
	return &Ledger{
		t: t,
	}
}

func (ledger *Ledger) NewCC(
	name string,
	bci core.BaseContractInterface,
	config string,
	opts ...core.ChaincodeOption,
) *core.Chaincode {
	mockStub := NewMockStub(ledger.t)
	mockStub.GetChannelIDReturns(name)
	ledger.stubs[name] = mockStub
	ledger.configs[name] = config

	cc, err := core.NewCC(bci, opts...)
	require.NoError(ledger.t, err)

	resp := cc.Init(mockStub)
	require.Empty(ledger.t, resp.GetMessage())

	ledger.chaincodes[name] = cc

	return cc
}

func (ledger *Ledger) GetStub(name string) *ChaincodeStub {
	return ledger.stubs[name]
}

func (ledger *Ledger) InvokeWithSign(name string, functionName string, signer *UserFoundation, parameters ...string) peer.Response {
	mockStub := ledger.stubs[name]
	config := ledger.configs[name]
	cc := ledger.chaincodes[name]

	mockStub.GetStateReturnsOnCall(0, []byte(config), nil)
	err := SetFunctionAndParametersWithSign(mockStub, signer, functionName, "", name, name, parameters...)
	require.NoError(ledger.t, err)
	ACLCheckSigner(ledger.t, mockStub, signer, false)
	ACLGetAccountInfo(ledger.t, mockStub, 1)

	resp := cc.Invoke(mockStub)
	require.Empty(ledger.t, resp.GetMessage())

	key, value := mockStub.PutStateArgsForCall(0)
	_, transactionID, err := mockStub.SplitCompositeKey(key)
	require.NoError(ledger.t, err)

	pending := &pbfound.PendingTx{}
	err = proto.Unmarshal(value, pending)
	require.NoError(ledger.t, err)

	err = SetCreator(mockStub, BatchRobotCert)
	require.NoError(ledger.t, err)

	mockStub.GetStateReturnsOnCall(1, []byte(config), nil)
	mockStub.GetStateReturnsOnCall(2, resp.GetPayload(), nil)

	dataIn, err := pb.Marshal(&pbfound.Batch{TxIDs: [][]byte{[]byte(transactionID[0])}})
	require.NoError(ledger.t, err)

	SetFunctionAndParameters(mockStub, "batchExecute", "", name, name, string(dataIn))
	return cc.Invoke(mockStub)
}
