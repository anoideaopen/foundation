package token

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/helpers"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

const (
	feeDecimals = 8
	// RateDecimal is the number of decimal places in the rate
	RateDecimal = 8
)

var ErrFeeAddressNotConfigured = errors.New("fee address is not set in token config")

// TxTransfer transfers tokens from one account to another
func (bt *BaseToken) TxTransfer(
	sender *types.Sender,
	recipient *types.Address,
	amount *big.Int,
	_ string, // ref
) error {
	if sender.Equal(recipient) {
		return errors.New("TxTransfer: sender and recipient are same users")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("TxTransfer: amount should be more than zero")
	}

	if err := bt.TokenBalanceTransfer(sender.Address(), recipient, amount, "transfer"); err != nil {
		return fmt.Errorf("TxTransfer: transferring tokens: %w", err)
	}

	if err := bt.transferFee(amount, sender.Address(), recipient); err != nil {
		return fmt.Errorf("TxTransfer: transferring fee for operation: %w", err)
	}

	return nil
}

func (bt *BaseToken) transferFee(
	amount *big.Int,
	sender *types.Address,
	recipient *types.Address,
) error {
	cfg, err := bt.loadConfig()
	if err != nil {
		return err
	}

	if cfg.GetFee() != nil && len(cfg.GetFeeAddress()) == 0 {
		return ErrFeeAddressNotConfigured
	}

	if err = validateFeeConfig(cfg); err != nil {
		return fmt.Errorf("validating fee in config: %w", err)
	}

	fee, err := bt.calcTransferFee(amount, sender, recipient)
	if err != nil {
		return fmt.Errorf("calculating transfer fee: %w", err)
	}

	if fee == nil || fee.Fee == nil || fee.Fee.Sign() != 1 {
		return nil
	}

	feeAddr := types.AddrFromBytes(cfg.GetFeeAddress())
	if cfg.GetFee().GetCurrency() == bt.ContractConfig().GetSymbol() {
		err = bt.TokenBalanceTransfer(sender, feeAddr, fee.Fee, "transfer fee")
		if err != nil {
			return fmt.Errorf(
				"failed to transfer fee from token balance, from %s to %s : %w",
				sender,
				feeAddr,
				err,
			)
		}
	} else {
		err = bt.AllowedBalanceTransfer(fee.Currency, sender, feeAddr, fee.Fee, "transfer fee")
		if err != nil {
			return fmt.Errorf(
				"failed to transfer fee from allowed balance, currency %s, from %s to %s : %w",
				fee.Currency,
				sender,
				feeAddr,
				err,
			)
		}
	}

	return nil
}

func (bt *BaseToken) calcTransferFee(amount *big.Int, sender *types.Address, recipient *types.Address) (*Predict, error) {
	if sender == nil {
		return nil, errors.New("sender can't be nil")
	}
	if recipient == nil {
		return nil, errors.New("recipient can't be nil")
	}
	if amount == nil {
		return nil, errors.New("amount can't be nil")
	}

	if sender.UserID == "" {
		fullSenderAddress, err := helpers.GetFullAddress(bt.GetStub(), sender.String())
		if err != nil {
			return nil, errors.New("failed to recive user id by sender address")
		}
		sender = (*types.Address)(fullSenderAddress)
	}
	if recipient.UserID == "" {
		fullRecipientAddress, err := helpers.GetFullAddress(bt.GetStub(), recipient.String())
		if err != nil {
			return nil, errors.New("failed to recive user id by recipient address")
		}
		recipient = (*types.Address)(fullRecipientAddress)
	}

	fee, err := bt.calcFee(amount)
	if err != nil {
		return nil, err
	}

	if !sender.IsUserIDSame(recipient) && fee.Fee.Sign() == 1 {
		return fee, nil
	}

	return &Predict{Fee: big.NewInt(0), Currency: bt.ContractConfig().GetSymbol()}, nil
}

