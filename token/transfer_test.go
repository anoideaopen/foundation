package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/assert"
)

const vtName = "Validation Token"

func TestBaseTokenTxTransfer(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	buyer := mock.NewWallet()
	seller := mock.NewWallet()

	vt := &VT{
		BaseToken{
			Name:     vtName,
			Symbol:   "VT",
			Decimals: 8,
		},
	}

	mock.NewChainCode("vt", vt, &core.ContractOptions{}, nil, issuer.Address())

	issuer.SignedInvoke("vt", "emitToken", "10")
	issuer.SignedInvoke("vt", "setRate", "buyToken", "usd", "100000000")
	issuer.SignedInvoke("vt", "setLimits", "buyToken", "usd", "1", "10")

	seller.AddAllowedBalance("vt", "usd", 5)

	err := seller.RawSignedInvokeWithErrorReturned("vt", "buyToken", "5", "usd")
	assert.NoError(t, err)

	if err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "0", ""); err != nil {
		assert.Equal(t, "amount should be more than zero", err.Error())
	}
	if err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "100", ""); err != nil {
		assert.Equal(t, "insufficient funds to process", err.Error())
	}
	err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "5", "")
	assert.NoError(t, err)
}

func TestTransferWithFee(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	feeAddressSetter := mock.NewWallet()
	feeSetter := mock.NewWallet()
	feeAggregator := mock.NewWallet()
	user := mock.NewWallet()

	vt := &VT{
		BaseToken{
			Name:     vtName,
			Symbol:   "VT",
			Decimals: 8,
		},
	}

	mock.NewChainCode("vt", vt, &core.ContractOptions{}, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	issuer.SignedInvoke("vt", "emitToken", "101")

	feeSetter.SignedInvoke("vt", "setFee", "VT", "500000", "1", "0")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	assert.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	assert.EqualError(t, err, "fee address is not set")

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())
	issuer.SignedInvoke("vt", "transfer", user.Address(), "100", "")

	issuer.BalanceShouldBe("vt", 0)
	user.BalanceShouldBe("vt", 100)
	feeAggregator.BalanceShouldBe("vt", 1)
}

func TestAllowedIndustrialBalanceTransfer(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	feeAddressSetter := mock.NewWallet()
	feeSetter := mock.NewWallet()
	user := mock.NewWallet()

	vt := &VT{
		BaseToken{
			Name:     vtName,
			Symbol:   "VT",
			Decimals: 8,
		},
	}

	mock.NewChainCode("vt", vt, &core.ContractOptions{}, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	const (
		ba1 = "BA02_GOLDBARLONDON.01"
		ba2 = "BA02_GOLDBARLONDON.02"
	)

	issuer.AddAllowedBalance("vt", ba1, 100000000)
	issuer.AddAllowedBalance("vt", ba2, 100000000)
	issuer.AllowedBalanceShouldBe("vt", ba1, 100000000)
	issuer.AllowedBalanceShouldBe("vt", ba2, 100000000)

	industrialAssets := []*types.MultiSwapAsset{
		{
			Group:  ba1,
			Amount: "50000000",
		},
		{
			Group:  ba2,
			Amount: "100000000",
		},
	}

	rawGA, err := json.Marshal(industrialAssets)
	assert.NoError(t, err)

	issuer.SignedInvoke("vt", "allowedIndustrialBalanceTransfer", user.Address(), string(rawGA), "ref")
	issuer.AllowedBalanceShouldBe("vt", ba1, 50000000)
	issuer.AllowedBalanceShouldBe("vt", ba2, 0)
	user.AllowedBalanceShouldBe("vt", ba1, 50000000)
	user.AllowedBalanceShouldBe("vt", ba2, 100000000)
}
