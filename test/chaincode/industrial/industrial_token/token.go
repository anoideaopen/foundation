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
}

// GetID returns token id
func (it *IndustrialToken) GetID() string {
	return it.ContractConfig().GetSymbol()
}

func (it *IndustrialToken) Issuer() *types.Address {
	if it.extConfig.GetIssuer() == nil {
		panic("issuer is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.GetIssuer().GetAddress())
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) FeeSetter() *types.Address {
	if it.extConfig.GetFeeSetter() == nil {
		panic("fee-setter is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.GetFeeSetter().GetAddress())
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) FeeAddressSetter() *types.Address {
	if it.extConfig.GetFeeAddressSetter() == nil {
		panic("fee-address-setter is not set")
	}

	addr, err := types.AddrFromBase58Check(it.extConfig.GetFeeAddressSetter().GetAddress())
	if err != nil {
		panic(err)
	}

	return addr
}

func (it *IndustrialToken) loadConfig() (*proto.Industrial, error) {
	data, err := it.GetStub().GetState("tokenMetadata")
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &proto.Industrial{}, nil
	}

	cfg := &proto.Industrial{}
	err = pb.Unmarshal(data, cfg)

	return cfg, err
}

func (it *IndustrialToken) saveConfig(cfg *proto.Industrial) error {
	data, err := pb.Marshal(cfg)
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
	cfg, err := it.loadConfig()
	if err != nil {
		return err
	}
	if cfg.GetFee() == nil {
		cfg.Fee = &proto.TokenFee{}
	}
	if currency == it.ContractConfig().GetSymbol() {
		cfg.Fee.Currency = currency
		cfg.Fee.Fee = fee.Bytes()
		cfg.Fee.Floor = floor.Bytes()
		cfg.Fee.Cap = cap.Bytes()
		return it.saveConfig(cfg)
	}
	for _, rate := range cfg.GetRates() {
		if rate.GetCurrency() == currency {
			cfg.Fee.Currency = currency
			cfg.Fee.Fee = fee.Bytes()
			cfg.Fee.Floor = floor.Bytes()
			cfg.Fee.Cap = cap.Bytes()
			return it.saveConfig(cfg)
		}
	}
	return errors.New("unknown currency")
}

// GetRateAndLimits returns token rate and limits from metadata
func (it *IndustrialToken) GetRateAndLimits(dealType string, currency string) (*proto.TokenRate, bool, error) {
	cfg, err := it.loadConfig()
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

// Initialize - token initialization
func (it *IndustrialToken) Initialize(groups []Group) error {
	cfg, err := it.loadConfig()
	if err != nil {
		return err
	}

	if cfg.GetInitialized() {
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

	cfg.Groups = industrialGroups
	cfg.Initialized = true

	for _, x := range industrialGroups {
		if err = it.IndustrialBalanceAdd(
			it.ContractConfig().GetSymbol()+"_"+x.GetId(),
			it.Issuer(),
			new(big.Int).SetBytes(x.GetEmission()),
			"initial emit",
		); err != nil {
			return err
		}
	}

	return it.saveConfig(cfg)
}
