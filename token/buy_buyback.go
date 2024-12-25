package token

import (
	"errors"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

// CheckLimitsAndPrice checks limits and price
func (bt *BaseToken) CheckLimitsAndPrice(method string, amount *big.Int, currency string) (*big.Int, error) {
	rate, exists, err := bt.GetRateAndLimits(method, currency)
	if err != nil {
		return big.NewInt(0), err
	}
	if !exists {
		return big.NewInt(0), errors.New("impossible to buy for this currency")
	}
	if !rate.InLimit(amount) {
		return big.NewInt(0), errors.New(ErrAmountOutOfLimits)
	}
	return rate.CalcPrice(amount, RateDecimal), nil
}

// TxBuyToken buys tokens for an asset
func (bt *BaseToken) TxBuyToken(sender *types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(bt.Issuer()) {
		return errors.New("impossible operation")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	price, err := bt.CheckLimitsAndPrice("buyToken", amount, currency)
	if err != nil {
		return err
	}

	if err = bt.AllowedBalanceTransfer(currency, sender.Address(), bt.Issuer(), price, "buyToken"); err != nil {
		return err
	}

	return bt.TokenBalanceTransfer(bt.Issuer(), sender.Address(), amount, "buyToken")
}

// TxBuyBack buys back tokens for an asset
func (bt *BaseToken) TxBuyBack(sender *types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(bt.Issuer()) {
		return errors.New("impossible operation")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	price, err := bt.CheckLimitsAndPrice("buyBack", amount, currency)
	if err != nil {
		return err
	}

	if err = bt.AllowedBalanceTransfer(currency, bt.Issuer(), sender.Address(), price, "buyBack"); err != nil {
		return err
	}

	return bt.TokenBalanceTransfer(sender.Address(), bt.Issuer(), amount, "buyBack")
}
