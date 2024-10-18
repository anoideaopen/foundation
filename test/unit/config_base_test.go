package unit

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type ConfigData struct {
	*proto.Config
}

// TestConfigToken chaincode with default TokenConfig fields
type TestConfigToken struct {
	token.BaseToken
}

// disabledFnContract is for testing disabled functions.
type disabledFnContract struct {
	core.BaseContract
}

func (*disabledFnContract) TxTestFunction(_ *types.Sender) error {
	return nil
}

func (*disabledFnContract) GetID() string {
	return "TEST"
}

var _ config.TokenConfigurator = &TestConfigToken{}

func (tct *TestConfigToken) QueryConfig() (ConfigData, error) {
	return ConfigData{
		&proto.Config{
			Contract: tct.ContractConfig(),
			Token:    tct.TokenConfig(),
		},
	}, nil
}

func (tct *TestConfigToken) TxSetEmitAmount(_ *types.Sender, amount string) error {
	const emitKey = "emit"
	if err := tct.GetStub().PutState(emitKey, []byte(amount)); err != nil {
		return fmt.Errorf("putting amount '%s' to state key '%s': %w",
			amount, emitKey, err)
	}

	return nil
}

func (tct *TestConfigToken) QueryEmitAmount() (string, error) {
	const emitKey = "emit"
	amountBytes, err := tct.GetStub().GetState(emitKey)
	if err != nil {
		return "", fmt.Errorf("getting data from state key '%s': %w", emitKey, err)
	}

	return string(amountBytes), nil
}

// TestInitWithCommonConfig tests chaincode initialization of token with common config.
func TestInitWithCommonConfig(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()
	issuer := ledgerMock.NewWallet()

	ttName, ttSymbol, ttDecimals := "test token", "TT", uint32(8)

	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol: ttSymbol,
			Options: &proto.ChaincodeOptions{
				DisableMultiSwaps: true,
			},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: issuer.Address()},
		},
		Token: &proto.TokenConfig{
			Name:     ttName,
			Decimals: ttDecimals,
			Issuer:   &proto.Wallet{Address: issuer.Address()},
		},
	}
	config, _ := protojson.Marshal(cfgEtl)

	step(t, "Init new chaincode", false, func() {
		message := ledgerMock.NewCC("tt", &TestConfigToken{}, string(config))
		require.Empty(t, message)
	})

	var cfg proto.Config
	step(t, "Fetch config", false, func() {
		data := user1.Invoke("tt", "config")
		require.NotEmpty(t, data)

		err := json.Unmarshal([]byte(data), &cfg)
		require.NoError(t, err)
	})

	step(t, "Validate contract config", false, func() {
		require.Equal(t, ttSymbol, cfg.Contract.Symbol)
		require.Equal(t, fixtures_test.RobotHashedCert, cfg.Contract.RobotSKI)
		require.Equal(t, false, cfg.Contract.Options.DisableSwaps)
		require.Equal(t, true, cfg.Contract.Options.DisableMultiSwaps)
	})

	step(t, "Validate token config", false, func() {
		require.Equal(t, ttName, cfg.Token.Name)
		require.Equal(t, ttDecimals, cfg.Token.Decimals)
		require.Equal(t, issuer.Address(), cfg.Token.Issuer.Address)
	})
}

