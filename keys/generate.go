package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/proto"
	"github.com/ddulesov/gogost/gost3410"
)

func GenerateEd25519Keys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pKey, sKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return pKey, sKey, nil
}

func GenerateSecp256k1Keys() (*ecdsa.PublicKey, *ecdsa.PrivateKey, error) {
	sKey, err := eth.NewKey()
	if err != nil {
		return nil, nil, err
	}
	return &sKey.PublicKey, sKey, nil
}

func GenerateGOSTKeys() (*gost3410.PublicKey, *gost3410.PrivateKey, error) {
	sKeyGOST, err := gost3410.GenPrivateKey(
		gost3410.CurveIdGostR34102001CryptoProXchAParamSet(),
		gost3410.Mode2001,
		rand.Reader,
	)
	if err != nil {
		return nil, nil, err
	}

	pKeyGOST, err := sKeyGOST.PublicKey()
	if err != nil {
		return nil, nil, err
	}

	return pKeyGOST, sKeyGOST, nil
}

// GenerateKeysByKeyType generates private and public keys based on specified key type
func GenerateKeysByKeyType(keyType proto.KeyType) (interface{}, interface{}, []byte, []byte, error) {
	var (
		sKey      interface{}
		pKey      interface{}
		sKeyBytes []byte
		pKeyBytes []byte
		err       error
	)
	switch keyType {
	case proto.KeyType_ed25519:
		pKeyBytes, sKeyBytes, err = GenerateEd25519Keys()
		pKey = pKeyBytes
		sKey = sKeyBytes
		if err != nil {
			return nil, nil, nil, nil, err
		}
	case proto.KeyType_secp256k1:
		pKey, sKey, err = GenerateSecp256k1Keys()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		sKeyBytes = eth.PrivateKeyBytes(sKey.(*ecdsa.PrivateKey))
		pKeyBytes = eth.PublicKeyBytes(pKey.(*ecdsa.PublicKey))
	case proto.KeyType_gost:
		pKey, sKey, err = GenerateGOSTKeys()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		sKeyBytes = sKey.(*gost3410.PrivateKey).Raw()
		pKeyBytes = pKey.(*gost3410.PublicKey).Raw()
	}

	return sKey, pKey, sKeyBytes, pKeyBytes, nil
}
