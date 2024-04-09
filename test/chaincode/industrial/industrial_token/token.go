package industrialtoken

import (
	"errors"
	"strings"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
)

// Group base struct
type Group struct {
	ID       string
	Emission uint64
	Maturity string
	Note     string
}

// ITInterface - base method for an industrial token prototype
type ITInterface interface {
	core.BaseContractInterface

	GetRateAndLimits(string, string) (*proto.TokenRate, bool, error)
}

// IndustrialToken base struct
type IndustrialToken struct {
	core.BaseContract

	extConfig *ExtConfig

	config *proto.Industrial
}

// GetID returns token id
func (it *IndustrialToken) GetID() string {
	return it.ContractConfig().Symbol
}

func (it *IndustrialToken) Issuer() *types.Address {
	if it.extConfig.Issuer == nil {
		panic("issuer is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.Issuer.Address)
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) FeeSetter() *types.Address {
	if it.extConfig.FeeSetter == nil {
		panic("fee-setter is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.FeeSetter.Address)
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) FeeAddressSetter() *types.Address {
	if it.extConfig.FeeAddressSetter == nil {
		panic("fee-address-setter is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.FeeAddressSetter.Address)
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) loadConfigUnlessLoaded() error {
	data, err := it.GetStub().GetState("tokenMetadata")
	if err != nil {
		return err
	}
	if it.config == nil {
		it.config = &proto.Industrial{}
	}

	if len(data) == 0 {
		return nil
	}
	return pb.Unmarshal(data, it.config)
}

func (it *IndustrialToken) saveConfig() error {
	data, err := pb.Marshal(it.config)
	if err != nil {
		return err
	}
	return it.GetStub().PutState("tokenMetadata", data)
}

func (it *IndustrialToken) setFee(
	currency string,
	fee *big.Int,
	floor *big.Int,
	cap *big.Int,
) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if it.config.Fee == nil {
		it.config.Fee = &proto.TokenFee{}
	}
	if currency == it.ContractConfig().Symbol {
		it.config.Fee.Currency = currency
		it.config.Fee.Fee = fee.Bytes()
		it.config.Fee.Floor = floor.Bytes()
		it.config.Fee.Cap = cap.Bytes()
		return it.saveConfig()
	}
	for _, rate := range it.config.Rates {
		if rate.Currency == currency {
			it.config.Fee.Currency = currency
			it.config.Fee.Fee = fee.Bytes()
			it.config.Fee.Floor = floor.Bytes()
			it.config.Fee.Cap = cap.Bytes()
			return it.saveConfig()
		}
	}
	return errors.New("unknown currency")
}

// GetRateAndLimits returns token rate and limits from metadata
func (it *IndustrialToken) GetRateAndLimits(dealType string, currency string) (*proto.TokenRate, bool, error) {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return nil, false, err
	}
	for _, r := range it.config.Rates {
		if r.DealType == dealType && r.Currency == currency {
			return r, true, nil
		}
	}
	return &proto.TokenRate{}, false, nil
}

// Initialize - token initialization
func (it *IndustrialToken) Initialize(groups []Group) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}

	if it.config.Initialized {
		return nil
	}

	industrialGroups := make([]*proto.IndustrialGroup, 0, len(groups))
	for _, group := range groups {
		if strings.Contains(group.ID, ",") {
			return errors.New("wrong group name")
		}

		maturity, err := time.Parse(timeFormat, group.Maturity)
		if err != nil {
			return err
		}
		industrialGroups = append(industrialGroups, &proto.IndustrialGroup{
			Id:       group.ID,
			Maturity: maturity.Unix(),
			Emission: new(big.Int).SetUint64(group.Emission).Bytes(),
			Note:     group.Note,
		})
	}

	it.config.Groups = industrialGroups
	it.config.Initialized = true

	for _, x := range industrialGroups {
		if err := it.IndustrialBalanceAdd(
			it.ContractConfig().Symbol+"_"+x.Id,
			it.Issuer(),
			new(big.Int).SetBytes(x.Emission),
			"initial emit",
		); err != nil {
			return err
		}
	}

	return it.saveConfig()
}
