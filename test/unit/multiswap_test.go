package unit

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/multiswap"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

// OnMultiSwapDoneEvent is a multi-swap done callback.
func (ct *CustomToken) OnMultiSwapDoneEvent(
	token string,
	owner *types.Address,
	assets []*pbfound.Asset,
) {
	type asset struct {
		group  string
		amount string
	}

	var al []asset
	for _, a := range assets {
		amount := new(big.Int)
		amount.SetBytes(a.Amount)
		al = append(al, asset{
			group:  a.Group,
			amount: amount.String(),
		})
	}

	fmt.Printf(
		"OnMultiSwapDoneEvent(): symbol: %s, token: %s, owner: %s, assets: %v\n",
		ct.ContractConfig().Symbol,
		token,
		owner.String(),
		al,
	)

	_ = ct.incrMultiSwapCallCount()
}

// incrMultiSwapCallCount increments OnMultiSwapDoneEvent function call counter.
// Counter stored in chaincode state.
func (ct *CustomToken) incrMultiSwapCallCount() error {
	calledBytes, _ := ct.GetStub().GetState(miltiswapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	fcc.Count++

	calledBytes, _ = json.Marshal(fcc)
	_ = ct.GetStub().PutState(miltiswapDoneEventCounter, calledBytes)

	return nil
}

// QueryMultiSwapDoneEventCallCount fetches OnMultiSwapDoneEvent call counter value.
// Counter stored in chaincode state.
func (ct *CustomToken) QueryMultiSwapDoneEventCallCount() (int, error) {
	calledBytes, _ := ct.GetStub().GetState(miltiswapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	return fcc.Count, nil
}

func TestMultiSwap(t *testing.T) {
	const (
		TokenCC           = "CC"
		TokenVT           = "VT"
		G1                = "A.101"
		G2                = "A.102"
		AllowedBalanceCC1 = TokenCC + "_" + G1
		AllowedBalanceCC2 = TokenCC + "_" + G2
		AllowedBalanceVT1 = TokenVT + "_" + G1
		AllowedBalanceVT2 = TokenVT + "_" + G2
	)

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	swapKeyEtl := "123"
	hashed := sha3.Sum256([]byte(swapKeyEtl))
	swapHash := hex.EncodeToString(hashed[:])

	msaCCBytes, err := json.Marshal(&types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  AllowedBalanceCC1,
				Amount: "1",
			},
			{
				Group:  AllowedBalanceCC2,
				Amount: "1",
			},
		},
	})
	require.NoError(t, err)

	msaCCpb := []*pbfound.Asset{
		{
			Group:  AllowedBalanceCC1,
			Amount: new(big.Int).SetUint64(1).Bytes(),
		},
		{
			Group:  AllowedBalanceCC2,
			Amount: new(big.Int).SetUint64(1).Bytes(),
		},
	}

	msaVTBytes, err := json.Marshal(&types.MultiSwapAssets{
		Assets: []*types.MultiSwapAsset{
			{
				Group:  AllowedBalanceVT1,
				Amount: "1",
			},
			{
				Group:  AllowedBalanceVT2,
				Amount: "1",
			},
		},
	})
	require.NoError(t, err)

	msaVTpb := []*pbfound.Asset{
		{
			Group:  AllowedBalanceVT1,
			Amount: new(big.Int).SetUint64(1).Bytes(),
		},
		{
			Group:  AllowedBalanceVT2,
			Amount: new(big.Int).SetUint64(1).Bytes(),
		},
	}

	for _, testCase := range []struct {
		description         string
		functionName        string
		isQuery             bool
		noBatch             bool
		errorMsg            string
		signUser            *mocks.UserFoundation
		codeResp            int32
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
		funcCheckQuery      func(t *testing.T, mockStub *mockstub.MockStub, payload []byte)
	}{
		{
			description:  "multiSwapBegin - disable swaps",
			functionName: "multiSwapBegin",
			errorMsg:     "method 'multiSwapBegin' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableMultiSwaps: true},
						RobotSKI: fixtures.RobotHashedCert,
						Admin:    &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
					Token: &pbfound.TokenConfig{
						Name:     "CC Token",
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{"", ""}
			},
		},
		{
			description:  "multiSwapCancel - disable swaps",
			functionName: "multiSwapCancel",
			errorMsg:     "method 'multiSwapCancel' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableMultiSwaps: true},
						RobotSKI: fixtures.RobotHashedCert,
						Admin:    &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
					Token: &pbfound.TokenConfig{
						Name:     "CC Token",
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{"", ""}
			},
		},
		{
			description:  "multiSwapDone - disable swaps",
			functionName: "multiSwapDone",
			errorMsg:     core.ErrSwapDisabled.Error(),
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableMultiSwaps: true},
						RobotSKI: fixtures.RobotHashedCert,
						Admin:    &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
					Token: &pbfound.TokenConfig{
						Name:     "CC Token",
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{"", ""}
			},
		},
		{
			description:  "multiSwapGet - disable swaps",
			functionName: "multiSwapGet",
			errorMsg:     "method 'multiSwapGet' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableMultiSwaps: true},
						RobotSKI: fixtures.RobotHashedCert,
						Admin:    &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
					Token: &pbfound.TokenConfig{
						Name:     "CC Token",
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{"", ""}
			},
		},
		{
			description:  "multiSwapBegin - ok",
			functionName: "multiSwapBegin",
			errorMsg:     "",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G1})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				userBalanceKey, err = mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G2})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				return []string{TokenCC, string(msaCCBytes), TokenVT, swapHash}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G1})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G2})
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{hex.EncodeToString(resp.GetId())})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == swapKey {
						s := &pbfound.MultiSwap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.MultiSwap{
							Id:      resp.GetId(),
							Creator: user.AddressBytes,
							Owner:   user.AddressBytes,
							Token:   "CC",
							Assets:  msaCCpb,
							From:    "CC",
							To:      "VT",
							Hash:    hashed[:],
							Timeout: 10800,
						}))
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
			description:  "multiSwapBegin back - ok",
			functionName: "multiSwapBegin",
			errorMsg:     "",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT1})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				userBalanceKey, err = mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT2})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				return []string{TokenVT, string(msaVTBytes), TokenVT, swapHash}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT1})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT2})
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{hex.EncodeToString(resp.GetId())})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == swapKey {
						s := &pbfound.MultiSwap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.MultiSwap{
							Id:      resp.GetId(),
							Creator: user.AddressBytes,
							Owner:   user.AddressBytes,
							Token:   "VT",
							Assets:  msaVTpb,
							From:    "CC",
							To:      "VT",
							Hash:    hashed[:],
							Timeout: 10800,
						}))
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
			description:  "multiSwapBegin - incorrect swap",
			functionName: "multiSwapBegin",
			errorMsg:     "incorrect swap",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G1})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				userBalanceKey, err = mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G2})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()

				return []string{"BA", string(msaCCBytes), "VT", swapHash}
			},
		},
		{
			description:  "multiswap answer - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					MultiSwaps: []*pbfound.MultiSwap{
						{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "VT",
							Assets:  msaVTpb,
							From:    "VT",
							To:      "CC",
							Hash:    hashed[:],
							Timeout: 10800,
						},
					},
				})
				require.NoError(t, err)

				return []string{string(dataIn)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				k, v := mockStub.PutStateArgsForCall(0)
				require.Equal(t, swapKey, k)

				s := &pbfound.MultiSwap{}
				err = pb.Unmarshal(v, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.MultiSwap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "VT",
					Assets:  msaVTpb,
					From:    "VT",
					To:      "CC",
					Hash:    hashed[:],
					Timeout: 300,
				}))
			},
		},
		{
			description:  "multiswap answer back - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(2).Bytes()

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					MultiSwaps: []*pbfound.MultiSwap{
						{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "CC",
							Assets:  msaCCpb,
							From:    "VT",
							To:      "CC",
							Hash:    hashed[:],
							Timeout: 10800,
						},
					},
				})
				require.NoError(t, err)
				return []string{string(dataIn)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == swapKey {
						s := &pbfound.MultiSwap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.MultiSwap{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "CC",
							Assets:  msaCCpb,
							From:    "VT",
							To:      "CC",
							Hash:    hashed[:],
							Timeout: 300,
						}))
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
			description:  "multiSwapGet - ok",
			functionName: "multiSwapGet",
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				s, err := pb.Marshal(&pbfound.MultiSwap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Assets:  msaCCpb,
					From:    "CC",
					To:      "VT",
					Hash:    hashed[:],
					Timeout: 10800,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[swapKey] = s

				return []string{mockStub.GetTxID()}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s := &pbfound.MultiSwap{}
				err = json.Unmarshal(payload, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.MultiSwap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Assets:  msaCCpb,
					From:    "CC",
					To:      "VT",
					Hash:    hashed[:],
					Timeout: 10800,
				}))
			},
		},
		{
			description:  "multiSwapGet - not found",
			functionName: "multiSwapGet",
			errorMsg:     "multiswap doesn't exist",
			isQuery:      true,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{mockStub.GetTxID()}
			},
		},
		{
			description:  "multiSwapDone - ok",
			functionName: "multiSwapDone",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.MultiSwap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "VT",
					Assets:  msaVTpb,
					From:    "VT",
					To:      "CC",
					Hash:    hashed[:],
					Timeout: 300,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[swapKey] = s

				return []string{mockStub.GetTxID(), swapKeyEtl}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT1})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, AllowedBalanceVT2})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(1).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
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
			description:  "multiSwapDone back - ok",
			functionName: "multiSwapDone",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.MultiSwap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "CC",
					Assets:  msaCCpb,
					From:    "VT",
					To:      "CC",
					Hash:    hashed[:],
					Timeout: 300,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[swapKey] = s

				return []string{mockStub.GetTxID(), swapKeyEtl}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G1})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, G2})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(1).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
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
			description:  "multiswap robot done - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.MultiSwap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Assets:  msaCCpb,
					From:    "CC",
					To:      "VT",
					Hash:    hashed[:],
					Timeout: 10800,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[swapKey] = s

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					MultiSwapsKeys: []*pbfound.SwapKey{
						{
							Id:  txID,
							Key: swapKeyEtl,
						},
					},
				})
				require.NoError(t, err)

				return []string{string(dataIn)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				k, v := mockStub.PutStateArgsForCall(0)
				require.Equal(t, givenBalanceKey, k)
				require.Equal(t, new(big.Int).SetUint64(2).Bytes(), v)
			},
		},
		{
			description:  "multiswap robot done back - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.MultiSwap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "VT",
					Assets:  msaVTpb,
					From:    "CC",
					To:      "VT",
					Hash:    hashed[:],
					Timeout: 10800,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[swapKey] = s

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					MultiSwapsKeys: []*pbfound.SwapKey{
						{
							Id:  txID,
							Key: swapKeyEtl,
						},
					},
				})
				require.NoError(t, err)
				return []string{string(dataIn)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				swapKey, err := mockStub.CreateCompositeKey(multiswap.MultiSwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				"CC Token",
				"CC",
				8,
				issuer.AddressBase58Check,
				"",
				"",
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&CustomToken{})
			require.NoError(t, err)

			parameters := testCase.funcPrepareMockStub(t, mockStub)

			var (
				txId string
				resp peer.Response
			)
			if testCase.isQuery {
				resp = mockStub.QueryChaincode(cc, testCase.functionName, parameters...)
			} else if testCase.noBatch {
				resp = mockStub.NbTxInvokeChaincode(cc, testCase.functionName, parameters...)
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
					testCase.funcCheckQuery(t, mockStub, resp.GetPayload())
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
