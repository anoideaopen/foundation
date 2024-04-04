package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
)

const rightKey = "acl_access_matrix"

// mockACL emulates alc chaincode, rights are stored in state
type mockACL struct{}

func (ma *mockACL) Init(_ shim.ChaincodeStubInterface) peer.Response { // stub
	return shim.Success(nil)
}

func (ma *mockACL) Invoke(stub shim.ChaincodeStubInterface) peer.Response { //nolint:funlen
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	case "checkAddress":
		addr, err := types.AddrFromBase58Check(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		data, err := proto.Marshal((*pb.Address)(addr))
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case "checkKeys":
		keys := strings.Split(args[0], "/")
		binPubKeys := make([][]byte, len(keys))
		for i, k := range keys {
			binPubKeys[i] = base58.Decode(k)
		}
		sort.Slice(binPubKeys, func(i, j int) bool {
			return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
		})

		hashed := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
		data, err := proto.Marshal(&pb.AclResponse{
			Account: &pb.AccountInfo{
				KycHash:    "123",
				GrayListed: false,
			},
			Address: &pb.SignedAddress{
				Address: &pb.Address{Address: hashed[:]},
				SignaturePolicy: &pb.SignaturePolicy{
					N: 2, //nolint:gomnd
				},
			},
		})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case "getAccountInfo":
		data, err := json.Marshal(&pb.AccountInfo{
			KycHash:     "123",
			GrayListed:  false,
			BlackListed: false,
		})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case acl.GetAccOpRightFn:
		if len(args) != acl.GetAccOpRightArgCount {
			return shim.Error(fmt.Sprintf(acl.WrongArgsCount, len(args), acl.GetAccOpRightArgCount))
		}

		ch, cc, role, operation, addr := args[0], args[1], args[2], args[3], args[4]
		haveRight, err := ma.getRight(stub, ch, cc, role, addr, operation)
		if err != nil {
			return shim.Error(err.Error())
		}

		rawResultData, err := proto.Marshal(&pb.HaveRight{HaveRight: haveRight})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(rawResultData)
	case acl.AddRightsFn:
		if len(args) != acl.AddRightsArgsCount {
			return shim.Error(fmt.Sprintf(acl.WrongArgsCount, len(args), acl.AddRightsArgsCount))
		}

		ch, cc, role, operation, addr := args[0], args[1], args[2], args[3], args[4]
		err := ma.addRight(stub, ch, cc, role, addr, operation)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case acl.RemoveRightsFn:
		if len(args) != acl.RemoveRightsArgsCount {
			return shim.Error(fmt.Sprintf(acl.WrongArgsCount, len(args), acl.RemoveRightsArgsCount))
		}

		ch, cc, role, operation, addr := args[0], args[1], args[2], args[3], args[4]
		err := ma.removeRight(stub, ch, cc, role, addr, operation)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	default:
		panic("should not be here")
	}
}

func (ma *mockACL) addRight(stub shim.ChaincodeStubInterface, channel, cc, role, addr, operation string) error {
	key, err := stub.CreateCompositeKey(rightKey, []string{channel, cc, role, operation})
	if err != nil {
		return err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return err
	}
	addresses := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err = proto.Unmarshal(rawAddresses, addresses)
		if err != nil {
			return err
		}
	}

	value, ver, err := base58.CheckDecode(addr)
	if err != nil {
		return err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for _, existedAddr := range addresses.Addresses {
		if address.String() == existedAddr.String() {
			return nil
		}
	}

	addresses.Addresses = append(addresses.Addresses, &address)
	rawAddresses, err = proto.Marshal(addresses)
	if err != nil {
		return err
	}

	err = stub.PutState(key, rawAddresses)
	if err != nil {
		return err
	}

	return nil
}

func (ma *mockACL) removeRight(stub shim.ChaincodeStubInterface, channel, cc, role, addr, operation string) error {
	key, err := stub.CreateCompositeKey(rightKey, []string{channel, cc, role, operation})
	if err != nil {
		return err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return err
	}
	addresses := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err := proto.Unmarshal(rawAddresses, addresses)
		if err != nil {
			return err
		}
	}

	value, ver, err := base58.CheckDecode(addr)
	if err != nil {
		return err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for i, existedAddr := range addresses.Addresses {
		if existedAddr.String() == address.String() {
			addresses.Addresses = append(addresses.Addresses[:i], addresses.Addresses[i+1:]...)
			rawAddresses, err = proto.Marshal(addresses)
			if err != nil {
				return err
			}
			err = stub.PutState(key, rawAddresses)
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (ma *mockACL) getRight(stub shim.ChaincodeStubInterface, channel, cc, role, addr, operation string) (bool, error) {
	key, err := stub.CreateCompositeKey(rightKey, []string{channel, cc, role, operation})
	if err != nil {
		return false, err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return false, err
	}

	if len(rawAddresses) == 0 {
		return false, nil
	}

	addrs := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err = proto.Unmarshal(rawAddresses, addrs)
		if err != nil {
			return false, err
		}
	}

	value, ver, err := base58.CheckDecode(addr)
	if err != nil {
		return false, err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for _, existedAddr := range addrs.Addresses {
		if existedAddr.String() == address.String() {
			return true, nil
		}
	}

	return false, nil
}
