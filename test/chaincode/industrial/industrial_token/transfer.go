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

	if err := it.IndustrialBalanceTransfer(it.ContractConfig().GetSymbol()+"_"+group, sender.Address(), to, amount, "transfer"); err != nil {
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

	cfg, err := it.loadConfig()
	if err != nil {
		return err
	}
	if !sender.Address().IsUserIDSame(to) && len(cfg.GetFeeAddress()) == 32 &&
		cfg.GetFee().GetCurrency() != "" {
		feeAddr := types.AddrFromBytes(cfg.GetFeeAddress())
		if cfg.GetFee().GetCurrency() == it.ContractConfig().GetSymbol() {
			return it.IndustrialBalanceTransfer(it.ContractConfig().GetSymbol()+"_"+group, sender.Address(), feeAddr, fee.Fee, "transfer fee")
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
	cfg, err := it.loadConfig()
	if err != nil {
		return err
	}
	cfg.FeeAddress = address.Bytes()
	return it.saveConfig(cfg)
}

func (it *IndustrialToken) calcFee(amount *big.Int) (Predict, error) {
	cfg, err := it.loadConfig()
	if err != nil {
		return Predict{}, err
	}

	if cfg.GetFee().GetFee() == nil || new(big.Int).SetBytes(cfg.GetFee().GetFee()).Cmp(big.NewInt(0)) == 0 {
		return Predict{Fee: big.NewInt(0), Currency: it.ContractConfig().GetSymbol()}, nil
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(amount, new(big.Int).SetBytes(cfg.GetFee().GetFee())),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10),
			new(big.Int).SetUint64(feeDecimals), nil))

	if cfg.GetFee().GetCurrency() != it.ContractConfig().GetSymbol() {
		rate, ok, err := it.GetRateAndLimits("buyToken", cfg.GetFee().GetCurrency())
		if err != nil {
			return Predict{}, err
		}
		if !ok {
			return Predict{}, errors.New("incorrect fee currency")
		}

		fee = new(big.Int).Div(
			new(big.Int).Mul(
				fee,
				new(big.Int).SetBytes(rate.GetRate()),
			),
			new(big.Int).Exp(
				new(big.Int).SetUint64(10),
				new(big.Int).SetUint64(RateDecimal),
				nil,
			),
		)
	}

	if fee.Cmp(new(big.Int).SetBytes(cfg.GetFee().GetFloor())) < 0 {
		fee = new(big.Int).SetBytes(cfg.GetFee().GetFloor())
	}
	c := new(big.Int).SetBytes(cfg.GetFee().GetCap())
	if c.Cmp(big.NewInt(0)) > 0 && fee.Cmp(c) > 0 {
		fee = new(big.Int).SetBytes(cfg.GetFee().GetCap())
	}

	return Predict{Fee: fee, Currency: cfg.GetFee().GetCurrency()}, nil
}
