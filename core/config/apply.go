package config

import (
	"fmt"

	"github.com/anoideaopen/foundation/internal/config"
	"github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// ContractConfigurable defines methods for contract configuration.
type ContractConfigurable interface {
	ValidateContractConfig(config []byte) error
	ApplyContractConfig(config *proto.ContractConfig) error
	ContractConfig() *proto.ContractConfig
}

// TokenConfigurable defines methods for token configuration.
type TokenConfigurable interface {
	ValidateTokenConfig(config []byte) error
	ApplyTokenConfig(config *proto.TokenConfig) error
	TokenConfig() *proto.TokenConfig
}

// ExternalConfigurable defines methods for external configuration.
type ExternalConfigurable interface {
	ValidateExternalConfig(cfgBytes []byte) error
	ApplyExternalConfig(cfgBytes []byte) error
}

// Apply applies various configurations to a contract.
//
// This function performs the following steps:
//  1. If the contract implements a method to set the ChaincodeStubInterface,
//     it sets the provided stub using SetStub.
//  2. Parses the base contract configuration from the provided bytes and applies it to the contract.
//  3. If the contract implements TokenConfigurable, parses and applies the token configuration.
//  4. If the contract implements ExternalConfigurable, applies the external configuration.
//
// Parameters:
// - contract: An object that implements the ContractConfigurable interface.
// - stub: The ChaincodeStubInterface to be set in the contract if supported.
// - cfgBytes: The configuration bytes to be parsed and applied.
//
// Returns:
// - An error if any step fails, providing information about the failure.
func Apply(contract ContractConfigurable, stub shim.ChaincodeStubInterface, cfgBytes []byte) error {
	// If the contract supports setting the stub, set it
	if ss, ok := contract.(interface {
		SetStub(shim.ChaincodeStubInterface)
	}); ok {
		ss.SetStub(stub)
	}

	contractCfg, err := config.ContractConfigFromBytes(cfgBytes)
	if err != nil {
		return fmt.Errorf("parsing base config: %w", err)
	}

	if contractCfg.GetOptions() == nil {
		contractCfg.Options = new(proto.ChaincodeOptions)
	}

	if err = contract.ApplyContractConfig(contractCfg); err != nil {
		return fmt.Errorf("applying base config: %w", err)
	}

	if tc, ok := contract.(TokenConfigurable); ok {
		tokenCfg, err := config.TokenConfigFromBytes(cfgBytes)
		if err != nil {
			return fmt.Errorf("parsing token config: %w", err)
		}

		if err = tc.ApplyTokenConfig(tokenCfg); err != nil {
			return fmt.Errorf("applying token config: %w", err)
		}
	}

	if ec, ok := contract.(ExternalConfigurable); ok {
		if err = ec.ApplyExternalConfig(cfgBytes); err != nil {
			return fmt.Errorf("applying external config: %w", err)
		}
	}

	return nil
}
