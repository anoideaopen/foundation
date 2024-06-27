package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"

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

func generateEd25519Keys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func generateSecp256k1Keys() (*ecdsa.PublicKey, *ecdsa.PrivateKey, error) {
	sKey, err := eth.NewKey()
	if err != nil {
		return nil, nil, err
	}
	return &sKey.PublicKey, sKey, nil
}

func generateGOSTKeys() (*gost3410.PublicKey, *gost3410.PrivateKey, error) {
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
func GenerateKeysByKeyType(keyType proto.KeyType) (*Keys, error) {
	keys := &Keys{KeyType: keyType}
	switch keyType {
	case proto.KeyType_ed25519:
		pKey, sKey, err := generateEd25519Keys()
		if err != nil {
			return nil, err
		}
		keys.PrivateKeyEd25519 = sKey
		keys.PublicKeyEd25519 = pKey
		keys.PrivateKeyBytes = sKey
		keys.PublicKeyBytes = pKey
	case proto.KeyType_secp256k1:
		pKey, sKey, err := generateSecp256k1Keys()
		if err != nil {
			return nil, err
		}
		keys.PrivateKeySecp256k1 = sKey
		keys.PublicKeySecp256k1 = pKey
		keys.PrivateKeyBytes = eth.PrivateKeyBytes(sKey)
		keys.PublicKeyBytes = eth.PublicKeyBytes(pKey)
	case proto.KeyType_gost:
		pKey, sKey, err := generateGOSTKeys()
		if err != nil {
			return nil, err
		}
		keys.PrivateKeyGOST = sKey
		keys.PublicKeyGOST = pKey
		keys.PrivateKeyBytes = sKey.Raw()
		keys.PublicKeyBytes = pKey.Raw()
	}

	keys.PublicKeyBase58 = base58.Encode(keys.PublicKeyBytes)
	return keys, nil
}

// GenerateAllKeys generates all kind of keys
func GenerateAllKeys() (*Keys, error) {
	var err error

	keys := &Keys{}
	keys.PublicKeyEd25519, keys.PrivateKeyEd25519, err = generateEd25519Keys()
	if err != nil {
		return nil, err
	}

	keys.PublicKeySecp256k1, keys.PrivateKeySecp256k1, err = generateSecp256k1Keys()
	if err != nil {
		return nil, err
	}

	keys.PublicKeyGOST, keys.PrivateKeyGOST, err = generateGOSTKeys()
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// GenerateEd25519FromBase58 generates ed25519 key from base58 encoded string
func GenerateEd25519FromBase58(base58encoded string) (*Keys, error) {
	keys := &Keys{}
	decoded, ver, err := base58.CheckDecode(base58encoded)
	if err != nil {
		return nil, err
	}
	sKey := ed25519.PrivateKey(append([]byte{ver}, decoded...))
	pKey, ok := sKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("error converting private key to public")
	}

	keys.KeyType = proto.KeyType_ed25519
	keys.PrivateKeyEd25519 = sKey
	keys.PublicKeyEd25519 = pKey
	return keys, nil
}

// GenerateEd25519FromHex generates ed25519 key from base58 encoded string
func GenerateEd25519FromHex(hexEncoded string) (*Keys, error) {
	keys := &Keys{}
	decoded, err := hex.DecodeString(hexEncoded)
	if err != nil {
		return nil, err
	}
	sKey := ed25519.PrivateKey(decoded)
	pKey, ok := sKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("error converting private key to public")
	}

	keys.KeyType = proto.KeyType_ed25519
	keys.PrivateKeyEd25519 = sKey
	keys.PublicKeyEd25519 = pKey
	return keys, nil
}
