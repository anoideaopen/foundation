package token

import (
	"encoding/hex"
	"encoding/json"
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

func TestMethods(t *testing.T) {
	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	cfg := &pbfound.Token{
		Rates: []*pbfound.TokenRate{
			{
				DealType: "distribute",
				Rate:     new(big.Int).SetUint64(1).Bytes(),
			},
		},
	}
	cfg1 := &pbfound.Token{
		Rates: []*pbfound.TokenRate{
			{
				DealType: "distribute",
				Rate:     new(big.Int).SetUint64(1).Bytes(),
				Min:      new(big.Int).SetUint64(1).Bytes(),
			},
		},
	}

	for _, testCase := range []struct {
		description         string
		functionName        string
		isQuery             bool
		errorMsg            string
		signUser            *mocks.UserFoundation
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
		funcCheckQuery      func(t *testing.T, payload []byte)
	}{
		{
			description:  "setRate-unauthorized",
			functionName: "setRate",
			errorMsg:     ErrUnauthorized,
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "1"}
			},
		},
		{
			description:  "setRate-0",
			functionName: "setRate",
			errorMsg:     "trying to set rate = 0",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "", "0"}
			},
		},
		{
			description:  "setRate-impossible",
			functionName: "setRate",
			errorMsg:     "currency is equals token: it is impossible",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{"distribute", "TT", "3"}
			},
		},
		{
			description:  "setRate-ok",
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
					require.True(t, pb.Equal(etl, cfg))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "deleteRate-ok",
			functionName: "deleteRate",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata
				return []string{"distribute", ""}
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
						Rates: []*pbfound.TokenRate{},
					}))

					return
				}
				require.Fail(t, "not found metadata")
			},
		},
		{
			description:  "metadata-ok",
			functionName: "metadata",
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{}
			},
			funcCheckQuery: func(t *testing.T, payload []byte) {
				var meta Metadata
				err = json.Unmarshal(payload, &meta)
				require.NoError(t, err)

				var tokenMethods = []string{"addDocs", "allowedBalanceOf", "lockedAllowedBalanceOf",
					"allowedIndustrialBalanceTransfer", "balanceOf", "lockedAllowedBalancesWithPagination",
					"lockedBalanceOf", "lockedTokenBalancesWithPagination", "buildInfo", "buyBack", "buyToken",
					"cancelCCTransferFrom", "allowedBalancesWithPagination",
					"channelTransferByAdmin", "channelMultiTransferByAdmin", "channelTransferByCustomer", "channelMultiTransferByCustomer", "channelTransferFrom",
					"channelTransferTo", "channelTransfersFrom", "commitCCTransferFrom", "coreChaincodeIDName",
					"createCCTransferTo", "deleteCCTransferFrom", "deleteCCTransferTo", "deleteDoc",
					"deleteRate", "documentsList", "getFeeTransfer", "getLockedAllowedBalance",
					"getLockedTokenBalance", "getNonce", "givenBalance", "givenBalancesWithPagination", "groupBalanceOf",
					"healthCheck", "lockAllowedBalance", "tokenBalancesWithPagination",
					"lockTokenBalance", "metadata", "multiSwapBegin", "multiSwapCancel", "multiSwapGet",
					"nameOfFiles", "predictFee", "setFee", "setFeeAddress", "setLimits", "setRate",
					"srcFile", "srcPartFile", "swapBegin", "swapCancel", "swapGet", "systemEnv", "transfer",
					"unlockAllowedBalance", "healthCheckNb", "unlockTokenBalance", "transferBalance"}
				require.ElementsMatch(t, tokenMethods, meta.Methods)
			},
		},
		{
			description:  "setLimits-unknown DealType",
			functionName: "setLimits",
			errorMsg:     "unknown DealType. Rate for deal type makarone and currency  was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"makarone", "", "1", "3"}
			},
		},
		{
			description:  "setLimits-unknown currency",
			functionName: "setLimits",
			errorMsg:     "unknown currency. Rate for deal type distribute and currency fish was not set",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(cfg)
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"distribute", "fish", "1", "3"}
			},
		},
		{
			description:  "setLimits-min limit is greater",
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
			description:  "setLimits-ok",
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
					require.True(t, pb.Equal(etl, cfg1))

					return
				}
				require.Fail(t, "not found metadata")
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
			if testCase.isQuery {
				resp = mockStub.QueryChaincode(cc, testCase.functionName, parameters...)
			} else {
				txId, resp = mockStub.TxInvokeChaincodeSigned(cc, testCase.functionName, testCase.signUser, "", "", "", parameters...)
			}

			// check result
			require.Equal(t, resp.GetStatus(), int32(shim.OK))
			require.Empty(t, resp.GetMessage())

			if testCase.isQuery {
				if testCase.funcCheckQuery != nil {
					testCase.funcCheckQuery(t, resp.GetPayload())
				}
				return
			}

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
