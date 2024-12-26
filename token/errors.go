package token

// Errors
const (
	ErrUnauthorized           = "unauthorized"
	ErrAmountEqualZero        = "amount should be more than zero"
	ErrWrongCurrency          = "impossible to buy for this currency"
	ErrMinLimitGreaterThanMax = "min limit is greater than max limit"
	ErrAmountOutOfLimits      = "amount out of limits"
)
