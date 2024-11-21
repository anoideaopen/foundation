package unit

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mocks"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type ConfigData struct {
	*pb.Config
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

var (
	_                config.TokenConfigurator = &TestConfigToken{}
	testFunctionName                          = "testFunction"
)

func (tct *TestConfigToken) QueryConfig() (ConfigData, error) {
	return ConfigData{
		&pb.Config{
			Contract: tct.ContractConfig(),
			Token:    tct.TokenConfig(),
		},
	}, nil
}

const configKey = "__config"

// TestInitWithCommonConfig tests chaincode initialization of token with common config.
func TestInitWithCommonConfig(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	mockStub := mocks.NewMockStub(t)

	ttName, ttSymbol, ttDecimals := "test token", "TT", uint32(8)

	cfgEtl := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol: ttSymbol,
			Options: &pb.ChaincodeOptions{
				DisableMultiSwaps: true,
			},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: issuer.AddressBase58Check},
		},
		Token: &pb.TokenConfig{
			Name:     ttName,
			Decimals: ttDecimals,
			Issuer:   &pb.Wallet{Address: issuer.AddressBase58Check},
		},
	}
	cfg, _ := protojson.Marshal(cfgEtl)
	var (
		cc *core.Chaincode
	)

	// Initializing new chaincode
	tct := &TestConfigToken{}
	cc, err = core.NewCC(tct)
	require.NoError(t, err)

	mockStub.GetStringArgsReturns([]string{string(cfg)})
	resp := cc.Init(mockStub)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := mockStub.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)
	require.NoError(t, err)

	// Validating contract config
	require.True(t, proto.Equal(&resultCfg, cfgEtl))

	// Requesting config from state
	mockStub.GetFunctionAndParametersReturns("config", []string{})
	cc.Invoke(mockStub)

	key = mockStub.GetStateArgsForCall(0)
	require.Equal(t, key, configKey)
}

func TestWithConfigMapperFunc(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	ttName, ttSymbol, ttDecimals := "test token", "TT", uint32(8)

	// Initializing new chaincode
	initArgs := []string{
		"",                            // PlatformSKI (backend) - deprecated
		fixtures_test.RobotHashedCert, // RobotSKI
		issuer.AddressBase58Check,     // IssuerAddress
		fixtures_test.AdminAddr,       // AdminAddress
	}
	tct := &TestConfigToken{}

	cc, err := core.NewCC(tct, core.WithConfigMapperFunc(
		func(args []string) (*pb.Config, error) {
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

			cfgEtl := &pb.Config{
				Contract: &pb.ContractConfig{
					Symbol: ttSymbol,
					Options: &pb.ChaincodeOptions{
						DisableMultiSwaps: true,
					},
					RobotSKI: robotSKI,
					Admin:    &pb.Wallet{Address: adminAddress},
				},
				Token: &pb.TokenConfig{
					Name:     ttName,
					Decimals: ttDecimals,
					Issuer:   &pb.Wallet{Address: issuerAddress},
				},
			}

			return cfgEtl, nil
		}),
	)
	require.NoError(t, err)

	mockStub.GetStringArgsReturns(initArgs)
	resp := cc.Init(mockStub)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := mockStub.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)
	require.NoError(t, err)

	// Validating contract config
	require.Equal(t, ttSymbol, resultCfg.Contract.Symbol)
	require.Equal(t, fixtures_test.RobotHashedCert, resultCfg.Contract.RobotSKI)
	require.Equal(t, false, resultCfg.Contract.Options.DisableSwaps)
	require.Equal(t, true, resultCfg.Contract.Options.DisableMultiSwaps)

	// Validating token config
	require.Equal(t, ttName, resultCfg.Token.Name)
	require.Equal(t, ttDecimals, resultCfg.Token.Decimals)
	require.Equal(t, issuer.AddressBase58Check, resultCfg.Token.Issuer.Address)
}

