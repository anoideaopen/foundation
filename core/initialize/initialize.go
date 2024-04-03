package initialize

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/msp"
)

const (
	// keyInit is the key for receiving initialization configuration from the state database.
	keyInit = "__init"
	// adminOU is the required OrganizationalUnit in the x509 certificate for Hyperledger admin.
	adminOU = "admin"
	// minChaincodeArgsCount is the minimum number of arguments expected when initializing the chaincode.
	minChaincodeArgsCount = 2
)

var (
	// ErrNilStub is returned when the stub is nil.
	ErrNilStub = errors.New("stub can't be nil")
	// ErrDecodeSerializedIdentity is returned when the SerializedIdentity.IdBytes can't be decoded.
	ErrDecodeSerializedIdentity = errors.New("block after decode SerializedIdentity.IdBytes can't be nil or empty")
)

// Config is global chaincode parameters from Chaincode Init arguments
type Config struct {
	// PlatformSKI is 0 index from init args
	PlatformSKI []byte
	// RobotSKI is 1 index from init args
	RobotSKI []byte
	// Args is subarray from init args from 2 index to last index arg
	Args []string
}

// InitChaincode initializes the chaincode with provided arguments.
// It validates the admin creator and stores necessary data (platformSKI, robotSKI, initArgs) in the state.
func InitChaincode(stub shim.ChaincodeStubInterface) error {
	if stub == nil {
		return ErrNilStub
	}

	err := validateAdminCreator(stub)
	if err != nil {
		return fmt.Errorf("failed to validate admin creator: %w", err)
	}

	args := stub.GetStringArgs()
	if len(args) < minChaincodeArgsCount {
		return fmt.Errorf("should set SKI of platformSKI and robotSKI certs. expected %d but found %d",
			minChaincodeArgsCount,
			len(args),
		)
	}
	platformSKI, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("failed to hex decode from string platformSKI %s: %w", args[0], err)
	}
	robotSKI, err := hex.DecodeString(args[1])
	if err != nil {
		return fmt.Errorf("failed to hex decode from string robotSKI %s: %w", args[1], err)
	}

	initArgs := Config{
		PlatformSKI: platformSKI,
		RobotSKI:    robotSKI,
		Args:        args[2:],
	}
	err = saveInitArgs(stub, initArgs)
	if err != nil {
		return fmt.Errorf("failed to save InitArgs %v to statedb: %w", initArgs, err)
	}

	return nil
}

// validateAdminCreator checks if the creator of the transaction is an admin.
func validateAdminCreator(stub shim.ChaincodeStubInterface) error {
	if stub == nil {
		return ErrNilStub
	}
	creator, err := stub.GetCreator()
	if err != nil {
		return fmt.Errorf("failed validate admin creator: %w", err)
	}

	var identity msp.SerializedIdentity
	if err = pb.Unmarshal(creator, &identity); err != nil {
		return fmt.Errorf("failed to unmarshal SerializedIdentity %v: %w", creator, err)
	}

	b, _ := pem.Decode(identity.IdBytes)
	if b == nil || len(b.Bytes) == 0 {
		return ErrDecodeSerializedIdentity
	}
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse x509 certificate: %w", err)
	}
	ouIsOk := false
	for _, ou := range parsed.Subject.OrganizationalUnit {
		if strings.ToLower(ou) == adminOU {
			ouIsOk = true
		}
	}
	if !ouIsOk {
		return fmt.Errorf("incorrect sender's OU, expected '%s' but found '%s'",
			adminOU,
			strings.Join(parsed.Subject.OrganizationalUnit, ","),
		)
	}

	return nil
}

// LoadInitArgs retrieves the initialization arguments from the state.
func LoadInitArgs(stub shim.ChaincodeStubInterface) (Config, error) {
	if stub == nil {
		return Config{}, ErrNilStub
	}
	data, err := stub.GetState(keyInit)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get InitArgs by key %s: %w", keyInit, err)
	}
	var initArgs proto.InitArgs
	if err = pb.Unmarshal(data, &initArgs); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal InitArgs: %w", err)
	}

	config := Config{
		PlatformSKI: initArgs.PlatformSKI,
		RobotSKI:    initArgs.RobotSKI,
		Args:        initArgs.Args,
	}

	return config, nil
}

// saveInitArgs saves the initialization arguments in the state.
func saveInitArgs(stub shim.ChaincodeStubInterface, initArgs Config) error {
	if stub == nil {
		return ErrNilStub
	}

	initArgsBytes, err := pb.Marshal(&proto.InitArgs{
		PlatformSKI: initArgs.PlatformSKI,
		RobotSKI:    initArgs.RobotSKI,
		Args:        initArgs.Args,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal InitArgs %v: %w", initArgs, err)
	}
	if err = stub.PutState(keyInit, initArgsBytes); err != nil {
		return fmt.Errorf("failed to put InitArgs %v by key %s: %w", initArgs, keyInit, err)
	}

	return nil
}
