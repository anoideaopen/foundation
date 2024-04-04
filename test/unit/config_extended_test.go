package unit

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/stretchr/testify/require"
)

//go:generate protoc -I=. -I=../../proto/ --go_out=. --validate_out=lang=go:. ext_config.proto

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
	var ec ExtConfig

	if err := json.Unmarshal(config, &ec); err != nil {
		return fmt.Errorf("unmarshalling ext config: %w", err)
	}

	if err := ec.Validate(); err != nil {
		return fmt.Errorf("validating ext config: %w", err)
	}

	return nil
}

func (tect *TestExtConfigToken) ApplyExtConfig(cfgBytes []byte) error {
	var extConfig ExtConfig
	if err := json.Unmarshal(cfgBytes, &extConfig); err != nil {
		return err
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

	config := fmt.Sprintf(
		`
{
	"contract": {
		"symbol":"%s",
		"robotSKI":"%s",
		"admin":{"address":"%s"}
	},
	"asset":"%s",
	"amount":"%s",
	"issuer":{"address":"%s"}
}`,
		"EXTCC",
		fixtures_test.RobotHashedCert,
		issuer.Address(),
		asset,
		amount,
		issuer.Address(),
	)

	tt := TestExtConfigToken{}
	step(t, "Init new chaincode", false, func() {
		initMsg := ledgerMock.NewCC(testTokenCCName, &tt, config)
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
