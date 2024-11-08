package balance

import (
	"fmt"
	"strconv"
)

// BalanceType represents different types of balance-related state keys in the ledger.
type BalanceType byte

// String returns the hexadecimal string representation of the BalanceType.
func (ot BalanceType) String() string {
	return strconv.FormatUint(uint64(ot), 16)
}

// Constants for different BalanceType values representing various balance state keys.
const (
	BalanceTypeToken                 BalanceType = 0x2b
	BalanceTypeTokenLocked           BalanceType = 0x2e
	BalanceTypeTokenExternalLocked   BalanceType = 0x32
	BalanceTypeAllowed               BalanceType = 0x2c
	BalanceTypeAllowedLocked         BalanceType = 0x2f
	BalanceTypeAllowedExternalLocked BalanceType = 0x31
	BalanceTypeGiven                 BalanceType = 0x2d
)

// BalanceTypeToStringMapValue returns string map value of the BalanceType
func BalanceTypeToStringMapValue(ot BalanceType) (string, error) {
	balanceTypeToStringMap := map[BalanceType]string{
		BalanceTypeToken:                 "Token",
		BalanceTypeTokenLocked:           "TokenLocked",
		BalanceTypeTokenExternalLocked:   "TokenExternalLocked",
		BalanceTypeAllowed:               "Allowed",
		BalanceTypeAllowedLocked:         "AllowedLocked",
		BalanceTypeAllowedExternalLocked: "AllowedExternalLocked",
		BalanceTypeGiven:                 "Given",
	}

	// Look up the BalanceType in the map.
	s, ok := balanceTypeToStringMap[ot]
	if !ok {
		return "", fmt.Errorf("unknown BalanceType: %s", ot.String())
	}

	return s, nil
}

// StringToBalanceType converts a string representation of a balance state key to its corresponding BalanceType.
func StringToBalanceType(s string) (BalanceType, error) {
	stringToBalanceTypeMap := map[string]BalanceType{
		"Token":                 BalanceTypeToken,
		"TokenLocked":           BalanceTypeTokenLocked,
		"TokenExternalLocked":   BalanceTypeTokenExternalLocked,
		"Allowed":               BalanceTypeAllowed,
		"AllowedLocked":         BalanceTypeAllowedLocked,
		"AllowedExternalLocked": BalanceTypeAllowedExternalLocked,
		"Given":                 BalanceTypeGiven,
	}

	// Look up the BalanceType in the map.
	ot, ok := stringToBalanceTypeMap[s]
	if !ok {
		return 0, fmt.Errorf("unknown BalanceType string: %s", s)
	}

	return ot, nil
}
