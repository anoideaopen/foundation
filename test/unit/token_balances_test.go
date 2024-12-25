package unit

import (
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

func (tt *TestToken) TxTokenBalanceLock(_ *types.Sender, address *types.Address, amount *big.Int) error {
	return tt.TokenBalanceLock(address, amount)
}

func (tt *TestToken) TxTokenBalanceUnlock(_ *types.Sender, address *types.Address, amount *big.Int) error {
	return tt.TokenBalanceUnlock(address, amount)
}

func (tt *TestToken) TxTokenBalanceTransferLocked(_ *types.Sender, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.TokenBalanceTransferLocked(from, to, amount, reason)
}

func (tt *TestToken) TxTokenBalanceBurnLocked(_ *types.Sender, address *types.Address, amount *big.Int, reason string) error {
	return tt.TokenBalanceBurnLocked(address, amount, reason)
}

func TestTokenBalances(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
		) []string
		funcInvokeChaincode func(
			cc *core.Chaincode,
			mockStub *mockstub.MockStub,
			functionName string,
			issuer *mocks.UserFoundation,
			user1 *mocks.UserFoundation,
			parameters ...string,
		) peer.Response
		funcCheckResult func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
			resp peer.Response,
		)
	}{
		{
			name:         "Lock balance",
			functionName: "tokenBalanceLock",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{user1.AddressBase58Check, "500"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, user1 *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation, resp peer.Response) {
				balanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				lockedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				balanceChecked := false
				lockedChecked := false

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == balanceKey {
						require.Equal(t, big.NewInt(500), new(big.Int).SetBytes(value))
						balanceChecked = true
					}
					if putStateKey == lockedBalanceKey {
						require.Equal(t, big.NewInt(500), new(big.Int).SetBytes(value))
						lockedChecked = true
					}
				}

				require.True(t, balanceChecked && lockedChecked)
			},
		},
		{
			name:         "Query locked balance",
			functionName: "lockedBalanceOf",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(500).Bytes()

				return []string{user1.AddressBase58Check}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, user1 *mocks.UserFoundation, parameters ...string) peer.Response {
				return mockStub.QueryChaincode(cc, functionName, parameters...)
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation, resp peer.Response) {
				require.Equal(t, "\"500\"", string(resp.GetPayload()))
			},
		},
		{
			name:         "Unlock balance",
			functionName: "tokenBalanceUnlock",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{user1.AddressBase58Check, "500"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, user1 *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation, resp peer.Response) {
				balanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				lockedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				balanceChecked := false
				lockedChecked := false

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == balanceKey {
						require.Equal(t, big.NewInt(500), new(big.Int).SetBytes(value))
						balanceChecked = true
					}
					if putStateKey == lockedBalanceKey {
						require.Equal(t, big.NewInt(500), new(big.Int).SetBytes(value))
						lockedChecked = true
					}
				}

				require.True(t, balanceChecked && lockedChecked)
			},
		},
		{
			name:         "Transfer locked balance",
			functionName: "tokenBalanceTransferLocked",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{user1.AddressBase58Check, user2.AddressBase58Check, "300", "test transfer locked balance"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, user1 *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation, resp peer.Response) {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				user2BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check})
				require.NoError(t, err)

				balanceChecked := false
				lockedChecked := false

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == user1BalanceKey {
						require.Equal(t, big.NewInt(700), new(big.Int).SetBytes(value))
						balanceChecked = true
					}
					if putStateKey == user2BalanceKey {
						require.Equal(t, big.NewInt(300), new(big.Int).SetBytes(value))
						lockedChecked = true
					}
				}

				require.True(t, balanceChecked && lockedChecked)
			},
		},
		{
			name:         "Burn locked balance",
			functionName: "tokenBalanceBurnLocked",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{user1.AddressBase58Check, "500", "test token burning locked"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer *mocks.UserFoundation, user1 *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1 *mocks.UserFoundation, user2 *mocks.UserFoundation, resp peer.Response) {
				lockedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				lockedChecked := false

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == lockedBalanceKey {
						require.Equal(t, big.NewInt(500), new(big.Int).SetBytes(value))
						lockedChecked = true
					}
				}

				require.True(t, lockedChecked)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig("tt", "TT", 8,
				issuer.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(&TestToken{})
			require.NoError(t, err)

			parameters := test.funcPrepareMockStub(t, mockStub, user1, user2)

			resp := test.funcInvokeChaincode(cc, mockStub, test.functionName, issuer, user1, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())
			test.funcCheckResult(t, mockStub, user1, user2, resp)
		})
	}
}
