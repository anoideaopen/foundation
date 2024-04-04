package gost

import (
	"github.com/ddulesov/gogost/gost3410"
)

// Verify verifies the signature for the specified message hash using the public key and
// GOST R 34.10-2012 algorithm on the cryptographic curve id-GostR3410-2001-CryptoPro-XchA-ParamSet.
// Returns true if the signature is true and false otherwise.
func Verify(pubKeyBytes, digest, signature []byte) (bool, error) {
	// Create a new instance of the public key for the GOST R 34.10-2001 algorithm
	// using parameters of the curve id-GostR3410-2001-CryptoPro-XchA-ParamSet.
	publicKey, err := gost3410.NewPublicKey(
		gost3410.CurveIdGostR34102001CryptoProXchAParamSet(),
		gost3410.Mode2001,
		pubKeyBytes,
	)
	if err != nil {
		return false, err
	}

	// Verify the signature, having previously inverted the byte order in the hash and signature.
	// This is necessary for compatibility with the elliptic curve of the Cryptopro Signature Server (with HSM).
	return publicKey.VerifyDigest(
		reverseBytes(digest),
		reverseBytes(signature),
	)
}

// reverseBytes returns a new byte slice that is the inverse of the passed slice.
func reverseBytes(in []byte) []byte {
	n := len(in)
	reversed := make([]byte, n)
	for i, b := range in {
		reversed[n-i-1] = b
	}

	return reversed
}
