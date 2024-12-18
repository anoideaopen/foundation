package mockstub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/anoideaopen/foundation/core/types"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
)

// ACL chaincode functions
const (
	FnCheckAddress    = "checkAddress"
	FnCheckKeys       = "checkKeys"
	FnGetAccountInfo  = "getAccountInfo"
	FnGetAccountsInfo = "getAccountsInfo"
)

// Key length
const (
	KeyLengthEd25519   = 32
	KeyLengthSecp256k1 = 65
	KeyLengthGOST      = 64

	PrefixUncompressedSecp259k1Key = 0x04
)

func MockACLCheckAddress(_ *MockStub, parameters ...string) peer.Response {
	address := parameters[0]

	addr, err := types.AddrFromBase58Check(address)
	if err != nil {
		return shim.Error(err.Error())
	}

	data, err := proto.Marshal((*pbfound.Address)(addr))
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func MockACLCheckKeys(_ *MockStub, parameters ...string) peer.Response {
	keysString := parameters[0]

	keys := strings.Split(keysString, "/")
	binPubKeys := make([][]byte, len(keys))
	for i, k := range keys {
		binPubKeys[i] = base58.Decode(k)
	}
	sort.Slice(binPubKeys, func(i, j int) bool {
		return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
	})

	hashed := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
	keyType, err := identifyKeyTypeByLength(binPubKeys[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	data, err := proto.Marshal(&pbfound.AclResponse{
		Account: &pbfound.AccountInfo{
			KycHash:    "123",
			GrayListed: false,
		},
		Address: &pbfound.SignedAddress{
			Address: &pbfound.Address{Address: hashed[:]},
			SignaturePolicy: &pbfound.SignaturePolicy{
				N: 2,
			},
		},
		KeyTypes: []pbfound.KeyType{
			keyType,
		},
	})
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func MockACLGetAccountInfo(_ *MockStub, _ ...string) peer.Response {
	data, err := json.Marshal(&pbfound.AccountInfo{
		KycHash:     "123",
		GrayListed:  false,
		BlackListed: false,
	})
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func MockACLGetAccountsInfo(mockStub *MockStub, parameters ...string) peer.Response {
	responses := make([]peer.Response, 0)
	for _, parameter := range parameters {
		var argsTmp []string
		err := json.Unmarshal([]byte(parameter), &argsTmp)
		if err != nil {
			return shim.Error(err.Error())
		}

		var response peer.Response
		functionName := argsTmp[0]

		if function, ok := mockStub.InvokeACLMap[functionName]; ok {
			response = function(mockStub, argsTmp[1:]...)
		} else {
			return shim.Error("mock stub does not support " + functionName + "function")
		}

		responses = append(responses, response)
	}
	b, err := json.Marshal(responses)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed get accounts info: marshal GetAccountsInfoResponse: %s", err))
	}
	return shim.Success(b)
}

func identifyKeyTypeByLength(key []byte) (pbfound.KeyType, error) {
	switch len(key) {
	case KeyLengthEd25519:
		return pbfound.KeyType_ed25519, nil
	case KeyLengthSecp256k1:
		if key[0] == PrefixUncompressedSecp259k1Key {
			return pbfound.KeyType_secp256k1, nil
		}
		return pbfound.KeyType_ed25519, errors.New("invalid key length")
	case KeyLengthGOST:
		return pbfound.KeyType_gost, nil
	default:
		return pbfound.KeyType_ed25519, errors.New("invalid key length")
	}
}
