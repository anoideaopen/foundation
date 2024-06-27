package eth

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

// Deprecated: use package keys/eth
// NewKey generates new secp256k1 key using Ethereum crypto functions
func NewKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

// Deprecated: use package keys/eth
// PublicKeyBytes returns bytes representation of secp256p1 public key
func PublicKeyBytes(publicKey *ecdsa.PublicKey) []byte {
	return crypto.FromECDSAPub(publicKey)
}

// Deprecated: use package keys/eth
// PrivateKeyFromBytes creates a secp256k1 private key from its bytes representation
func PrivateKeyFromBytes(bytes []byte) (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(bytes)
}

// Deprecated: use package keys/eth
// PrivateKeyBytes returns bytes representation of secp256p1 private key
func PrivateKeyBytes(privateKey *ecdsa.PrivateKey) []byte {
	return crypto.FromECDSA(privateKey)
}
