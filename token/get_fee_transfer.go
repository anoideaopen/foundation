package token

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

type FeeTransferRequestDTO struct {
	SenderAddress    *types.Address `json:"sender_address,omitempty"`
	RecipientAddress *types.Address `json:"recipient_address,omitempty"`
	Amount           *big.Int       `json:"amount,omitempty"`
}

func (r FeeTransferRequestDTO) Validate() error {
	if r.SenderAddress == nil || r.SenderAddress.String() == "" {
		return errors.New("sender address can't be empty")
	}
	if r.RecipientAddress == nil || r.RecipientAddress.String() == "" {
		return errors.New("recipient address can't be empty")
	}
	if r.Amount == nil || r.Amount.Cmp(big.NewInt(0)) < 0 {
		return errors.New("amount must be non-negative")
	}
	return nil
}

type FeeTransferResponseDTO struct {
	FeeAddress *types.Address `json:"fee_address,omitempty"`
	Amount     *big.Int       `json:"amount,omitempty"`
	Currency   string         `json:"currency,omitempty"`
}

func (bt *BaseToken) QueryGetFeeTransfer(req FeeTransferRequestDTO) (*FeeTransferResponseDTO, error) {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := validateFeeConfig(bt.config); err != nil {
		return nil, fmt.Errorf("failed to validate config for fee: %w", err)
	}

	if len(bt.config.GetFeeAddress()) == 0 {
		return nil, ErrFeeAddressNotConfigured
	}

	fee, err := bt.calcTransferFee(req.Amount, req.SenderAddress, req.RecipientAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to calc transfer fee: %w", err)
	}

	resp := &FeeTransferResponseDTO{
		FeeAddress: types.AddrFromBytes(bt.config.GetFeeAddress()),
		Amount:     fee.Fee,
		Currency:   fee.Currency,
	}
	return resp, nil
}
