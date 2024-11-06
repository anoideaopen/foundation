package unit

import (
	"embed"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
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

	cc, err := core.NewCC(tt, core.WithSrcFS(&f))
	require.NoError(t, err)

	mockStub.GetChannelIDReturns(testTokenCCName)

	mockStub.GetFunctionAndParametersReturns("nameOfFiles", []string{})
	mockStub.GetStateReturns([]byte(config), nil)

	resp := cc.Invoke(mockStub)
	var files []string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &files))

	mockStub.GetFunctionAndParametersReturns("srcFile", []string{"version_test.go"})

	resp = cc.Invoke(mockStub)
	var file string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &file))
	require.Equal(t, "unit", file[8:12])
	l := len(file)
	l += 10
	lStr := strconv.Itoa(l)

	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "8", "12"})

	resp = cc.Invoke(mockStub)
	var partFile string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile)

	time.Sleep(10 * time.Second)

	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", "12"})

	resp = cc.Invoke(mockStub)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])

	time.Sleep(10 * time.Second)

	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", lStr})

	resp = cc.Invoke(mockStub)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])
}

func TestEmbedSrcFilesWithoutFS(t *testing.T) {
	const errMsg = "embed fs is nil"

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
	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	mockStub.GetChannelIDReturns(testTokenCCName)

	mockStub.GetStateReturns([]byte(config), nil)
	mockStub.GetFunctionAndParametersReturns("nameOfFiles", []string{})

	resp := cc.Invoke(mockStub)
	msg := resp.GetMessage()
	require.Equal(t, msg, errMsg)

	mockStub.GetFunctionAndParametersReturns("srcFile", []string{"embed_test.go"})

	resp = cc.Invoke(mockStub)
	msg = resp.GetMessage()
	require.Equal(t, msg, errMsg)

	mockStub.GetFunctionAndParametersReturns("srcPartFile", []string{"embed_test.go", "8", "13"})

	resp = cc.Invoke(mockStub)
	msg = resp.GetMessage()
	require.Equal(t, msg, errMsg)
}

func TestBuildInfo(t *testing.T) {
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
	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	mockStub.GetChannelIDReturns(testTokenCCName)

	mockStub.GetStateReturns([]byte(config), nil)
	mockStub.GetFunctionAndParametersReturns("buildInfo", []string{})

	resp := cc.Invoke(mockStub)
	biData := resp.GetPayload()
	require.NotEmpty(t, biData)

	var bi debug.BuildInfo
	err = json.Unmarshal([]byte(biData), &bi)
	require.NoError(t, err)
	require.NotNil(t, bi)
}

func TestSysEnv(t *testing.T) {
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

	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	mockStub.GetChannelIDReturns(testTokenCCName)

	mockStub.GetStateReturns([]byte(config), nil)
	mockStub.GetFunctionAndParametersReturns("systemEnv", []string{})

	resp := cc.Invoke(mockStub)
	sysEnv := resp.GetPayload()
	require.NotEmpty(t, sysEnv)

	systemEnv := make(map[string]string)
	err = json.Unmarshal(sysEnv, &systemEnv)
	require.NoError(t, err)
	_, ok := systemEnv["/etc/issue"]
	require.True(t, ok)
}

func TestCoreChaincodeIdName(t *testing.T) {
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
	cc, err := core.NewCC(tt)
	require.NoError(t, err)

	mockStub.GetChannelIDReturns(testTokenCCName)

	mockStub.GetStateReturns([]byte(config), nil)
	mockStub.GetFunctionAndParametersReturns("coreChaincodeIDName", []string{})

	resp := cc.Invoke(mockStub)
	ChNameData := resp.GetPayload()
	require.NotEmpty(t, ChNameData)

	var name string
	err = json.Unmarshal(ChNameData, &name)
	require.NoError(t, err)
	require.NotEmpty(t, name)
}
