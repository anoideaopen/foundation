package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

const (
	// accInfoPrefix         = "accountinfo"
	replaceTxChangePrefix = "replacetx"
	signedTxChangePrefix  = "signedtx"
)

// AddAddrIfChanged looks to ACL for pb.Address saved for specific pubkeys
// and checks addr changed or not (does have pb.Address SignedTx field or not)
// if the address has changed in the ACL, we also fix it in the token channel
func AddAddrIfChanged(stub shim.ChaincodeStubInterface, addrMsgFromACL *pb.SignedAddress) error { //nolint:gocognit
	// check if it multisig, and it's pubkeys changed or not
	if addrMsgFromACL.Address.IsMultisig { //nolint:nestif
		changeTx, err := shim.CreateCompositeKey(
			replaceTxChangePrefix,
			[]string{base58.CheckEncode(addrMsgFromACL.Address.Address[1:], addrMsgFromACL.Address.Address[0])},
		)
		if err != nil {
			return err
		}

		signedChangeTxBytes, err := stub.GetState(changeTx)
		if err != nil {
			return err
		}

		// if there is no public key change transaction in the token channel, but such a transaction is present in the ACL
		if len(signedChangeTxBytes) == 0 && len(addrMsgFromACL.SignaturePolicy.ReplaceKeysSignedTx) != 0 {
			m, err := json.Marshal(addrMsgFromACL.SignaturePolicy.ReplaceKeysSignedTx)
			if err != nil {
				return err
			}

			err = stub.PutState(changeTx, m)
			if err != nil {
				return err
			}
			// if public key change transaction is present in both channels
		} else if len(signedChangeTxBytes) != 0 && len(addrMsgFromACL.SignaturePolicy.ReplaceKeysSignedTx) != 0 {
			var signedChangeTx []string
			if err = json.Unmarshal(signedChangeTxBytes, &signedChangeTx); err != nil {
				return fmt.Errorf("failed to unmarshal replace tx: %w", err)
			}

			for index, replaceTx := range addrMsgFromACL.SignaturePolicy.ReplaceKeysSignedTx {
				if replaceTx != signedChangeTx[index] {
					// pubkeys in multisig already changed, put new pb.SignedAddress to token channel too
					m, err := json.Marshal(addrMsgFromACL.SignaturePolicy.ReplaceKeysSignedTx)
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

	chngTx, err := shim.CreateCompositeKey(signedTxChangePrefix, []string{base58.CheckEncode(addrMsgFromACL.Address.Address[1:], addrMsgFromACL.Address.Address[0])})
	if err != nil {
		return err
	}
	signedChangeTxBytes, err := stub.GetState(chngTx)
	if err != nil {
		return err
	}
	// if there is no public key change transaction in the token channel, but such a transaction is present in the ACL

	if len(signedChangeTxBytes) == 0 && len(addrMsgFromACL.SignedTx) != 0 { //nolint:nestif
		m, err := json.Marshal(addrMsgFromACL.SignedTx)
		if err != nil {
			return err
		}

		err = stub.PutState(chngTx, m)
		if err != nil {
			return err
		}
		// if public key change transaction is present in both channels
	} else if len(signedChangeTxBytes) != 0 && len(addrMsgFromACL.SignedTx) != 0 {
		var signedChangeTx []string
		if err = json.Unmarshal(signedChangeTxBytes, &signedChangeTx); err != nil {
			return fmt.Errorf("failed to unmarshal signed tx: %w", err)
		}

		// check if pb.SignedAddress from ACL has the same SignedTx as pb.SignedAddress saved in the token channel
		for index, changePubkeyTx := range addrMsgFromACL.SignedTx {
			if changePubkeyTx != signedChangeTx[index] {
				// pubkey already changed, put new SignedTx to token channel too
				m, err := json.Marshal(addrMsgFromACL.SignedTx)
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
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte("checkKeys"),
		[]byte(keys),
	}, "acl")

	if resp.Status != http.StatusOK {
		return nil, errors.New(resp.Message)
	}

	if len(resp.Payload) == 0 {
		return nil, errors.New("empty response")
	}

	addrMsg := &pb.AclResponse{}
	if err := proto.Unmarshal(resp.Payload, addrMsg); err != nil {
		return nil, err
	}

	return addrMsg, nil
}

// GetFullAddress returns pb.Address from the ACL
func GetFullAddress(stub shim.ChaincodeStubInterface, key string) (*pb.Address, error) {
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte("checkAddress"),
		[]byte(key),
	}, "acl")

	if resp.Status != http.StatusOK {
		return nil, errors.New(resp.Message)
	}

	if len(resp.Payload) == 0 {
		return nil, errors.New("empty response")
	}

	addrMsg := &pb.Address{}
	if err := proto.Unmarshal(resp.Payload, addrMsg); err != nil {
		return nil, err
	}

	return addrMsg, nil
}

// GetAccountInfo returns pb.AccountInfo from the ACL
func GetAccountInfo(stub shim.ChaincodeStubInterface, addr string) (*pb.AccountInfo, error) {
	resp := stub.InvokeChaincode("acl", [][]byte{
		[]byte("getAccountInfo"),
		[]byte(addr),
	}, "acl")

	if resp.Status != http.StatusOK {
		return nil, errors.New(resp.Message)
	}
	if len(resp.Payload) == 0 {
		return nil, errors.New("empty response")
	}

	infoMsg := pb.AccountInfo{}
	if err := json.Unmarshal(resp.Payload, &infoMsg); err != nil {
		return nil, err
	}
	return &infoMsg, nil
}
