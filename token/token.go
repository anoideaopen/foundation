package token

import (
	"errors"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
)

const (
	// FeeSetterArgPos is the position of the fee setter in the init args
	FeeSetterArgPos = 1
	// FeeAddressSetterArgPos is the position of the fee address setter in the init args
	FeeAddressSetterArgPos = 2
	metadataKey            = "tokenMetadata"
)

// Tokener is the interface for tokens
type Tokener interface {
	core.BaseContractInterface
	EmissionAdd(*big.Int) error
	EmissionSub(*big.Int) error
	GetRateAndLimits(string, string) (*proto.TokenRate, bool, error)
}

// BaseToken is the base token
type BaseToken struct {
	core.BaseContract
	Name            string
	Symbol          string
	Decimals        uint
	UnderlyingAsset string

	config *proto.Token
}

// Issuer returns the issuer of the token
func (bt *BaseToken) Issuer() *types.Address {
	addr, err := types.AddrFromBase58Check(bt.GetInitArg(0))
	if err != nil {
		panic(err)
	}
	return addr
}

// FeeSetter returns the fee setter of the token
func (bt *BaseToken) FeeSetter() *types.Address {
	addr, err := types.AddrFromBase58Check(bt.GetInitArg(FeeSetterArgPos))
	if err != nil {
		panic(err)
	}
	return addr
}

// FeeAddressSetter returns the fee address setter of the token
func (bt *BaseToken) FeeAddressSetter() *types.Address {
	addr, err := types.AddrFromBase58Check(bt.GetInitArg(FeeAddressSetterArgPos))
	if err != nil {
		panic(err)
	}
	return addr
}

// GetID returns the ID of the token
func (bt *BaseToken) GetID() string {
	return bt.Symbol
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
	if currency == bt.Symbol {
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
