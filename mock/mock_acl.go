package mock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/anoideaopen/foundation/core/acl"
	"github.com/anoideaopen/foundation/core/types"
	st "github.com/anoideaopen/foundation/mock/stub"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	keyRight        = "acl_access_matrix"
	keyAddressRight = "acl_access_matrix_principal_addresses"
)

// mockACL emulates alc chaincode, rights are stored in state
type mockACL struct{}

func (ma *mockACL) Init(_ shim.ChaincodeStubInterface) peer.Response { // stub
	return shim.Success(nil)
}

func (ma *mockACL) Invoke(stub shim.ChaincodeStubInterface) peer.Response { //nolint:funlen,gocognit
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	case acl.FnCheckAddress:
		addr, err := types.AddrFromBase58Check(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		data, err := proto.Marshal((*pb.Address)(addr))
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case acl.FnCheckKeys:
		keys := strings.Split(args[0], "/")
		binPubKeys := make([][]byte, len(keys))
		for i, k := range keys {
			binPubKeys[i] = base58.Decode(k)
		}
		sort.Slice(binPubKeys, func(i, j int) bool {
			return bytes.Compare(binPubKeys[i], binPubKeys[j]) < 0
		})

		hashed := sha3.Sum256(bytes.Join(binPubKeys, []byte("")))
		keyType := getWalletKeyType(stub, base58.CheckEncode(hashed[1:], hashed[0]))

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
			KeyTypes: []pb.KeyType{
				keyType,
			},
		})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case acl.FnGetAccountInfo:
		data, err := json.Marshal(&pb.AccountInfo{
			KycHash:     "123",
			GrayListed:  false,
			BlackListed: false,
		})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(data)
	case acl.FnGetAccOpRight:
		if len(args) != acl.ArgsQtyGetAccOpRight {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyGetAccOpRight))
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
	case acl.FnAddRights:
		if len(args) != acl.ArgsQtyAddRights {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyAddRights))
		}

		ch, cc, role, operation, addr := args[0], args[1], args[2], args[3], args[4]
		err := ma.addRight(stub, ch, cc, role, addr, operation)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case acl.FnRemoveRights:
		if len(args) != acl.ArgsQtyRemoveRights {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyRemoveRights))
		}

		ch, cc, role, operation, addr := args[0], args[1], args[2], args[3], args[4]
		err := ma.removeRight(stub, ch, cc, role, addr, operation)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case acl.FnGetAccountsInfo:
		responses := make([]peer.Response, 0)
		for _, a := range args {
			var argsTmp []string
			err := json.Unmarshal([]byte(a), &argsTmp)
			if err != nil {
				continue
			}
			argsTmp2 := make([][]byte, 0, len(argsTmp))
			for _, a2 := range argsTmp {
				argsTmp2 = append(argsTmp2, []byte(a2))
			}
			st1, ok := stub.(*st.Stub)
			if !ok {
				continue
			}
			st1.Args = argsTmp2
			resp := ma.Invoke(stub)
			responses = append(responses, resp)
		}
		b, err := json.Marshal(responses)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed get accounts info: marshal GetAccountsInfoResponse: %s", err))
		}
		return shim.Success(b)
	case acl.FnAddAddressRightForNominee:
		if len(args) != acl.ArgsQtyAddAddressRightForNominee {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyAddAddressRightForNominee))
		}

		channelName, chaincodeName, nomineeAddress, principalAddress := args[0], args[1], args[2], args[3]
		err := ma.addAddressRightForNominee(stub, channelName, chaincodeName, nomineeAddress, principalAddress)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case acl.FnRemoveAddressRightFromNominee:
		if len(args) != acl.ArgsQtyRemoveAddressRightFromNominee {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyRemoveAddressRightFromNominee))
		}

		channelName, chaincodeName, nomineeAddress, principalAddress := args[0], args[1], args[2], args[3]
		err := ma.removeAddressRightFromNominee(stub, channelName, chaincodeName, nomineeAddress, principalAddress)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(nil)
	case acl.FnGetAddressRightForNominee:
		if len(args) != acl.ArgsQtyGetAddressRightForNominee {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyGetAddressRightForNominee))
		}

		channelName, chaincodeName, nomineeAddress, principalAddress := args[0], args[1], args[2], args[3]
		haveRight, err := ma.getAddressRightForNominee(stub, channelName, chaincodeName, nomineeAddress, principalAddress)
		if err != nil {
			return shim.Error(err.Error())
		}
		rawResultData, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(&pb.HaveRight{HaveRight: haveRight})
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(rawResultData)
	case acl.FnGetAddressesListForNominee:
		if len(args) != acl.ArgsQtyGetAddressesListForNominee {
			return shim.Error(fmt.Sprintf(acl.ErrWrongArgsCount, len(args), acl.ArgsQtyGetAddressesListForNominee))
		}

		channelName, chaincodeName, nomineeAddress := args[0], args[1], args[2]
		addresses, err := ma.getAddressesListForNominee(stub, channelName, chaincodeName, nomineeAddress)
		if err != nil {
			return shim.Error(err.Error())
		}

		rawAddresses, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(addresses)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success(rawAddresses)
	default:
		panic("should not be here")
	}
}

