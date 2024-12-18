package unit

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"
)

type QueryTestToken struct {
	token.BaseToken
}

func (tt *QueryTestToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceAdd(token, address, amount, reason)
}

func (tt *QueryTestToken) QueryAllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceSub(token, address, amount, reason)
}

func (tt *QueryTestToken) QueryAllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceLock(token, address, amount)
}

func (tt *QueryTestToken) QueryAllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceUnLock(token, address, amount)
}

func (tt *QueryTestToken) QueryAllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *QueryTestToken) QueryAllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceBurnLocked(token, address, amount, reason)
}

func (tt *QueryTestToken) QueryAllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	return tt.AllowedBalanceGetAll(address)
}

type InvokeTestToken struct {
	token.BaseToken
}

func (tt *InvokeTestToken) TxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceAdd(token, address, amount, reason)
}

func (tt *InvokeTestToken) TxAllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceSub(token, address, amount, reason)
}

func (tt *InvokeTestToken) TxAllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceLock(token, address, amount)
}

func (tt *InvokeTestToken) TxAllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceUnLock(token, address, amount)
}

func (tt *InvokeTestToken) TxAllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *InvokeTestToken) TxAllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceBurnLocked(token, address, amount, reason)
}

// Checking query stub does not put any record into the state
func TestQuery(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                      string
		functionName              string
		preparePayloadEqual       func(t *testing.T) []byte
		prepareFunctionParameters func(user1, user2 *mocks.UserFoundation) []string
		prepareMockStubAdditional func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation)
	}{
		{
			name:         "Query allowed balance add",
			functionName: "allowedBalanceAdd",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
		},
		{
			name:         "Query allowed balance sub",
			functionName: "allowedBalanceSub",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
		},
		{
			name:         "Query allowed balance lock",
			functionName: "allowedBalanceLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
		},
		{
			name:         "Query allowed balance unlock",
			functionName: "allowedBalanceUnLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
		},
		{
			name:         "Query allowed transfer locked",
			functionName: "allowedBalanceTransferLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, user2.AddressBase58Check, "100", "reason"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
		},
		{
			name:         "Query allowed balance burn locked",
			functionName: "allowedBalanceBurnLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
		},
		{
			name:         "Query allowed balances get all",
			functionName: "allowedBalanceGetAll",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{user1.AddressBase58Check}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				balances := map[string]string{"vt": "100", "fiat": "200"}
				rawBalances, err := json.Marshal(balances)
				require.NoError(t, err)

				return rawBalances
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				mockIterator := &mocks.StateIterator{}
				mockIterator.HasNextReturnsOnCall(0, false)
				mockStub.GetStateByPartialCompositeKeyReturns(mockIterator, nil)

				key1, err := shim.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vt"})
				require.NoError(t, err)

				key2, err := shim.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "fiat"})
				require.NoError(t, err)

				mockIterator.HasNextReturnsOnCall(0, true)
				mockIterator.HasNextReturnsOnCall(1, true)
				mockIterator.HasNextReturnsOnCall(2, false)

				mockIterator.NextReturnsOnCall(0, &queryresult.KV{
					Key:   key1,
					Value: big.NewInt(100).Bytes(),
				}, nil)
				mockIterator.NextReturnsOnCall(1, &queryresult.KV{
					Key:   key2,
					Value: big.NewInt(200).Bytes(),
				}, nil)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(&QueryTestToken{})
			require.NoError(t, err)

			// preparing stub
			mockStub.SetConfig(config)

			if test.prepareMockStubAdditional != nil {
				test.prepareMockStubAdditional(t, mockStub, issuer, user1)
			}

			// invoking chaincode
			resp := mockStub.QueryChaincode(cc, test.functionName, test.prepareFunctionParameters(user1, user2)...)
			require.Empty(t, resp.GetMessage())
			require.Equal(t, test.preparePayloadEqual(t), resp.GetPayload())
			require.Equal(t, 0, mockStub.PutStateCallCount())
		})
	}
}

