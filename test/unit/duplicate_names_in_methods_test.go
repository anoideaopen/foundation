package unit

import (
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
)

func TestDuplicateNames(t *testing.T) {
	chName := "dn"

	t.Run("variant 1", func(t *testing.T) {
		dn := &DuplicateNamesT1{
			token.BaseToken{
				Symbol: strings.ToUpper(chName),
			},
		}

		_, err := core.NewCC(dn, nil)
		assert.ErrorContains(t, err, core.ErrMethodAlreadyDefined)
	})

	t.Run("variant 2", func(t *testing.T) {
		dn := &DuplicateNamesT2{
			token.BaseToken{
				Symbol: strings.ToUpper(chName),
			},
		}

		_, err := core.NewCC(dn, nil)
		assert.ErrorContains(t, err, core.ErrMethodAlreadyDefined)
	})

	t.Run("variant 3", func(t *testing.T) {
		dn := &DuplicateNamesT3{
			token.BaseToken{
				Symbol: strings.ToUpper(chName),
			},
		}

		_, err := core.NewCC(dn, nil)
		assert.ErrorContains(t, err, core.ErrMethodAlreadyDefined)
	})
}

// Tokens with some duplicate names in methods

type DuplicateNamesT1 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT1) NBTxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT1) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", dnt.AllowedBalanceAdd(token, address, amount, reason)
}

type DuplicateNamesT2 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT2) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT2) TxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

type DuplicateNamesT3 struct {
	token.BaseToken
}

func (dnt *DuplicateNamesT3) NBTxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}

func (dnt *DuplicateNamesT3) TxAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return dnt.AllowedBalanceAdd(token, address, amount, reason)
}
