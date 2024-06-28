package keys

import (
	"fmt"

	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/keys/gost"
	pb "github.com/anoideaopen/foundation/proto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

const (
	KeyLengthEd25519   = 32
	KeyLengthSecp256k1 = 65
	KeyLengthGOST      = 64
)

const PrefixUncompressedSecp259k1Key = 0x04

func ValidateKeyLength(key []byte) bool {
	if len(key) == KeyLengthEd25519 {
		return true
	}
	if len(key) == KeyLengthSecp256k1 && key[0] == PrefixUncompressedSecp259k1Key {
		return true
	}
	if len(key) == KeyLengthGOST {
		return true
	}
	return false
}

func verifyEd25519ByMessage(publicKeyBytes []byte, message []byte, signature []byte) bool {
	digestSHA3 := getDigestSHA3(message)
	return verifyEd25519ByDigest(publicKeyBytes, digestSHA3, signature)
}

func verifySecp256k1ByMessage(publicKeyBytes []byte, message []byte, signature []byte) bool {
	digestEth := getDigestEth(message)
	return verifySecp256k1ByDigest(publicKeyBytes, digestEth, signature)
}

func verifyGostByMessage(publicKeyBytes []byte, message []byte, signature []byte) (bool, error) {
	digestGOST := getDigestGost(message)
	return verifyGostByDigest(publicKeyBytes, digestGOST, signature)
}

func getDigestSHA3(message []byte) []byte {
	digestSHA3Raw := sha3.Sum256(message)
	return digestSHA3Raw[:]
}

func getDigestEth(message []byte) []byte {
	digestSHA3 := getDigestSHA3(message)
	return eth.Hash(digestSHA3)
}

func getDigestGost(message []byte) []byte {
	digestGOSTRaw := gost.Sum256(message)
	return digestGOSTRaw[:]
}

func verifyEd25519ByDigest(publicKeyBytes, digestSHA3, signature []byte) bool {
	return len(publicKeyBytes) == ed25519.PublicKeySize && ed25519.Verify(publicKeyBytes, digestSHA3, signature)
}

func verifySecp256k1ByDigest(publicKeyBytes, digestEth, signature []byte) bool {
	return eth.Verify(publicKeyBytes, digestEth, signature)
}

func verifyGostByDigest(publicKeyBytes, digestGOST, signature []byte) (bool, error) {
	return gost.Verify(publicKeyBytes, digestGOST, signature)
}

// VerifySignatureByKeyType returns true if signature corresponds to signed message
func VerifySignatureByKeyType(keyType pb.KeyType, publicKeyBytes, message, signature []byte) (bool, error) {
	var err error

	valid := false
	switch keyType {
	case pb.KeyType_ed25519:
		valid = verifyEd25519ByMessage(publicKeyBytes, message, signature)
	case pb.KeyType_secp256k1:
		valid = verifySecp256k1ByMessage(publicKeyBytes, message, signature)
	case pb.KeyType_gost:
		valid, err = verifyGostByMessage(publicKeyBytes, message, signature)
		if err != nil {
			return false, fmt.Errorf("incorrect signature: %w", err)
		}
	default:
		return false, fmt.Errorf("invalid key type: %s", keyType.String())
	}

	return valid, nil
}
