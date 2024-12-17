package unit

import (
	"net/http"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestKeyTypesEmission(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name    string
		keyType pbfound.KeyType
	}{
		{
			name:    "ed25519 emission test",
			keyType: pbfound.KeyType_ed25519,
		},
		{
			name:    "secp256k1 emission test",
			keyType: pbfound.KeyType_secp256k1,
		},
		{
			name:    "gost emission test",
			keyType: pbfound.KeyType_gost,
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)
			cs := mockStub.GetStub()

			issuer, err := mocks.NewUserFoundation(test.keyType)
			require.NoError(t, err)

			user, err := mocks.NewUserFoundation(test.keyType)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(&FiatTestToken{})
			require.NoError(t, err)

			issuerAddress := sha3.Sum256(issuer.PublicKeyBytes)

			pending := &pbfound.PendingTx{
				Method: "emit",
				Sender: &pbfound.Address{
					UserID:       issuer.UserID,
					Address:      issuerAddress[:],
					IsIndustrial: false,
					IsMultisig:   false,
				},
				Args:  []string{user.AddressBase58Check, "1000"},
				Nonce: uint64(time.Now().UnixNano() / 1000000),
			}
			pendingMarshalled, err := pb.Marshal(pending)
			require.NoError(t, err)

			mocks.ACLGetAccountInfo(t, cs, 0)
			cs.GetStateReturnsOnCall(0, []byte(config), nil)
			cs.GetStateReturnsOnCall(1, pendingMarshalled, nil)
			cs.GetStateReturnsOnCall(2, big.NewInt(1000).Bytes(), nil)

			dataIn, err := pb.Marshal(&pbfound.Batch{TxIDs: [][]byte{[]byte("testTxID")}})
			require.NoError(t, err)

			err = mocks.SetCreator(cs, BatchRobotCert)
			require.NoError(t, err)

			cs.GetFunctionAndParametersReturns("batchExecute", []string{string(dataIn)})

			// invoking chaincode
			resp := cc.Invoke(cs)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())

			// checking put state
			require.Equal(t, 3, cs.PutStateCallCount())
			var i int
			for i = 0; i < cs.PutStateCallCount(); i++ {
				key, value := cs.PutStateArgsForCall(i)
				if key != "tokenMetadata" {
					prefix, keys, err := cs.SplitCompositeKey(key)
					require.NoError(t, err)

					if prefix == balance.BalanceTypeToken.String() {
						require.Equal(t, keys[0], user.AddressBase58Check)
						require.Equal(t, big.NewInt(1000).Bytes(), value)
					}

					if prefix == "0" {
						require.Equal(t, keys[1], issuer.AddressBase58Check)
						require.Equal(t, big.NewInt(1000).Bytes(), value)
					}
				}
			}
		})
	}
}