func TestWithConfigMapperFunc(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()
	issuer := ledgerMock.NewWallet()

	ttName, ttSymbol, ttDecimals := "test token", "TT", uint32(8)
	step(t, "Init new chaincode", false, func() {
		initArgs := []string{
			"",                            // PlatformSKI (backend) - deprecated
			fixtures_test.RobotHashedCert, // RobotSKI
			issuer.Address(),              // IssuerAddress
			fixtures_test.AdminAddr,       // AdminAddress
		}
		message := ledgerMock.NewCCArgsArr("tt", &TestConfigToken{}, initArgs, core.WithConfigMapperFunc(
			func(args []string) (*proto.Config, error) {
				const requiredArgsCount = 4

				if len(args) != requiredArgsCount {
					return nil, fmt.Errorf(
						"required args length '%s' is '%d', passed %d",
						ttSymbol,
						requiredArgsCount,
						len(args),
					)
				}

				_ = args[0] // PlatformSKI (backend) - deprecated

				robotSKI := args[1]
				if robotSKI == "" {
					return nil, fmt.Errorf("robot ski is empty")
				}

				issuerAddress := args[2]
				if issuerAddress == "" {
					return nil, fmt.Errorf("issuer address is empty")
				}

				adminAddress := args[3]
				if adminAddress == "" {
					return nil, fmt.Errorf("admin address is empty")
				}

				cfgEtl := &proto.Config{
					Contract: &proto.ContractConfig{
						Symbol: ttSymbol,
						Options: &proto.ChaincodeOptions{
							DisableMultiSwaps: true,
						},
						RobotSKI: robotSKI,
						Admin:    &proto.Wallet{Address: adminAddress},
					},
					Token: &proto.TokenConfig{
						Name:     ttName,
						Decimals: ttDecimals,
						Issuer:   &proto.Wallet{Address: issuerAddress},
					},
				}

				return cfgEtl, nil
			}),
		)
		require.Empty(t, message)
	})

	var cfg proto.Config
	step(t, "Fetch config", false, func() {
		data := user1.Invoke("tt", "config")
		require.NotEmpty(t, data)

		err := json.Unmarshal([]byte(data), &cfg)
		require.NoError(t, err)
	})

	step(t, "Validate contract config", false, func() {
		require.Equal(t, ttSymbol, cfg.Contract.Symbol)
		require.Equal(t, fixtures_test.RobotHashedCert, cfg.Contract.RobotSKI)
		require.Equal(t, false, cfg.Contract.Options.DisableSwaps)
		require.Equal(t, true, cfg.Contract.Options.DisableMultiSwaps)
	})

	step(t, "Validate token config", false, func() {
		require.Equal(t, ttName, cfg.Token.Name)
		require.Equal(t, ttDecimals, cfg.Token.Decimals)
		require.Equal(t, issuer.Address(), cfg.Token.Issuer.Address)
	})
}

func TestWithConfigMapperFuncFromArgs(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()
	issuer := ledgerMock.NewWallet()

	ttSymbol := "tt"
	step(t, "Init new chaincode", false, func() {
		initArgs := []string{
			"",                            // PlatformSKI (backend) - deprecated
			fixtures_test.RobotHashedCert, // RobotSKI
			issuer.Address(),              // IssuerAddress
			fixtures_test.AdminAddr,       // AdminAddress
		}
		message := ledgerMock.NewCCArgsArr(ttSymbol, &TestConfigToken{}, initArgs, core.WithConfigMapperFunc(
			func(args []string) (*proto.Config, error) {
				return config.FromArgsWithIssuerAndAdmin(ttSymbol, args)
			}),
		)
		require.Empty(t, message)
	})

	var cfg proto.Config
	step(t, "Fetch config", false, func() {
		data := user1.Invoke("tt", "config")
		require.NotEmpty(t, data)

		err := json.Unmarshal([]byte(data), &cfg)
		require.NoError(t, err)
	})

	step(t, "Validate contract config", false, func() {
		require.Equal(t, strings.ToUpper(ttSymbol), cfg.Contract.Symbol)
		require.Equal(t, fixtures_test.RobotHashedCert, cfg.Contract.RobotSKI)
		require.Equal(t, false, cfg.Contract.Options.DisableSwaps)
		require.Equal(t, false, cfg.Contract.Options.DisableMultiSwaps)
	})
}