func (ma *mockACL) addRight(stub shim.ChaincodeStubInterface, channel, cc, role, addr, operation string) error {
	key, err := stub.CreateCompositeKey(keyRight, []string{channel, cc, role, operation})
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

	for _, existedAddr := range addresses.GetAddresses() {
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
	key, err := stub.CreateCompositeKey(keyRight, []string{channel, cc, role, operation})
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

	for i, existedAddr := range addresses.GetAddresses() {
		if existedAddr.String() == address.String() {
			addresses.Addresses = append(addresses.Addresses[:i], addresses.GetAddresses()[i+1:]...)
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
	key, err := stub.CreateCompositeKey(keyRight, []string{channel, cc, role, operation})
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

	for _, existedAddr := range addrs.GetAddresses() {
		if existedAddr.String() == address.String() {
			return true, nil
		}
	}

	return false, nil
}

func (ma *mockACL) addAddressRightForNominee(
	stub shim.ChaincodeStubInterface,
	channelName string,
	chaincodeName string,
	nomineeAddress string,
	principalAddress string,
) error {
	key, err := stub.CreateCompositeKey(keyAddressRight, []string{channelName, chaincodeName, nomineeAddress})
	if err != nil {
		return err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return err
	}
	addresses := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err = protojson.Unmarshal(rawAddresses, addresses)
		if err != nil {
			return err
		}
	}

	value, ver, err := base58.CheckDecode(principalAddress)
	if err != nil {
		return err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for _, existedAddr := range addresses.GetAddresses() {
		if address.String() == existedAddr.String() {
			return nil
		}
	}

	addresses.Addresses = append(addresses.Addresses, &address)
	rawAddresses, err = protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(addresses)
	if err != nil {
		return err
	}

	err = stub.PutState(key, rawAddresses)
	if err != nil {
		return err
	}

	return nil
}

func (ma *mockACL) removeAddressRightFromNominee(
	stub shim.ChaincodeStubInterface,
	channelName string,
	chaincodeName string,
	nomineeAddress string,
	principalAddress string,
) error {
	key, err := stub.CreateCompositeKey(keyAddressRight, []string{channelName, chaincodeName, nomineeAddress})
	if err != nil {
		return err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return err
	}
	addresses := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err := protojson.Unmarshal(rawAddresses, addresses)
		if err != nil {
			return err
		}
	}

	value, ver, err := base58.CheckDecode(principalAddress)
	if err != nil {
		return err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for i, existedAddr := range addresses.GetAddresses() {
		if existedAddr.String() == address.String() {
			addresses.Addresses = append(addresses.Addresses[:i], addresses.GetAddresses()[i+1:]...)
			rawAddresses, err = protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(addresses)
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

func (ma *mockACL) getAddressRightForNominee(
	stub shim.ChaincodeStubInterface,
	channelName string,
	chaincodeName string,
	nomineeAddress string,
	principalAddress string,
) (bool, error) {
	key, err := stub.CreateCompositeKey(keyAddressRight, []string{channelName, chaincodeName, nomineeAddress})
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
		err = protojson.Unmarshal(rawAddresses, addrs)
		if err != nil {
			return false, err
		}
	}

	value, ver, err := base58.CheckDecode(principalAddress)
	if err != nil {
		return false, err
	}
	address := pb.Address{Address: append([]byte{ver}, value...)[:32]}

	for _, existedAddr := range addrs.GetAddresses() {
		if existedAddr.String() == address.String() {
			return true, nil
		}
	}

	return false, nil
}

func (ma *mockACL) getAddressesListForNominee(
	stub shim.ChaincodeStubInterface,
	channelName string,
	chaincodeName string,
	nomineeAddress string,
) (*pb.Accounts, error) {
	key, err := stub.CreateCompositeKey(keyAddressRight, []string{channelName, chaincodeName, nomineeAddress})
	if err != nil {
		return nil, err
	}

	rawAddresses, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

	addrs := &pb.Accounts{Addresses: []*pb.Address{}}
	if len(rawAddresses) != 0 {
		err = protojson.Unmarshal(rawAddresses, addrs)
		if err != nil {
			return nil, err
		}
	}

	return addrs, nil
}
