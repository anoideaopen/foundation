package token

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestBaseTokenTransfer(t *testing.T) {
	const (
		ba1 = "BA02_GOLDBARLONDON.01"
		ba2 = "BA02_GOLDBARLONDON.02"
	)

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	buyer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	seller, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description         string
		functionName        string
		isQuery             bool
		errorMsg            string
		codeResp            int32
		signUser            *mocks.UserFoundation
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
		funcCheckQuery      func(t *testing.T, payload []byte)
	}{
		{
			description:  "predictFee - ok",
			functionName: "predictFee",
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"100"}
			},
			funcCheckQuery: func(t *testing.T, payload []byte) {
				predict := &Predict{}
				err = json.Unmarshal(payload, &predict)
				require.NoError(t, err)
				require.Equal(t, &Predict{
					Currency: "VT",
					Fee:      new(big.Int).SetUint64(1),
				}, predict)
			},
		},
		{
			description:  "buyToken - ok",
			functionName: "buyToken",
			errorMsg:     "",
			signUser:     seller,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "usd",
							Rate:     new(big.Int).SetUint64(100000000).Bytes(),
							Min:      new(big.Int).SetUint64(1).Bytes(),
							Max:      new(big.Int).SetUint64(10).Bytes(),
						},
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(10).Bytes()

				userBalanceKey, err = mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{seller.AddressBase58Check, "usd"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(5).Bytes()

				return []string{"5", "usd"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				issuerUSDBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, "usd"})
				require.NoError(t, err)

				sellerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{seller.AddressBase58Check})
				require.NoError(t, err)

				sellerUSDBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{seller.AddressBase58Check, "usd"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == issuerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(5).Bytes(), v)
						j++
					} else if k == issuerUSDBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(5).Bytes(), v)
						j++
					} else if k == sellerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(5).Bytes(), v)
						j++
					} else if k == sellerUSDBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					}

					if j == 4 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "transfer - amount should be more than zero",
			functionName: "transfer",
			errorMsg:     ErrAmountEqualZero,
			signUser:     seller,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{buyer.AddressBase58Check, "0", ""}
			},
		},
		{
			description:  "transfer - insufficient balance",
			functionName: "transfer",
			errorMsg:     "insufficient balance",
			signUser:     seller,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				sellerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{seller.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[sellerVTBalanceKey] = new(big.Int).SetUint64(5).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer - ok",
			functionName: "transfer",
			errorMsg:     "",
			signUser:     seller,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				sellerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{seller.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[sellerVTBalanceKey] = new(big.Int).SetUint64(5).Bytes()

				return []string{buyer.AddressBase58Check, "5", ""}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				sellerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{seller.AddressBase58Check})
				require.NoError(t, err)

				buyerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{buyer.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == buyerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(5).Bytes(), v)
						j++
					} else if k == sellerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					}

					if j == 2 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "transfer with fee - [negative] trying to transfer when sender equals address to",
			functionName: "transfer",
			errorMsg:     "sender and recipient are same users",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{issuer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer with fee - [negative] trying to transfer negative amount",
			functionName: "transfer",
			errorMsg:     "invalid argument value: '-100': validation failed: 'negative number': validate TxTransfer, argument 2",
			signUser:     issuer,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "-100", ""}
			},
		},
		{
			description:  "transfer with fee - [negative] trying to transfer zero amount",
			functionName: "transfer",
			errorMsg:     ErrAmountEqualZero,
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "0", ""}
			},
		},
		{
			description:  "transfer with fee - [negative] trying to transfer when fee address is not set",
			functionName: "transfer",
			errorMsg:     ErrFeeAddressNotConfigured.Error(),
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer with fee - ok",
			functionName: "transfer",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
					FeeAddress: feeAggregator.AddressBytes,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				buyerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{buyer.AddressBase58Check})
				require.NoError(t, err)

				feeAggregatorVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{feeAggregator.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == buyerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(100).Bytes(), v)
						j++
					} else if k == issuerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == feeAggregatorVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(1).Bytes(), v)
						j++
					}

					if j == 3 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "transfer with fee - empty currency",
			functionName: "transfer",
			errorMsg:     "config fee currency can't be empty",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
					FeeAddress: feeAggregator.AddressBytes,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer with fee - with nil token fee",
			functionName: "transfer",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					FeeAddress: feeAggregator.AddressBytes,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				buyerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{buyer.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == buyerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(100).Bytes(), v)
						j++
					} else if k == issuerVTBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(1).Bytes(), v)
						j++
					}

					if j == 2 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "transfer with fee - with wrong address",
			functionName: "transfer",
			errorMsg:     "config fee address has a wrong len. actual 4 but expected 32",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "VT",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
					FeeAddress: []byte("1111"),
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer with fee - with wrong symbol",
			functionName: "transfer",
			errorMsg:     "incorrect fee currency",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "asd",
						Fee:      new(big.Int).SetUint64(500000).Bytes(),
						Floor:    new(big.Int).SetUint64(1).Bytes(),
					},
					FeeAddress: feeAggregator.AddressBytes,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				issuerVTBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKey] = new(big.Int).SetUint64(101).Bytes()

				return []string{buyer.AddressBase58Check, "100", ""}
			},
		},
		{
			description:  "transfer allowed industrial balance",
			functionName: "allowedIndustrialBalanceTransfer",
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				issuerVTBalanceKeyBA1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, ba1})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKeyBA1] = new(big.Int).SetUint64(100000000).Bytes()

				issuerVTBalanceKeyBA2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, ba2})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[issuerVTBalanceKeyBA2] = new(big.Int).SetUint64(100000000).Bytes()

				industrialAssets := []*types.MultiSwapAsset{
					{
						Group:  ba1,
						Amount: "50000000",
					},
					{
						Group:  ba2,
						Amount: "100000000",
					},
				}
				rawGA, err := json.Marshal(industrialAssets)
				require.NoError(t, err)

				return []string{buyer.AddressBase58Check, string(rawGA), "ref"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				issuerVTBalanceKeyBA1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, ba1})
				require.NoError(t, err)

				issuerVTBalanceKeyBA2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, ba2})
				require.NoError(t, err)

				buyerVTBalanceKeyBA1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{buyer.AddressBase58Check, ba1})
				require.NoError(t, err)

				buyerVTBalanceKeyBA2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{buyer.AddressBase58Check, ba2})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == issuerVTBalanceKeyBA1 {
						require.Equal(t, new(big.Int).SetUint64(50000000).Bytes(), v)
						j++
					} else if k == issuerVTBalanceKeyBA2 {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == buyerVTBalanceKeyBA1 {
						require.Equal(t, new(big.Int).SetUint64(50000000).Bytes(), v)
						j++
					} else if k == buyerVTBalanceKeyBA2 {
						require.Equal(t, new(big.Int).SetUint64(100000000).Bytes(), v)
						j++
					}

					if j == 4 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				"vt",
				"VT",
				8,
				issuer.AddressBase58Check,
				feeSetter.AddressBase58Check,
				feeAddressSetter.AddressBase58Check,
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&VT{})
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
			if testCase.codeResp == int32(shim.ERROR) {
				require.Equal(t, resp.GetStatus(), testCase.codeResp)
				require.Contains(t, resp.GetMessage(), testCase.errorMsg)
				require.Empty(t, resp.GetPayload())
				return
			}

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
