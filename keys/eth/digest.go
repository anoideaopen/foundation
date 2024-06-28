package eth

import "github.com/ethereum/go-ethereum/accounts"

// Hash calculates a hash for given message using Ethereum crypto functions
func Hash(message []byte) []byte {
	return accounts.TextHash(message)
}
