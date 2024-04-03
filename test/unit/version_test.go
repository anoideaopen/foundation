package unit

import (
	"embed"
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"

	"github.com/anoideaopen/foundation/core"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
)

//go:embed *.go
var f embed.FS

func TestEmbedSrcFiles(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	tt := &token.BaseToken{
		Name:     "Test Token",
		Symbol:   "TT",
		Decimals: 8,
	}

	mock.NewChainCode("tt", tt, &core.ContractOptions{}, &f, issuer.Address())

	rawFiles := issuer.Invoke("tt", "nameOfFiles")
	var files []string
	assert.NoError(t, json.Unmarshal([]byte(rawFiles), &files))

	rawFile := issuer.Invoke("tt", "srcFile", "version_test.go")
	var file string
	assert.NoError(t, json.Unmarshal([]byte(rawFile), &file))
	assert.Equal(t, "unit", file[8:12])
	l := len(file)
	l += 10
	lStr := strconv.Itoa(l)

	rawPartFile := issuer.Invoke("tt", "srcPartFile", "version_test.go", "8", "12")
	var partFile string
	assert.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	assert.Equal(t, "unit", partFile)

	rawPartFile = issuer.Invoke("tt", "srcPartFile", "version_test.go", "-1", "12")
	assert.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	assert.Equal(t, "unit", partFile[8:12])

	rawPartFile = issuer.Invoke("tt", "srcPartFile", "version_test.go", "-1", lStr)
	assert.NoError(t, json.Unmarshal([]byte(rawPartFile), &partFile))
	assert.Equal(t, "unit", partFile[8:12])
}

func TestEmbedSrcFilesWithoutFS(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	tt := &token.BaseToken{
		Name:     "Test Token",
		Symbol:   "TT",
		Decimals: 8,
	}

	mock.NewChainCode("tt", tt, &core.ContractOptions{}, nil, issuer.Address())

	err := issuer.InvokeWithError("tt", "nameOfFiles")
	assert.Error(t, err)

	err = issuer.InvokeWithError("tt", "srcFile", "embed_test.go")
	assert.Error(t, err)

	err = issuer.InvokeWithError("tt", "srcPartFile", "embed_test.go", "8", "13")
	assert.Error(t, err)
}

func TestBuildInfo(t *testing.T) {
	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()
	lm.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, issuer.Address())

	biData := issuer.Invoke(testTokenCCName, "buildInfo")
	assert.NotEmpty(t, biData)

	var bi debug.BuildInfo
	err := json.Unmarshal([]byte(biData), &bi)
	assert.NoError(t, err)
	assert.NotNil(t, bi)
}

func TestSysEnv(t *testing.T) {
	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()
	lm.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, issuer.Address())

	sysEnv := issuer.Invoke(testTokenCCName, "systemEnv")
	assert.NotEmpty(t, sysEnv)

	systemEnv := make(map[string]string)
	err := json.Unmarshal([]byte(sysEnv), &systemEnv)
	assert.NoError(t, err)
	_, ok := systemEnv["/etc/issue"]
	assert.True(t, ok)
}

func TestCoreChaincodeIdName(t *testing.T) {
	tt := &TestToken{
		token.BaseToken{
			Name:     testTokenName,
			Symbol:   testTokenSymbol,
			Decimals: 8,
		},
	}

	lm := ma.NewLedger(t)
	issuer := lm.NewWallet()
	lm.NewChainCode(testTokenCCName, tt, &core.ContractOptions{}, nil, issuer.Address())

	ChNameData := issuer.Invoke(testTokenCCName, "coreChaincodeIDName")
	assert.NotEmpty(t, ChNameData)

	var name string
	err := json.Unmarshal([]byte(ChNameData), &name)
	assert.NoError(t, err)
	assert.NotEmpty(t, name)
}
