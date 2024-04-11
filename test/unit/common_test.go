package unit

import (
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"google.golang.org/protobuf/encoding/protojson"
)

func step(t *testing.T, name string, skip bool, f func()) {
	if skip {
		t.Log(fmt.Sprintf("⚠warning⚠ step '%s' is skipped", name))
		return
	}

	t.Log(name)
	f()
}

// makeBaseTokenConfig creates config for token, based on BaseToken.
// If feeSetter is not set or empty, Token.FeeSetter will be nil.
// If feeAddressSetter is not set or empty, Token.FeeAddressSetter will be nil.
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
			RobotSKI: fixtures_test.RobotHashedCert,
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
