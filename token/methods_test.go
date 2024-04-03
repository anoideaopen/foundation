package token

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/assert"
)

func TestBaseTokenSetLimits(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	tt := &BaseToken{
		Name:     "Test Token",
		Symbol:   "TT",
		Decimals: 8,
	}

	mock.NewChainCode("tt", tt, &core.ContractOptions{}, nil, issuer.Address())

	issuer.SignedInvoke("tt", "setRate", "distribute", "", "1")

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "makarone", "", "1", "3"); err != nil {
		assert.Equal(t, "unknown DealType. Rate for deal type makarone and currency  was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "fish", "1", "3"); err != nil {
		assert.Equal(t, "unknown currency. Rate for deal type distribute and currency fish was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "", "10", "3"); err != nil {
		assert.Equal(t, "min limit is greater than max limit", err.Error())
	}

	err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "", "1", "0")
	assert.NoError(t, err)
}

func TestIndustrialTokenSetRate(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	outsider := mock.NewWallet()

	tt := &BaseToken{
		Name:     "Test Token",
		Symbol:   "TT",
		Decimals: 8,
	}

	mock.NewChainCode("tt", tt, &core.ContractOptions{}, nil, issuer.Address())

	if err := outsider.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "1"); err != nil {
		assert.Equal(t, "unauthorized", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "0"); err != nil {
		assert.Equal(t, "trying to set rate = 0", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "TT", "3"); err != nil {
		assert.Equal(t, "currency is equals token: it is impossible", err.Error())
	}
	err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "1")
	assert.NoError(t, err)

	rawMD := issuer.Invoke("tt", "metadata")
	md := &Metadata{}

	assert.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates := md.Rates
	assert.Len(t, rates, 1)

	issuer.SignedInvoke("tt", "deleteRate", "distribute", "")

	rawMD = issuer.Invoke("tt", "metadata")
	md = &Metadata{}

	assert.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates = md.Rates
	assert.Len(t, rates, 0)
}
