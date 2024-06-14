package unit

import (
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestInitWithPositionalParameters(t *testing.T) {
	t.Parallel()

	robotSKI := fixtures_test.RobotHashedCert

	tt := []struct {
		channel       string
		args          []string
		bci           core.BaseContractInterface
		initMsg       string
		adminIsIssuer bool // set to true if admin has same address as issuer
	}{
		{
			channel: "nft",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.AdminAddr,
			},
			bci: &core.BaseContract{},
		},
		{
			channel: "ct",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.IssuerAddr,
				fixtures_test.AdminAddr,
			},
			bci: &token.BaseToken{},
		},
		{
			channel: "nmmmulti",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.AdminAddr,
			},
			bci:     &core.BaseContract{},
			initMsg: "",
		},
		{
			channel: "curusd",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.IssuerAddr,
				fixtures_test.FeeSetterAddr,
				fixtures_test.FeeAddressSetterAddr,
			},
			bci:           &core.BaseContract{},
			initMsg:       "",
			adminIsIssuer: true,
		},
		{
			channel: "non-handled-channel",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.AdminAddr,
			},
			bci:     &core.BaseContract{},
			initMsg: "chaincode 'non-handled-channel' does not have positional args initialization",
		},
		{
			channel: "otf",
			args: []string{
				"<backend_ski>,deprecated",
				robotSKI,
				fixtures_test.IssuerAddr,
				fixtures_test.FeeSetterAddr,
			},
			bci:           &core.BaseContract{},
			initMsg:       "",
			adminIsIssuer: true,
		},
	}

	for _, test := range tt {
		t.Run(test.channel, func(t *testing.T) {
			ledger := mock.NewLedger(t)

			initMsg := ledger.NewCCArgsArr(test.channel, test.bci, test.args)
			if test.initMsg != "" {
				require.Contains(t, initMsg, test.initMsg)
				return
			} else {
				require.Empty(t, initMsg)
			}

			stub := ledger.GetStubByKey(test.channel)

			cfgBytes, err := config.Load(stub)
			require.NoError(t, err)

			cfg, err := config.FromBytes(cfgBytes)
			require.NoError(t, err)

			symbolExpected := strings.ToUpper(test.channel)

			require.Equal(t, symbolExpected, cfg.GetContract().Symbol)
			require.Equal(t, robotSKI, cfg.GetContract().RobotSKI)
			if test.adminIsIssuer {
				require.Equal(t, fixtures_test.IssuerAddr, cfg.GetContract().GetAdmin().GetAddress())
			} else {
				require.Equal(t, fixtures_test.AdminAddr, cfg.GetContract().GetAdmin().GetAddress())
			}

			if _, ok := test.bci.(token.Tokener); ok {
				require.Equal(t, fixtures_test.IssuerAddr, cfg.GetToken().GetIssuer().GetAddress())
			}
		})
	}
}
