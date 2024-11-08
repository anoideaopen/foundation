package mocks

import (
	"encoding/json"

	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
)

func MockGetACLResponse(user *UserFoundation) (peer.Response, error) {
	ownerAddress := sha3.Sum256(user.PublicKeyBytes)
	addressBytes := ownerAddress[:]

	accountInfo := getAccountInfo()

	aclResp, err := proto.Marshal(&pbfound.AclResponse{
		Account: &accountInfo,
		Address: &pbfound.SignedAddress{
			Address: &pbfound.Address{
				UserID:       user.UserID,
				Address:      addressBytes,
				IsIndustrial: false,
				IsMultisig:   false,
			},
			SignedTx:        nil,
			SignaturePolicy: nil,
			Reason:          "",
			ReasonId:        0,
			AdditionalKeys:  nil,
		},
		KeyTypes: []pbfound.KeyType{user.KeyType},
	})
	if err != nil {
		return peer.Response{}, err
	}

	return peer.Response{
		Status:  shim.OK,
		Message: "",
		Payload: aclResp,
	}, nil
}

func MockGetAccountInfo() (peer.Response, error) {
	accountInfo := getAccountInfo()
	aclResp, err := json.Marshal(&accountInfo)
	if err != nil {
		return peer.Response{}, err
	}

	return peer.Response{
		Status:  shim.OK,
		Message: "",
		Payload: aclResp,
	}, nil
}

func getAccountInfo() pbfound.AccountInfo {
	return pbfound.AccountInfo{
		KycHash:     "",
		GrayListed:  false,
		BlackListed: false,
	}
}
