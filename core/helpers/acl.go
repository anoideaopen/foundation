package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/anoideaopen/foundation/core/logger"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

const (
	// accInfoPrefix         = "accountinfo"
	replaceTxChangePrefix = "replacetx"
	signedTxChangePrefix  = "signedtx"

	FnGetAccountsInfo = "getAccountsInfo"
	FnGetAccountInfo  = "getAccountInfo"
	FnCheckAddress    = "checkAddress"
	FnCheckKeys       = "checkKeys"
)

// AddAddrIfChanged looks to ACL for pb.Address saved for specific pubkeys
// and checks addr changed or not (does have pb.Address SignedTx field or not)
// if the address has changed in the ACL, we also fix it in the token channel
func AddAddrIfChanged(stub shim.ChaincodeStubInterface, addrMsgFromACL *pb.SignedAddress) error { //nolint:gocognit
	// check if it multisig, and it's pubkeys changed or not
	if addrMsgFromACL.GetAddress().GetIsMultisig() { //nolint:nestif
		changeTx, err := shim.CreateCompositeKey(
			replaceTxChangePrefix,
			[]string{base58.CheckEncode(addrMsgFromACL.GetAddress().GetAddress()[1:], addrMsgFromACL.GetAddress().GetAddress()[0])},
		)
		if err != nil {
			return err
		}

		signedChangeTxBytes, err := stub.GetState(changeTx)
		if err != nil {
			return err
		}

		// if there is no public key change transaction in the token channel, but such a transaction is present in the ACL
		if len(signedChangeTxBytes) == 0 && len(addrMsgFromACL.GetSignaturePolicy().GetReplaceKeysSignedTx()) != 0 {
			m, err := json.Marshal(addrMsgFromACL.GetSignaturePolicy().GetReplaceKeysSignedTx())
			if err != nil {
				return err
			}

			err = stub.PutState(changeTx, m)
			if err != nil {
				return err
			}
			// if public key change transaction is present in both channels
		} else if len(signedChangeTxBytes) != 0 && len(addrMsgFromACL.GetSignaturePolicy().GetReplaceKeysSignedTx()) != 0 {
			var signedChangeTx []string
			if err = json.Unmarshal(signedChangeTxBytes, &signedChangeTx); err != nil {
				return fmt.Errorf("failed to unmarshal replace tx: %w", err)
			}

			for index, replaceTx := range addrMsgFromACL.GetSignaturePolicy().GetReplaceKeysSignedTx() {
				if replaceTx != signedChangeTx[index] {
					// pubkeys in multisig already changed, put new pb.SignedAddress to token channel too
					m, err := json.Marshal(addrMsgFromACL.GetSignaturePolicy().GetReplaceKeysSignedTx())
					if err != nil {
						return err
					}

					err = stub.PutState(changeTx, m)
					if err != nil {
						return err
					}
					break
				}
			}
		}
	}

	chngTx, err := shim.CreateCompositeKey(signedTxChangePrefix, []string{base58.CheckEncode(addrMsgFromACL.GetAddress().GetAddress()[1:], addrMsgFromACL.GetAddress().GetAddress()[0])})
	if err != nil {
		return err
	}
	signedChangeTxBytes, err := stub.GetState(chngTx)
	if err != nil {
		return err
	}
	// if there is no public key change transaction in the token channel, but such a transaction is present in the ACL

	if len(signedChangeTxBytes) == 0 && len(addrMsgFromACL.GetSignedTx()) != 0 { //nolint:nestif
		m, err := json.Marshal(addrMsgFromACL.GetSignedTx())
		if err != nil {
			return err
		}

		err = stub.PutState(chngTx, m)
		if err != nil {
			return err
		}
		// if public key change transaction is present in both channels
	} else if len(signedChangeTxBytes) != 0 && len(addrMsgFromACL.GetSignedTx()) != 0 {
		var signedChangeTx []string
		if err = json.Unmarshal(signedChangeTxBytes, &signedChangeTx); err != nil {
			return fmt.Errorf("failed to unmarshal signed tx: %w", err)
		}

		// check if pb.SignedAddress from ACL has the same SignedTx as pb.SignedAddress saved in the token channel
		for index, changePubkeyTx := range addrMsgFromACL.GetSignedTx() {
			if changePubkeyTx != signedChangeTx[index] {
				// pubkey already changed, put new SignedTx to token channel too
				m, err := json.Marshal(addrMsgFromACL.GetSignedTx())
				if err != nil {
					return err
				}

				err = stub.PutState(chngTx, m)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

// CheckACL checks if the address is in the ACL
func CheckACL(stub shim.ChaincodeStubInterface, keys []string) (*pb.AclResponse, error) {
	return GetAddress(stub, strings.Join(keys, "/"))
}

// GetAddress returns pb.AclResponse from the ACL
func GetAddress(stub shim.ChaincodeStubInterface, keys string) (*pb.AclResponse, error) {
	logger.Logger().Debugf("invoke acl chaincode, %s: %s: %s", stub.GetTxID(), FnCheckKeys, keys)
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte(FnCheckKeys),
		[]byte(keys),
	}, "acl")

	if resp.GetStatus() != http.StatusOK {
		return nil, errors.New(resp.GetMessage())
	}

	if len(resp.GetPayload()) == 0 {
		return nil, errors.New("empty response")
	}

	addrMsg := &pb.AclResponse{}
	if err := proto.Unmarshal(resp.GetPayload(), addrMsg); err != nil {
		return nil, err
	}

	return addrMsg, nil
}

// GetFullAddress returns pb.Address from the ACL
func GetFullAddress(stub shim.ChaincodeStubInterface, key string) (*pb.Address, error) {
	logger.Logger().Debugf("invoke acl chaincode, %s: %s: %s", stub.GetTxID(), FnCheckAddress, key)
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte(FnCheckAddress),
		[]byte(key),
	}, "acl")

	if resp.GetStatus() != http.StatusOK {
		return nil, errors.New(resp.GetMessage())
	}

	if len(resp.GetPayload()) == 0 {
		return nil, errors.New("empty response")
	}

	addrMsg := &pb.Address{}
	if err := proto.Unmarshal(resp.GetPayload(), addrMsg); err != nil {
		return nil, err
	}

	return addrMsg, nil
}

