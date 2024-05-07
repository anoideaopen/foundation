package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/require"
)

func (tt *TestToken) TxTokenBalanceLock(_ *types.Sender, address *types.Address, amount *big.Int) error {
	return tt.TokenBalanceLock(address, amount)
}

func (tt *TestToken) QueryTokenBalanceGetLocked(address *types.Address) (*big.Int, error) {
	return tt.TokenBalanceGetLocked(address)
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
		lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
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
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Token balance unlock test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceUnlock", user1.Address(), "500")
		lockedBalance = user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
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
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance transfer test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceTransferLocked", user1.Address(), user2.Address(), "500", "transfer")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
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
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	require.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance burn test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceBurnLocked", user1.Address(), "500", "burn")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
		require.Equal(t, lockedBalanceUser1, "\"0\"")
	})
}
