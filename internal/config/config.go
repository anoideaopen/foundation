package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/proto"
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

type State interface {
	// GetState returns the value of the specified `key` from the
	// ledger. Note that GetState doesn't read data from the Write Set, which
	// has not been committed to the ledger. In other words, GetState doesn't
	// consider data modified by PutState that has not been committed.
	// If the key does not exist in the state database, (nil, nil) is returned.
	GetState(key string) ([]byte, error)

	// PutState puts the specified `key` and `value` into the transaction's
	// Write Set as a data-write proposal. PutState doesn't affect the ledger
	// until the transaction is validated and successfully committed.
	// Simple keys must not be an empty string and must not start with a
	// null character (0x00) in order to avoid range query collisions with
	// composite keys, which internally get prefixed with 0x00 as composite
	// key namespace. In addition, if using CouchDB, keys can only contain
	// valid UTF-8 strings and cannot begin with an underscore ("_").
	PutState(key string, value []byte) error
}

// SaveConfig saves configuration data to the state using the provided State interface.
//
// If the provided cfgBytes slice is empty, the function returns an ErrCfgBytesEmpty error.
//
// If there is an error while saving the data to the state, an error is returned with
// additional information about the error.
func SaveConfig(state State, cfgBytes []byte) error {
	if len(cfgBytes) == 0 {
		return ErrCfgBytesEmpty
	}

	if err := state.PutState(keyConfig, cfgBytes); err != nil {
		return fmt.Errorf("putting config data to state: %w", err)
	}

	return nil
}

// LoadRawConfig retrieves and returns the raw configuration data from the state
// using the provided State interface.
//
// The function returns the configuration data as a byte slice and nil error if successful.
//
// If there is an error while loading the data from the state,
// an error is returned with additional information about the error.
//
// If the retrieved configuration data is empty, the function returns an ErrCfgBytesEmpty error.
func LoadRawConfig(state State) ([]byte, error) {
	cfgBytes, err := state.GetState(keyConfig)
	if err != nil {
		return nil, fmt.Errorf("loading raw config: %w", err)
	}
	if len(cfgBytes) == 0 {
		return nil, ErrCfgBytesEmpty
	}

	return cfgBytes, nil
}

// ContractConfigFromBytes parses the provided byte slice containing JSON-encoded contract configuration
// and returns a pointer to a proto.ContractConfig struct.
//
// The function uses protojson.Unmarshal to deserialize the JSON-encoded data into the *proto.ContractConfig struct.
// If the unmarshalling process fails, an error is returned with additional information about the failure.
//
// If the deserialized ContractConfig struct has nil Options, a new proto.ChaincodeOptions is created and assigned.
// If the BatchPrefix in Options is empty, it is set to the default BatchPrefix constant.
func ContractConfigFromBytes(cfgBytes []byte) (*proto.ContractConfig, error) {
	var cfg proto.Config
	if err := protojson.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling failed: %w", err)
	}

	if cfg.GetContract().GetOptions() == nil {
		cfg.Contract.Options = new(proto.ChaincodeOptions)
	}

	return cfg.GetContract(), nil
}

// TokenConfigFromBytes parses the provided byte slice containing JSON-encoded token configuration
// and returns a pointer to a proto.TokenConfig struct.
//
// The function uses protojson.Unmarshal to deserialize the JSON-encoded data into a temporary struct
// with a "token" field of type proto.TokenConfig. If the unmarshalling process fails, an error
// is returned with additional information about the failure.
//
// The function returns a pointer to the deserialized proto.TokenConfig struct.
func TokenConfigFromBytes(cfgBytes []byte) (*proto.TokenConfig, error) {
	var cfg proto.Config
	if err := protojson.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, err
	}

	return cfg.GetToken(), nil
}

// IsJSONConfig checks if the provided arguments represent a valid JSON configuration.
//
// The function returns true if there is exactly one argument in the initialization args slice,
// and if the content of that argument is a valid JSON.
func IsJSONConfig(args []string) bool {
	return len(args) == 1 && json.Valid([]byte(args[0]))
}

