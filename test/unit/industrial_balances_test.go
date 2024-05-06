package unit

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/require"
)

func (tt *TestToken) TxIndustrialBalanceAdd(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceAdd(token, address, amount, reason)
}

func (tt *TestToken) QueryIndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	return tt.IndustrialBalanceGet(address)
}

func (tt *TestToken) TxIndustrialBalanceSub(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceSub(token, address, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceTransfer(_ *types.Sender, token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceTransfer(token, from, to, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceLock(_ *types.Sender, token string, address *types.Address, amount *big.Int) error {
	return tt.IndustrialBalanceLock(token, address, amount)
}

func (tt *TestToken) QueryIndustrialBalanceGetLocked(address *types.Address) (map[string]string, error) {
	return tt.IndustrialBalanceGetLocked(address)
}

func (tt *TestToken) TxIndustrialBalanceUnLock(_ *types.Sender, token string, address *types.Address, amount *big.Int) error {
	return tt.IndustrialBalanceUnLock(token, address, amount)
}

func (tt *TestToken) TxIndustrialBalanceTransferLocked(_ *types.Sender, token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceBurnLocked(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceBurnLocked(token, address, amount, reason)
}

// TestIndustrialBalanceAdd - Checking that industrial balance can be added
func TestIndustrialBalanceAdd(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "123"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user := ledgerMock.NewWallet()

	t.Run("Industrial balance add", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceAdd", testTokenWithGroup, user.Address(), balanceAddAmount, "add balance")

		balanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user.Address())
		balance, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, balanceAddAmount, balance, "require that balance equals "+balanceAddAmount)
	})
}

// TestIndustrialBalanceSub - Checking that industrial balance sub is working
func TestIndustrialBalanceSub(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user := ledgerMock.NewWallet()

	balanceAddAmount := "123"
	subAmount := "23"
	balanceAfterSubExpected := "100"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	owner.SignedInvoke(
		testTokenWithGroup,
		"industrialBalanceAdd",
		testTokenWithGroup,
		user.Address(),
		balanceAddAmount,
		"add balance for "+balanceAddAmount,
	)
	owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user.Address())

	t.Run("Industrial balance sub", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceSub", testTokenWithGroup, user.Address(), subAmount, "sub balance for "+subAmount)
		balanceAfterSubResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user.Address())
		balanceAfterSub, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterSubResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, balanceAfterSubExpected, balanceAfterSub, "require that balance equals "+balanceAfterSubExpected)
	})
}

// TestIndustrialBalanceTransfer - Checking that industrial balance transfer is working
func TestIndustrialBalanceTransfer(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "123"
	transferAmount := "122"
	balanceAfterTransferUser1Expected := "1"
	balanceAfterTransferUser2Expected := "122"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user1 := ledgerMock.NewWallet()
	user2 := ledgerMock.NewWallet()

	owner.SignedInvoke(testTokenWithGroup, "industrialBalanceAdd", testTokenWithGroup, user1.Address(), balanceAddAmount, "add balance for "+balanceAddAmount)
	owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())

	t.Run("Industrial balance transfer", func(t *testing.T) {
		user1.SignedInvoke(testTokenWithGroup, "industrialBalanceTransfer", testGroup, user1.Address(), user2.Address(), transferAmount, "transfer balance for "+transferAmount)

		balanceAfterSubUser2Response := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user2.Address())
		balanceAfterSubUser2, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterSubUser2Response, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, balanceAfterTransferUser2Expected, balanceAfterSubUser2, "require that balance equals "+balanceAfterTransferUser2Expected)

		balanceAfterSubUser1Response := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
		balanceAfterSubUser1, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterSubUser1Response, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, balanceAfterTransferUser1Expected, balanceAfterSubUser1, "require that balance equals "+balanceAfterTransferUser1Expected)
	})
}

// TestIndustrialBalanceLockAndGetLocked - Checking that industrial balance can be locked
func TestIndustrialBalanceLockAndGetLocked(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "1000"
	lockAmount := "500"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user1 := ledgerMock.NewWallet()

	owner.SignedInvoke(
		testTokenWithGroup,
		"industrialBalanceAdd",
		testTokenWithGroup,
		user1.Address(),
		balanceAddAmount,
		"add industrial balance for "+balanceAddAmount,
	)

	balanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balance, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, balanceAddAmount, balance, "require that balance equals "+balanceAddAmount)

	t.Run("Industrial balance lock and get test", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceLock", testTokenWithGroup, user1.Address(), lockAmount)

		balanceAfterLockResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
		balanceAfterLock, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterLockResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, lockAmount, balanceAfterLock, "require that balance equals "+balanceAfterLock)

		lockedBalanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
		lockedBalance, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, lockAmount, lockedBalance, "require that locked balance equals "+lockedBalance)
	})
}