func TestBaseTokenTx(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()
	issuer := ledgerMock.NewWallet()

	ttName, ttSymbol, ttDecimals := "test token", "TT", uint32(8)

	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol: ttSymbol,
			Options: &proto.ChaincodeOptions{
				DisableMultiSwaps: true,
			},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: issuer.Address()},
		},
		Token: &proto.TokenConfig{
			Name:     ttName,
			Decimals: ttDecimals,
			Issuer:   &proto.Wallet{Address: issuer.Address()},
		},
	}
	config, _ := protojson.Marshal(cfgEtl)

	t.Run("Init new chaincode", func(t *testing.T) {
		initMsg := ledgerMock.NewCC(testTokenCCName, &TestConfigToken{}, string(config))
		require.Empty(t, initMsg)
	})

	const emitAmount = "42"

	t.Run("Tx emit", func(t *testing.T) {
		err := user1.RawSignedInvokeWithErrorReturned(testTokenCCName, "setEmitAmount", emitAmount)
		require.NoError(t, err)
	})

	t.Run("Query emit", func(t *testing.T) {
		data := user1.Invoke(testTokenCCName, "emitAmount")
		require.NotEmpty(t, data)

		var amount string
		err := json.Unmarshal([]byte(data), &amount)
		require.NoError(t, err)
		require.Equal(t, emitAmount, amount)
	})
}

func TestDisabledFunctions(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)
	user1 := ledgerMock.NewWallet()

	tt1 := disabledFnContract{}
	cfgEtl := &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol:   "TT1",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
	}
	config1, _ := protojson.Marshal(cfgEtl)
	step(t, "Init new tt1 chaincode", false, func() {
		message := ledgerMock.NewCC("tt1", &tt1, string(config1))
		require.Empty(t, message)
	})

	step(t, "Call TxTestFunction", false, func() {
		err := user1.RawSignedInvokeWithErrorReturned("tt1", "testFunction")
		require.NoError(t, err)
	})

	tt2 := disabledFnContract{}
	cfgEtl = &proto.Config{
		Contract: &proto.ContractConfig{
			Symbol: "TT2",
			Options: &proto.ChaincodeOptions{
				DisabledFunctions: []string{"TxTestFunction"},
			},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &proto.Wallet{Address: fixtures_test.AdminAddr},
		},
	}
	config2, _ := protojson.Marshal(cfgEtl)

	step(t, "Init new tt2 chaincode", false, func() {
		message := ledgerMock.NewCC("tt2", &tt2, string(config2))
		require.Empty(t, message, message)
	})

	step(t, "[negative] call TxTestFunction", false, func() {
		err := user1.RawSignedInvokeWithErrorReturned("tt2", "testFunction")
		require.EqualError(t, err, "invoke: finding method: method 'testFunction' not found")
	})
}

func TestInitWithEmptyConfig(t *testing.T) {
	t.Parallel()

	ledgerMock := mock.NewLedger(t)

	config := `{}`

	step(t, "Init new chaincode", false, func() {
		initMsg := ledgerMock.NewCC(testTokenCCName, &TestConfigToken{}, config)
		require.Contains(t, initMsg, "contract config is not set")
	})

	return
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	allowedSymbols := []string{`TT`, `TT-2`, `TT-2.0`, `TT-2.A`, `TT-23.AB`}
	for _, s := range allowedSymbols {
		cfg := &proto.Config{
			Contract: &proto.ContractConfig{
				Symbol:   s,
				RobotSKI: fixtures_test.RobotHashedCert,
			},
		}
		require.NoError(t, cfg.Validate(), s)
	}

	disallowedSymbols := []string{`TT_1`, `TT-2.4.6`, `TT-.1`, `TT-1.`, `TT-1..2`}
	for _, s := range disallowedSymbols {
		cfg := &proto.Config{
			Contract: &proto.ContractConfig{
				Symbol:   s,
				RobotSKI: fixtures_test.RobotHashedCert,
			},
		}
		require.Error(t, cfg.Validate(), s)
	}
}
