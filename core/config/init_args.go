package config

import (
	"fmt"
	"strings"

	"github.com/anoideaopen/foundation/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// FromInitArgs parses positional initialization arguments and generates JSON-config of []byte type.
// Accepts the channel name (chaincode) and the list of positional initialization parameters.
// Only needed to maintain backward compatibility.
// Marked for deletion after all deploy tools will be switched to JSON-config initialization of chaincodes.
// Deprecated
func FromInitArgs(channel string, args []string) ([]byte, error) {
	const minArgsCount = 2
	argsCount := len(args)
	if argsCount < minArgsCount {
		return nil, fmt.Errorf("minimum required args length is '%d', passed %d",
			argsCount, minArgsCount)
	}

	var (
		cfg *proto.Config
		err error
	)

	symbol := strings.ToUpper(channel)

	switch channel {
	case "nft", "dcdac", "ndm", "rub", "it":
		cfg, err = ConfigWithAdmin(symbol, args)
	case "ct", "hermitage", "dcrsb", "minetoken", "invclass", "vote":
		cfg, err = ConfigWithIssuerAndAdmin(symbol, args)
	case "nmmmulti", "invmulti", "dcmulti":
		cfg, err = ConfigWithAdmin(symbol, args)
	case "curaed", "curbhd", "curtry", "currub", "curusd":
		cfg, err = ConfigWithIssuerFeeSetterAndFeeAddressSetter(symbol, args)
	case "otf":
		cfg, err = ConfigWithIssuerAndFeeSetter(symbol, args)
	default:
		return nil, fmt.Errorf(
			"chaincode '%s' does not have positional args initialization, args: %v",
			channel,
			args,
		)
	}

	if err != nil {
		return nil, err
	}

	cfgBytes, err := protojson.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling config: %w", err)
	}

	return cfgBytes, nil
}

// ConfigWithAdmin configures the proto.Config with an admin address.
// Args: [platformSKI (deprecated), robotSKI, adminAddress]
func ConfigWithAdmin(symbol string, args []string) (*proto.Config, error) {
	const requiredArgsCount = 3
	if len(args) != requiredArgsCount {
		return nil, fmt.Errorf("required args length is '%d', passed %d",
			requiredArgsCount, len(args))
	}

	_ = args[0] // PlatformSKI (backend) - deprecated
	robotSKI := args[1]
	adminAddress := args[2]
	if adminAddress == "" {
		return nil, ErrAdminEmpty
	}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   symbol,
			Admin:    &proto.Wallet{Address: adminAddress},
			RobotSKI: robotSKI,
		},
		Token: &proto.TokenConfig{
			Name:   symbol,
			Issuer: &proto.Wallet{Address: adminAddress},
		},
	}
	return cfg, nil
}

// ConfigWithIssuerAndAdmin configures the proto.Config with an issuer and admin address.
// Args: [platformSKI (deprecated), robotSKI, issuerAddress, adminAddress]
func ConfigWithIssuerAndAdmin(symbol string, args []string) (*proto.Config, error) {
	const requiredArgsCount = 4
	if len(args) != requiredArgsCount {
		return nil, fmt.Errorf("required args length is '%d', passed %d",
			requiredArgsCount, len(args))
	}

	_ = args[0] // PlatformSKI (backend) - deprecated
	robotSKI := args[1]
	issuerAddress := args[2]
	if issuerAddress == "" {
		return nil, ErrIssuerEmpty
	}
	adminAddress := args[3]
	if adminAddress == "" {
		return nil, ErrAdminEmpty
	}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   symbol,
			Admin:    &proto.Wallet{Address: adminAddress},
			RobotSKI: robotSKI,
		},
		Token: &proto.TokenConfig{
			Name:   symbol,
			Issuer: &proto.Wallet{Address: issuerAddress},
		},
	}
	return cfg, nil
}

// ConfigWithIssuerFeeSetterAndFeeAddressSetter configures the proto.Config with an issuer, fee setter, and fee admin setter address.
// Args: [platformSKI (deprecated), robotSKI, issuerAddress, feeSetter, feeAddressSetter]
func ConfigWithIssuerFeeSetterAndFeeAddressSetter(symbol string, args []string) (*proto.Config, error) {
	const requiredArgsCount = 5
	if len(args) != requiredArgsCount {
		return nil, fmt.Errorf("required args length is '%d', passed %d",
			requiredArgsCount, len(args))
	}

	_ = args[0] // PlatformSKI (backend) - deprecated
	robotSKI := args[1]
	issuerAddress := args[2]
	if issuerAddress == "" {
		return nil, ErrIssuerEmpty
	}
	feeSetter := args[3]
	if feeSetter == "" {
		return nil, ErrFeeSetterEmpty
	}
	feeAddressSetter := args[4]
	if feeAddressSetter == "" {
		return nil, ErrFeeAddressSetterEmpty
	}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   symbol,
			Admin:    &proto.Wallet{Address: issuerAddress},
			RobotSKI: robotSKI,
		},
		Token: &proto.TokenConfig{
			Name:             symbol,
			Issuer:           &proto.Wallet{Address: issuerAddress},
			FeeSetter:        &proto.Wallet{Address: feeSetter},
			FeeAddressSetter: &proto.Wallet{Address: feeAddressSetter},
		},
	}
	return cfg, nil
}

// ConfigWithIssuerAndFeeSetter configures the proto.Config with an issuer and fee setter address.
// Args: [platformSKI (deprecated), robotSKI, issuerAddress, feeSetter]
func ConfigWithIssuerAndFeeSetter(symbol string, args []string) (*proto.Config, error) {
	const requiredArgsCount = 4
	if len(args) != requiredArgsCount {
		return nil, fmt.Errorf("required args length is '%d', passed %d",
			requiredArgsCount, len(args))
	}

	_ = args[0] // PlatformSKI (backend) - deprecated
	robotSKI := args[1]
	issuerAddress := args[2]
	if issuerAddress == "" {
		return nil, ErrIssuerEmpty
	}
	feeSetter := args[3]
	if feeSetter == "" {
		return nil, ErrFeeSetterEmpty
	}

	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   symbol,
			Admin:    &proto.Wallet{Address: issuerAddress},
			RobotSKI: robotSKI,
		},
		Token: &proto.TokenConfig{
			Name:      symbol,
			Issuer:    &proto.Wallet{Address: issuerAddress},
			FeeSetter: &proto.Wallet{Address: feeSetter},
		},
	}
	return cfg, nil
}
