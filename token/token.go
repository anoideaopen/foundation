package token

import (
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"google.golang.org/protobuf/encoding/protojson"
)

const metadataKey = "tokenMetadata"

// Tokener is the interface for tokens
type Tokener interface {
	core.BaseContractInterface
	config.TokenConfigurator

	EmissionAdd(*big.Int) error
	EmissionSub(*big.Int) error
	GetRateAndLimits(string, string) (*proto.TokenRate, bool, error)
}

var (
	_ Tokener                    = &BaseToken{}
	_ core.BaseContractInterface = &BaseToken{}
)

// BaseToken represents a chaincode with configurable token-attributes.
// Implements core.BaseContractInterface by embedding core.BaseContract.
type BaseToken struct {
	core.BaseContract

	// stores token-specific attributes.
	tokenConfig *proto.TokenConfig
}

// Issuer returns the issuer of the token
func (bt *BaseToken) Issuer() *types.Address {
	addr, err := types.AddrFromBase58Check(bt.tokenConfig.GetIssuer().GetAddress())
	if err != nil {
		panic(err)
	}
	return addr
}

// FeeSetter returns the fee setter of the token
func (bt *BaseToken) FeeSetter() *types.Address {
	if bt.TokenConfig().GetFeeSetter().GetAddress() == "" {
		panic("feeSetter is not set or empty")
	}

	addr, err := types.AddrFromBase58Check(bt.TokenConfig().GetFeeSetter().GetAddress())
	if err != nil {
		panic(fmt.Sprintf("parsing address: %s", err))
	}

	return addr
}

// FeeAddressSetter returns the fee address setter of the token
func (bt *BaseToken) FeeAddressSetter() *types.Address {
	if bt.tokenConfig.GetFeeAddressSetter().GetAddress() == "" {
		panic("feeAddressSetter is not set or empty")
	}

	addr, err := types.AddrFromBase58Check(bt.tokenConfig.GetFeeAddressSetter().GetAddress())
	if err != nil {
		panic(err)
	}
	return addr
}

// GetID returns the ID of the token
func (bt *BaseToken) GetID() string {
	return bt.TokenConfig().GetName()
}

func (bt *BaseToken) loadConfig() (*proto.Token, error) {
	data, err := bt.GetStub().GetState(metadataKey)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &proto.Token{}, nil
	}

	cfg := &proto.Token{}
	err = pb.Unmarshal(data, cfg)

	return cfg, err
}

func (bt *BaseToken) saveConfig(cfg *proto.Token) error {
	data, err := pb.Marshal(cfg)
	if err != nil {
		return err
	}
	return bt.GetStub().PutState(metadataKey, data)
}

// EmissionAdd adds emission
func (bt *BaseToken) EmissionAdd(amount *big.Int) error {
	cfg, err := bt.loadConfig()
	if err != nil {
		return err
	}
	if cfg.GetTotalEmission() == nil {
		cfg.TotalEmission = new(big.Int).Bytes()
	}

	cfg.TotalEmission = new(big.Int).Add(new(big.Int).SetBytes(cfg.GetTotalEmission()), amount).Bytes()
	return bt.saveConfig(cfg)
}

// EmissionSub subtracts emission
func (bt *BaseToken) EmissionSub(amount *big.Int) error {
	cfg, err := bt.loadConfig()
	if err != nil {
		return err
	}
	if cfg.GetTotalEmission() == nil {
		cfg.TotalEmission = new(big.Int).Bytes()
	}
	if new(big.Int).SetBytes(cfg.GetTotalEmission()).Cmp(amount) < 0 {
		return errors.New("emission can't become negative")
	}
	cfg.TotalEmission = new(big.Int).Sub(new(big.Int).SetBytes(cfg.GetTotalEmission()), amount).Bytes()
	return bt.saveConfig(cfg)
}

func (bt *BaseToken) setFee(currency string, fee *big.Int, floor *big.Int, cap *big.Int) error {
	cfg, err := bt.loadConfig()
	if err != nil {
		return err
	}

	if cfg.GetFee() == nil {
		cfg.Fee = &proto.TokenFee{}
	}

	if currency == bt.ContractConfig().GetSymbol() {
		cfg.Fee.Currency = currency
		cfg.Fee.Fee = fee.Bytes()
		cfg.Fee.Floor = floor.Bytes()
		cfg.Fee.Cap = cap.Bytes()
		return bt.saveConfig(cfg)
	}

	for _, rate := range cfg.GetRates() {
		if rate.GetCurrency() == currency {
			cfg.Fee.Currency = currency
			cfg.Fee.Fee = fee.Bytes()
			cfg.Fee.Floor = floor.Bytes()
			cfg.Fee.Cap = cap.Bytes()
			return bt.saveConfig(cfg)
		}
	}

	return errors.New("unknown currency")
}

// GetRateAndLimits returns rate and limits for the deal type and currency
func (bt *BaseToken) GetRateAndLimits(dealType string, currency string) (*proto.TokenRate, bool, error) {
	cfg, err := bt.loadConfig()
	if err != nil {
		return nil, false, err
	}
	for _, r := range cfg.GetRates() {
		if r.GetDealType() == dealType && r.GetCurrency() == currency {
			return r, true, nil
		}
	}
	return &proto.TokenRate{}, false, nil
}

func (bt *BaseToken) ValidateTokenConfig(config []byte) error {
	var cfg proto.Config

	if err := protojson.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("unmarshalling token config data failed: %w", err)
	}

	return cfg.Validate()
}

func (bt *BaseToken) ApplyTokenConfig(config *proto.TokenConfig) error {
	bt.tokenConfig = config

	return nil
}

func (bt *BaseToken) TokenConfig() *proto.TokenConfig {
	if bt.tokenConfig == nil {
		panic("token config is not set")
	}

	return bt.tokenConfig
}
