package unit

import (
	"embed"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

const issuerAddress = "SkXcT15CDtiEFWSWcT3G8GnWfG2kAJw9yW28tmPEeatZUvRct"

//go:embed *.go
var f embed.FS

func TestEmbedSrcFiles(t *testing.T) {
	t.Parallel()

	mockStub := mockstub.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetChannelIDReturns(testTokenCCName)

	cs.GetFunctionAndParametersReturns("nameOfFiles", []string{})
	cs.GetStateReturns([]byte(config), nil)

	resp := cc.Invoke(cs)
	var files []string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &files))

	cs.GetFunctionAndParametersReturns("srcFile", []string{"version_test.go"})

	resp = cc.Invoke(cs)
	var file string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &file))
	require.Equal(t, "unit", file[8:12])
	l := len(file)
	l += 10
	lStr := strconv.Itoa(l)

	cs.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "8", "12"})

	resp = cc.Invoke(cs)
	var partFile string
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile)

	time.Sleep(10 * time.Second)

	cs.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", "12"})

	resp = cc.Invoke(cs)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])

	time.Sleep(10 * time.Second)

	cs.GetFunctionAndParametersReturns("srcPartFile", []string{"version_test.go", "-1", lStr})

	resp = cc.Invoke(cs)
	require.NoError(t, json.Unmarshal(resp.GetPayload(), &partFile))
	require.Equal(t, "unit", partFile[8:12])
}

func TestEmbedSrcFilesWithoutFS(t *testing.T) {
	const errMsg = "embed fs is nil"

	t.Parallel()

	mockStub := mockstub.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetChannelIDReturns(testTokenCCName)

	cs.GetStateReturns([]byte(config), nil)
	cs.GetFunctionAndParametersReturns("nameOfFiles", []string{})

	resp := cc.Invoke(cs)
	msg := resp.GetMessage()
	require.Equal(t, msg, errMsg)

	cs.GetFunctionAndParametersReturns("srcFile", []string{"embed_test.go"})

	resp = cc.Invoke(cs)
	msg = resp.GetMessage()
	require.Equal(t, msg, errMsg)

	cs.GetFunctionAndParametersReturns("srcPartFile", []string{"embed_test.go", "8", "13"})

	resp = cc.Invoke(cs)
	msg = resp.GetMessage()
	require.Equal(t, msg, errMsg)
}

func TestBuildInfo(t *testing.T) {
	t.Parallel()

	mockStub := mockstub.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetChannelIDReturns(testTokenCCName)

	cs.GetStateReturns([]byte(config), nil)
	cs.GetFunctionAndParametersReturns("buildInfo", []string{})

	resp := cc.Invoke(cs)
	biData := resp.GetPayload()
	require.NotEmpty(t, biData)

	var bi debug.BuildInfo
	err = json.Unmarshal(biData, &bi)
	require.NoError(t, err)
	require.NotNil(t, bi)
}

func TestSysEnv(t *testing.T) {
	t.Parallel()

	mockStub := mockstub.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetChannelIDReturns(testTokenCCName)

	cs.GetStateReturns([]byte(config), nil)
	cs.GetFunctionAndParametersReturns("systemEnv", []string{})

	resp := cc.Invoke(cs)
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

	mockStub := mockstub.NewMockStub(t)
	cs := mockStub.GetStub()

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

	cs.GetChannelIDReturns(testTokenCCName)

	cs.GetStateReturns([]byte(config), nil)
	cs.GetFunctionAndParametersReturns("coreChaincodeIDName", []string{})

	resp := cc.Invoke(cs)
	ChNameData := resp.GetPayload()
	require.NotEmpty(t, ChNameData)

	var name string
	err = json.Unmarshal(ChNameData, &name)
	require.NoError(t, err)
	require.NotEmpty(t, name)
}
