package token

import (
	"encoding/hex"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestSetRate(t *testing.T) {
	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description         string
		functionName        string
		errorMsg            string
		signUser            *mocks.UserFoundation
		codeResp            int32
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
	}{
		{
			description:  "setRate - positive test with valid parameters",
			functionName: "setRate",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "1"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k != metadataKey {
						continue
					}
					etl := &pbfound.Token{}
					err = pb.Unmarshal(v, etl)
					require.NoError(t, err)
					require.True(t, pb.Equal(etl, &pbfound.Token{
						Rates: []*pbfound.TokenRate{
							{
								DealType: "distribute",
								Rate:     new(big.Int).SetUint64(1).Bytes(),
							},
						},
					}))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "setRate - negative test with invalid issuer",
			functionName: "setRate",
			errorMsg:     ErrUnauthorized,
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "1"}
			},
		},
		{
			description:  "setRate - negative test with invalid rate parameter set to zero",
			functionName: "setRate",
			errorMsg:     "trying to set rate = 0",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "0"}
			},
		},
		{
			description:  "setRate - negative test with invalid rate parameter set to string",
			functionName: "setRate",
			errorMsg:     "invalid argument value: 'wonder': for type '*big.Int'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "wonder"}
			},
		},
		{
			description:  "setRate - negative test with invalid rate parameter set to minus value",
			functionName: "setRate",
			errorMsg:     "validation failed: 'negative number'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "-3"}
			},
		},
		{
			description:  "setRate - negative test with invalid currency parameter set to equals token",
			functionName: "setRate",
			errorMsg:     "currency is equals token: it is impossible",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "TT", "3"}
			},
		},
		{
			description:  "setRate - negative test with incorrect number of parameters",
			functionName: "setRate",
			errorMsg:     "incorrect number of keys or signs",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "", ""}
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				"Test Token",
				"TT",
				8,
				issuer.AddressBase58Check,
				feeSetter.AddressBase58Check,
				feeAddressSetter.AddressBase58Check,
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&BaseToken{})
			require.NoError(t, err)

			parameters := testCase.funcPrepareMockStub(t, mockStub)

			var (
				txId string
				resp peer.Response
			)

			txId, resp = mockStub.TxInvokeChaincodeSigned(cc, testCase.functionName, testCase.signUser, "", "", "", parameters...)

			// check result
			if testCase.codeResp == int32(shim.ERROR) {
				require.Equal(t, resp.GetStatus(), testCase.codeResp)
				require.Contains(t, resp.GetMessage(), testCase.errorMsg)
				require.Empty(t, resp.GetPayload())
				return
			}

			require.Equal(t, resp.GetStatus(), int32(shim.OK))
			require.Empty(t, resp.GetMessage())

			bResp := &pbfound.BatchResponse{}
			err = pb.Unmarshal(resp.GetPayload(), bResp)
			require.NoError(t, err)

			var respb *pbfound.TxResponse
			for _, r := range bResp.GetTxResponses() {
				if hex.EncodeToString(r.GetId()) == txId {
					respb = r
					break
				}
			}

			if len(testCase.errorMsg) != 0 {
				require.Contains(t, respb.GetError().GetError(), testCase.errorMsg)
				return
			}

			if testCase.funcCheckResponse != nil {
				testCase.funcCheckResponse(t, mockStub, respb)
			}
		})
	}
}
