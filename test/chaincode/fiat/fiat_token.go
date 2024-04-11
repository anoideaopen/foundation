package main

import (
	"errors"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/token"
)

// FiatToken - base struct
type FiatToken struct {
	token.BaseToken
}

// NewFiatToken creates fiat token
func NewFiatToken() *FiatToken {
	return &FiatToken{token.BaseToken{}}
}

// TxEmit - emits fiat token
func (ft *FiatToken) TxEmit(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := ft.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return ft.EmissionAdd(amount)
}

func (ft *FiatToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", ft.AllowedBalanceAdd(token, address, amount, reason)
}
