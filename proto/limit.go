package proto

import "github.com/anoideaopen/foundation/core/types/big"

// InLimit checks if the amount is in the limit
func (x *TokenRate) InLimit(amount *big.Int) bool {
	maxLimit := new(big.Int).SetBytes(x.GetMax())
	minLimit := new(big.Int).SetBytes(x.GetMin())

	return amount.Cmp(minLimit) >= 0 && (maxLimit.Cmp(big.NewInt(0)) == 0 || amount.Cmp(maxLimit) <= 0)
}

// CalcPrice calculates the price
func (x *TokenRate) CalcPrice(amount *big.Int, rateDecimal uint64) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(
			amount, new(big.Int).SetBytes(x.GetRate()),
		),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10),
			new(big.Int).SetUint64(rateDecimal), nil,
		),
	)
}
