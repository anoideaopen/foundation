package token

import (
	"encoding/json"

	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
)

func makeBaseTokenConfig(
	name, symbol string,
	decimals uint,
	issuer, feeSetter, feeAddressSetter string,
) string {
	ow := &pb.Wallet{}
	if issuer == "" {
		ow.Address = fixtures_test.AdminAddr
	} else {
		ow.Address = issuer
	}

	fsw := &pb.Wallet{}
	if feeSetter != "" {
		fsw.Address = feeSetter
	} else {
		fsw = nil
	}

	fasw := &pb.Wallet{}
	if feeAddressSetter != "" {
		fasw.Address = feeAddressSetter
	} else {
		fasw = nil
	}

	cfg := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   symbol,
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    fixtures_test.Admin,
		},
		Token: &pb.TokenConfig{
			Name:             name,
			Decimals:         uint32(decimals),
			Issuer:           ow,
			FeeSetter:        fsw,
			FeeAddressSetter: fasw,
		},
	}

	cfgBytes, _ := json.Marshal(cfg)

	return string(cfgBytes)
}
