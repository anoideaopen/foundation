package acl

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// acl errors
const (
	// NoRights       = "you have no right to make '%s' operation with chaincode '%s' with role '%s'"
	ErrWrongArgsCount = "wrong arguments count, get: %d, want: %d"
)

// access matrix functions args count
const (
	ArgsQtyGetAccOpRight                 = 5
	ArgsQtyAddRights                     = 5
	ArgsQtyRemoveRights                  = 5
	ArgsQtyAddAddressRightForNominee     = 4
	ArgsQtyRemoveAddressRightFromNominee = 4
	ArgsQtyGetAddressRightForNominee     = 4
	ArgsQtyGetAddressesListForNominee    = 3
)

// GetAccountRight checks permission for user doing operation with chaincode in channel with role
// params[0] -> channel name
// params[1] -> chaincode name
// params[2] -> role
// params[3] -> operation name
// params[4] -> user address
func GetAccountRight(stub shim.ChaincodeStubInterface, params []string) (*pb.HaveRight, error) {
	if len(params) != ArgsQtyGetAccOpRight {
		return nil, fmt.Errorf(ErrWrongArgsCount, len(params), ArgsQtyGetAccOpRight)
	}

	args := [][]byte{[]byte(FnGetAccOpRight)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CcACL, args, ChACL)
	if resp.GetStatus() != shim.OK {
		return nil, errors.New(resp.GetMessage())
	}

	var r pb.HaveRight
	if err := proto.Unmarshal(resp.GetPayload(), &r); err != nil {
		return nil, err
	}

	return &r, nil
}

// IsIssuerAccountRight checks whether the specified address holds the Issuer right by querying ACL account rights.
// It utilizes the provided BaseContractInterface (bci) to interact with the smart contract.
// The function returns a boolean indicating if the address is an issuer and an error if encountered.
//
// Parameters:
//   - bci: The BaseContractInterface representing the smart contract interface.
//   - address: A pointer to the Address being checked for issuer rights.
//
// Returns:
//   - bool: True if the address is an issuer, false otherwise.
//   - error: Any error encountered during the process, or nil if successful.
func IsIssuerAccountRight(bci core.BaseContractInterface, address *types.Address) (bool, error) {
	chaincodeStubInterface := bci.GetStub()
	chaincode := bci.GetID()
	channelID := chaincodeStubInterface.GetChannelID()
	// get account right for any operations by empty string
	anyOperation := ""

	params := []string{channelID, chaincode, Issuer.String(), anyOperation, address.String()}
	haveRight, err := GetAccountRight(chaincodeStubInterface, params)
	if err != nil {
		return false, fmt.Errorf("getting account right: %w", err)
	}

	if haveRight != nil && !haveRight.GetHaveRight() {
		return false, nil
	}

	return true, nil
}

// GetAddressRightForNominee returns if nominee have right to transfer from specified address
// args[0] - channelName
// args[1] - chaincodeName
// args[2] - nomineeAddress
// args[3] - principalAddress
func GetAddressRightForNominee(stub shim.ChaincodeStubInterface, params []string) (*pb.HaveRight, error) {
	if len(params) != ArgsQtyGetAddressRightForNominee {
		return nil, fmt.Errorf(ErrWrongArgsCount, len(params), ArgsQtyGetAddressRightForNominee)
	}

	args := [][]byte{[]byte(FnGetAddressRightForNominee)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CcACL, args, ChACL)
	if resp.GetStatus() != shim.OK {
		return nil, errors.New(resp.GetMessage())
	}

	var r pb.HaveRight
	if err := protojson.Unmarshal(resp.GetPayload(), &r); err != nil {
		return nil, err
	}

	return &r, nil
}

// GetAddressesListForNominee returns principal addresses for nominee
// args[0] - channelName
// args[1] - chaincodeName
// args[2] - nomineeAddress
func GetAddressesListForNominee(stub shim.ChaincodeStubInterface, params []string) (*pb.Accounts, error) {
	if len(params) != ArgsQtyGetAddressesListForNominee {
		return nil, fmt.Errorf(ErrWrongArgsCount, len(params), ArgsQtyGetAddressesListForNominee)
	}

	args := [][]byte{[]byte(FnGetAddressesListForNominee)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CcACL, args, ChACL)
	if resp.GetStatus() != shim.OK {
		return nil, errors.New(resp.GetMessage())
	}

	var r pb.Accounts
	if err := protojson.Unmarshal(resp.GetPayload(), &r); err != nil {
		return nil, err
	}

	return &r, nil
}
