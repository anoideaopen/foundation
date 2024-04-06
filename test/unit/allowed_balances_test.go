package unit

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

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
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	issuer := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &TestToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		issuer.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("vt", &TestToken{}, vtConfig)
	require.Empty(t, initMsg)

	nt := &TestToken{}
	configNT := fmt.Sprintf(
		`
{
	"symbol": "%s",
	"robotSKI":"%s",
	"admin":{"address":"%s"},
	"token":{
		"name":"%s",
		"decimals":%d,
		"issuer":{"address":"%s"}
	}
}`,
		"NT",
		fixtures_test.RobotHashedCert,
		issuer.Address(),
		"NT Token",
		8,
		issuer.Address(),
	)
	ledger.NewCC("nt", nt, configNT)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	user2 := ledger.NewWallet()

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	txID := user1.SignedInvoke("cc", "swapBegin", "CC", "VT", "450", swapHash)
	user1.BalanceShouldBe("cc", 550)
	ledger.WaitSwapAnswer("vt", txID, time.Second*5)
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
	ledger.WaitSwapAnswer("vt", txID2, time.Second*5)
	user1.Invoke("vt", "swapDone", txID2, swapKey)

	t.Run("Allowed balances get all", func(t *testing.T) {
		balance := owner.Invoke("vt", "allowedBalanceGetAll", user1.Address())
		assert.Equal(t, "{\"CC\":\"600\"}", balance)
	})
}
