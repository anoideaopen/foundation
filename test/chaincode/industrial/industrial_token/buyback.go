package industrialtoken

import (
	"errors"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

// CheckLimitsAndPrice - checks limits and prices
func (it *IndustrialToken) CheckLimitsAndPrice(
	method string,
	amount *big.Int,
	currency string,
) (*big.Int, error) {
	rate, exists, err := it.GetRateAndLimits(method, currency)
	if err != nil {
		return big.NewInt(0), err
	}
	if !exists {
		return big.NewInt(0), errors.New("impossible to buy for this currency")
	}
	if !rate.InLimit(amount) {
		return big.NewInt(0), errors.New("amount out of limits")
	}
	return rate.CalcPrice(amount, RateDecimal), nil
}

// TxIndustrialBuyBack - method for token buyback
func (it *IndustrialToken) TxIndustrialBuyBack(
	sender *types.Sender,
	group string,
	amount *big.Int,
	currency string,
) error {
	if sender.Equal(it.Issuer()) {
		return errors.New("impossible operation")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	price, err := it.CheckLimitsAndPrice("buyBack", amount, currency)
	if err != nil {
		return err
	}

	if err = it.AllowedBalanceTransfer(currency, it.Issuer(), sender.Address(), price, "buyBack"); err != nil {
		return err
	}

	return it.IndustrialBalanceTransfer(
		it.ContractConfig().GetSymbol()+"_"+group,
		sender.Address(),
		it.Issuer(),
		amount,
		"buyBack",
	)
}
