package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/ddulesov/gogost/gost3410"

	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/proto"
)

func signEd25519Validate(privateKeyBytes ed25519.PrivateKey, message []byte) ([]byte, error) {
	digestSHA3 := getDigestSHA3(message)
	signature := ed25519.Sign(privateKeyBytes, digestSHA3)
	publicKeyBytes := privateKeyBytes.Public().(ed25519.PublicKey)
	if !verifyEd25519ByDigest(publicKeyBytes, digestSHA3, signature) {
		return nil, errors.New("ed25519 signature rejected")
	}

	return signature, nil
}

func signSecp256k1Validate(privateKey ecdsa.PrivateKey, message []byte) ([]byte, error) {
	digestEth := getDigestEth(message)
	signature, err := eth.Sign(digestEth, &privateKey)
	if err != nil {
		return nil, fmt.Errorf("error signing message: %w", err)
	}
	publicKeyBytes := eth.PublicKeyBytes(privateKey.Public().(*ecdsa.PublicKey))
	if !verifySecp256k1ByDigest(publicKeyBytes, digestEth, signature) {
		return nil, errors.New("secp256k1 signature rejected")
	}

	return signature, nil
}

func signGostValidate(privateKey gost3410.PrivateKey, message []byte) ([]byte, error) {
	digest := getDigestGost(message)
	digestReverse := reverseBytes(digest)
	signature, err := privateKey.SignDigest(digestReverse, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error signing message with GOST key type: %w", err)
	}
	signature = reverseBytes(signature)

	pKey, err := privateKey.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("error calculating GOST public key: %w", err)
	}
	pKeyBytes := pKey.Raw()
	valid, err := verifyGostByDigest(pKeyBytes, digest, signature)
	if err != nil {
		return nil, fmt.Errorf("error verifying GOST signature: %w", err)
	}

	if !valid {
		return nil, errors.New("GOST signature rejected")
	}

	return signature, nil
}

// SignMessageByKeyType signs message depending on specified key type
func SignMessageByKeyType(keyType proto.KeyType, sKey interface{}, message []byte) ([]byte, error) {
	var (
		signature []byte
		err       error
	)
	switch keyType {
	case proto.KeyType_ed25519:
		signature, err = signEd25519Validate(sKey.([]byte), message)
		if err != nil {
			return nil, err
		}
	case proto.KeyType_secp256k1:
		signature, err = signSecp256k1Validate(*sKey.(*ecdsa.PrivateKey), message)
		if err != nil {
			return nil, err
		}
	case proto.KeyType_gost:
		signature, err = signGostValidate(*sKey.(*gost3410.PrivateKey), message)
	default:
		return nil, fmt.Errorf("unexpected key type: %s", keyType.String())
	}

	return signature, nil
}

func reverseBytes(in []byte) []byte {
	n := len(in)
	reversed := make([]byte, n)
	for i, b := range in {
		reversed[n-i-1] = b
	}

	return reversed
}
