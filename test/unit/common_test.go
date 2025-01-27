package unit

import (
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures"
	"google.golang.org/protobuf/encoding/protojson"
)

// makeBaseTokenConfig creates config for token, based on BaseToken.
// If feeSetter is not set or empty, Token.FeeSetter will be nil.
// If feeAddressSetter is not set or empty, Token.FeeAddressSetter will be nil.
// Deprecated. Use MockStub.CreateAndSetConfig instead
func makeBaseTokenConfig(
	name, symbol string,
	decimals uint,
	issuer string,
	feeSetter string,
	feeAddressSetter string,
	admin string,
	tracingCollectorEndpoint *proto.CollectorEndpoint,
) string {
	cfg := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   symbol,
			RobotSKI: fixtures.RobotHashedCert,
		},
		Token: &proto.TokenConfig{
			Name:     name,
			Decimals: uint32(decimals),
			Issuer:   &proto.Wallet{Address: issuer},
		},
	}

	if feeSetter != "" {
		cfg.Token.FeeSetter = &proto.Wallet{Address: feeSetter}
	}

	if feeAddressSetter != "" {
		cfg.Token.FeeAddressSetter = &proto.Wallet{Address: feeAddressSetter}
	}

	if admin != "" {
		cfg.Contract.Admin = &proto.Wallet{Address: admin}
	}

	cfg.Contract.TracingCollectorEndpoint = tracingCollectorEndpoint

	cfgBytes, _ := protojson.Marshal(cfg)

	return string(cfgBytes)
}
