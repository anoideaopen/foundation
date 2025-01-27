package unit

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures"
	pb "github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestChannelTransfer(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	id := uuid.NewString()

	itemsCCpb := []*pbfound.CCTransferItem{
		{Token: "CC_1", Amount: new(big.Int).SetInt64(450).Bytes()},
		{Token: "CC_2", Amount: new(big.Int).SetInt64(900).Bytes()},
	}
	itemsCC := []core.TransferItem{
		{Token: "CC_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "CC_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsCCJSON, err := json.Marshal(itemsCC)
	require.NoError(t, err)

	itemsVTpb := []*pbfound.CCTransferItem{
		{Token: "VT_1", Amount: new(big.Int).SetInt64(450).Bytes()},
		{Token: "VT_2", Amount: new(big.Int).SetInt64(900).Bytes()},
	}
	itemsVT := []core.TransferItem{
		{Token: "VT_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "VT_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsVTJSON, err := json.Marshal(itemsVT)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description         string
		functionName        string
		isQuery             bool
		noSign              bool
		noBatch             bool
		errorMsg            string
		signUser            *mocks.UserFoundation
		codeResp            int32
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
		funcCheckQuery      func(t *testing.T, mockStub *mockstub.MockStub, payload []byte)
	}{
		{
			description:  "channelTransferByCustomer forward - ok",
			functionName: "channelTransferByCustomer",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				return []string{id, "VT", "CC", "450"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							Token:            "CC",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: true,
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
			description:  "channelTransferByCustomer backward - ok",
			functionName: "channelTransferByCustomer",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				return []string{id, "VT", "VT", "450"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							Token:            "VT",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: false,
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
			description:  "channelTransferByCustomer - CC-to-CC transfer",
			functionName: "channelTransferByCustomer",
			errorMsg:     cctransfer.ErrInvalidChannel.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{id, "CC", "CC", "450"}
			},
		},
		{
			description:  "channelTransferByCustomer - transferring the wrong tokens",
			functionName: "channelTransferByCustomer",
			errorMsg:     cctransfer.ErrInvalidToken.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{id, "VT", "FIAT", "450"}
			},
		},
		{
			description:  "channelTransferByCustomer - insufficient funds",
			functionName: "channelTransferByCustomer",
			errorMsg:     "failed to subtract token balance: insufficient balance",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{id, "VT", "CC", "450"}
			},
		},
		{
			description:  "channelTransferByCustomer - such a transfer is already in place",
			functionName: "channelTransferByCustomer",
			errorMsg:     cctransfer.ErrIDTransferExist.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				tr, err := pb.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = tr

				return []string{id, "VT", "CC", "450"}
			},
		},
		{
			description:  "channelTransferByAdmin forward - ok",
			functionName: "channelTransferByAdmin",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				return []string{id, "VT", user.AddressBase58Check, "CC", "450"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							Token:            "CC",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: true,
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
			description:  "channelTransferByAdmin backward - ok",
			functionName: "channelTransferByAdmin",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				return []string{id, "VT", user.AddressBase58Check, "VT", "450"}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							Token:            "VT",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: false,
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
			description:  "channelTransferByAdmin - admin in ContractConfig was not set",
			functionName: "channelTransferByAdmin",
			errorMsg:     cctransfer.ErrAdminNotSet.Error(),
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:   "CC",
						RobotSKI: fixtures.RobotHashedCert,
					},
					Token: &pbfound.TokenConfig{
						Name:     "CC Token",
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{id, "VT", user.AddressBase58Check, "CC", "450"}
			},
		},
		{
			description:  "channelTransferByAdmin - admin function sent by someone other than admin",
			functionName: "channelTransferByAdmin",
			errorMsg:     cctransfer.ErrUnauthorisedNotAdmin.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{id, "VT", user.AddressBase58Check, "CC", "450"}
			},
		},
		{
			description:  "channelTransferByAdmin - the admin sends the transfer to himself",
			functionName: "channelTransferByAdmin",
			errorMsg:     cctransfer.ErrInvalidIDUser.Error(),
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{id, "VT", issuer.AddressBase58Check, "CC", "450"}
			},
		},
		{
			description:  "channelMultiTransferByCustomer forward - ok",
			functionName: "channelMultiTransferByCustomer",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey1] = new(big.Int).SetUint64(450).Bytes()

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey2] = new(big.Int).SetUint64(900).Bytes()

				return []string{id, "VT", string(itemsCCJSON)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
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
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(1350).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							User:             user.AddressBytes,
							ForwardDirection: true,
							Items:            itemsCCpb,
						}))
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
			description:  "channelMultiTransferByCustomer backward - ok",
			functionName: "channelMultiTransferByCustomer",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey1] = new(big.Int).SetUint64(450).Bytes()

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey2] = new(big.Int).SetUint64(900).Bytes()

				return []string{id, "VT", string(itemsVTJSON)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
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
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							User:             user.AddressBytes,
							ForwardDirection: false,
							Items:            itemsVTpb,
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
			description:  "channelMultiTransferByCustomer - invalid argument token",
			functionName: "channelMultiTransferByCustomer",
			errorMsg:     cctransfer.ErrInvalidToken.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				itemsFIAT, err := json.Marshal([]core.TransferItem{
					{Token: "FIAT_1", Amount: new(big.Int).SetInt64(450)},
					{Token: "FIAT_2", Amount: new(big.Int).SetInt64(900)},
				})
				require.NoError(t, err)

				return []string{id, "VT", string(itemsFIAT)}
			},
		},
		{
			description:  "channelMultiTransferByCustomer - invalid items count found 0",
			functionName: "channelMultiTransferByCustomer",
			errorMsg:     "invalid argument transfer items count found 0 but expected from 1 to 100",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				items := make([]core.TransferItem, 0)
				itemsJSON, err := json.Marshal(items)
				require.NoError(t, err)

				return []string{id, "VT", string(itemsJSON)}
			},
		},
		{
			description:  "channelMultiTransferByCustomer - invalid argument token already exists",
			functionName: "channelMultiTransferByCustomer",
			errorMsg:     cctransfer.ErrInvalidTokenAlreadyExists.Error(),
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				items := []core.TransferItem{
					{Token: "CC_1", Amount: new(big.Int).SetInt64(450)},
					{Token: "CC_1", Amount: new(big.Int).SetInt64(450)},
				}
				itemsJSON, err := json.Marshal(items)
				require.NoError(t, err)

				return []string{id, "VT", string(itemsJSON)}
			},
		},
		{
			description:  "channelMultiTransferByCustomer - invalid items count found 101",
			functionName: "channelMultiTransferByCustomer",
			errorMsg:     "invalid argument transfer items count found 101 but expected from 1 to 100",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				items := make([]core.TransferItem, 0, 101)
				for i := 0; i < 101; i++ {
					itemToken := fmt.Sprintf("CC_%d", i)
					items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
				}
				itemsJSON, err := json.Marshal(items)
				require.NoError(t, err)

				return []string{id, "VT", string(itemsJSON)}
			},
		},
		{
			description:  "channelMultiTransferByCustomer - invalid argument token",
			functionName: "channelMultiTransferByCustomer",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				items := make([]core.TransferItem, 0, 100)
				for i := 0; i < 100; i++ {
					n := strconv.Itoa(i)
					itemToken := "CC_" + n
					items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
					userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, n})
					require.NoError(t, err)
					mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1).Bytes()
				}
				itemsJSON, err := json.Marshal(items)
				require.NoError(t, err)

				return []string{id, "VT", string(itemsJSON)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(100).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.Len(t, s.Items, 100)
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
			description:  "channelMultiTransferByAdmin forward - ok",
			functionName: "channelMultiTransferByAdmin",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey1] = new(big.Int).SetUint64(450).Bytes()

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey2] = new(big.Int).SetUint64(900).Bytes()

				return []string{id, "VT", user.AddressBase58Check, string(itemsCCJSON)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
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
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(1350).Bytes(), v)
						j++
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							User:             user.AddressBytes,
							ForwardDirection: true,
							Items:            itemsCCpb,
						}))
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
			description:  "channelMultiTransferByAdmin backward - ok",
			functionName: "channelMultiTransferByAdmin",
			signUser:     issuer,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey1] = new(big.Int).SetUint64(450).Bytes()

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey2] = new(big.Int).SetUint64(900).Bytes()

				return []string{id, "VT", user.AddressBase58Check, string(itemsVTJSON)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
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
					} else if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							User:             user.AddressBytes,
							ForwardDirection: false,
							Items:            itemsVTpb,
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
			description:  "createCCTransferTo single forward - ok",
			functionName: "createCCTransferTo",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				nonceBalanceKey, err := mockStub.CreateCompositeKey(hex.EncodeToString([]byte{core.StateKeyNonce}), []string{"VT", core.CreateTo.String(), user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == nonceBalanceKey {
						n := new(pbfound.Nonce)
						err = pb.Unmarshal(v, n)
						require.NoError(t, err)
						require.Equal(t, uint64(0), n.GetNonce()[0])
						j++
					} else if k == "/transfer/to/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "VT",
							To:               "CC",
							Token:            "VT",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: true,
							IsCommit:         true,
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
			description:  "createCCTransferTo single backward - ok",
			functionName: "createCCTransferTo",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:     id,
					From:   "VT",
					To:     "CC",
					Token:  "CC",
					User:   user.AddressBytes,
					Amount: new(big.Int).SetUint64(450).Bytes(),
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				nonceBalanceKey, err := mockStub.CreateCompositeKey(hex.EncodeToString([]byte{core.StateKeyNonce}), []string{"VT", core.CreateTo.String(), user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == nonceBalanceKey {
						n := new(pbfound.Nonce)
						err = pb.Unmarshal(v, n)
						require.NoError(t, err)
						require.Equal(t, uint64(0), n.GetNonce()[0])
						j++
					} else if k == "/transfer/to/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:       id,
							From:     "VT",
							To:       "CC",
							Token:    "CC",
							User:     user.AddressBytes,
							Amount:   new(big.Int).SetUint64(450).Bytes(),
							IsCommit: true,
						}))
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
			description:  "createCCTransferTo multi forward - ok",
			functionName: "createCCTransferTo",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					User:             user.AddressBytes,
					ForwardDirection: true,
					Items:            itemsVTpb,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
				require.NoError(t, err)

				nonceBalanceKey, err := mockStub.CreateCompositeKey(hex.EncodeToString([]byte{core.StateKeyNonce}), []string{"VT", core.CreateTo.String(), user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
						require.Equal(t, new(big.Int).SetUint64(900).Bytes(), v)
						j++
					} else if k == nonceBalanceKey {
						n := new(pbfound.Nonce)
						err = pb.Unmarshal(v, n)
						require.NoError(t, err)
						require.Equal(t, uint64(0), n.GetNonce()[0])
						j++
					} else if k == "/transfer/to/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "VT",
							To:               "CC",
							User:             user.AddressBytes,
							Items:            itemsVTpb,
							ForwardDirection: true,
							IsCommit:         true,
						}))
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
			description:  "createCCTransferTo multi backward - ok",
			functionName: "createCCTransferTo",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(1350).Bytes()

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:    id,
					From:  "VT",
					To:    "CC",
					User:  user.AddressBytes,
					Items: itemsCCpb,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				nonceBalanceKey, err := mockStub.CreateCompositeKey(hex.EncodeToString([]byte{core.StateKeyNonce}), []string{"VT", core.CreateTo.String(), user.AddressBase58Check})
				require.NoError(t, err)

				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
						require.Equal(t, new(big.Int).SetUint64(900).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
						j++
					} else if k == nonceBalanceKey {
						n := new(pbfound.Nonce)
						err = pb.Unmarshal(v, n)
						require.NoError(t, err)
						require.Equal(t, uint64(0), n.GetNonce()[0])
						j++
					} else if k == "/transfer/to/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:       id,
							From:     "VT",
							To:       "CC",
							User:     user.AddressBytes,
							Items:    itemsCCpb,
							IsCommit: true,
						}))
						j++
					}

					if j == 5 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "createCCTransferTo - the transfer went into the wrong channel",
			functionName: "createCCTransferTo",
			errorMsg:     cctransfer.ErrInvalidChannel.Error(),
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "FIAT",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "createCCTransferTo - incorrect data format",
			functionName: "createCCTransferTo",
			errorMsg:     "invalid character '(' looking for beginning of value",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{"(09345345-0934]"}
			},
		},
		{
			description:  "createCCTransferTo - From and To channels are equal",
			functionName: "createCCTransferTo",
			errorMsg:     cctransfer.ErrInvalidChannel.Error(),
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "createCCTransferTo - token is not equal to one of the channels",
			functionName: "createCCTransferTo",
			errorMsg:     cctransfer.ErrInvalidToken.Error(),
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "FIAT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "createCCTransferTo - misdirection of changes in balances",
			functionName: "createCCTransferTo",
			errorMsg:     cctransfer.ErrInvalidToken.Error(),
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: false,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "createCCTransferTo - the transfer is already in place",
			functionName: "createCCTransferTo",
			errorMsg:     cctransfer.ErrIDTransferExist.Error(),
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				dataTmp, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/to/"+id] = dataTmp

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "createCCTransferTo - trying to create again but already removed",
			functionName: "createCCTransferTo",
			errorMsg:     "nonce 0 already exists",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				nonceBalanceKey, err := mockStub.CreateCompositeKey(hex.EncodeToString([]byte{core.StateKeyNonce}), []string{"VT", core.CreateTo.String(), user.AddressBase58Check})
				require.NoError(t, err)
				n := &pbfound.Nonce{
					Nonce: []uint64{0},
				}
				b, err := pb.Marshal(n)
				mockStub.GetStateCallsMap[nonceBalanceKey] = b

				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)

				return []string{string(data)}
			},
		},
		{
			description:  "commitCCTransferFrom - ok",
			functionName: "commitCCTransferFrom",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == "/transfer/from/"+id {
						s := &pbfound.CCTransfer{}
						err = protojson.Unmarshal(v, s)
						require.NoError(t, err)
						require.True(t, pb.Equal(s, &pbfound.CCTransfer{
							Id:               id,
							From:             "CC",
							To:               "VT",
							Token:            "CC",
							User:             user.AddressBytes,
							Amount:           new(big.Int).SetUint64(450).Bytes(),
							ForwardDirection: true,
							IsCommit:         true,
						}))
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
			description:  "commitCCTransferFrom - transfer not found",
			functionName: "commitCCTransferFrom",
			errorMsg:     cctransfer.ErrNotFound.Error(),
			noBatch:      true,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
			},
		},
		{
			description:  "commitCCTransferFrom - transfer is already committed",
			functionName: "commitCCTransferFrom",
			errorMsg:     cctransfer.ErrTransferCommit.Error(),
			noBatch:      true,
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
		},
		{
			description:  "deleteCCTransferTo - ok",
			functionName: "deleteCCTransferTo",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/to/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				k := mockStub.DelStateArgsForCall(0)
				require.Equal(t, "/transfer/to/"+id, k)
			},
		},
		{
			description:  "deleteCCTransferTo - there's a From but we don't delete it.",
			functionName: "deleteCCTransferTo",
			noBatch:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
		},
		{
			description:  "deleteCCTransferTo - transfer not found",
			functionName: "deleteCCTransferTo",
			noBatch:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
			},
		},
		{
			description:  "deleteCCTransferFrom - ok",
			functionName: "deleteCCTransferFrom",
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				k := mockStub.DelStateArgsForCall(0)
				require.Equal(t, "/transfer/from/"+id, k)
			},
		},
		{
			description:  "deleteCCTransferFrom - the transfer is not committed",
			functionName: "deleteCCTransferFrom",
			noBatch:      true,
			errorMsg:     cctransfer.ErrTransferNotCommit.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
		},
		{
			description:  "deleteCCTransferFrom - there's a To but we don't delete it.",
			functionName: "deleteCCTransferFrom",
			noBatch:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/to/"+id] = data

				return []string{id}
			},
		},
		{
			description:  "deleteCCTransferFrom - transfer not found",
			functionName: "deleteCCTransferFrom",
			noBatch:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
			},
		},
		{
			description:  "channelTransferFrom - ok",
			functionName: "channelTransferFrom",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				s := &pbfound.CCTransfer{}
				err = protojson.Unmarshal(payload, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				}))
			},
		},
		{
			description:  "channelTransferFrom - transfer not found",
			functionName: "channelTransferFrom",
			isQuery:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
			},
		},
		{
			description:  "channelTransferTo - ok",
			functionName: "channelTransferTo",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := protojson.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/to/"+id] = data

				return []string{id}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				s := &pbfound.CCTransfer{}
				err = protojson.Unmarshal(payload, s)
				require.NoError(t, err)
				require.True(t, pb.Equal(s, &pbfound.CCTransfer{
					Id:               id,
					From:             "VT",
					To:               "CC",
					Token:            "VT",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				}))
			},
		},
		{
			description:  "channelTransferTo - transfer not found",
			functionName: "channelTransferTo",
			isQuery:      true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			codeResp:     int32(shim.ERROR),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
			},
		},
		{
			description:  "cancelCCTransferFrom single forward - ok",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.DelStateCallCount(); i++ {
					k := mockStub.DelStateArgsForCall(i)
					if k == "/transfer/from/"+id {
						j++
					}

					if j == 1 {
						break
					}
				}
				if j != 1 {
					require.Fail(t, "not found checking data")
				}

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				j = 0
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
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
			description:  "cancelCCTransferFrom single backward - ok",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:     id,
					From:   "CC",
					To:     "VT",
					Token:  "VT",
					User:   user.AddressBytes,
					Amount: new(big.Int).SetUint64(450).Bytes(),
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.DelStateCallCount(); i++ {
					k := mockStub.DelStateArgsForCall(i)
					if k == "/transfer/from/"+id {
						j++
					}

					if j == 1 {
						break
					}
				}
				if j != 1 {
					require.Fail(t, "not found checking data")
				}

				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT"})
				require.NoError(t, err)

				j = 0
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
			description:  "cancelCCTransferFrom multi forward - ok",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(1350).Bytes()

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					User:             user.AddressBytes,
					ForwardDirection: true,
					Items:            itemsCCpb,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.DelStateCallCount(); i++ {
					k := mockStub.DelStateArgsForCall(i)
					if k == "/transfer/from/"+id {
						j++
					}

					if j == 1 {
						break
					}
				}
				if j != 1 {
					require.Fail(t, "not found checking data")
				}

				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check, "2"})
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)

				j = 0
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
						require.Equal(t, new(big.Int).SetUint64(900).Bytes(), v)
						j++
					} else if k == givenBalanceKey {
						require.Equal(t, new(big.Int).SetUint64(0).Bytes(), v)
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
			description:  "cancelCCTransferFrom multi backward - ok",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:    id,
					From:  "CC",
					To:    "VT",
					User:  user.AddressBytes,
					Items: itemsVTpb,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.DelStateCallCount(); i++ {
					k := mockStub.DelStateArgsForCall(i)
					if k == "/transfer/from/"+id {
						j++
					}

					if j == 1 {
						break
					}
				}
				if j != 1 {
					require.Fail(t, "not found checking data")
				}

				userBalanceKey1, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_1"})
				require.NoError(t, err)

				userBalanceKey2, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "VT_2"})
				require.NoError(t, err)

				j = 0
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if k == userBalanceKey1 {
						require.Equal(t, new(big.Int).SetUint64(450).Bytes(), v)
						j++
					} else if k == userBalanceKey2 {
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
			description:  "cancelCCTransferFrom - transfer completed",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			errorMsg:     cctransfer.ErrTransferCommit.Error(),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				givenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeGiven.String(), []string{"VT"})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[givenBalanceKey] = new(big.Int).SetUint64(450).Bytes()

				data, err := json.Marshal(&pbfound.CCTransfer{
					Id:               id,
					From:             "CC",
					To:               "VT",
					Token:            "CC",
					User:             user.AddressBytes,
					Amount:           new(big.Int).SetUint64(450).Bytes(),
					ForwardDirection: true,
					IsCommit:         true,
				})
				require.NoError(t, err)
				mockStub.GetStateCallsMap["/transfer/from/"+id] = data

				return []string{id}
			},
		},
		{
			description:  "cancelCCTransferFrom - transfer not found",
			functionName: "cancelCCTransferFrom",
			noSign:       true,
			errorMsg:     cctransfer.ErrNotFound.Error(),
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				err = mocks.SetCreator(mockStub.ChaincodeStub, mocks.BatchRobotCert)
				require.NoError(t, err)

				return []string{id}
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
			} else if testCase.noSign {
				txId, resp = mockStub.TxInvokeChaincode(cc, testCase.functionName, parameters...)
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
			if string(resp.GetPayload()) != "null" {
				err = pb.Unmarshal(resp.GetPayload(), bResp)
				require.NoError(t, err)
			}

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

func TestQueryAllTransfersFrom(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	id := uuid.NewString()

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
	meta1 := &peer.QueryResponseMetadata{
		FetchedRecordsCount: 2,
		Bookmark:            "/transfer/from/" + id,
	}
	meta2 := &peer.QueryResponseMetadata{
		FetchedRecordsCount: 1,
	}
	mockIterator := &mocks.StateIterator{}
	mockStub.GetStateByRangeWithPaginationReturnsOnCall(0, mockIterator, meta1, nil)
	mockStub.GetStateByRangeWithPaginationReturnsOnCall(1, mockIterator, meta1, nil)
	mockStub.GetStateByRangeWithPaginationReturnsOnCall(2, mockIterator, meta2, nil)
	mockIterator.HasNextReturnsOnCall(0, true)
	mockIterator.HasNextReturnsOnCall(1, true)
	mockIterator.HasNextReturnsOnCall(2, false)
	mockIterator.HasNextReturnsOnCall(3, true)
	mockIterator.HasNextReturnsOnCall(4, true)
	mockIterator.HasNextReturnsOnCall(5, false)
	mockIterator.HasNextReturnsOnCall(6, true)
	mockIterator.HasNextReturnsOnCall(7, false)

	cct := &pbfound.CCTransfer{
		Id:               id,
		From:             "CC",
		To:               "VT",
		Token:            "CC",
		User:             user.AddressBytes,
		Amount:           new(big.Int).SetUint64(450).Bytes(),
		ForwardDirection: true,
	}
	bCct, err := pb.Marshal(cct)
	require.NoError(t, err)

	mockIterator.NextReturns(&queryresult.KV{
		Key:   "/transfer/from/" + id,
		Value: bCct,
	}, nil)

	cc, err := core.NewCC(&CustomToken{})
	require.NoError(t, err)

	var resp peer.Response

	b := ""
	for {
		resp = mockStub.QueryChaincode(cc, "channelTransfersFrom", "2", b)
		require.Equal(t, resp.GetStatus(), int32(shim.OK))
		require.Empty(t, resp.GetMessage())
		require.NotEmpty(t, resp.GetPayload())

		res := new(pbfound.CCTransfers)
		err = json.Unmarshal(resp.GetPayload(), res)
		require.NoError(t, err)
		for _, c := range res.GetCcts() {
			require.True(t, pb.Equal(cct, c))
		}
		if res.Bookmark == "" {
			break
		}
		b = res.Bookmark
	}

	resp = mockStub.QueryChaincode(cc, "channelTransfersFrom", "2", "pfi/transfer/from/"+id)
	require.Equal(t, resp.GetStatus(), int32(shim.ERROR))
	require.Empty(t, resp.GetPayload())
	require.EqualError(t, cctransfer.ErrInvalidBookmark, resp.GetMessage())

	resp = mockStub.QueryChaincode(cc, "channelTransfersFrom", "-2", "/transfer/from/"+id)
	require.Equal(t, resp.GetStatus(), int32(shim.ERROR))
	require.Empty(t, resp.GetPayload())
	require.EqualError(t, cctransfer.ErrPageSizeLessOrEqZero, resp.GetMessage())
}
