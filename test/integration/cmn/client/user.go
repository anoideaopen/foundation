package client

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

type UserFoundation struct {
	PrivateKey         ed25519.PrivateKey
	PublicKey          ed25519.PublicKey
	PublicKeyBase58    string
	AddressBase58Check string
	UserID             string
}

func NewUserFoundation() *UserFoundation {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return &UserFoundation{}
	}
	publicKeyBase58 := base58.Encode(publicKey)
	hash := sha3.Sum256(publicKey)
	addressBase58Check := base58.CheckEncode(hash[1:], hash[0])

	return &UserFoundation{
		PrivateKey:         privateKey,
		PublicKey:          publicKey,
		PublicKeyBase58:    publicKeyBase58,
		AddressBase58Check: addressBase58Check,
		UserID:             "testuser",
	}
}

func UserFoundationFromPrivateKey(privateKey ed25519.PrivateKey) (*UserFoundation, error) {
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("type requireion failed")
	}

	publicKeyBase58 := base58.Encode(publicKey)
	hash := sha3.Sum256(publicKey)
	addressBase58Check := base58.CheckEncode(hash[1:], hash[0])

	return &UserFoundation{
		PrivateKey:         privateKey,
		PublicKey:          publicKey,
		PublicKeyBase58:    publicKeyBase58,
		AddressBase58Check: addressBase58Check,
		UserID:             "testuser",
	}, nil
}

func UserFoundationFromBase58CheckPrivateKey(base58Check string) (*UserFoundation, error) {
	decode, ver, err := base58.CheckDecode(base58Check)
	if err != nil {
		return nil, fmt.Errorf("check decode: %w", err)
	}
	privateKey := ed25519.PrivateKey(append([]byte{ver}, decode...))

	return UserFoundationFromPrivateKey(privateKey)
}

func (u *UserFoundation) Sign(args ...string) (publicKeyBase58 string, signMsg []byte, err error) {
	publicKeyBase58 = u.PublicKeyBase58
	msg := make([]string, 0, len(args)+1)
	msg = append(msg, args...)
	msg = append(msg, publicKeyBase58)

	bytesToSign := sha3.Sum256([]byte(strings.Join(msg, "")))

	signMsg = signMessage(u.PrivateKey, bytesToSign[:])
	err = verifyEd25519(u.PublicKey, bytesToSign[:], signMsg)
	if err != nil {
		return "", nil, err
	}

	return
}

func (u *UserFoundation) SetUserID(id string) {
	if len(id) != 0 {
		u.UserID = id
	}
}

// MultiSig - added multi sign
func MultiSig(args []string, users ...*UserFoundation) (publicKeysBase58 []string, signMsgs [][]byte, err error) {
	msg := make([]string, 0, len(args)+len(users))
	msg = append(msg, args...)
	for _, i := range users {
		msg = append(msg, i.PublicKeyBase58)
		publicKeysBase58 = append(publicKeysBase58, i.PublicKeyBase58)
	}

	bytesToSign := sha3.Sum256([]byte(strings.Join(msg, "")))

	for _, i := range users {
		sMsg := signMessage(i.PrivateKey, bytesToSign[:])
		err = verifyEd25519(i.PublicKey, bytesToSign[:], sMsg)
		if err != nil {
			return nil, nil, err
		}
		signMsgs = append(signMsgs, sMsg)
	}

	return
}

// signMessage - sign arguments with private key in ed25519
func signMessage(privateKey ed25519.PrivateKey, msgToSign []byte) []byte {
	sig := ed25519.Sign(privateKey, msgToSign)
	return sig
}

// verifyEd25519 - verify publicKey with message and signed message
func verifyEd25519(publicKey ed25519.PublicKey, bytesToSign []byte, sMsg []byte) error {
	if !ed25519.Verify(publicKey, bytesToSign, sMsg) {
		err := errors.New("valid signature rejected")
		return err
	}
	return nil
}
