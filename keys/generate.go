package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"

	"github.com/anoideaopen/foundation/keys/eth"
	"github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/ddulesov/gogost/gost3410"
)

type Keys struct {
	KeyType             proto.KeyType
	PublicKeyEd25519    ed25519.PublicKey
	PrivateKeyEd25519   ed25519.PrivateKey
	PublicKeySecp256k1  *ecdsa.PublicKey
	PrivateKeySecp256k1 *ecdsa.PrivateKey
	PublicKeyGOST       *gost3410.PublicKey
	PrivateKeyGOST      *gost3410.PrivateKey
	PublicKeyBytes      []byte
	PrivateKeyBytes     []byte
	PublicKeyBase58     string
}

func GenerateEd25519Keys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
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
func GenerateKeysByKeyType(keyType proto.KeyType) (Keys, error) {
	var keys Keys
	switch keyType {
	case proto.KeyType_ed25519:
		pKey, sKey, err := GenerateEd25519Keys()
		if err != nil {
			return keys, err
		}
		keys.PrivateKeyEd25519 = sKey
		keys.PublicKeyEd25519 = pKey
		keys.PrivateKeyBytes = sKey
		keys.PublicKeyBytes = pKey
	case proto.KeyType_secp256k1:
		pKey, sKey, err := GenerateSecp256k1Keys()
		if err != nil {
			return keys, err
		}
		keys.PrivateKeySecp256k1 = sKey
		keys.PublicKeySecp256k1 = pKey
		keys.PrivateKeyBytes = eth.PrivateKeyBytes(sKey)
		keys.PublicKeyBytes = eth.PublicKeyBytes(pKey)
	case proto.KeyType_gost:
		pKey, sKey, err := GenerateGOSTKeys()
		if err != nil {
			return keys, err
		}
		keys.PrivateKeyGOST = sKey
		keys.PublicKeyGOST = pKey
		keys.PrivateKeyBytes = sKey.Raw()
		keys.PublicKeyBytes = pKey.Raw()
	}

	keys.PublicKeyBase58 = base58.Encode(keys.PublicKeyBytes)

	return keys, nil
}
