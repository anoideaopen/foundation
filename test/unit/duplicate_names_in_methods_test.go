package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/routing/reflectx"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestDuplicateNames(t *testing.T) {
	tt := []struct {
		name string
		bci  core.BaseContractInterface
		err  error
	}{
		{
			name: "no duplicated functions",
			bci:  &token.BaseToken{},
			err:  nil,
		},
		{
			name: "variant #1",
			bci:  &DuplicateNamesT1{},
			err:  reflectx.ErrMethodAlreadyDefined,
		},
		{
			name: "variant #2",
			bci:  &DuplicateNamesT2{},
			err:  reflectx.ErrMethodAlreadyDefined,
		},
		{
			name: "variant #3",
			bci:  &DuplicateNamesT3{},
			err:  reflectx.ErrMethodAlreadyDefined,
		},
	}

	t.Parallel()
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			_, err := core.NewCC(test.bci)
			require.ErrorIs(t, err, test.err)
		})
	}
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
