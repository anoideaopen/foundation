package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
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
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	user1 := ledgerMock.NewWallet()

	ledgerMock.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, owner.Address())
	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")

	t.Run("Token balance get test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
		user1.BalanceShouldBe(testTokenCCName, 500)
		lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
		assert.Equal(t, lockedBalance, "\"500\"")
	})
}

// TestTokenBalanceUnlock - Checking that token balance can be unlocked
func TestTokenBalanceUnlock(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	user1 := ledgerMock.NewWallet()

	ledgerMock.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, owner.Address())
	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	assert.Equal(t, lockedBalance, "\"500\"")

	t.Run("Token balance unlock test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceUnlock", user1.Address(), "500")
		lockedBalance = user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
		assert.Equal(t, lockedBalance, "\"0\"")
		user1.BalanceShouldBe(testTokenCCName, 1000)
	})
}

// TestTokenBalanceTransferLocked - Checking that locked token balance can be transferred
func TestTokenBalanceTransferLocked(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	user1 := ledgerMock.NewWallet()
	user2 := ledgerMock.NewWallet()

	ledgerMock.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, owner.Address())
	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	assert.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance transfer test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceTransferLocked", user1.Address(), user2.Address(), "500", "transfer")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
		assert.Equal(t, lockedBalanceUser1, "\"0\"")
		user2.BalanceShouldBe(testTokenCCName, 500)
	})
}

// TestTokenBalanceBurnLocked - Checking that locked token balance can be burned
func TestTokenBalanceBurnLocked(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	user1 := ledgerMock.NewWallet()

	ledgerMock.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, owner.Address())
	owner.SignedInvoke(testTokenCCName, "emissionAdd", user1.Address(), "1000")
	owner.SignedInvoke(testTokenCCName, "tokenBalanceLock", user1.Address(), "500")
	user1.BalanceShouldBe(testTokenCCName, 500)
	lockedBalance := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
	assert.Equal(t, lockedBalance, "\"500\"")

	t.Run("Locked balance burn test", func(t *testing.T) {
		owner.SignedInvoke(testTokenCCName, "tokenBalanceBurnLocked", user1.Address(), "500", "burn")
		lockedBalanceUser1 := user1.Invoke(testTokenCCName, "tokenBalanceGetLocked", user1.Address())
		assert.Equal(t, lockedBalanceUser1, "\"0\"")
	})
}
