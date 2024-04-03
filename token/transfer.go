package token

import (
	"encoding/json"
	"errors"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

const (
	feeDecimals = 8
	// RateDecimal is the number of decimal places in the rate
	RateDecimal = 8
)

// TxTransfer transfers tokens from one account to another
func (bt *BaseToken) TxTransfer(sender *types.Sender, to *types.Address, amount *big.Int, _ string) error { // ref
	if sender.Equal(to) {
		return errors.New("impossible operation")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}

	if bt.config.Fee != nil && len(bt.config.FeeAddress) == 0 {
		return errors.New("fee address is not set")
	}

	if err := bt.TokenBalanceTransfer(sender.Address(), to, amount, "transfer"); err != nil {
		return err
	}

	fee, err := bt.calcFee(amount)
	if err != nil {
		return err
	}

	stub := bt.GetStub()
	fullAdr, err := helpers.GetFullAddress(stub, to.String())
	if err != nil {
		return err
	}
	to = (*types.Address)(fullAdr)

	if !sender.Address().IsUserIDSame(to) && fee.Fee.Cmp(new(big.Int).SetInt64(0)) != 0 {
		if types.IsValidAddressLen(bt.config.FeeAddress) && bt.config.Fee != nil && bt.config.Fee.Currency != "" {
			feeAddr := types.AddrFromBytes(bt.config.FeeAddress)
			if bt.config.Fee.Currency == bt.Symbol {
				return bt.TokenBalanceTransfer(sender.Address(), feeAddr, fee.Fee, "transfer fee")
			}
			return bt.AllowedBalanceTransfer(fee.Currency, sender.Address(), feeAddr, fee.Fee, "transfer fee")
		}
	}

	return nil
}

// TxAllowedIndustrialBalanceTransfer transfers tokens from one account to another
func (bt *BaseToken) TxAllowedIndustrialBalanceTransfer(sender *types.Sender, to *types.Address, rawAssets string, _ string) error { // ref
	if sender.Equal(to) {
		return errors.New("impossible operation")
	}

	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}

	var industrialAssets []*types.MultiSwapAsset
	if err := json.Unmarshal([]byte(rawAssets), &industrialAssets); err != nil {
		return err
	}
	assets, err := types.ConvertToAsset(industrialAssets)
	if err != nil {
		return err
	}

	for _, industrialAsset := range assets {
		if new(big.Int).SetBytes(industrialAsset.Amount).Cmp(big.NewInt(0)) == 0 {
			return errors.New("amount should be more than zero")
		}
	}

	return bt.AllowedIndustrialBalanceTransfer(sender.Address(), to, assets, "transfer")
}

// Predict is a struct for fee prediction
type Predict struct {
	// Currency is the currency of the fee
	Currency string `json:"currency"`
	// Fee is the predicted fee
	Fee *big.Int `json:"fee"`
}

// QueryPredictFee returns the predicted fee
func (bt *BaseToken) QueryPredictFee(amount *big.Int) (*Predict, error) {
	return bt.calcFee(amount)
}

// TxSetFee sets the fee
func (bt *BaseToken) TxSetFee(sender *types.Sender, currency string, fee *big.Int, floor *big.Int, cap *big.Int) error {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !sender.Equal(bt.FeeSetter()) {
		return errors.New("unauthorized")
	}
	if fee.Cmp(new(big.Int).SetInt64(100000000)) > 0 { //nolint:gomnd
		return errors.New("fee should be equal or less than 100%")
	}
	if cap.Cmp(big.NewInt(0)) > 0 && floor.Cmp(cap) > 0 {
		return errors.New("incorrect limits")
	}
	return bt.setFee(currency, fee, floor, cap)
}

// TxSetFeeAddress sets the fee address
func (bt *BaseToken) TxSetFeeAddress(sender *types.Sender, address *types.Address) error {
	if !sender.Equal(bt.FeeAddressSetter()) {
		return errors.New("unauthorized")
	}

	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	bt.config.FeeAddress = address.Bytes()
	return bt.saveConfig()
}

func (bt *BaseToken) calcFee(amount *big.Int) (*Predict, error) {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return &Predict{}, err
	}

	if bt.config.Fee == nil || bt.config.Fee.Fee == nil || new(big.Int).SetBytes(bt.config.Fee.Fee).Cmp(big.NewInt(0)) == 0 {
		return &Predict{Fee: big.NewInt(0), Currency: bt.Symbol}, nil
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(
			amount,
			new(big.Int).SetBytes(bt.config.Fee.Fee),
		),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10), //nolint:gomnd
			new(big.Int).SetUint64(feeDecimals),
			nil,
		),
	)

	if bt.config.Fee.Currency != bt.Symbol {
		rate, ok, err := bt.GetRateAndLimits("buyToken", bt.config.Fee.Currency)
		if err != nil {
			return &Predict{}, err
		}
		if !ok {
			return &Predict{}, errors.New("incorrect fee currency")
		}

		fee = new(big.Int).Div(
			new(big.Int).Mul(
				fee,
				new(big.Int).SetBytes(rate.Rate),
			),
			new(big.Int).Exp(
				new(big.Int).SetUint64(10), //nolint:gomnd
				new(big.Int).SetUint64(RateDecimal),
				nil,
			),
		)
	}

	if fee.Cmp(new(big.Int).SetBytes(bt.config.Fee.Floor)) < 0 {
		fee = new(big.Int).SetBytes(bt.config.Fee.Floor)
	}

	cp := new(big.Int).SetBytes(bt.config.Fee.Cap)
	if cp.Cmp(big.NewInt(0)) > 0 && fee.Cmp(cp) > 0 {
		fee = new(big.Int).SetBytes(bt.config.Fee.Cap)
	}

	return &Predict{Fee: fee, Currency: bt.config.Fee.Currency}, nil
}
