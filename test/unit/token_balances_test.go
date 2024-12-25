package unit

import (
	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
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
		},
		{
			name:         "Transfer locked balance",
			functionName: "tokenBalanceTransferLocked",
		},
		{
			name:         "Burn locked balance",
			functionName: "tokenBalanceBurnLocked",
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

// TestTokenBalanceLockAndGetLocked - Checking that token balance can be locked
func TestTokenBalanceLockAndGetLocked(t *testing.T) {
	t.Parallel()

	lm := mock.NewLedger(t)
	issuer := lm.NewWallet()

	config := makeBaseTokenConfig("tt", "TT", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := lm.NewCC("tt", &TestToken{}, config)
	require.Empty(t, initMsg)

	user1 := lm.NewWallet()
	err := issuer.RawSignedInvokeWithErrorReturned("tt", "emissionAdd", user1.Address(), "1000")
	require.NoError(t, err)

	t.Run("Token balance get test", func(t *testing.T) {
		issuer.SignedInvoke("tt", "tokenBalanceLock", user1.Address(), "500")
		user1.BalanceShouldBe("tt", 500)
		lockedBalance := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
		require.Equal(t, lockedBalance, "\"500\"")
	})
}

// TestTokenBalanceUnlock - Checking that token balance can be unlocked
func TestTokenBalanceUnlock(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC(testTokenCCName, &TestToken{}, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")

	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Token balance unlock test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceUnlock", user1.Address(), "500")
		lockedBalance = user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
		require.Equal(t, lockedBalance, "\"0\"")
		user1.BalanceShouldBe(testTokenCCName, 1000)
	})
}

// TestTokenBalanceTransferLocked - Checking that locked token balance can be transferred
func TestTokenBalanceTransferLocked(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC(testTokenCCName, tt, ttConfig)

	user1 := ledger.NewWallet()
	user2 := ledger.NewWallet()

	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance transfer test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceTransferLocked", user1.Address(), user2.Address(), "500", "transfer")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
		require.Equal(t, lockedBalanceUser1, "\"0\"")
		user2.BalanceShouldBe(testTokenCCName, 500)
	})
}

// TestTokenBalanceBurnLocked - Checking that locked token balance can be burned
func TestTokenBalanceBurnLocked(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC(testTokenCCName, tt, ttConfig)

	user1 := ledger.NewWallet()

	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance burn test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceBurnLocked", user1.Address(), "500", "burn")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "lockedBalanceOf", user1.Address())
		require.Equal(t, lockedBalanceUser1, "\"0\"")
	})
}
