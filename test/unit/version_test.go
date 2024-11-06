package unit

import (
	"embed"
	"encoding/hex"
	"encoding/json"
	"github.com/anoideaopen/foundation/core"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"runtime/debug"
	"strconv"
	"testing"
	"time"
)

const issuerAddress = "SkXcT15CDtiEFWSWcT3G8GnWfG2kAJw9yW28tmPEeatZUvRct"

//go:embed *.go
var f embed.FS

func TestEmbedSrcFiles(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(
		testTokenName,
		testTokenSymbol,
		8,
		issuerAddress,
		"",
		"",
		"",
		nil,
	)
	cc, err := mocks.NewCC(mockStub, tt, config, core.WithSrcFS(&f))
	require.NoError(t, err)
	mockStub.GetChannelIDReturns("tt")

	txID := [16]byte(uuid.New())
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("nameOfFiles", []string{})
	mockStub.GetStateReturns([]byte(config), nil)

	resp := cc.Invoke(mockStub)
	var files []string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &files))

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcFile", []string{"version_test.go"})

	resp = cc.Invoke(mockStub)
	var file string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &file))
	require.Equal(t, "unit", file[8:12])
	l := len(file)
	l += 10
	lStr := strconv.Itoa(l)

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "8", "12"})

	resp = cc.Invoke(mockStub)
	var partFile string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile)

	time.Sleep(10 * time.Second)

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", "12"})

	resp = cc.Invoke(mockStub)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])

	time.Sleep(10 * time.Second)

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", lStr})

	resp = cc.Invoke(mockStub)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])
}

func TestEmbedSrcFilesWithoutFS(t *testing.T) {
	t.Parallel()

	mockStub := mocks.NewMockStub(t)

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(
		testTokenName,
		testTokenSymbol,
		8,
		issuerAddress,
		"",
		"",
		"",
		nil,
	)
	cc, err := mocks.NewCC(mockStub, tt, config, core.WithSrcFS(&f))
	require.NoError(t, err)
	mockStub.GetChannelIDReturns("tt")

	txID := [16]byte(uuid.New())
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("nameOfFiles", []string{})

	//err := issuer.InvokeWithError("tt", "nameOfFiles")
	resp := cc.Invoke(mockStub)
	msg := resp.GetMessage()
	require.Equal(t, msg, "invoke: loading raw config: config bytes is empty")
	//require.Error(t, err)

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcFile", []string{"embed_test.go"})

	//err = issuer.InvokeWithError("tt", "srcFile", "embed_test.go")
	resp = cc.Invoke(mockStub)
	msg = resp.GetMessage()
	require.Equal(t, msg, "invoke: loading raw config: config bytes is empty")

	//require.Error(t, err)

	txID = uuid.New()
	mockStub.GetTxIDReturns(hex.EncodeToString(txID[:]))
	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"embed_test.go", "8", "13"})

	//err = issuer.InvokeWithError("tt", "srcPartFile", "embed_test.go", "8", "13")
	resp = cc.Invoke(mockStub)
	msg = resp.GetMessage()
	require.Equal(t, msg, "invoke: loading raw config: config bytes is empty")

	//require.Error(t, err)
}

func TestBuildInfo(t *testing.T) {
	t.Parallel()

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := lm.NewCC("tt", tt, config)
	require.Empty(t, initMsg)

	biData := issuer.Invoke(testTokenCCName, "buildInfo")
	require.NotEmpty(t, biData)

	var bi debug.BuildInfo
	err := json.Unmarshal([]byte(biData), &bi)
	require.NoError(t, err)
	require.NotNil(t, bi)
}

func TestSysEnv(t *testing.T) {
	t.Parallel()

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := lm.NewCC("tt", tt, config)
	require.Empty(t, initMsg)

	sysEnv := issuer.Invoke(testTokenCCName, "systemEnv")
	require.NotEmpty(t, sysEnv)

	systemEnv := make(map[string]string)
	err := json.Unmarshal([]byte(sysEnv), &systemEnv)
	require.NoError(t, err)
	_, ok := systemEnv["/etc/issue"]
	require.True(t, ok)
}

func TestCoreChaincodeIdName(t *testing.T) {
	t.Parallel()

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := lm.NewCC("tt", tt, config)
	require.Empty(t, initMsg)

	ChNameData := issuer.Invoke(testTokenCCName, "coreChaincodeIDName")
	require.NotEmpty(t, ChNameData)

	var name string
	err := json.Unmarshal([]byte(ChNameData), &name)
	require.NoError(t, err)
	require.NotEmpty(t, name)
}
