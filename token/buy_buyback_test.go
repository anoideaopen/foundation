package token

import (
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

const keyMetadata = "tokenMetadata"

func TestBaseTokenTxBuy(t *testing.T) {
	t.Parallel()

	var (
		vEmitAmount    = big.NewInt(10)
		vAllowedAmount = big.NewInt(5)
		vBuyAmount     = big.NewInt(1)
		vDealType      = "buyToken"
		vCurrency      = "usd"
		vRate          = big.NewInt(100000000)
		vLimitMin      = big.NewInt(1)
		vLimitMax      = big.NewInt(10)
	)

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			issuer,
			user *mocks.UserFoundation,
		) []string
		funcInvokeChaincode func(
			cc *core.Chaincode,
			mockStub *mockstub.MockStub,
			functionName string,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
			parameters ...string,
		) peer.Response
		funcCheckResult func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
			resp peer.Response,
		)
	}{
		{
			name:         "Emit token",
			functionName: "emitToken",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, issuer, user *mocks.UserFoundation) []string {
				return []string{vEmitAmount.String()}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, issuer, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, issuer, user *mocks.UserFoundation, resp peer.Response) {
				issuerBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == issuerBalanceKey {
						require.Equal(t, vEmitAmount, new(big.Int).SetBytes(value))
						checked = true
						break
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Setting rate",
			functionName: "setRate",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				return []string{vDealType, vCurrency, vRate.String()}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						expectedRate := &pbfound.TokenRate{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
						}

						require.True(t, proto.Equal(expectedRate, tokenMetadata.Rates[0]))
						checked = true
						break
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Setting limits",
			functionName: "setLimits",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation) []string {
				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{vDealType, vCurrency, vLimitMin.String(), vLimitMax.String()}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						expectedRate := &pbfound.TokenRate{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      vLimitMin.Bytes(),
							Max:      vLimitMax.Bytes(),
						}

						require.True(t, proto.Equal(expectedRate, tokenMetadata.Rates[0]))
						checked = true
						break
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "[negative] Trying to buy zero tokens",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				issuerTokenKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				userAllowedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[issuerTokenKey] = vEmitAmount.Bytes()
				mockStub.GetStateCallsMap[userAllowedKey] = vAllowedAmount.Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      vLimitMin.Bytes(),
							Max:      vLimitMax.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"0", vCurrency}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
				resp peer.Response,
			) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, ErrAmountEqualZero, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] Trying to use wrong currency",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				issuerTokenKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				userAllowedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[issuerTokenKey] = vEmitAmount.Bytes()
				mockStub.GetStateCallsMap[userAllowedKey] = vAllowedAmount.Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      vLimitMin.Bytes(),
							Max:      vLimitMax.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{vBuyAmount.String(), "rub"}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
				resp peer.Response,
			) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, ErrWrongCurrency, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] Trying to buy tokens above limits",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				issuerTokenKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				userAllowedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[issuerTokenKey] = vEmitAmount.Bytes()
				mockStub.GetStateCallsMap[userAllowedKey] = vAllowedAmount.Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      big.NewInt(2).Bytes(),
							Max:      vLimitMax.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{vBuyAmount.String(), vCurrency}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
				resp peer.Response,
			) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, ErrAmountOutOfLimits, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] Trying to buy tokens below limits",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				issuerTokenKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				userAllowedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[issuerTokenKey] = vEmitAmount.Bytes()
				mockStub.GetStateCallsMap[userAllowedKey] = vAllowedAmount.Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      vLimitMin.Bytes(),
							Max:      vLimitMax.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"20", vCurrency}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
				resp peer.Response,
			) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, ErrAmountOutOfLimits, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "Buy tokens",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				issuerTokenKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				userAllowedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[issuerTokenKey] = vEmitAmount.Bytes()
				mockStub.GetStateCallsMap[userAllowedKey] = vAllowedAmount.Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: vDealType,
							Currency: vCurrency,
							Rate:     vRate.Bytes(),
							Min:      vLimitMin.Bytes(),
							Max:      vLimitMax.Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{vBuyAmount.String(), vCurrency}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				issuer *mocks.UserFoundation,
				user *mocks.UserFoundation,
				resp peer.Response,
			) {
				keyUserBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				keyUserAllowedBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				keyIssuerBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{issuer.AddressBase58Check})
				require.NoError(t, err)

				keyIssuerAllowedBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{issuer.AddressBase58Check, vCurrency})
				require.NoError(t, err)

				checkedUserBalance := false
				checkedUserAllowed := false
				checkedIssuerBalance := false
				checkedIssuerAllowed := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyUserBalance {
						require.Equal(t, big.NewInt(1).Bytes(), value)
						checkedUserBalance = true
					}
					if putStateKey == keyUserAllowedBalance {
						require.Equal(t, big.NewInt(4).Bytes(), value)
						checkedUserAllowed = true
					}
					if putStateKey == keyIssuerBalance {
						require.Equal(t, big.NewInt(9).Bytes(), value)
						checkedIssuerBalance = true
					}
					if putStateKey == keyIssuerAllowedBalance {
						require.Equal(t, big.NewInt(1).Bytes(), value)
						checkedIssuerAllowed = true
					}
				}
				require.True(t, checkedUserBalance && checkedUserAllowed && checkedIssuerBalance && checkedIssuerAllowed)
			},
		},
	}

	for _, testCase := range testCollection {
		t.Run(testCase.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig(
				"Validation Token",
				"VT",
				uint(8),
				issuer.AddressBase58Check,
				"",
				"",
				"",
				nil,
			)

			cc, err := core.NewCC(&VT{})
			require.NoError(t, err)

			parameters := testCase.funcPrepareMockStub(t, mockStub, issuer, user)
			resp := testCase.funcInvokeChaincode(cc, mockStub, testCase.functionName, issuer, user, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())
			testCase.funcCheckResult(t, mockStub, issuer, user, resp)
		})
	}
}

// ToDo: add buyback test
