package token

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/assert"
)

func TestBaseTokenTxBuy(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	vt := &VT{
		BaseToken{
			Name:     "Validation Token",
			Symbol:   "VT",
			Decimals: 8,
		},
	}

	mock.NewChainCode("vt", vt, &core.ContractOptions{}, nil, issuer.Address())

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
