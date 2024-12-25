package token

import (
	"errors"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
)

// VT is a test token
type VT struct {
	// BaseToken is the base token
	BaseToken
}

// TxEmitToken emits tokens
func (vt *VT) TxEmitToken(sender *types.Sender, amount *big.Int) error {
	if !sender.Equal(vt.Issuer()) {
		return errors.New(ErrUnauthorized)
	}
	if err := vt.TokenBalanceAdd(vt.Issuer(), amount, "emitToken"); err != nil {
		return err
	}
	return vt.EmissionAdd(amount)
}
