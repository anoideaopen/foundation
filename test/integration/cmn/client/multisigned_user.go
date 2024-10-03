package client

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"sort"
	"strings"

	"github.com/anoideaopen/foundation/keys"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

const MultisignKeyDelimiter = "/"

type UserFoundationMultisigned struct {
	Users              []*UserFoundation
	AddressBase58Check string
	UserID             string
}

type PrivateKeyWithType struct {
	KeyType         string
	PrivateKeyBytes []byte
}

// NewUserFoundationMultisigned creates multisigned user based on specified key type and policy
func NewUserFoundationMultisigned(keyType pbfound.KeyType, n int) (*UserFoundationMultisigned, error) {
	var pKeys [][]byte
	userMultisigned := &UserFoundationMultisigned{
		Users:  make([]*UserFoundation, 0),
		UserID: "testUserMultisigned",
	}
	for i := 0; i < n; i++ {
		user, err := NewUserFoundation(keyType)
		if err != nil {
			return nil, err
		}
		userMultisigned.Users = append(userMultisigned.Users, user)
		pKeys = append(pKeys, user.PublicKeyBytes)
	}

	binPubKeys := make([][]byte, len(pKeys))
	copy(binPubKeys, pKeys)
	sort.Slice(binPubKeys, func(i, j int) bool {
		return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
	})

	hashedAddr := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
	userMultisigned.AddressBase58Check = base58.CheckEncode(hashedAddr[1:], hashedAddr[0])
	return userMultisigned, nil
}

func UserFoundationMultisignedFromEd25519PrivateKeys(keys []PrivateKeyWithType) (*UserFoundationMultisigned, error) {
	var pKeys [][]byte
	userMultisigned := &UserFoundationMultisigned{
		Users:  make([]*UserFoundation, 0),
		UserID: "testUserMultisigned",
	}
	for _, keyWithType := range keys {
		user, err := UserFoundationFromEd25519PrivateKey(keyWithType.PrivateKeyBytes)
		if err != nil {
			return nil, err
		}
		userMultisigned.Users = append(userMultisigned.Users, user)
		pKeys = append(pKeys, user.PublicKeyBytes)
	}
	binPubKeys := make([][]byte, len(pKeys))
	copy(binPubKeys, pKeys)
	sort.Slice(binPubKeys, func(i, j int) bool {
		return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
	})

	hashedAddr := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
	userMultisigned.AddressBase58Check = base58.CheckEncode(hashedAddr[1:], hashedAddr[0])
	return userMultisigned, nil
}

func UserFoundationMultisignedFromBase58CheckPrivateKey(keysBase58Check []string) (*UserFoundationMultisigned, error) {
	var privateKeys []PrivateKeyWithType
	for _, keyBase58Check := range keysBase58Check {
		decode, ver, err := base58.CheckDecode(keyBase58Check)
		if err != nil {
			return nil, fmt.Errorf("check decode: %w", err)
		}
		privateKey := ed25519.PrivateKey(append([]byte{ver}, decode...))
		privateKeys = append(privateKeys, PrivateKeyWithType{KeyType: pbfound.KeyType_ed25519.String(), PrivateKeyBytes: privateKey})
	}

	return UserFoundationMultisignedFromEd25519PrivateKeys(privateKeys)
}

// Sign adds sign for multisigned user
func (u *UserFoundationMultisigned) Sign(args ...string) (publicKeysBase58 []string, signMsgs [][]byte, err error) {
	msg := make([]string, 0, len(args)+len(u.Users))
	msg = append(msg, args...)
	for _, user := range u.Users {
		msg = append(msg, user.PublicKeyBase58)
		publicKeysBase58 = append(publicKeysBase58, user.PublicKeyBase58)
	}

	message := []byte(strings.Join(msg, ""))

	for _, user := range u.Users {
		_, signature, err := keys.SignMessageByKeyType(user.KeyType, user.Keys, message)
		if err != nil {
			return nil, nil, err
		}
		signMsgs = append(signMsgs, signature)
	}

	return
}

func (u *UserFoundationMultisigned) SignWithUsers(users []*UserFoundation, args ...string) (publicKeysBase58 []string, signMsgs [][]byte, err error) {
	msg := make([]string, 0, len(args)+len(users))
	msg = append(msg, args...)
	for _, user := range users {
		msg = append(msg, user.PublicKeyBase58)
		publicKeysBase58 = append(publicKeysBase58, user.PublicKeyBase58)
	}

	message := []byte(strings.Join(msg, ""))

	for _, user := range users {
		_, signature, err := keys.SignMessageByKeyType(user.KeyType, user.Keys, message)
		if err != nil {
			return nil, nil, err
		}
		signMsgs = append(signMsgs, signature)
	}

	return
}

// PublicKey - returns public key for multisigned user based on keys of its users
func (u *UserFoundationMultisigned) PublicKey() string {
	var multisignedKeys string
	for _, user := range u.Users {
		multisignedKeys = multisignedKeys + user.PublicKeyBase58 + MultisignKeyDelimiter
	}

	return strings.TrimRight(multisignedKeys, MultisignKeyDelimiter)
}
