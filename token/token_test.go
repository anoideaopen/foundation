package token

import (
	"encoding/hex"
	"errors"
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

const (
	testTokenSymbol = "TT"
	testTokenCCName = "tt"

	testTokenGetIssuerFnName           = "getIssuer"
	testTokenGetFeeSetterFnName        = "getFeeSetter"
	testTokenGetFeeAddressSetterFnName = "getFeeAddressSetter"

	testEmissionAddFnName   = "emissionAdd"
	testEmissionSubFnName   = "emissionSub"
	testSetFeeSubFnName     = "setFee"
	testSetFeeAddressFnName = "setFeeAddress"
)

func TestTokenSeries(t *testing.T) {
	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

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
			description:  "Issuer address check",
			functionName: testTokenGetIssuerFnName,
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{}
			},
			funcCheckQuery: func(t *testing.T, payload []byte) {
				actualIssuerAddr := string(payload)
				require.Contains(t, actualIssuerAddr, issuer.AddressBase58Check)
			},
		},
		{
			description:  "FeeSetter address check",
			functionName: testTokenGetFeeSetterFnName,
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{}
			},
			funcCheckQuery: func(t *testing.T, payload []byte) {
				actualFeeSetterAddr := string(payload)
				require.Contains(t, actualFeeSetterAddr, feeSetter.AddressBase58Check)
			},
		},
		{
			description:  "FeeAddressSetter address check",
			functionName: testTokenGetFeeAddressSetterFnName,
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{}
			},
			funcCheckQuery: func(t *testing.T, payload []byte) {
				actualFeeAddressSetterAddr := string(payload)
				require.Contains(t, actualFeeAddressSetterAddr, feeAddressSetter.AddressBase58Check)
			},
		},
		{
			description:  "Checking that emission is working",
			functionName: testEmissionAddFnName,
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{user.AddressBase58Check, "1000"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == metadataKey {
						etl := &pbfound.Token{}
						err = pb.Unmarshal(v, etl)
						require.NoError(t, err)
						require.True(t, pb.Equal(etl, &pbfound.Token{
							TotalEmission: new(big.Int).SetUint64(1000).Bytes(),
						}))
						j++
					} else if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(1000).Bytes(), v)
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
			description:  "Checking that emission sub is working",
			functionName: testEmissionSubFnName,
			errorMsg:     "",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				rawMetadata, err := pb.Marshal(&pbfound.Token{
					TotalEmission: new(big.Int).SetUint64(1000).Bytes(),
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1000).Bytes()

				return []string{user.AddressBase58Check, "100"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == metadataKey {
						etl := &pbfound.Token{}
						err = pb.Unmarshal(v, etl)
						require.NoError(t, err)
						require.True(t, pb.Equal(etl, &pbfound.Token{
							TotalEmission: new(big.Int).SetUint64(900).Bytes(),
						}))
						j++
					} else if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(900).Bytes(), v)
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
			description:  "Checking that setting fee is working - setFeeAddress",
			functionName: testSetFeeAddressFnName,
			errorMsg:     "",
			signUser:     feeAddressSetter,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{feeAggregator.AddressBase58Check}
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
						FeeAddress: feeAggregator.AddressBytes,
					}))

					return
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "Checking that setting fee is working - setFee",
			functionName: testSetFeeSubFnName,
			errorMsg:     "",
			signUser:     feeSetter,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{testTokenSymbol, "500000", "100", "100000"}
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
						Fee: &pbfound.TokenFee{
							Currency: testTokenSymbol,
							Fee:      new(big.Int).SetUint64(500000).Bytes(),
							Floor:    new(big.Int).SetUint64(100).Bytes(),
							Cap:      new(big.Int).SetUint64(100000).Bytes(),
						},
					}))

					return
				}
				require.Fail(t, "not found checking data")
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				testTokenCCName,
				testTokenSymbol,
				8,
				issuer.AddressBase58Check,
				feeSetter.AddressBase58Check,
				feeAddressSetter.AddressBase58Check,
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&TestToken{})
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

// TestToken helps to test base token roles.
type TestToken struct {
	BaseToken
}

func (tt *TestToken) QueryGetIssuer() (string, error) {
	addr := tt.Issuer().String()
	return addr, nil
}

func (tt *TestToken) QueryGetFeeSetter() (string, error) {
	addr := tt.FeeSetter().String()
	return addr, nil
}

func (tt *TestToken) QueryGetFeeAddressSetter() (string, error) {
	addr := tt.FeeAddressSetter().String()
	return addr, nil
}

func (tt *TestToken) TxEmissionAdd(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New(ErrUnauthorized)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(ErrAmountEqualZero)
	}
	if err := tt.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return tt.EmissionAdd(amount)
}

func (tt *TestToken) TxEmissionSub(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New(ErrUnauthorized)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(ErrAmountEqualZero)
	}
	if err := tt.TokenBalanceSub(address, amount, "txEmitSub"); err != nil {
		return err
	}
	return tt.EmissionSub(amount)
}
