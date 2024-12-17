package unit

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

func TestExternalLocks(t *testing.T) {
	testCollection := []struct {
		name         string
		functionName string
		invokeFunc   func(
			mockStub *mockstub.MockStub,
			cc *core.Chaincode,
			functionName string,
			params string,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
		) (string, peer.Response)
		checkResponseFunc   func(t *testing.T, resp peer.Response)
		prepareMockStubFunc func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			cc *core.Chaincode,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
		) string
		checkResultFunc func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string)
	}{
		{
			name:         "external token lock test",
			functionName: "lockTokenBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test1",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) {
				bal, err := mockStub.GetTokenBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(400), bal)

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				externalLockedInfo, err := mockStub.GetTokenExternalLockedInfo(txID)
				require.NoError(t, err)
				require.Equal(t, "600", externalLockedInfo.InitAmount)
			},
		},
		{
			name:         "external allowed lock test",
			functionName: "lockAllowedBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test2",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) {
				bal, err := mockStub.GetAllowedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(400), bal)

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				externalLockedInfo, err := mockStub.GetAllowedExternalLockedInfo(txID)
				require.NoError(t, err)
				require.Equal(t, "600", externalLockedInfo.InitAmount)
			},
		},
		{
			name:         "[negative] wrong user token lock test",
			functionName: "lockTokenBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test3",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrUnauthorisedNotAdmin.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] wrong user allowed lock test",
			functionName: "lockAllowedBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test4",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrUnauthorisedNotAdmin.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] token lock more than added test",
			functionName: "lockTokenBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "1100",
					Reason:  "test5",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, "insufficient balance", payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] allowed lock more than added test",
			functionName: "lockAllowedBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "1100",
					Reason:  "test6",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, "insufficient balance", payload.TxResponses[0].Error.Error)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			user, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", issuer.AddressBase58Check, nil)

			mockStub.SetConfig(config)

			cc, err := core.NewCC(&CustomToken{})
			require.NoError(t, err)

			// prepare mock stub
			params := test.prepareMockStubFunc(t, mockStub, cc, issuer, user)

			// invoking chaincode
			txID, resp := test.invokeFunc(mockStub, cc, test.functionName, params, issuer, user)

			// checking result
			require.Equal(t, int32(http.StatusOK), resp.Status)
			require.Empty(t, resp.Message)
			if test.checkResponseFunc != nil {
				test.checkResponseFunc(t, resp)
			}
			if test.checkResultFunc != nil {
				test.checkResultFunc(t, mockStub, user, txID)
			}
		})
	}
}

func TestExternalUnlocks(t *testing.T) {
	testCollection := []struct {
		name         string
		functionName string
		invokeFunc   func(
			mockStub *mockstub.MockStub,
			cc *core.Chaincode,
			functionName string,
			params string,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
		) (string, peer.Response)
		lockBalanceFunc func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			cc *core.Chaincode,
			issuer *mocks.UserFoundation,
			user *mocks.UserFoundation,
		) string
		checkResponseFunc   func(t *testing.T, resp peer.Response)
		prepareMockStubFunc func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user *mocks.UserFoundation,
			txID string,
		) string
		checkResultFunc func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string)
	}{
		{
			name:         "external token unlock test",
			functionName: "unlockTokenBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test1",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockTokenBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "150",
					Reason:  "test1",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) {
				bal, err := mockStub.GetTokenBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(550), bal)

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(450), lockedBalance)

				externalLockedInfo, err := mockStub.GetTokenExternalLockedInfo(txID)
				require.NoError(t, err)
				require.Equal(t, "600", externalLockedInfo.InitAmount)
				require.Equal(t, "450", externalLockedInfo.CurrentAmount)
			},
		},
		{
			name:         "external allowed lock unlock test",
			functionName: "unlockAllowedBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test2",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockAllowedBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "150",
					Reason:  "test2",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResultFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) {
				bal, err := mockStub.GetAllowedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(550), bal)

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(450), lockedBalance)

				externalLockedInfo, err := mockStub.GetAllowedExternalLockedInfo(txID)
				require.NoError(t, err)
				require.Equal(t, "600", externalLockedInfo.InitAmount)
				require.Equal(t, "450", externalLockedInfo.CurrentAmount)
			},
		},
		{
			name:         "[negative] wrong user token unlock test",
			functionName: "unlockTokenBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test3",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockTokenBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "150",
					Reason:  "test3",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrUnauthorisedNotAdmin.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] wrong user allowed unlock test",
			functionName: "unlockAllowedBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test4",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockAllowedBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "150",
					Reason:  "test4",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrUnauthorisedNotAdmin.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] token locking twice test",
			functionName: "lockTokenBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test5",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockTokenBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test5",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrAlreadyExist.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] allowed locking twice test",
			functionName: "lockAllowedBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test6",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockAllowedBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test6",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, core.ErrAlreadyExist.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] token unlock negative test",
			functionName: "unlockTokenBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddTokenBalance(user, big.NewInt(1000))
				require.NoError(t, err)
				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "600",
					Reason:  "test7",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockTokenBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetTokenLockedBalance(user)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "cc",
					Amount:  "-100",
					Reason:  "test7",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, balance.ErrAmountMustBeNonNegative.Error(), payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] allowed unlock negative test",
			functionName: "unlockAllowedBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				err := mockStub.AddAllowedBalance(user, "vk", big.NewInt(1000))
				require.NoError(t, err)

				request := &proto.BalanceLockRequest{
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "600",
					Reason:  "test8",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				txID, resp := mockStub.TxInvokeChaincodeSigned(cc, "lockAllowedBalance", issuer, "", "", "", string(data))
				require.Equal(t, int32(http.StatusOK), resp.GetStatus())

				lockedBalance, err := mockStub.GetAllowedLockedBalance(user, "vk")
				require.NoError(t, err)
				require.Equal(t, big.NewInt(600), lockedBalance)

				return txID
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, user *mocks.UserFoundation, txID string) string {
				request := &proto.BalanceLockRequest{
					Id:      txID,
					Address: user.AddressBase58Check,
					Token:   "vk",
					Amount:  "-100",
					Reason:  "test8",
					Docs:    nil,
					Payload: nil,
				}

				data, err := json.Marshal(request)
				require.NoError(t, err)

				return string(data)
			},
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName, params string, issuer, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			checkResponseFunc: func(t *testing.T, resp peer.Response) {
				payload := &proto.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, balance.ErrAmountMustBeNonNegative.Error(), payload.TxResponses[0].Error.Error)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			user, err := mocks.NewUserFoundation(proto.KeyType_ed25519)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", issuer.AddressBase58Check, nil)

			mockStub.SetConfig(config)

			cc, err := core.NewCC(&CustomToken{})
			require.NoError(t, err)

			// locking balance
			txID := test.lockBalanceFunc(t, mockStub, cc, issuer, user)

			// prepare mock stub
			params := test.prepareMockStubFunc(t, mockStub, user, txID)

			// unlocking balance
			txID, resp := test.invokeFunc(mockStub, cc, test.functionName, params, issuer, user)
			require.Equal(t, int32(http.StatusOK), resp.Status)
			require.Empty(t, resp.Message)
			if test.checkResponseFunc != nil {
				test.checkResponseFunc(t, resp)
			}
			if test.checkResultFunc != nil {
				test.checkResultFunc(t, mockStub, user, txID)
			}
		})
	}
}
