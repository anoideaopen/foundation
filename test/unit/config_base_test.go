package unit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/mocks"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
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

// TestInitWithPositionedArgs tests chaincode initialization of token with positioned arguments. Deprecated functionality
func TestInitWithPositionedArgs(t *testing.T) {
	t.Parallel()

	robotSKI := fixtures_test.RobotHashedCert

	testsCollection := []struct {
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

	for _, test := range testsCollection {
		t.Run(test.channel, func(t *testing.T) {
			mockStub := mocks.NewMockStub(t)
			cs := mockStub.GetStub()

			cc, err := core.NewCC(test.bci)
			require.NoError(t, err)

			cs.GetChannelIDReturns(test.channel)
			cs.GetStringArgsReturns(test.args)
			resp := cc.Init(cs)
			message := resp.GetMessage()
			if message != "" {
				require.Contains(t, message, test.initMsg)
				return
			} else {
				require.Empty(t, message)
			}

			// Checking config was set to state
			key, value := cs.PutStateArgsForCall(0)
			require.Equal(t, key, configKey)

			cfg, err := config.FromBytes(value)
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

// TestInitWithCommonConfig tests chaincode initialization of token with common config.
func TestInitWithCommonConfig(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetStringArgsReturns([]string{string(cfg)})
	resp := cc.Init(cs)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := cs.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)
	require.NoError(t, err)

	// Validating contract config
	require.True(t, proto.Equal(&resultCfg, cfgEtl))

	// Requesting config from state
	cs.GetFunctionAndParametersReturns("config", []string{})
	cc.Invoke(cs)

	key = cs.GetStateArgsForCall(0)
	require.Equal(t, key, configKey)
}

func TestWithConfigMapperFunc(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	// Initializing new chaincode
	initArgs := []string{
		"test token",                  // Chaincode Name
		"TT",                          // Token Symbol
		"8",                           // Decimals
		"",                            // PlatformSKI (backend) - deprecated
		fixtures_test.RobotHashedCert, // RobotSKI
		issuer.AddressBase58Check,     // IssuerAddress
		fixtures_test.AdminAddr,       // AdminAddress
	}
	tct := &TestConfigToken{}

	expectedConfig, err := getExpectedConfigFromArgs(initArgs)
	require.NoError(t, err)

	cc, err := core.NewCC(tct, core.WithConfigMapperFunc(getExpectedConfigFromArgs))
	require.NoError(t, err)

	cs.GetStringArgsReturns(initArgs)
	resp := cc.Init(cs)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := cs.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)
	require.NoError(t, err)

	// Validating contract config
	require.True(t, proto.Equal(&resultCfg, expectedConfig))
}

func TestWithConfigMapperFuncFromArgs(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	// Initializing new chaincode
	initArgs := []string{
		"",                            // Chaincode Name
		"tt",                          // Token Symbol
		"",                            // Decimals
		"",                            // PlatformSKI (backend) - deprecated
		fixtures_test.RobotHashedCert, // RobotSKI
		issuer.AddressBase58Check,     // IssuerAddress
		fixtures_test.AdminAddr,       // AdminAddress
	}
	tct := &TestConfigToken{}

	expectedConfig, err := getExpectedConfigFromArgs(initArgs)
	require.NoError(t, err)

	cc, err := core.NewCC(tct, core.WithConfigMapperFunc(
		func(args []string) (*pb.Config, error) {
			return config.FromArgsWithIssuerAndAdmin(args[1], args[3:])
		}))
	require.NoError(t, err)

	cs.GetStringArgsReturns(initArgs)
	resp := cc.Init(cs)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := cs.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)

	// Validating config
	require.True(t, proto.Equal(&resultCfg, expectedConfig))
}

func TestDisabledFunctions(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

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

	// Calling TxTestFunction while it's not disabled
	cs.GetStateReturns(config1, nil)

	err = mocks.SetFunctionAndParametersWithSign(cs, user1, testFunctionName, "", "", "")
	require.NoError(t, err)

	resp := cc.Invoke(cs)
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
	cs.GetStateReturns(config2, nil)
	err = mocks.SetFunctionAndParametersWithSign(cs, user1, testFunctionName, "", "", "")
	require.NoError(t, err)

	resp = cc.Invoke(cs)
	require.Equal(t, "invoke: finding method: method 'testFunction' not found", resp.GetMessage())
}

func TestInitWithEmptyConfig(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

	cfg := `{}`

	// Init new chaincode
	cc, err := core.NewCC(&TestConfigToken{})
	require.NoError(t, err)

	cs.GetStringArgsReturns([]string{cfg})
	resp := cc.Init(cs)
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

func getExpectedConfigFromArgs(args []string) (*pb.Config, error) {
	const requiredArgsCount = 7

	if len(args) != requiredArgsCount {
		return nil, fmt.Errorf(
			"required args length is '%d', got %d",
			requiredArgsCount,
			len(args),
		)
	}

	var (
		ttDecimals uint64
		err        error
	)

	ttName := args[0]
	ttSymbol := strings.ToUpper(args[1])
	if args[2] == "" {
		ttDecimals = 0
	} else {
		ttDecimals, err = strconv.ParseUint(args[2], 10, 32)
		if err != nil {
			return nil, err
		}
	}

	if ttName == "" && ttSymbol != "" {
		ttName = ttSymbol
	}

	_ = args[3] // PlatformSKI (backend) - deprecated

	robotSKI := args[4]
	if robotSKI == "" {
		return nil, fmt.Errorf("robot ski is empty")
	}

	issuerAddress := args[5]
	if issuerAddress == "" {
		return nil, fmt.Errorf("issuer address is empty")
	}

	adminAddress := args[6]
	if adminAddress == "" {
		return nil, fmt.Errorf("admin address is empty")
	}

	cfgEtl := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   ttSymbol,
			RobotSKI: robotSKI,
			Admin:    &pb.Wallet{Address: adminAddress},
		},
		Token: &pb.TokenConfig{
			Name:     ttName,
			Decimals: uint32(ttDecimals),
			Issuer:   &pb.Wallet{Address: issuerAddress},
		},
	}

	return cfgEtl, nil
}

// Extended config

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
		cfgFull pb.Config
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
		cfgFull   pb.Config
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

	mockStub := mocks.NewMockStub(t)
	cs := mockStub.GetStub()

	issuer, err := mocks.NewUserFoundation(pb.KeyType_ed25519)
	require.NoError(t, err)

	asset, amount := "SOME_ASSET", "42"

	extCfgEtl := &ExtConfig{
		Asset:  asset,
		Amount: amount,
		Issuer: &pb.Wallet{Address: issuer.AddressBase58Check},
	}
	cfgEtl := &pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   "EXTCC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: issuer.AddressBase58Check},
		},
	}
	cfgEtl.ExtConfig, err = anypb.New(extCfgEtl)
	require.NoError(t, err)
	cfg, err := protojson.Marshal(cfgEtl)
	require.NoError(t, err)

	cc, err := core.NewCC(&TestExtConfigToken{})
	require.NoError(t, err)

	// Init new chaincode
	cs.GetStringArgsReturns([]string{string(cfg)})
	resp := cc.Init(cs)
	require.Empty(t, resp.GetMessage())

	// Checking config was set to state
	var resultCfg pb.Config
	key, value := cs.PutStateArgsForCall(0)
	require.Equal(t, key, configKey)

	err = protojson.Unmarshal(value, &resultCfg)
	require.NoError(t, err)

	// Validating contract config
	require.True(t, proto.Equal(&resultCfg, cfgEtl))

	// Read and validate ExtConfig data
	cs.GetStateReturns(cfg, nil)
	cs.GetFunctionAndParametersReturns("extConfig", []string{})

	resp = cc.Invoke(cs)
	require.NotEmpty(t, resp.GetPayload())

	var m ExtConfig
	err = json.Unmarshal(resp.GetPayload(), &m)
	require.NoError(t, err)

	require.True(t, proto.Equal(&m, extCfgEtl))
}
