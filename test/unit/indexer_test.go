package unit

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/stretchr/testify/require"
)

var (
	fiat = NewFiatTestToken(&token.BaseToken{})
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

	mockStub := mockstub.NewMockStub(t)

	mockStub.CreateAndSetConfig(
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
	balanceTypeStr, err := balance.BalanceTypeToStringMapValue(balance.BalanceTypeToken)
	require.NoError(t, err)

	resp := mockStub.QueryChaincode(cc, "indexCreated", []string{balanceTypeStr}...)
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

	mockStub.GetFunctionAndParametersReturns("createIndex", []string{balanceTypeStr})
	resp = cc.Invoke(mockStub)
	require.Equal(t, "", resp.Message)

	// checking index created and owners appears
	indexCreated, ownersAfterIndexing := checkIndexAndOwners(t, mockStub.ChaincodeStub)
	require.True(t, indexCreated)
	require.Equal(t, 3, ownersAfterIndexing)
}

func TestAutoBalanceIndexing(t *testing.T) {
	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)
	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	mockStub := mockstub.NewMockStub(t)

	mockStub.CreateAndSetConfig(
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
	index, ownersAutoIndexed := checkIndexAndOwners(t, mockStub.ChaincodeStub)
	require.False(t, index)
	require.Equal(t, 0, ownersAutoIndexed)

	// emulating batch execute for emitting tokens to user
	_, resp := mockStub.TxInvokeChaincodeSigned(cc, "emitIndustrial", owner, "", "", "", []string{user.AddressBase58Check, "1000", "usdt"}...)
	require.Equal(t, "", resp.Message)

	// checking inverse balance appears
	_, ownersAutoIndexed = checkIndexAndOwners(t, mockStub.ChaincodeStub)
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
