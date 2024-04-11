package industrialtoken

import (
	"errors"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

// Decimals const
const (
	feeDecimals = 8
	RateDecimal = 8
)

type Predict struct {
	Currency string   `json:"currency"`
	Fee      *big.Int `json:"fee"`
}

// TxTransferIndustrial transfers token to user address
func (it *IndustrialToken) TxTransferIndustrial(sender *types.Sender, to *types.Address, group string, amount *big.Int, _ string) error {
	if sender.Equal(to) {
		return errors.New("impossible operation")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := it.IndustrialBalanceTransfer(it.ContractConfig().Symbol+"_"+group, sender.Address(), to, amount, "transfer"); err != nil {
		return err
	}

	fee, err := it.calcFee(amount)
	if err != nil {
		return err
	}

	stub := it.GetStub()
	fullAdr, err := helpers.GetFullAddress(stub, to.String())
	if err != nil {
		return err
	}
	to = (*types.Address)(fullAdr)

	if !sender.Address().IsUserIDSame(to) && len(it.config.FeeAddress) == 32 && it.config.Fee != nil && it.config.Fee.Currency != "" {
		feeAddr := types.AddrFromBytes(it.config.FeeAddress)
		if it.config.Fee.Currency == it.ContractConfig().Symbol {
			return it.IndustrialBalanceTransfer(it.ContractConfig().Symbol+"_"+group, sender.Address(), feeAddr, fee.Fee, "transfer fee")
		}
		return it.AllowedBalanceTransfer(fee.Currency, sender.Address(), feeAddr, fee.Fee, "transfer fee")
	}

	return nil
}

// QueryPredictFee calculates fee
func (it *IndustrialToken) QueryPredictFee(amount *big.Int) (Predict, error) {
	return it.calcFee(amount)
}

// TxSetFee sets fee values to config
func (it *IndustrialToken) TxSetFee(sender *types.Sender, currency string, fee *big.Int, floor *big.Int, cap *big.Int) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !sender.Address().Equal(it.FeeSetter()) {
		return errors.New("unauthorized")
	}
	if cap.Cmp(big.NewInt(0)) > 0 && floor.Cmp(cap) > 0 {
		return errors.New("incorrect limits")
	}
	return it.setFee(currency, fee, floor, cap)
}

// TxSetFeeAddress sets fee address
func (it *IndustrialToken) TxSetFeeAddress(sender *types.Sender, address *types.Address) error {
	if !sender.Address().Equal(it.FeeAddressSetter()) {
		return errors.New("unauthorized")
	}
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	it.config.FeeAddress = address.Bytes()
	return it.saveConfig()
}

func (it *IndustrialToken) calcFee(amount *big.Int) (Predict, error) {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return Predict{}, err
	}

	if it.config.Fee == nil || it.config.Fee.Fee == nil || new(big.Int).SetBytes(it.config.Fee.Fee).Cmp(big.NewInt(0)) == 0 {
		return Predict{Fee: big.NewInt(0), Currency: it.ContractConfig().Symbol}, nil
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(amount, new(big.Int).SetBytes(it.config.Fee.Fee)),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10),
			new(big.Int).SetUint64(feeDecimals), nil))

	if it.config.Fee.Currency != it.ContractConfig().Symbol {
		rate, ok, err := it.GetRateAndLimits("buyToken", it.config.Fee.Currency)
		if err != nil {
			return Predict{}, err
		}
		if !ok {
			return Predict{}, errors.New("incorrect fee currency")
		}

		fee = new(big.Int).Div(new(big.Int).Mul(fee, new(big.Int).SetBytes(rate.Rate)), new(big.Int).Exp(new(big.Int).SetUint64(10), new(big.Int).SetUint64(RateDecimal), nil))
	}

	if fee.Cmp(new(big.Int).SetBytes(it.config.Fee.Floor)) < 0 {
		fee = new(big.Int).SetBytes(it.config.Fee.Floor)
	}
	c := new(big.Int).SetBytes(it.config.Fee.Cap)
	if c.Cmp(big.NewInt(0)) > 0 && fee.Cmp(c) > 0 {
		fee = new(big.Int).SetBytes(it.config.Fee.Cap)
	}

	return Predict{Fee: fee, Currency: it.config.Fee.Currency}, nil
}
