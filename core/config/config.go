package config

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"google.golang.org/protobuf/encoding/protojson"
)

// keyConfig is a key for storing a configuration data in json format.
const keyConfig = "__config"

// BatchPrefix is a prefix for batched transactions
const BatchPrefix = "batchTransactions"

var ErrCfgBytesEmpty = errors.New("config bytes is empty")

// positional args specific errors
var (
	ErrAdminEmpty            = errors.New("'admin' address is empty")
	ErrIssuerEmpty           = errors.New("'issuer' address is empty")
	ErrFeeSetterEmpty        = errors.New("'fee-setter' address is empty")
	ErrFeeAddressSetterEmpty = errors.New("'fee-address-setter' address is empty")
)

// Save saves configuration data to the state using the provided State interface.
//
// If the provided cfgBytes slice is empty, the function returns an ErrCfgBytesEmpty error.
//
// If there is an error while saving the data to the state, an error is returned with
// additional information about the error.
func Save(stub shim.ChaincodeStubInterface, cfgBytes []byte) error {
	if len(cfgBytes) == 0 {
		return ErrCfgBytesEmpty
	}

	if err := stub.PutState(keyConfig, cfgBytes); err != nil {
		return fmt.Errorf("putting config data to state: %w", err)
	}

	return nil
}

// Load retrieves and returns the raw configuration data from the state
// using the provided State interface.
//
// The function returns the configuration data as a byte slice and nil error if successful.
//
// If there is an error while loading the data from the state,
// an error is returned with additional information about the error.
//
// If the retrieved configuration data is empty, the function returns an ErrCfgBytesEmpty error.
func Load(stub shim.ChaincodeStubInterface) ([]byte, error) {
	cfgBytes, err := stub.GetState(keyConfig)
	if err != nil {
		return nil, fmt.Errorf("loading raw config: %w", err)
	}

	if len(cfgBytes) == 0 {
		return nil, ErrCfgBytesEmpty
	}

	return cfgBytes, nil
}

// FromBytes parses the provided byte slice containing JSON-encoded configuration
// and returns a pointer to a proto.Config struct.
func FromBytes(cfgBytes []byte) (*proto.Config, error) {
	cfg := new(proto.Config)

	if err := protojson.Unmarshal(cfgBytes, cfg); err != nil {
		return nil, err
	}

	if cfg.GetContract() == nil {
		cfg.Contract = new(proto.ContractConfig)
	}

	if cfg.GetToken() == nil {
		cfg.Token = new(proto.TokenConfig)
	}

	if cfg.GetContract().GetOptions() == nil {
		cfg.Contract.Options = new(proto.ChaincodeOptions)
	}

	return cfg, nil
}

// IsJSON checks if the provided arguments represent a valid JSON configuration.
//
// The function returns true if there is exactly one argument in the initialization args slice,
// and if the content of that argument is a valid JSON.
func IsJSON(args []string) bool {
	return len(args) == 1 && json.Valid([]byte(args[0]))
}
