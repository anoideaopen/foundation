package unit

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestMetadataMethods(t *testing.T) {
	t.Parallel()

	mockStub := mockstub.NewMockStub(t)

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.Nil(t, err)

	tt := &token.BaseToken{}
	config := mockStub.CreateAndSetConfig("Test Token", "TT", 8,
		issuer.AddressBase58Check, "", "", "", nil)

	cc, err := core.NewCC(tt)
	require.Nil(t, err)

	mockStub.GetStringArgsReturns([]string{config})
	cc.Init(mockStub)

	resp := mockStub.QueryChaincode(cc, "metadata", []string{}...)
	require.Empty(t, resp.GetMessage())

	var meta token.Metadata
	err = json.Unmarshal(resp.GetPayload(), &meta)
	require.NoError(t, err)
}