// GetAccountInfo returns pb.AccountInfo from the ACL
func GetAccountInfo(stub shim.ChaincodeStubInterface, addr string) (*pb.AccountInfo, error) {
	logger.Logger().Debugf("invoke acl chaincode, %s: %s: %s", stub.GetTxID(), FnGetAccountInfo, addr)
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte(FnGetAccountInfo),
		[]byte(addr),
	}, "acl")

	if resp.GetStatus() != http.StatusOK {
		return nil, fmt.Errorf(
			"ACL status is not OK: status code: %d, message: '%s', payload: '%s'",
			resp.GetStatus(),
			resp.GetMessage(),
			string(resp.GetPayload()),
		)
	}

	if len(resp.GetPayload()) == 0 {
		return nil, errors.New("empty response")
	}

	infoMsg := pb.AccountInfo{}
	if err := json.Unmarshal(resp.GetPayload(), &infoMsg); err != nil {
		return nil, err
	}

	return &infoMsg, nil
}

// GetAccountsInfo execute group requests in single invoke request each of them contains own peer.Response
func GetAccountsInfo(stub shim.ChaincodeStubInterface, bytes [][]byte) ([]peer.Response, error) {
	logger.Logger().Debugf("invoke acl chaincode: %s", FnGetAccountsInfo)
	args := append([][]byte{[]byte(FnGetAccountsInfo)}, bytes...)
	resp := stub.InvokeChaincode("acl", args, "acl")

	if resp.GetStatus() != http.StatusOK {
		return nil, fmt.Errorf(
			"ACL status is not OK: status code: %d, message: '%s', payload: '%s'",
			resp.GetStatus(),
			resp.GetMessage(),
			string(resp.GetPayload()),
		)
	}

	if len(resp.GetPayload()) == 0 {
		return nil, errors.New("invoke acl method getAccountsInfo: empty response")
	}

	var responses []peer.Response
	if err := json.Unmarshal(resp.GetPayload(), &responses); err != nil {
		return nil, fmt.Errorf("invoke acl method getAccountsInfo: failed to unmarshal response: %w", err)
	}

	return responses, nil
}
