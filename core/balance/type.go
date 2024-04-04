package balance

import "fmt"

// BalanceType represents different types of balance-related state keys in the ledger.
type BalanceType byte //nolint:revive

// String returns the hexadecimal string representation of the BalanceType.
func (ot BalanceType) String() string {
	return fmt.Sprintf("%x", byte(ot))
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
