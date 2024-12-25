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
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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
				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == tokenBalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(400), bal)
					}

					if putStateKey == tokenBalanceLockedKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(600), bal)
					}

					if putStateKey == tokenBalanceLockedInfoKey {
						info := &proto.TokenBalanceLock{}
						err = json.Unmarshal(value, info)
						require.NoError(t, err)
						require.Equal(t, "cc", info.Token)
						require.Equal(t, "600", info.InitAmount)
					}
				}
			},
		},
		{
			name:         "external allowed lock test",
			functionName: "lockAllowedBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, issuer, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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
				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)

					if putStateKey == allowedBalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(400), bal)
					}

					if putStateKey == allowedBalanceLockedKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(600), bal)
					}

					if putStateKey == allowedBalanceLockedInfoKey {
						result := &proto.AllowedBalanceLock{}
						err = json.Unmarshal(value, result)
						require.NoError(t, err)
						require.Equal(t, "vk", result.Token)
						require.Equal(t, "600", result.InitAmount)
					}
				}
			},
		},
		{
			name:         "[negative] wrong user token lock test",
			functionName: "lockTokenBalance",
			invokeFunc: func(mockStub *mockstub.MockStub, cc *core.Chaincode, functionName string, params string, issuer *mocks.UserFoundation, user *mocks.UserFoundation) (string, peer.Response) {
				return mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", params)
			},
			prepareMockStubFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer, user *mocks.UserFoundation) string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

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

			mockStub.CreateAndSetConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", issuer.AddressBase58Check, nil)

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
				txID := mockStub.GetTxID()

				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[tokenBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.TokenBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "cc",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test1",
				}

				rawInfo, err := json.Marshal(info)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceLockedInfoKey] = rawInfo

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
				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)

					if putStateKey == tokenBalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(550), bal)
					}

					if putStateKey == tokenBalanceLockedKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(450), bal)
					}

					if putStateKey == tokenBalanceLockedInfoKey {
						info := &proto.TokenBalanceLock{}
						err := json.Unmarshal(value, info)
						require.NoError(t, err)
						require.Equal(t, "cc", info.Token)
						require.Equal(t, "600", info.InitAmount)
						require.Equal(t, "450", info.CurrentAmount)
					}
				}
			},
		},
		{
			name:         "external allowed lock unlock test",
			functionName: "unlockAllowedBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				txID := mockStub.GetTxID()

				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[allowedBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[allowedBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.AllowedBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "vk",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test2",
				}
				rawInfo, err := json.Marshal(info)

				mockStub.GetStateCallsMap[allowedBalanceLockedInfoKey] = rawInfo

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
				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == allowedBalanceKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(550), bal)
					}

					if putStateKey == allowedBalanceLockedKey {
						bal := new(big.Int).SetBytes(value)
						require.Equal(t, big.NewInt(450), bal)
					}

					if putStateKey == allowedBalanceLockedInfoKey {
						info := &proto.AllowedBalanceLock{}
						err := json.Unmarshal(value, info)
						require.NoError(t, err)
						require.Equal(t, "vk", info.Token)
						require.Equal(t, "600", info.InitAmount)
						require.Equal(t, "450", info.CurrentAmount)
					}
				}
			},
		},
		{
			name:         "[negative] wrong user token unlock test",
			functionName: "unlockTokenBalance",
			lockBalanceFunc: func(t *testing.T, mockStub *mockstub.MockStub, cc *core.Chaincode, issuer *mocks.UserFoundation, user *mocks.UserFoundation) string {
				txID := mockStub.GetTxID()

				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[tokenBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.TokenBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "cc",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test3",
				}

				rawInfo, err := json.Marshal(info)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceLockedInfoKey] = rawInfo

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
				txID := mockStub.GetTxID()

				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[allowedBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[allowedBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.AllowedBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "vk",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test4",
				}
				rawInfo, err := json.Marshal(info)

				mockStub.GetStateCallsMap[allowedBalanceLockedInfoKey] = rawInfo

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
				txID := mockStub.GetTxID()

				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[tokenBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.TokenBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "cc",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test5",
				}

				rawInfo, err := json.Marshal(info)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceLockedInfoKey] = rawInfo

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
				txID := mockStub.GetTxID()

				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[allowedBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[allowedBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.AllowedBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "vk",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test6",
				}
				rawInfo, err := json.Marshal(info)

				mockStub.GetStateCallsMap[allowedBalanceLockedInfoKey] = rawInfo

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
				txID := mockStub.GetTxID()

				tokenBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				tokenBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[tokenBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.TokenBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "cc",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test7",
				}

				rawInfo, err := json.Marshal(info)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[tokenBalanceLockedInfoKey] = rawInfo

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
				txID := mockStub.GetTxID()

				allowedBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedLocked.String(), []string{user.AddressBase58Check, "vk"})
				require.NoError(t, err)

				allowedBalanceLockedInfoKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{txID})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[allowedBalanceKey] = big.NewInt(400).Bytes()
				mockStub.GetStateCallsMap[allowedBalanceLockedKey] = big.NewInt(600).Bytes()

				info := &proto.AllowedBalanceLock{
					Id:            txID,
					Address:       user.AddressBase58Check,
					Token:         "vk",
					InitAmount:    "600",
					CurrentAmount: "600",
					Reason:        "test8",
				}
				rawInfo, err := json.Marshal(info)

				mockStub.GetStateCallsMap[allowedBalanceLockedInfoKey] = rawInfo

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

			mockStub.CreateAndSetConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", issuer.AddressBase58Check, nil)

			cc, err := core.NewCC(&CustomToken{})
			require.NoError(t, err)

			// locking balance
			txID := test.lockBalanceFunc(t, mockStub, cc, issuer, user)

			// prepare mock stub
			params := test.prepareMockStubFunc(t, mockStub, user, txID)

			// invoking method
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