// TestIndustrialBalanceUnLock - Checking that industrial balance can be unlocked
func TestIndustrialBalanceUnLock(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "1000"
	lockAmount := "500"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user1 := ledgerMock.NewWallet()

	owner.SignedInvoke(
		testTokenWithGroup,
		"industrialBalanceAdd",
		testTokenWithGroup,
		user1.Address(),
		balanceAddAmount,
		"add industrial balance for "+balanceAddAmount,
	)

	balanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balance, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, balanceAddAmount, balance, "require that balance equals "+balanceAddAmount)

	owner.SignedInvoke(testTokenWithGroup, "industrialBalanceLock", testTokenWithGroup, user1.Address(), lockAmount)

	balanceAfterLockResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balanceAfterLock, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterLockResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, balanceAfterLock, "require that balance equals "+balanceAfterLock)

	lockedBalanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
	lockedBalance, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, lockedBalance, "require that locked balance equals "+lockedBalance)

	t.Run("Industrial balance unlock test", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceUnLock", testTokenWithGroup, user1.Address(), "300")
		lockedBalanceAfterUnlockResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
		lockedBalanceAfterUnlock, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceAfterUnlockResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, "200", lockedBalanceAfterUnlock, "require that locked balance equals "+lockedBalance)
	})
}

// TestIndustrialBalanceTransferLocked - Checking that locked industrial balance can be transferred
func TestIndustrialBalanceTransferLocked(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "1000"
	lockAmount := "500"
	transferAmount := "300"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user1 := ledgerMock.NewWallet()
	user2 := ledgerMock.NewWallet()

	owner.SignedInvoke(
		testTokenWithGroup,
		"industrialBalanceAdd",
		testTokenWithGroup,
		user1.Address(),
		balanceAddAmount,
		"add industrial balance for "+balanceAddAmount,
	)

	balanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balance, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, balanceAddAmount, balance, "require that balance equals "+balanceAddAmount)
	owner.SignedInvoke(testTokenWithGroup, "industrialBalanceLock", testTokenWithGroup, user1.Address(), lockAmount)

	balanceAfterLockResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balanceAfterLock, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterLockResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, balanceAfterLock, "require that balance equals "+balanceAfterLock)

	lockedBalanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
	lockedBalance, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, lockedBalance, "require that locked balance equals "+lockedBalance)

	t.Run("Industrial balance transfer locked", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceTransferLocked", testTokenWithGroup, user1.Address(), user2.Address(), transferAmount, "transfer locked")

		balanceAfterTransferUser1Response := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
		balanceAfterTransferUser1, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterTransferUser1Response, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, "200", balanceAfterTransferUser1, "require that locked balance equals "+lockedBalance)

		balanceAfterTransferUser2Response := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user2.Address())
		balanceAfterTransferUser2, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterTransferUser2Response, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, transferAmount, balanceAfterTransferUser2, "require that locked balance equals "+lockedBalance)
	})
}

// TestIndustrialBalanceBurnLocked - Checking that locked industrial balance can be burned
func TestIndustrialBalanceBurnLocked(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	balanceAddAmount := "1000"
	lockAmount := "500"

	tt := &TestToken{}
	ttConfig := makeBaseTokenConfig(testTokenWithGroup, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	ledgerMock.NewCC(testTokenWithGroup, tt, ttConfig)

	user1 := ledgerMock.NewWallet()

	owner.SignedInvoke(testTokenWithGroup, "industrialBalanceAdd", testTokenWithGroup, user1.Address(), balanceAddAmount, "add industrial balance for "+balanceAddAmount)

	balanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balance, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, balanceAddAmount, balance, "require that balance equals "+balanceAddAmount)

	owner.SignedInvoke(testTokenWithGroup, "industrialBalanceLock", testTokenWithGroup, user1.Address(), lockAmount)

	balanceAfterLockResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGet", user1.Address())
	balanceAfterLock, err := GetIndustrialBalanceFromResponseByGroup(balanceAfterLockResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, balanceAfterLock, "require that balance equals "+balanceAfterLock)

	lockedBalanceResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
	lockedBalance, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceResponse, testGroup)
	if err != nil {
		require.FailNow(t, err.Error())
	}
	require.Equal(t, lockAmount, lockedBalance, "require that locked balance equals "+lockedBalance)

	t.Run("Industrial balance burn locked", func(t *testing.T) {
		owner.SignedInvoke(testTokenWithGroup, "industrialBalanceBurnLocked", testTokenWithGroup, user1.Address(), "300", "burn locked")

		lockedBalanceAfterBurnResponse := owner.Invoke(testTokenWithGroup, "industrialBalanceGetLocked", user1.Address())
		lockedBalanceAfterBurn, err := GetIndustrialBalanceFromResponseByGroup(lockedBalanceAfterBurnResponse, testGroup)
		if err != nil {
			require.FailNow(t, err.Error())
		}
		require.Equal(t, "200", lockedBalanceAfterBurn, "require that locked balance equals "+lockedBalance)
	})
}

func GetIndustrialBalanceFromResponseByGroup(response string, group string) (string, error) {
	var balanceMap map[string]string
	err := json.Unmarshal([]byte(response), &balanceMap)
	if err != nil {
		return "", err
	}
	bl := balanceMap[group]
	if bl == "" {
		return "", errors.New("cant find balance for group: " + group)
	}
	return bl, nil
}