func TestWithConfigMapperFuncFromArgs(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)
	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	ttSymbol := "tt"

	// Initializing new chaincode
	initArgs := []string{
		"",                            // PlatformSKI (backend) - deprecated
		fixtures_test.RobotHashedCert, // RobotSKI
		issuer.AddressBase58Check,     // IssuerAddress
		fixtures_test.AdminAddr,       // AdminAddress
	}
	tct := &TestConfigToken{}

	cc, err := core.NewCC(tct, core.WithConfigMapperFunc(
		func(args []string) (*pb.Config, error) {
			return config.FromArgsWithIssuerAndAdmin(ttSymbol, args)
		}))
	require.NoError(t, err)

	mockStub.GetStringArgsReturns(initArgs)
	resp := cc.Init(mockStub)
	require.Empty(t, resp.GetMessage())

	//Checking config was set to state
	var resultCfg pb.Config
	key, value := mockStub.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)

	// Validating config
	require.Equal(t, strings.ToUpper(ttSymbol), resultCfg.Contract.Symbol)
	require.Equal(t, fixtures_test.RobotHashedCert, resultCfg.Contract.RobotSKI)
	require.Equal(t, fixtures_test.AdminAddr, resultCfg.Contract.Admin.Address)
	require.Equal(t, issuer.AddressBase58Check, resultCfg.Token.Issuer.Address)
}

func TestDisabledFunctions(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)

	user1, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	tt := &disabledFnContract{}
	cfgEtl := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   "TT1",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: fixtures_test.AdminAddr},
		},
	}

	config1, err := protojson.Marshal(cfgEtl)
	require.NoError(t, err)

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	//Calling TxTestFunction while it's not disabled
	ctorArgs := prepareArgsWithSign(t, user1, testFunctionName, "", "")
	mockStub.GetStateReturns(config1, nil)
	mockStub.GetFunctionAndParametersReturns(testFunctionName, ctorArgs)

	resp := cc.Invoke(mockStub)
	require.Empty(t, resp.GetMessage())

	cfgEtl = &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol: "TT2",
			Options: &pb.ChaincodeOptions{
				DisabledFunctions: []string{"TxTestFunction"},
			},
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: fixtures_test.AdminAddr},
		},
	}
	config2, _ := protojson.Marshal(cfgEtl)

	//Calling TxTestFunction while it's disabled
	ctorArgs = prepareArgsWithSign(t, user1, testFunctionName, "", "")
	mockStub.GetStateReturns(config2, nil)
	mockStub.GetFunctionAndParametersReturns(testFunctionName, ctorArgs)

	resp = cc.Invoke(mockStub)
	require.Equal(t, "invoke: finding method: method 'testFunction' not found", resp.GetMessage())
}

func TestInitWithEmptyConfig(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)

	cfg := `{}`

	// Init new chaincode
	cc, err := core.NewCC(&TestConfigToken{})
	require.NoError(t, err)

	mockStub.GetStringArgsReturns([]string{cfg})
	resp := cc.Init(mockStub)
	require.Contains(t, resp.GetMessage(), "contract config is not set")
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	allowedSymbols := []string{`TT`, `TT2`, `TT-2`, `TT-2.0`, `TT-2.A`, `TT-23.AB`, `TT_2.0`}
	for _, s := range allowedSymbols {
		cfg := &pb.Config{
			Contract: &pb.ContractConfig{
				Symbol:   s,
				RobotSKI: fixtures_test.RobotHashedCert,
			},
		}
		require.NoError(t, cfg.Validate(), s)
	}

	disallowedSymbols := []string{`2T`, `TT+1`, `TT-2.4.6`, `TT-.1`, `TT-1.`, `TT-1..2`}
	for _, s := range disallowedSymbols {
		cfg := &pb.Config{
			Contract: &pb.ContractConfig{
				Symbol:   s,
				RobotSKI: fixtures_test.RobotHashedCert,
			},
		}
		require.Error(t, cfg.Validate(), s)
	}
}

func prepareArgsWithSign(
	t *testing.T,
	user *mocks.UserFoundation,
	functionName,
	channelName,
	chaincodeName string,
	args ...string,
) []string {
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	ctorArgs := append(append([]string{functionName, channelName, chaincodeName}, args...), nonce)

	//ctorArgs := append([]string{functionName, channelName, chaincodeName}, nonce)
	pubKey, sMsg, err := user.Sign(ctorArgs...)
	require.NoError(t, err)

	return append(ctorArgs, pubKey, base58.Encode(sMsg))
}
