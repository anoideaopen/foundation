package token

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
)

const metadataKey = "tokenMetadata"

// Tokener is the interface for tokens
type Tokener interface {
	core.BaseContractInterface
	core.TokenConfigurable

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

	// stores emission amount, fees and rates.
	config *proto.Token
}

// Issuer returns the issuer of the token
func (bt *BaseToken) Issuer() *types.Address {
	addr, err := types.AddrFromBase58Check(bt.tokenConfig.Issuer.Address)
	if err != nil {
		panic(err)
	}
	return addr
}

// FeeSetter returns the fee setter of the token
func (bt *BaseToken) FeeSetter() *types.Address {
	if bt.TokenConfig().FeeSetter == nil || bt.TokenConfig().FeeSetter.Address == "" {
		panic("feeSetter is not set or empty")
	}

	addr, err := types.AddrFromBase58Check(bt.TokenConfig().FeeSetter.Address)
	if err != nil {
		panic(fmt.Sprintf("parsing address: %s", err))
	}

	return addr
}

// FeeAddressSetter returns the fee address setter of the token
func (bt *BaseToken) FeeAddressSetter() *types.Address {
	if bt.TokenConfig().FeeAddressSetter == nil || bt.tokenConfig.FeeAddressSetter.Address == "" {
		panic("feeAddressSetter is not set or empty")
	}

	addr, err := types.AddrFromBase58Check(bt.tokenConfig.FeeAddressSetter.Address)
	if err != nil {
		panic(err)
	}
	return addr
}

// GetID returns the ID of the token
func (bt *BaseToken) GetID() string {
	return bt.TokenConfig().Name
}

func (bt *BaseToken) loadConfigUnlessLoaded() error {
	data, err := bt.GetStub().GetState(metadataKey)
	if err != nil {
		return err
	}
	if bt.config == nil {
		bt.config = &proto.Token{}
	}

	if len(data) == 0 {
		return nil
	}
	return pb.Unmarshal(data, bt.config)
}

func (bt *BaseToken) saveConfig() error {
	data, err := pb.Marshal(bt.config)
	if err != nil {
		return err
	}
	return bt.GetStub().PutState(metadataKey, data)
}

// EmissionAdd adds emission
func (bt *BaseToken) EmissionAdd(amount *big.Int) error {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if bt.config.TotalEmission == nil {
		bt.config.TotalEmission = new(big.Int).Bytes()
	}
	bt.config.TotalEmission = new(big.Int).Add(new(big.Int).SetBytes(bt.config.TotalEmission), amount).Bytes()
	return bt.saveConfig()
}

// EmissionSub subtracts emission
func (bt *BaseToken) EmissionSub(amount *big.Int) error {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if bt.config.TotalEmission == nil {
		bt.config.TotalEmission = new(big.Int).Bytes()
	}
	if new(big.Int).SetBytes(bt.config.TotalEmission).Cmp(amount) < 0 {
		return errors.New("emission can't become negative")
	}
	bt.config.TotalEmission = new(big.Int).Sub(new(big.Int).SetBytes(bt.config.TotalEmission), amount).Bytes()
	return bt.saveConfig()
}

func (bt *BaseToken) setFee(currency string, fee *big.Int, floor *big.Int, cap *big.Int) error {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return err
	}

	if bt.config.Fee == nil {
		bt.config.Fee = &proto.TokenFee{}
	}

	if currency == bt.ContractConfig().Symbol {
		bt.config.Fee.Currency = currency
		bt.config.Fee.Fee = fee.Bytes()
		bt.config.Fee.Floor = floor.Bytes()
		bt.config.Fee.Cap = cap.Bytes()
		return bt.saveConfig()
	}

	for _, rate := range bt.config.Rates {
		if rate.Currency == currency {
			bt.config.Fee.Currency = currency
			bt.config.Fee.Fee = fee.Bytes()
			bt.config.Fee.Floor = floor.Bytes()
			bt.config.Fee.Cap = cap.Bytes()
			return bt.saveConfig()
		}
	}

	return errors.New("unknown currency")
}

// GetRateAndLimits returns rate and limits for the deal type and currency
func (bt *BaseToken) GetRateAndLimits(dealType string, currency string) (*proto.TokenRate, bool, error) {
	if err := bt.loadConfigUnlessLoaded(); err != nil {
		return nil, false, err
	}
	for _, r := range bt.config.Rates {
		if r.DealType == dealType && r.Currency == currency {
			return r, true, nil
		}
	}
	return &proto.TokenRate{}, false, nil
}

func (bt *BaseToken) ValidateTokenConfig(config []byte) error {
	fullCfg := struct {
		proto.TokenConfig `json:"token"`
	}{}

	if err := json.Unmarshal(config, &fullCfg); err != nil {
		return fmt.Errorf("unmarshalling token config data failed: %w", err)
	}

	return fullCfg.Validate()
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
