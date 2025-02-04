package unit

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
)

func (tt *TestToken) TxIndustrialBalanceAdd(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceAdd(token, address, amount, reason)
}

func (tt *TestToken) QueryIndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	return tt.IndustrialBalanceGet(address)
}

func (tt *TestToken) TxIndustrialBalanceSub(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceSub(token, address, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceTransfer(_ *types.Sender, token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceTransfer(token, from, to, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceLock(_ *types.Sender, token string, address *types.Address, amount *big.Int) error {
	return tt.IndustrialBalanceLock(token, address, amount)
}

func (tt *TestToken) QueryIndustrialBalanceGetLocked(address *types.Address) (map[string]string, error) {
	return tt.IndustrialBalanceGetLocked(address)
}

func (tt *TestToken) TxIndustrialBalanceUnLock(_ *types.Sender, token string, address *types.Address, amount *big.Int) error {
	return tt.IndustrialBalanceUnLock(token, address, amount)
}

func (tt *TestToken) TxIndustrialBalanceTransferLocked(_ *types.Sender, token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *TestToken) TxIndustrialBalanceBurnLocked(_ *types.Sender, token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.IndustrialBalanceBurnLocked(token, address, amount, reason)
}

// TestIndustrialBalances - industrial balances test
func TestIndustrialBalances(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string
		funcInvoke          func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response
		funcCheckResult     func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response)
	}{
		{
			name:         "industrial balances get",
			functionName: "industrialBalanceGet",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				mockIterator := &mocks.StateIterator{}
				mockStub.GetStateByPartialCompositeKeyReturns(mockIterator, nil)

				mockIterator.HasNextReturnsOnCall(0, true)
				mockIterator.HasNextReturnsOnCall(1, false)

				mockIterator.NextReturnsOnCall(0, &queryresult.KV{
					Key:   key,
					Value: big.NewInt(1000).Bytes(),
				}, nil)

				return []string{user1.AddressBase58Check}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				return mockStub.QueryChaincode(cc, functionName, parameters...)
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				payload := map[string]string{}

				err := json.Unmarshal(resp.Payload, &payload)
				require.NoError(t, err)

				value, ok := payload[testGroup]
				require.True(t, ok)

				require.Equal(t, "1000", value)
			},
		},
		{
			name:         "industrial balances add",
			functionName: "industrialBalanceAdd",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, "100", "add balance"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == key {
						require.Equal(t, big.NewInt(1100), new(big.Int).SetBytes(value))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "industrial balances sub",
			functionName: "industrialBalanceSub",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, "100", "sub balance"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == key {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "industrial balances lock",
			functionName: "industrialBalanceLock",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, "100"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				keyLocked, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				keyUnlocked, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checkedLocked := false
				checkedUnlocked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyLocked {
						require.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
						checkedLocked = true
					}
					if putStateKey == keyUnlocked {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checkedUnlocked = true
					}
				}
				require.True(t, checkedLocked && checkedUnlocked)
			},
		},
		{
			name:         "industrial balance get locked",
			functionName: "industrialBalanceGetLocked",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				mockIterator := &mocks.StateIterator{}
				mockStub.GetStateByPartialCompositeKeyReturns(mockIterator, nil)

				mockIterator.HasNextReturnsOnCall(0, true)
				mockIterator.HasNextReturnsOnCall(1, false)

				mockIterator.NextReturnsOnCall(0, &queryresult.KV{
					Key:   key,
					Value: big.NewInt(1000).Bytes(),
				}, nil)

				return []string{user1.AddressBase58Check}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				return mockStub.QueryChaincode(cc, functionName, parameters...)
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				payload := map[string]string{}

				err := json.Unmarshal(resp.Payload, &payload)
				require.NoError(t, err)

				value, ok := payload[testGroup]
				require.True(t, ok)

				require.Equal(t, "1000", value)
			},
		},
		{
			name:         "industrial balances unlock",
			functionName: "industrialBalanceUnLock",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, "100"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				keyLocked, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				keyUnlocked, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checkedLocked := false
				checkedUnlocked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyLocked {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checkedLocked = true
					}
					if putStateKey == keyUnlocked {
						require.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
						checkedUnlocked = true
					}
				}
				require.True(t, checkedLocked && checkedUnlocked)
			},
		},
		{
			name:         "industrial balances transfer",
			functionName: "industrialBalanceTransfer",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, user2.AddressBase58Check, "100", "transfer balance"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				keyBalanceUser1, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				keyBalanceUser2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checkedUser1 := false
				checkedUser2 := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyBalanceUser1 {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checkedUser1 = true
					}
					if putStateKey == keyBalanceUser2 {
						require.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
						checkedUser2 = true
					}
				}
				require.True(t, checkedUser1 && checkedUser2)
			},
		},
		{
			name:         "industrial balances transfer locked",
			functionName: "industrialBalanceTransferLocked",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, user2.AddressBase58Check, "100", "transfer balance"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				keyBalanceUser1, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				keyBalanceUser2, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checkedUser1 := false
				checkedUser2 := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyBalanceUser1 {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checkedUser1 = true
					}
					if putStateKey == keyBalanceUser2 {
						require.Equal(t, big.NewInt(100), new(big.Int).SetBytes(value))
						checkedUser2 = true
					}
				}
				require.True(t, checkedUser1 && checkedUser2)
			},
		},
		{
			name:         "industrial balances burn locked",
			functionName: "industrialBalanceBurnLocked",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation) []string {
				key, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[key] = big.NewInt(1000).Bytes()

				return []string{testTokenWithGroup, user1.AddressBase58Check, "100", "transfer balance"}
			},
			funcInvoke: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, signer *mocks.UserFoundation, parameters ...string) *peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, signer, "", "", "", parameters...)
				return resp
			},
			funcCheckResult: func(t *testing.T, mockStub *mockstub.MockStub, user1, user2 *mocks.UserFoundation, resp *peer.Response) {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeTokenLocked.String(), []string{user1.AddressBase58Check, testGroup})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyBalance {
						require.Equal(t, big.NewInt(900), new(big.Int).SetBytes(value))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig(testTokenWithGroup, testTokenSymbol, 8,
				owner.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(&TestToken{})
			require.NoError(t, err)

			parameters := test.funcPrepareMockStub(t, mockStub, user1, user2)

			resp := test.funcInvoke(cc, mockStub, test.functionName, owner, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())

			test.funcCheckResult(t, mockStub, user1, user2, resp)
		})
	}
}
