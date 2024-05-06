package proto

import "strings"

// TokenSymbol returns the token symbol
func (x *Swap) TokenSymbol() string {
	parts := strings.Split(x.GetToken(), "_")
	return parts[0]
}
