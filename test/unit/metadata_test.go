package unit

import (
	"encoding/json"
	"testing"

	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestMetadataMethods(t *testing.T) {
	t.Parallel()

	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()

	tt := &token.BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("tt", tt, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	rsp := user1.Invoke("tt", "metadata")

	var meta token.Metadata
	err := json.Unmarshal([]byte(rsp), &meta)
	require.NoError(t, err)

	// require.ElementsMatch(t, tokenMethods, meta.Methods)
}
