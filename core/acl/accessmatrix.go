package acl

import (
	"errors"
	"fmt"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// acl errors
const (
	// NoRights       = "you have no right to make '%s' operation with chaincode '%s' with role '%s'"
	EmptyResponse  = "ACL empty response"
	WrongArgsCount = "wrong arguments count, get: %d, want: %d"
)

// access matrix functions args count
const (
	GetAccOpRightArgCount   = 5
	GetAccAllRightsArgCount = 5
	AddRightsArgsCount      = 5
	RemoveRightsArgsCount   = 5
)

// GetAccountAllRights fetch all permissions, available for account,
// params[0] -> channel name
// params[1] -> chaincode name
// params[2] -> role
// params[3] -> operation name
// params[4] -> account address
func GetAccountAllRights(stub shim.ChaincodeStubInterface, params []string) (*pb.AccountRights, error) {
	if len(params) != GetAccAllRightsArgCount {
		return nil, fmt.Errorf(WrongArgsCount, len(params), GetAccAllRightsArgCount)
	}

	args := [][]byte{[]byte(GetAccAllRightsFn)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CC, args, Ch)
	if resp.Status != shim.OK {
		return nil, errors.New(resp.Message)
	}
	if len(resp.Payload) == 0 {
		return nil, errors.New(EmptyResponse)
	}

	var ar pb.AccountRights
	if err := proto.Unmarshal(resp.Payload, &ar); err != nil {
		return nil, err
	}

	return &ar, nil
}

// GetAccountRight checks permission for user doing operation with chaincode in channel with role
// params[0] -> channel name
// params[1] -> chaincode name
// params[2] -> role
// params[3] -> operation name
// params[4] -> user address
func GetAccountRight(stub shim.ChaincodeStubInterface, params []string) (*pb.HaveRight, error) {
	if len(params) != GetAccOpRightArgCount {
		return nil, fmt.Errorf(WrongArgsCount, len(params), GetAccOpRightArgCount)
	}

	args := [][]byte{[]byte(GetAccOpRightFn)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CC, args, Ch)
	if resp.Status != shim.OK {
		return nil, errors.New(resp.Message)
	}

	var r pb.HaveRight
	if err := proto.Unmarshal(resp.Payload, &r); err != nil {
		return nil, err
	}

	return &r, nil
}

// GetOperationAllRights fetch all permissions for user with chaincode in channel with specified role
// params[0] -> channel name
// params[1] -> chaincode name
// params[2] -> role
// params[3] -> operation name
// params[4] -> user address
func GetOperationAllRights(stub shim.ChaincodeStubInterface, params []string) (*pb.AccountRights, error) {
	if len(params) != GetAccAllRightsArgCount {
		return nil, fmt.Errorf(WrongArgsCount, len(params), GetAccAllRightsArgCount)
	}

	args := [][]byte{[]byte(GetAccAllRightsFn)}
	for _, param := range params {
		args = append(args, []byte(param))
	}
	resp := stub.InvokeChaincode(CC, args, Ch)
	if resp.Status != shim.OK {
		return nil, errors.New(resp.Message)
	}

	var ar pb.AccountRights
	if err := proto.Unmarshal(resp.Payload, &ar); err != nil {
		return nil, err
	}

	return &ar, nil
}
