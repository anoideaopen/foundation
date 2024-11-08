package unit

import (
	"encoding/json"
	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"golang.org/x/crypto/sha3"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

var (
	fiat = NewFiatTestToken(token.BaseToken{})
	usdt = "USDT"
)

func TestCreateIndex(t *testing.T) {
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)
	user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)
	user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)
	user3, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	mockStub := mocks.NewMockStub(t)

	config := makeBaseTokenConfig(
		"Test Token",
		"TT",
		8,
		owner.AddressBase58Check,
		"",
		"",
		"",
		nil,
	)

	cc, err := core.NewCC(fiat)
	require.NoError(t, err)

	// Checking for index absence
	var index bool
	mockStub.GetStateReturnsOnCall(0, []byte(config), nil)
	mockStub.GetStateReturnsOnCall(1, []byte{}, nil)
	balanceTypeStr, err := balance.BalanceTypeToStringMapValue(balance.BalanceTypeToken)
	require.NoError(t, err)
	mockStub.GetFunctionAndParametersReturns("indexCreated", []string{balanceTypeStr})

	resp := cc.Invoke(mockStub)
	require.Nil(t, err)
	require.NoError(t, json.Unmarshal(resp.Payload, &index))
	require.False(t, index)

	mockIterator := &mocks.StateIterator{}
	mockIterator.HasNextReturnsOnCall(0, false)
	mockStub.GetStateByPartialCompositeKeyReturns(mockIterator, nil)

	// Creating index
	key1, err := shim.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, usdt})
	require.NoError(t, err)

	key2, err := shim.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check, usdt})
	require.NoError(t, err)

	key3, err := shim.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user3.AddressBase58Check, usdt})
	require.NoError(t, err)

	key4, err := shim.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
	require.NoError(t, err)

	mockIterator.HasNextReturnsOnCall(0, true)
	mockIterator.HasNextReturnsOnCall(1, true)
	mockIterator.HasNextReturnsOnCall(2, true)
	mockIterator.HasNextReturnsOnCall(3, true)
	mockIterator.HasNextReturnsOnCall(4, false)
	mockIterator.NextReturnsOnCall(0, &queryresult.KV{
		Key:   key1,
		Value: []byte("1000"),
	}, nil)
	mockIterator.NextReturnsOnCall(1, &queryresult.KV{
		Key:   key2,
		Value: []byte("1000"),
	}, nil)
	mockIterator.NextReturnsOnCall(2, &queryresult.KV{
		Key:   key3,
		Value: []byte("1000"),
	}, nil)
	mockIterator.NextReturnsOnCall(3, &queryresult.KV{
		Key:   key4,
		Value: []byte("1000"),
	}, nil)

	mockStub.GetStateReturns([]byte(config), nil)
	mockStub.GetFunctionAndParametersReturns("createIndex", []string{balanceTypeStr})
	resp = cc.Invoke(mockStub)
	require.Equal(t, "", resp.Message)

	// checking index created and owners appears
	indexCreated, ownersAfterIndexing := checkIndexAndOwners(t, mockStub)
	require.True(t, indexCreated)
	require.Equal(t, 3, ownersAfterIndexing)
}

func TestAutoBalanceIndexing(t *testing.T) {
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)
	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	mockStub := mocks.NewMockStub(t)

	config := makeBaseTokenConfig(
		"Test Token",
		"TT",
		8,
		owner.AddressBase58Check,
		"",
		"",
		"",
		nil,
	)

	cc, err := core.NewCC(fiat)
	require.NoError(t, err)

	// checking there's no owners
	index, ownersAutoIndexed := checkIndexAndOwners(t, mockStub)
	require.False(t, index)
	require.Equal(t, 0, ownersAutoIndexed)

	// emulating batch execute for emitting tokens to user
	ownerAddress := sha3.Sum256(owner.PublicKeyBytes)

	pending := &pbfound.PendingTx{
		Method: "emitIndustrial",
		Sender: &pbfound.Address{
			UserID:       owner.UserID,
			Address:      ownerAddress[:],
			IsIndustrial: false,
			IsMultisig:   false,
		},
		Args: []string{
			user.AddressBase58Check,
			"1000",
			usdt,
		},
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
	require.Equal(t, "", resp.Message)

	// checking inverse balance appears
	_, ownersAutoIndexed = checkIndexAndOwners(t, mockStub)
	require.Equal(t, 1, ownersAutoIndexed)
}

func checkIndexAndOwners(t *testing.T, stub *mocks.ChaincodeStub) (bool, int) {
	indexKey, err := stub.CreateCompositeKey(balance.IndexCreatedKey, []string{balance.BalanceTypeToken.String()})
	require.NoError(t, err)

	indexCreated := false
	var ownersAfterIndexing []string

	// Checking index to be constructed.
	for i := 0; i < stub.PutStateCallCount(); i++ {
		key, value := stub.PutStateArgsForCall(i)
		if key == indexKey {
			indexCreated, err = strconv.ParseBool(string(value))
			require.NoError(t, err)
		}
		prefix, args, err := stub.SplitCompositeKey(key)
		require.NoError(t, err)
		if prefix == balance.InverseBalanceObjectType {
			ownersAfterIndexing = append(ownersAfterIndexing, args[2])
		}
	}

	return indexCreated, len(ownersAfterIndexing)
}