// ParseArgsArr parses positional initialization arguments and generates JSON-config of []byte type.
// Accepts the channel name (chaincode) and the list of positional initialization parameters.
// Only needed to maintain backward compatibility.
// Marked for deletion after all deploy tools will be switched to JSON-config initialization of chaincodes.
// Deprecated
func ParseArgsArr(channel string, args []string) ([]byte, error) { //nolint:funlen,gocognit
	const minArgsCount = 2
	argsCount := len(args)
	if argsCount < minArgsCount {
		return nil, fmt.Errorf("minimum required args length is '%d', passed %d",
			argsCount, minArgsCount)
	}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{},
		Token:    &proto.TokenConfig{},
	}

	_ = args[0] // PlatformSKI (backend) - deprecated
	robotSKI := args[1]
	cfg.Contract.RobotSKI = robotSKI

	switch channel {
	case "nft", "dcdac", "ndm", "rub", "it":
		const requiredArgsCount = 3
		if argsCount != requiredArgsCount {
			return nil, fmt.Errorf("required args length '%s' is '%d', passed %d",
				channel, requiredArgsCount, argsCount)
		}

		adminAddress := args[2]
		if adminAddress == "" {
			return nil, ErrAdminEmpty
		}

		symbol := strings.ToUpper(channel)
		cfg.Contract.Symbol = symbol
		cfg.Contract.Admin = &proto.Wallet{Address: adminAddress}
		cfg.Token.Name = symbol
		cfg.Token.Issuer = &proto.Wallet{Address: adminAddress}
	case "ct", "hermitage", "dcrsb", "minetoken", "invclass", "vote":
		const requiredArgsCount = 4
		if argsCount != requiredArgsCount {
			return nil, fmt.Errorf("required args length for '%s' is '%d', passed %d",
				channel, requiredArgsCount, argsCount)
		}

		issuerAddress := args[2]
		if issuerAddress == "" {
			return nil, ErrIssuerEmpty
		}
		adminAddress := args[3]
		if adminAddress == "" {
			return nil, ErrAdminEmpty
		}

		symbol := strings.ToUpper(channel)
		cfg.Contract.Symbol = symbol
		cfg.Contract.Admin = &proto.Wallet{Address: adminAddress}
		cfg.Token.Name = symbol
		cfg.Token.Issuer = &proto.Wallet{Address: issuerAddress}
	case "nmmmulti", "invmulti", "dcmulti":
		const requiredArgsCount = 3
		if argsCount != requiredArgsCount {
			return nil, fmt.Errorf("required args length for '%s' is '%d', passed %d",
				channel, requiredArgsCount, argsCount)
		}

		adminAddress := args[2]
		if adminAddress == "" {
			return nil, ErrAdminEmpty
		}

		symbol := strings.ToUpper(channel)
		cfg.Contract.Symbol = symbol
		cfg.Contract.Admin = &proto.Wallet{Address: adminAddress}
		cfg.Token.Name = symbol
		cfg.Token.Issuer = &proto.Wallet{Address: adminAddress}
	case "curaed", "curbhd", "curtry", "currub", "curusd":
		const requiredArgsCount = 5
		if argsCount != requiredArgsCount {
			return nil, fmt.Errorf("required args length for '%s' is '%d', passed %d",
				channel, requiredArgsCount, argsCount)
		}

		issuerAddress := args[2]
		if issuerAddress == "" {
			return nil, ErrIssuerEmpty
		}
		feeSetter := args[3]
		if feeSetter == "" {
			return nil, ErrFeeSetterEmpty
		}
		feeAdminSetter := args[4]
		if feeAdminSetter == "" {
			return nil, ErrFeeAddressSetterEmpty
		}

		symbol := strings.ToUpper(channel)
		cfg.Contract.Symbol = symbol
		cfg.Contract.Admin = &proto.Wallet{Address: issuerAddress}

		cfg.Token.Name = symbol
		cfg.Token.Issuer = &proto.Wallet{Address: issuerAddress}
		cfg.Token.FeeSetter = &proto.Wallet{Address: feeSetter}
		cfg.Token.FeeSetter = &proto.Wallet{Address: feeAdminSetter}
	case "otf":
		const requiredArgsCount = 4
		if argsCount != requiredArgsCount {
			return nil, fmt.Errorf("required args length for '%s' is '%d', passed %d",
				channel, requiredArgsCount, argsCount)
		}

		issuerAddress := args[2]
		if issuerAddress == "" {
			return nil, ErrIssuerEmpty
		}
		feeSetter := args[3]
		if feeSetter == "" {
			return nil, ErrFeeSetterEmpty
		}

		symbol := strings.ToUpper(channel)
		cfg.Contract.Symbol = symbol
		cfg.Contract.Admin = &proto.Wallet{Address: issuerAddress}

		cfg.Token.Name = symbol
		cfg.Token.Issuer = &proto.Wallet{Address: issuerAddress}
		cfg.Token.FeeSetter = &proto.Wallet{Address: feeSetter}
	default:
		return nil, fmt.Errorf(
			"chaincode '%s' does not have positional args initialization, args: %v",
			channel,
			args,
		)
	}

	cfgBytes, err := protojson.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling config: %w", err)
	}

	return cfgBytes, nil
}
