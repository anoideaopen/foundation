package unit

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestConfigToken chaincode with extended TokenConfig fields
type TestExtConfigToken struct {
	core.BaseContract
	ExtConfig
}

// GetID returns chaincode identifier. It required by core.BaseContractInterface.
func (tect *TestExtConfigToken) GetID() string {
	return "TEST"
}

func (tect *TestExtConfigToken) ValidateExtConfig(config []byte) error {
	var (
		ec      ExtConfig
		cfgFull proto.Config
	)

	if err := protojson.Unmarshal(config, &cfgFull); err != nil {
		return fmt.Errorf("unmarshalling config: %w", err)
	}

	if cfgFull.ExtConfig.MessageIs(&ec) {
		if err := cfgFull.ExtConfig.UnmarshalTo(&ec); err != nil {
			return fmt.Errorf("unmarshalling ext config: %w", err)
		}
	}

	if err := ec.Validate(); err != nil {
		return fmt.Errorf("validating ext config: %w", err)
	}

	return nil
}

func (tect *TestExtConfigToken) ApplyExtConfig(cfgBytes []byte) error {
	var (
		extConfig ExtConfig
		cfgFull   proto.Config
	)

	if err := protojson.Unmarshal(cfgBytes, &cfgFull); err != nil {
		return fmt.Errorf("unmarshalling config: %w", err)
	}

	if cfgFull.ExtConfig.MessageIs(&extConfig) {
		if err := cfgFull.ExtConfig.UnmarshalTo(&extConfig); err != nil {
			return fmt.Errorf("unmarshalling ext config: %w", err)
		}
	}

	tect.Asset = extConfig.Asset
	tect.Amount = extConfig.Amount
	tect.Issuer = extConfig.Issuer

	return nil
}

// QueryMetadata returns Metadata
func (tect *TestExtConfigToken) QueryExtConfig() (*ExtConfig, error) {
	return &tect.ExtConfig, nil
}

// TestInitWithExtConfig tests chaincode initialization of token with common config.
func TestInitWithExtConfig(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()
	issuer := ledgerMock.NewWallet()

	asset, amount := "SOME_ASSET", "42"

	extCfgEtl := &ExtConfig{
		Asset:  asset,
		Amount: amount,
		Issuer: &proto.Wallet{Address: issuer.Address()},
	}
	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "EXTCC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: issuer.Address()},
		},
	}
	cfgEtl.ExtConfig, _ = anypb.New(extCfgEtl)
	config, _ := protojson.Marshal(cfgEtl)

	tt := TestExtConfigToken{}
	step(t, "Init new chaincode", false, func() {
		initMsg := ledgerMock.NewCC(testTokenCCName, &tt, string(config))
		require.Empty(t, initMsg)
	})

	step(t, "Read and validate ExtConfig data", false, func() {
		data := user1.Invoke(testTokenCCName, "extConfig")
		require.NotEmpty(t, data)

		var m ExtConfig
		err := json.Unmarshal([]byte(data), &m)
		require.NoError(t, err)

		require.Equal(t, asset, m.Asset)
		require.Equal(t, amount, m.Amount)
		require.Equal(t, issuer.Address(), m.Issuer.Address)
	})
}