// TxAllowedIndustrialBalanceTransfer transfers tokens from one account to another
func (bt *BaseToken) TxAllowedIndustrialBalanceTransfer(sender *types.Sender, recipient *types.Address, rawAssets string, _ string) error { // ref
	if sender.Equal(recipient) {
		return errors.New("impossible operation, the sender and recipient of the transfer cannot be equal")
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
		if new(big.Int).SetBytes(industrialAsset.GetAmount()).Cmp(big.NewInt(0)) == 0 {
			return errors.New("amount should be more than zero")
		}
	}

	return bt.AllowedIndustrialBalanceTransfer(sender.Address(), recipient, assets, "transfer")
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
	if !sender.Equal(bt.FeeSetter()) {
		return errors.New("TxSetFee: unauthorized, sender is not a fee setter")
	}

	if fee.Cmp(new(big.Int).SetInt64(100000000)) > 0 { //nolint:gomnd
		return errors.New("TxSetFee: fee should be equal or less than 100%")
	}

	if cap.Cmp(big.NewInt(0)) > 0 && floor.Cmp(cap) > 0 {
		return errors.New("TxSetFee: incorrect limits")
	}

	if err := bt.setFee(currency, fee, floor, cap); err != nil {
		return fmt.Errorf("TxSetFee: setting fee: %w", err)
	}

	return nil
}

// TxSetFeeAddress sets the fee address
func (bt *BaseToken) TxSetFeeAddress(sender *types.Sender, address *types.Address) error {
	if !sender.Equal(bt.FeeAddressSetter()) {
		return errors.New(ErrUnauthorized)
	}

	cfg, err := bt.loadConfig()
	if err != nil {
		return err
	}
	cfg.FeeAddress = address.Bytes()
	return bt.saveConfig(cfg)
}

func (bt *BaseToken) calcFee(amount *big.Int) (*Predict, error) {
	cfg, err := bt.loadConfig()
	if err != nil {
		return &Predict{}, err
	}

	if cfg.GetFee().GetFee() == nil || new(big.Int).SetBytes(cfg.GetFee().GetFee()).Cmp(big.NewInt(0)) == 0 {
		return &Predict{Fee: big.NewInt(0), Currency: bt.ContractConfig().GetSymbol()}, nil
	}

	fee := new(big.Int).Div(
		new(big.Int).Mul(
			amount,
			new(big.Int).SetBytes(cfg.GetFee().GetFee()),
		),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10), //nolint:gomnd
			new(big.Int).SetUint64(feeDecimals),
			nil,
		),
	)

	if cfg.GetFee().GetCurrency() != bt.ContractConfig().GetSymbol() {
		rate, ok, err := bt.GetRateAndLimits("buyToken", cfg.GetFee().GetCurrency())
		if err != nil {
			return &Predict{}, err
		}
		if !ok {
			return &Predict{}, errors.New("incorrect fee currency")
		}

		fee = new(big.Int).Div(
			new(big.Int).Mul(
				fee,
				new(big.Int).SetBytes(rate.GetRate()),
			),
			new(big.Int).Exp(
				new(big.Int).SetUint64(10), //nolint:gomnd
				new(big.Int).SetUint64(RateDecimal),
				nil,
			),
		)
	}

	if fee.Cmp(new(big.Int).SetBytes(cfg.GetFee().GetFloor())) < 0 {
		fee = new(big.Int).SetBytes(cfg.GetFee().GetFloor())
	}

	cp := new(big.Int).SetBytes(cfg.GetFee().GetCap())
	if cp.Cmp(big.NewInt(0)) > 0 && fee.Cmp(cp) > 0 {
		fee = new(big.Int).SetBytes(cfg.GetFee().GetCap())
	}

	return &Predict{Fee: fee, Currency: cfg.GetFee().GetCurrency()}, nil
}

func validateFeeConfig(config *proto.Token) error {
	if config == nil || config.GetFee() == nil {
		return nil
	}

	if len(config.GetFeeAddress()) == 0 {
		return ErrFeeAddressNotConfigured
	}

	if !types.IsValidAddressLen(config.GetFeeAddress()) {
		return fmt.Errorf("config fee address has a wrong len. actual %d but expected %d",
			len(config.GetFeeAddress()),
			types.AddressLength,
		)
	}

	if config.GetFee().GetCurrency() == "" {
		return errors.New("config fee currency can't be empty")
	}

	return nil
}
