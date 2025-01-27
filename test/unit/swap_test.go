package unit

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/swap"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures"
	"github.com/anoideaopen/foundation/token"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	swapDoneEventCounter      = "__swap_done_event_counter"
	miltiswapDoneEventCounter = "__multiswap_done_event_counter"
)

type FnCallCount struct {
	Count int `json:"count"`
}

type CustomToken struct {
	token.BaseToken
}

// OnSwapDoneEvent is a swap done callback.
func (ct *CustomToken) OnSwapDoneEvent(
	token string,
	owner *types.Address,
	amount *big.Int,
) {
	fmt.Printf(
		"OnSwapEvent(): symbol: %s, token: %s, owner: %s, amount: %s\n",
		ct.ContractConfig().Symbol,
		token,
		owner.String(),
		amount.String(),
	)

	_ = ct.incSwapCallCount()
}

// incSwapCallCount increments OnSwapDoneEvent function call counter.
// Counter stored in chaincode state.
func (ct *CustomToken) incSwapCallCount() error {
	calledBytes, _ := ct.GetStub().GetState(swapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	fcc.Count++

	calledBytes, _ = json.Marshal(fcc)
	_ = ct.GetStub().PutState(swapDoneEventCounter, calledBytes)

	return nil
}

// QuerySwapDoneEventCallCount fetches OnSwapDoneEvent call counter value.
// Counter stored in chaincode state.
func (ct *CustomToken) QuerySwapDoneEventCallCount() (int, error) {
	calledBytes, _ := ct.GetStub().GetState(swapDoneEventCounter)
	var fcc FnCallCount
	_ = json.Unmarshal(calledBytes, &fcc)

	return fcc.Count, nil
}

func TestSwap(t *testing.T) {
	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	swapKeyEtl := "123"
	hashed := sha3.Sum256([]byte(swapKeyEtl))
	swapHash := hex.EncodeToString(hashed[:])

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
			description:  "swapBegin - disable swaps",
			functionName: "swapBegin",
			errorMsg:     "method 'swapBegin' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableSwaps: true},
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
			description:  "swapCancel - disable swaps",
			functionName: "swapCancel",
			errorMsg:     "method 'swapCancel' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableSwaps: true},
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
			description:  "swapDone - disable swaps",
			functionName: "swapDone",
			errorMsg:     core.ErrSwapDisabled.Error(),
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableSwaps: true},
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
			description:  "swapGet - disable swaps",
			functionName: "swapGet",
			errorMsg:     "method 'swapGet' not found",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						Options:  &pbfound.ChaincodeOptions{DisableSwaps: true},
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
			description:  "swapBegin - ok",
			functionName: "swapBegin",
			errorMsg:     "",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1000).Bytes()

				return []string{"CC", "VT", "450", swapHash}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{hex.EncodeToString(resp.GetId())})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(550).Bytes(), v)
						j++
					} else if k == swapKey {
						s := &pbfound.Swap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.Swap{
							Id:      resp.GetId(),
							Creator: user.AddressBytes,
							Owner:   user.AddressBytes,
							Token:   "CC",
							Amount:  new(big.Int).SetUint64(450).Bytes(),
							From:    "CC",
							To:      "VT",
							Hash:    hashed[:],
							Timeout: 10800,
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
			description:  "swapBegin back - ok",
			functionName: "swapBegin",
			errorMsg:     "",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1000).Bytes()

				return []string{"VT", "VT", "450", swapHash}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{hex.EncodeToString(resp.GetId())})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(550).Bytes(), v)
						j++
					} else if k == swapKey {
						s := &pbfound.Swap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.Swap{
							Id:      resp.GetId(),
							Creator: user.AddressBytes,
							Owner:   user.AddressBytes,
							Token:   "VT",
							Amount:  new(big.Int).SetUint64(450).Bytes(),
							From:    "CC",
							To:      "VT",
							Hash:    hashed[:],
							Timeout: 10800,
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
			description:  "swapBegin - incorrect swap",
			functionName: "swapBegin",
			errorMsg:     "incorrect swap",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1000).Bytes()

				return []string{"BA", "VT", "450", swapHash}
			},
		},
		{
			description:  "answer - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					Swaps: []*pbfound.Swap{
						{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "VT",
							Amount:  new(big.Int).SetUint64(450).Bytes(),
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
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				k, v := mockStub.PutStateArgsForCall(0)
				require.Equal(t, swapKey, k)

				s := &pbfound.Swap{}
				err = pb.Unmarshal(v, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.Swap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "VT",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
					From:    "VT",
					To:      "CC",
					Hash:    hashed[:],
					Timeout: 300,
				}))
			},
		},
		{
			description:  "answer back - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				dataIn, err := pb.Marshal(&pbfound.Batch{
					Swaps: []*pbfound.Swap{
						{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "CC",
							Amount:  new(big.Int).SetUint64(450).Bytes(),
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

				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
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
						s := &pbfound.Swap{}
						err = pb.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.Swap{
							Id:      txID,
							Creator: []byte("0000"),
							Owner:   user.AddressBytes,
							Token:   "CC",
							Amount:  new(big.Int).SetUint64(450).Bytes(),
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
			description:  "swapGet - ok",
			functionName: "swapGet",
			errorMsg:     "",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				s, err := pb.Marshal(&pbfound.Swap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
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

				s := &pbfound.Swap{}
				err = json.Unmarshal(payload, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.Swap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
					From:    "CC",
					To:      "VT",
					Hash:    hashed[:],
					Timeout: 10800,
				}))
			},
		},
		{
			description:  "swapGet - not found",
			functionName: "swapGet",
			errorMsg:     "swap doesn't exist by key",
			isQuery:      true,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{mockStub.GetTxID()}
			},
		},
		{
			description:  "swapDone - ok",
			functionName: "swapDone",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.Swap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "VT",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
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
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "swapDone back - ok",
			functionName: "swapDone",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.Swap{
					Id:      txID,
					Creator: []byte("0000"),
					Owner:   user.AddressBytes,
					Token:   "CC",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
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
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "robot done - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.Swap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "CC",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
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
					Keys: []*pbfound.SwapKey{
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
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)
				require.Equal(t, swapKey, mockStub.DelStateArgsForCall(0))

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				k, v := mockStub.PutStateArgsForCall(0)
				require.Equal(t, givenBalanceKey, k)
				require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
			},
		},
		{
			description:  "robot done back - ok",
			functionName: "batchExecute",
			errorMsg:     "",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
				require.NoError(t, err)

				txID, err := hex.DecodeString(mockStub.GetTxID())
				require.NoError(t, err)

				s, err := pb.Marshal(&pbfound.Swap{
					Id:      txID,
					Creator: user.AddressBytes,
					Owner:   user.AddressBytes,
					Token:   "VT",
					Amount:  new(big.Int).SetUint64(450).Bytes(),
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
					Keys: []*pbfound.SwapKey{
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
				swapKey, err := mockStub.CreateCompositeKey(swap.SwapCompositeType, []string{mockStub.GetTxID()})
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
