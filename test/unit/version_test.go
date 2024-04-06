package unit

import (
	"embed"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

//go:embed *.go
var f embed.FS

func TestEmbedSrcFiles(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("tt", tt, config, core.WithSrcFS(&f))
	require.Empty(t, initMsg)

	rawFiles := issuer.Invoke("tt", "nameOfFiles")
	var files []string
	require.NoError(t, json.Unmarshal([]byte(rawFiles), &files))

	rawFile := issuer.Invoke("tt", "srcFile", "version_test.go")
	var file string
	require.NoError(t, json.Unmarshal([]byte(rawFile), &file))
	require.Equal(t, "unit", file[8:12])
	l := len(file)
	l += 10
	lStr := strconv.Itoa(l)

	rawPartFile := issuer.Invoke("tt", "srcPartFile", "version_test.go", "8", "12")
	var partFile string
	require.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	require.Equal(t, "unit", partFile)

	time.Sleep(10 * time.Second)

	rawPartFile = issuer.Invoke("tt", "srcPartFile", "version_test.go", "-1", "12")
	require.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	require.Equal(t, "unit", partFile[8:12])

	time.Sleep(10 * time.Second)

	rawPartFile = issuer.Invoke("tt", "srcPartFile", "version_test.go", "-1", lStr)
	require.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	require.Equal(t, "unit", partFile[8:12])
}

func TestEmbedSrcFilesWithoutFS(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)
	ledger.NewCC("tt", tt, config)

	err := issuer.InvokeWithError("tt", "nameOfFiles")
	require.Error(t, err)

	err = issuer.InvokeWithError("tt", "srcFile", "embed_test.go")
	require.Error(t, err)

	err = issuer.InvokeWithError("tt", "srcPartFile", "embed_test.go", "8", "13")
	require.Error(t, err)
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
