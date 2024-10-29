package token

import (
	"encoding/json"
	"testing"

	ma "github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/require"
)

func TestBaseTokenSetLimits(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	issuer.SignedInvoke("tt", "setRate", "distribute", "", "1")

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "makarone", "", "1", "3"); err != nil {
		require.Equal(t, "unknown DealType. Rate for deal type makarone and currency  was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "fish", "1", "3"); err != nil {
		require.Equal(t, "unknown currency. Rate for deal type distribute and currency fish was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "", "10", "3"); err != nil {
		require.Equal(t, "min limit is greater than max limit", err.Error())
	}

	err := issuer.RawSignedInvokeWithErrorReturned("tt", "setLimits", "distribute", "", "1", "0")
	require.NoError(t, err)
}

func TestIndustrialTokenSetRate(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	outsider := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	if err := outsider.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "1"); err != nil {
		require.Equal(t, "unauthorized", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "0"); err != nil {
		require.Equal(t, "trying to set rate = 0", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "TT", "3"); err != nil {
		require.Equal(t, "currency is equals token: it is impossible", err.Error())
	}
	err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "1")
	require.NoError(t, err)

	rawMD := issuer.Invoke("tt", "metadata")
	md := &Metadata{}

	require.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates := md.Rates
	require.Len(t, rates, 1)

	issuer.SignedInvoke("tt", "deleteRate", "distribute", "")

	rawMD = issuer.Invoke("tt", "metadata")
	md = &Metadata{}

	require.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates = md.Rates
	require.Len(t, rates, 0)
}

func TestMetadataMethods(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), "", "")
	initMsg := ledger.NewCC("tt", tt, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	rsp := user1.Invoke("tt", "metadata")

	var meta Metadata
	err := json.Unmarshal([]byte(rsp), &meta)
	require.NoError(t, err)

	var tokenMethods = []string{"addDocs", "allowedBalanceOf", "lockedAllowedBalanceOf",
		"allowedIndustrialBalanceTransfer", "balanceOf", "lockedAllowedBalancesWithPagination",
		"lockedBalanceOf", "lockedTokenBalancesWithPagination", "buildInfo", "buyBack", "buyToken",
		"cancelCCTransferFrom", "allowedBalancesWithPagination",
		"channelTransferByAdmin", "channelMultiTransferByAdmin", "channelTransferByCustomer", "channelMultiTransferByCustomer", "channelTransferFrom",
		"channelTransferTo", "channelTransfersFrom", "commitCCTransferFrom", "coreChaincodeIDName",
		"createCCTransferTo", "deleteCCTransferFrom", "deleteCCTransferTo", "deleteDoc",
		"deleteRate", "documentsList", "getFeeTransfer", "getLockedAllowedBalance",
		"getLockedTokenBalance", "getNonce", "givenBalance", "givenBalancesWithPagination", "groupBalanceOf",
		"healthCheck", "lockAllowedBalance", "tokenBalancesWithPagination",
		"lockTokenBalance", "metadata", "multiSwapBegin", "multiSwapCancel", "multiSwapGet",
		"nameOfFiles", "predictFee", "setFee", "setFeeAddress", "setLimits", "setRate",
		"srcFile", "srcPartFile", "swapBegin", "swapCancel", "swapGet", "systemEnv", "transfer",
		"unlockAllowedBalance", "healthCheckNb", "unlockTokenBalance", "transferBalance"}
	require.ElementsMatch(t, tokenMethods, meta.Methods)
}
