package token

import (
	"encoding/hex"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestSetLimits(t *testing.T) {
	cfg := &pbfound.Token{
		Rates: []*pbfound.TokenRate{
			{
				DealType: "distribute",
				Rate:     new(big.Int).SetUint64(1).Bytes(),
			},
		},
	}

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
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
			description:  "setLimits - positive test with maxLimit set to a valid value",
			functionName: "setLimits",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "1", "10"}
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
								Min:      new(big.Int).SetUint64(1).Bytes(),
								Max:      new(big.Int).SetUint64(10).Bytes(),
							},
						},
					}))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "setLimits - positive test with maxLimit set to a valid unlimited value",
			functionName: "setLimits",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "1", "0"}
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
								Min:      new(big.Int).SetUint64(1).Bytes(),
							},
						},
					}))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "setLimits - positive test with min limit parameter set to zero",
			functionName: "setLimits",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "0", "3"}
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
								Max:      new(big.Int).SetUint64(3).Bytes(),
							},
						},
					}))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "setLimits - negative test when min limit is greater than max limit",
			functionName: "setLimits",
			errorMsg:     "min limit is greater than max limit",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "10", "3"}
			},
		},
		{
			description:  "setLimits - negative test with invalid min limit parameter set to minus value",
			functionName: "setLimits",
			errorMsg:     "validation failed: 'negative number'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "-1", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid min limit parameter set to string",
			functionName: "setLimits",
			errorMsg:     "invalid argument value: 'wonder': for type '*big.Int'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "wonder", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid max limit parameter set to minus value",
			functionName: "setLimits",
			errorMsg:     "validation failed: 'negative number'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "1", "-1"}
			},
		},
		{
			description:  "setLimits - negative test with invalid max limit parameter set to string",
			functionName: "setLimits",
			errorMsg:     "invalid argument value: 'wonder': for type '*big.Int'",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "1", "wonder"}
			},
		},
		{
			description:  "setLimits - negative test with invalid currency parameter set to equals token",
			functionName: "setLimits",
			errorMsg:     "unknown currency. Rate for deal type distribute and currency TT was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "TT", "1", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid currency parameter set to wrong  string",
			functionName: "setLimits",
			errorMsg:     "unknown currency. Rate for deal type distribute and currency wonder was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "wonder", "1", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid currency parameter set to Numeric",
			functionName: "setLimits",
			errorMsg:     "unknown currency. Rate for deal type distribute and currency 353 was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "353", "1", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid deal Type parameter set to wrong string",
			functionName: "setLimits",
			errorMsg:     "unknown DealType. Rate for deal type wonder and currency  was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"wonder", "", "0", "10"}
			},
		},
		{
			description:  "setLimits - negative test with invalid deal Type parameter set to numeric",
			functionName: "setLimits",
			errorMsg:     "unknown DealType. Rate for deal type 353 and currency  was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"353", "", "1", "10"}
			},
		},
		{
			description:  "setLimits - negative test with incorrect number of parameters",
			functionName: "setLimits",
			errorMsg:     "incorrect number of keys or signs",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "", "", "1", "10"}
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
				resp *peer.Response
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
