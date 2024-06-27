package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/proto"
	"github.com/ddulesov/gogost/gost3410"
)

func signEd25519Validate(privateKeyBytes ed25519.PrivateKey, message []byte) ([]byte, []byte, error) {
	digestSHA3 := getDigestSHA3(message)
	signature := ed25519.Sign(privateKeyBytes, digestSHA3)
	publicKeyBytes, ok := privateKeyBytes.Public().(ed25519.PublicKey)
	if !ok {
		return nil, nil, errors.New("error converting private key to public")
	}
	if !verifyEd25519ByDigest(publicKeyBytes, digestSHA3, signature) {
		return nil, nil, errors.New("ed25519 signature rejected")
	}

	return digestSHA3, signature, nil
}

func signSecp256k1Validate(privateKey ecdsa.PrivateKey, message []byte) ([]byte, []byte, error) {
	digestEth := getDigestEth(message)
	signature, err := eth.Sign(digestEth, &privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error signing message: %w", err)
	}
	publicKey, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("error converting private key to public")
	}
	publicKeyBytes := eth.PublicKeyBytes(publicKey)
	if !verifySecp256k1ByDigest(publicKeyBytes, digestEth, signature) {
		return nil, nil, errors.New("secp256k1 signature rejected")
	}

	return digestEth, signature, nil
}

func signGostValidate(privateKey gost3410.PrivateKey, message []byte) ([]byte, []byte, error) {
	digest := getDigestGost(message)
	digestReverse := reverseBytes(digest)
	signature, err := privateKey.SignDigest(digestReverse, rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("error signing message with GOST key type: %w", err)
	}
	signature = reverseBytes(signature)

	pKey, err := privateKey.PublicKey()
	if err != nil {
		return nil, nil, fmt.Errorf("error calculating GOST public key: %w", err)
	}
	pKeyBytes := pKey.Raw()
	valid, err := verifyGostByDigest(pKeyBytes, digest, signature)
	if err != nil {
		return nil, nil, fmt.Errorf("error verifying GOST signature: %w", err)
	}

	if !valid {
		return nil, nil, errors.New("GOST signature rejected")
	}

	return digestReverse, signature, nil
}

// SignMessageByKeyType signs message depending on specified key type
func SignMessageByKeyType(keyType proto.KeyType, keys *Keys, message []byte) ([]byte, []byte, error) {
	switch keyType {
	case proto.KeyType_ed25519:
		return signEd25519Validate(keys.PrivateKeyEd25519, message)
	case proto.KeyType_secp256k1:
		return signSecp256k1Validate(*keys.PrivateKeySecp256k1, message)
	case proto.KeyType_gost:
		return signGostValidate(*keys.PrivateKeyGOST, message)
	default:
		return nil, nil, fmt.Errorf("unexpected key type: %s", keyType.String())
	}
}

func reverseBytes(in []byte) []byte {
	n := len(in)
	reversed := make([]byte, n)
	for i, b := range in {
		reversed[n-i-1] = b
	}

	return reversed
}
