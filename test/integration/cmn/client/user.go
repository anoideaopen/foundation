package client

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/keys"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type UserFoundation struct {
	*keys.Keys
	AddressBase58Check string
	UserID             string
}

func NewUserFoundation(keyType pbfound.KeyType) (*UserFoundation, error) {
	keysStr, err := keys.GenerateKeysByKeyType(keyType)
	if err != nil {
		return nil, err
	}

	hash := sha3.Sum256(keysStr.PublicKeyBytes)
	addressBase58Check := base58.CheckEncode(hash[1:], hash[0])

	return &UserFoundation{
		Keys:               keysStr,
		AddressBase58Check: addressBase58Check,
		UserID:             "testuser",
	}, nil
}

func UserFoundationFromEd25519PrivateKey(privateKey ed25519.PrivateKey) (*UserFoundation, error) {
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("type requireion failed")
	}

	publicKeyBase58 := base58.Encode(publicKey)
	hash := sha3.Sum256(publicKey)
	addressBase58Check := base58.CheckEncode(hash[1:], hash[0])

	return &UserFoundation{
		Keys: &keys.Keys{
			KeyType:           pbfound.KeyType_ed25519,
			PrivateKeyEd25519: privateKey,
			PublicKeyEd25519:  publicKey,
			PrivateKeyBytes:   privateKey,
			PublicKeyBytes:    publicKey,
			PublicKeyBase58:   publicKeyBase58,
		},
		AddressBase58Check: addressBase58Check,
		UserID:             "testuser",
	}, nil
}

func UserFoundationFromEd25519Base58CheckPrivateKey(base58Check string) (*UserFoundation, error) {
	decode, ver, err := base58.CheckDecode(base58Check)
	if err != nil {
		return nil, fmt.Errorf("check decode: %w", err)
	}
	privateKey := ed25519.PrivateKey(append([]byte{ver}, decode...))

	return UserFoundationFromEd25519PrivateKey(privateKey)
}

func (u *UserFoundation) Sign(args ...string) (publicKeyBase58 string, signMsg []byte, err error) {
	publicKeyBase58 = u.PublicKeyBase58
	msg := make([]string, 0, len(args)+1)
	msg = append(msg, args...)
	msg = append(msg, publicKeyBase58)

	message := []byte(strings.Join(msg, ""))

	if signMsg, err = u.sign(message); err != nil {
		return "", nil, err
	}

	return
}

func (u *UserFoundation) sign(message []byte) (signMsg []byte, err error) {
	_, signature, err := keys.SignMessageByKeyType(u.KeyType, u.Keys, message)
	return signature, err
}

func (u *UserFoundation) SetUserID(id string) {
	if len(id) != 0 {
		u.UserID = id
	}
}
