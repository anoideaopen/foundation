package unit

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

// implemented through query requests, an error comes through tx
// relate github.com/anoideaopen/foundation/-/issues/44 https://github.com/anoideaopen/foundation/-/issues/45

func (tt *TestToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceAdd(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceSub(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceLock(token, address, amount)
}

func (tt *TestToken) QueryAllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceUnLock(token, address, amount)
}

func (tt *TestToken) QueryAllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceBurnLocked(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	return tt.AllowedBalanceGetAll(address)
}

func TestQuery(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	cc := &TestToken{
		token.BaseToken{
			Symbol: "CC",
		},
	}
	ledgerMock.NewChainCode("cc", cc, nil, nil, owner.Address())

	vt := &TestToken{
		token.BaseToken{
			Symbol: "VT",
		},
	}
	ledgerMock.NewChainCode("vt", vt, nil, nil, owner.Address())

	nt := &TestToken{
		token.BaseToken{
			Symbol: "NT",
		},
	}
	ledgerMock.NewChainCode("nt", nt, nil, nil, owner.Address())

	user1 := ledgerMock.NewWallet()
	user1.AddBalance("cc", 1000)

	user2 := ledgerMock.NewWallet()

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("cc", "swapBegin", "CC", "VT", "450", swapHash)
	user1.BalanceShouldBe("cc", 550)
	ledgerMock.WaitSwapAnswer("vt", txID, time.Second*5)
	user1.Invoke("vt", "swapDone", txID, swapKey)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)

	t.Run("Query allowed balance add  [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceAdd", "CC", user1.Address(), "50", "add balance")
		assert.NoError(t, err)
	})

	t.Run("Query allowed balance sub  [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceSub", "CC", user1.Address(), "50", "sub balance")
		assert.NoError(t, err)
	})

	t.Run("Query allowed balance lock  [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceLock", "CC", user1.Address(), "50")
		assert.NoError(t, err)
	})

	t.Run("Query allowed balance unlock [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceUnLock", "CC", user1.Address(), "50")
		assert.Errorf(t, err, "method PutState is not implemented for query")
	})

	t.Run("Query allowed balance transfer locked [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceTransferLocked", "CC", user1.Address(), user2.Address(), "50", "transfer")
		assert.Errorf(t, err, "method PutState is not implemented for query")

		user2.AllowedBalanceShouldBe("vt", "CC", 0)
	})

	t.Run("Query allowed balance burn locked [negative]", func(t *testing.T) {
		err := owner.InvokeWithError("vt", "allowedBalanceBurnLocked", "CC", user1.Address(), "50", "transfer")
		assert.Errorf(t, err, "method PutState is not implemented for query")
	})

	txID2 := user1.SignedInvoke("cc", "swapBegin", "CC", "VT", "150", swapHash)
	user1.BalanceShouldBe("cc", 400)
	ledgerMock.WaitSwapAnswer("vt", txID2, time.Second*5)
	user1.Invoke("vt", "swapDone", txID2, swapKey)

	t.Run("Allowed balances get all", func(t *testing.T) {
		balance := owner.Invoke("vt", "allowedBalanceGetAll", user1.Address())
		assert.Equal(t, "{\"CC\":\"600\"}", balance)
	})
}
