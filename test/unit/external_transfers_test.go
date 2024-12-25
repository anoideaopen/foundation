package unit

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExternalTransfers(t *testing.T) {
	testCollection := []struct {
		name             string
		amountToTransfer string
		invokeFunc       func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response
		checkResultFunc  func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string)
	}{
		{
			name:             "happy path",
			amountToTransfer: "600",
			invokeFunc: func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, "transferBalance", owner, "", "", "", parameters...)
				return resp
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string) {
				balance1Checked := false
				balance2Checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)

					if putStateKey == user1BalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, 0, bal.Cmp(big.NewInt(400)))
						balance1Checked = true
					}

					if putStateKey == user2BalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, 0, bal.Cmp(big.NewInt(1100)))
						balance2Checked = true
					}
				}
				require.True(t, balance1Checked && balance2Checked)
			},
		},
		{
			name:             "[negative] unauthorized invoke",
			amountToTransfer: "600",
			invokeFunc: func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, "transferBalance", user, "", "", "", parameters...)
				return resp
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrUnauthorisedNotAdmin.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:             "[negative] insufficient funds",
			amountToTransfer: "1100",
			invokeFunc: func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, "transferBalance", owner, "", "", "", parameters...)
				return resp
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, balance.ErrInsufficientBalance.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:             "[negative] negative amount transfer",
			amountToTransfer: "-100",
			invokeFunc: func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, "transferBalance", owner, "", "", "", parameters...)
				return resp
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrAmountMustBeGreaterThanZero.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:             "[negative] zero amount transfer",
			amountToTransfer: "0",
			invokeFunc: func(cc *core.Chaincode, mockStub *mockstub.MockStub, owner, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, "transferBalance", owner, "", "", "", parameters...)
				return resp
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, resp peer.Response, user1BalanceKey, user2BalanceKey string) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrAmountMustBeGreaterThanZero.Error(), payload.TxResponses[0].Error.Error)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig("CC Token", "CC", 8,
				owner.AddressBase58Check, "", "", owner.AddressBase58Check, nil)

			cc, err := core.NewCC(&CustomToken{})
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			// adding balances
			balanceUser1Key, err := mockStub.CreateCompositeKeyStub(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
			require.NoError(t, err)

			mockStub.GetStateCallsMap[balanceUser1Key] = big.NewInt(1000).Bytes()

			balanceUser2Key, err := mockStub.CreateCompositeKeyStub(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check})
			require.NoError(t, err)

			mockStub.GetStateCallsMap[balanceUser2Key] = big.NewInt(500).Bytes()

			// preparing transfer request
			transferRequest := &proto.TransferRequest{
				RequestId:       "",
				Basis:           proto.TransferBasis_TRANSFER_BASIS_INHERITANCE,
				AdministratorId: owner.AddressBase58Check,
				DocumentType:    proto.DocumentType_DOCUMENT_TYPE_INHERITANCE,
				DocumentNumber:  "1",
				DocumentDate:    timestamppb.New(time.Now()),
				DocumentHashes: []string{
					"hash1", "hash2",
				},
				FromAddress:    user1.AddressBase58Check,
				ToAddress:      user2.AddressBase58Check,
				Token:          "",
				Amount:         test.amountToTransfer,
				Reason:         "test transfer",
				BalanceType:    proto.BalanceType_BALANCE_TYPE_TOKEN,
				AdditionalInfo: nil,
			}

			data, err := json.Marshal(transferRequest)
			require.NoError(t, err)

			resp := test.invokeFunc(cc, mockStub, owner, user1, []string{string(data)}...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())

			test.checkResultFunc(t, mockStub, resp, balanceUser1Key, balanceUser2Key)
		})
	}
}