func TestAllowedBalanceInvoke(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                      string
		functionName              string
		checkPutState             func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte)
		prepareFunctionParameters func(user1, user2 *mocks.UserFoundation) []string
		prepareMockStubAdditional func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation)
	}{
		{
			name:         "Allowed balance add",
			functionName: "allowedBalanceAdd",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, big.NewInt(100).String(), "reason"}
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 3, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowed.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(100).Bytes(), value)
					}

					if prefix == "inverse_balance" {
						require.Equal(t, keys[0], balance.BalanceTypeAllowed.String())
						require.Equal(t, keys[1], "VT")
						require.Equal(t, keys[2], user1.AddressBase58Check)
						require.Equal(t, big.NewInt(100).Bytes(), value)
					}
				}
			},
		},
		{
			name:         "Allowed balance sub",
			functionName: "allowedBalanceSub",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, big.NewInt(100).String(), "reason"}
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 3, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowed.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}

					if keys[0] == "inverse_balance" {
						require.Equal(t, keys[0], balance.BalanceTypeAllowed.String())
						require.Equal(t, keys[1], "VT")
						require.Equal(t, keys[2], user1.AddressBase58Check)
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}
				}
			},
		},
		{
			name:         "Allowed balance lock",
			functionName: "allowedBalanceLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, big.NewInt(100).String()}
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 5, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowed.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}

					if prefix == balance.BalanceTypeAllowedLocked.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(100).Bytes(), value)
					}

					if prefix == "inverse_balance" {
						require.Equal(t, keys[1], "VT")
						require.Equal(t, keys[2], user1.AddressBase58Check)

						if keys[0] == balance.BalanceTypeAllowed.String() {
							require.Equal(t, big.NewInt(900).Bytes(), value)
						}
						if keys[0] == balance.BalanceTypeAllowedLocked.String() {
							require.Equal(t, big.NewInt(100).Bytes(), value)
						}
					}
				}
			},
		},
		{
			name:         "Allowed balance unlock",
			functionName: "allowedBalanceUnLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, big.NewInt(100).String()}
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 5, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowed.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(100).Bytes(), value)
					}

					if prefix == balance.BalanceTypeAllowedLocked.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}

					if prefix == "inverse_balance" {
						require.Equal(t, keys[1], "VT")
						require.Equal(t, keys[2], user1.AddressBase58Check)

						if keys[0] == balance.BalanceTypeAllowed.String() {
							require.Equal(t, big.NewInt(100).Bytes(), value)
						}
						if keys[0] == balance.BalanceTypeAllowedLocked.String() {
							require.Equal(t, big.NewInt(900).Bytes(), value)
						}
					}
				}
			},
		},
		{
			name:         "Allowed balance transfer locked",
			functionName: "allowedBalanceTransferLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, user2.AddressBase58Check, big.NewInt(100).String(), "reason"}
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 5, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowed.String() {
						require.Equal(t, keys[0], user2.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(100).Bytes(), value)
					}

					if prefix == balance.BalanceTypeAllowedLocked.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}

					if prefix == "inverse_balance" {
						require.Equal(t, keys[1], "VT")

						if keys[0] == balance.BalanceTypeAllowed.String() {
							require.Equal(t, keys[2], user2.AddressBase58Check)
							require.Equal(t, big.NewInt(100).Bytes(), value)
						}
						if keys[0] == balance.BalanceTypeAllowedLocked.String() {
							require.Equal(t, keys[2], user1.AddressBase58Check)
							require.Equal(t, big.NewInt(900).Bytes(), value)
						}
					}
				}
			},
		},
		{
			name:         "Allowed balance burn locked",
			functionName: "allowedBalanceBurnLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, big.NewInt(100).String(), "reason"}
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()
			},
			checkPutState: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, payload []byte) {
				require.Equal(t, 3, mockStub.PutStateCallCount())
				var i int
				for i = 0; i < mockStub.PutStateCallCount(); i++ {
					key, value := mockStub.PutStateArgsForCall(i)
					prefix, keys, err := mockStub.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeAllowedLocked.String() {
						require.Equal(t, keys[0], user1.AddressBase58Check)
						require.Equal(t, keys[1], "VT")
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}

					if prefix == "inverse_balance" {
						require.Equal(t, keys[0], balance.BalanceTypeAllowedLocked.String())
						require.Equal(t, keys[1], "VT")
						require.Equal(t, keys[2], user1.AddressBase58Check)
						require.Equal(t, big.NewInt(900).Bytes(), value)
					}
				}
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", "", nil)

			mockStub.SetConfig(config)

			cc, err := core.NewCC(&InvokeTestToken{})
			require.NoError(t, err)

			// preparing stub
			if test.prepareMockStubAdditional != nil {
				test.prepareMockStubAdditional(t, mockStub, issuer, user1)
			}

			// invoking chaincode
			_, resp := mockStub.TxInvokeChaincode(
				cc,
				test.functionName,
				test.prepareFunctionParameters(user1, user2)...)
			require.Empty(t, resp.GetMessage())
			test.checkPutState(t, mockStub, user1, user2, resp.GetPayload())
		})
	}
}
