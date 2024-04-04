package token

import (
	"testing"

	ma "github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseTokenTxBuy(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}

	ttName, ttSymbol, ttDecimals := "Validation Token", "VT", uint(8)
	config := makeBaseTokenConfig(ttName, ttSymbol, ttDecimals,
		issuer.Address(), feeSetter.Address(), "")

	msg := ledger.NewCC("vt", vt, config)
	require.Empty(t, msg)

	issuer.SignedInvoke("vt", "emitToken", "10")
	issuer.SignedInvoke("vt", "setRate", "buyToken", "usd", "100000000")
	issuer.SignedInvoke("vt", "setLimits", "buyToken", "usd", "1", "10")

	user.AddAllowedBalance("vt", "usd", 5)
	if err := user.RawSignedInvokeWithErrorReturned("vt", "buyToken", "0", "usd"); err != nil {
		assert.Equal(t, "amount should be more than zero", err.Error())
	}
	if err := user.RawSignedInvokeWithErrorReturned("vt", "buyToken", "1", "rub"); err != nil {
		assert.Equal(t, "impossible to buy for this currency", err.Error())
	}
	if err := user.RawSignedInvokeWithErrorReturned("vt", "buyToken", "100", "usd"); err != nil {
		assert.Equal(t, "amount out of limits", err.Error())
	}
	err := user.RawSignedInvokeWithErrorReturned("vt", "buyToken", "1", "usd")
	assert.NoError(t, err)
}
