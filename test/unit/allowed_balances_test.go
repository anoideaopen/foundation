package unit

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"
)

func (tt *TestToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceAdd(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceSub(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceLock(token, address, amount)
}

func (tt *TestToken) QueryAllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	return tt.AllowedBalanceUnLock(token, address, amount)
}

func (tt *TestToken) QueryAllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceTransferLocked(token, from, to, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	return tt.AllowedBalanceBurnLocked(token, address, amount, reason)
}

func (tt *TestToken) QueryAllowedBalanceGetAll(address *types.Address) (map[string]string, error) {
	return tt.AllowedBalanceGetAll(address)
}

// Checking query stub does not put any record into the state
func TestQuery(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                      string
		needACLAccess             bool
		functionName              string
		resultMessage             string
		preparePayloadEqual       func(t *testing.T) []byte
		prepareFunctionParameters func(user1, user2 *mocks.UserFoundation) []string
		prepareMockStubAdditional func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation)
	}{
		{
			name:          "Query allowed balance add",
			needACLAccess: true,
			functionName:  "allowedBalanceAdd",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
		},
		{
			name:          "Query allowed balance sub",
			needACLAccess: true,
			functionName:  "allowedBalanceSub",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, []byte("1000"), nil)
			},
		},
		{
			name:          "Query allowed balance lock",
			needACLAccess: true,
			functionName:  "allowedBalanceLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, []byte("1000"), nil)
			},
		},
		{
			name:          "Query allowed balance unlock",
			needACLAccess: true,
			functionName:  "allowedBalanceUnLock",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, []byte("1000"), nil)
			},
		},
		{
			name:          "Query allowed transfer locked",
			needACLAccess: true,
			functionName:  "allowedBalanceTransferLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, user2.AddressBase58Check, "100", "reason"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mocks.ACLGetAccountInfo(t, mockStub, 1)
				mockStub.GetStateReturnsOnCall(1, []byte("1000"), nil)
			},
		},
		{
			name:          "Query allowed balance burn locked",
			needACLAccess: true,
			functionName:  "allowedBalanceBurnLocked",
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{"VT", user1.AddressBase58Check, "100", "reason"}
			},
			resultMessage: "",
			preparePayloadEqual: func(t *testing.T) []byte {
				return []byte("null")
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mockStub.GetStateReturnsOnCall(1, []byte("1000"), nil)
			},
		},
		{
			name:          "Query allowed balances get all",
			functionName:  "allowedBalanceGetAll",
			needACLAccess: true,
			prepareFunctionParameters: func(user1, user2 *mocks.UserFoundation) []string {
				return []string{user1.AddressBase58Check}
			},
			preparePayloadEqual: func(t *testing.T) []byte {
				balances := map[string]string{"vt": "100", "fiat": "200"}
				rawBalances, err := json.Marshal(balances)
				require.NoError(t, err)

				return rawBalances
			},
			prepareMockStubAdditional: func(t *testing.T, mockStub *mocks.ChaincodeStub, owner, user *mocks.UserFoundation) {
				mockIterator := &mocks.StateIterator{}
				mockIterator.HasNextReturnsOnCall(0, false)
				mockStub.GetStateByPartialCompositeKeyReturns(mockIterator, nil)

				key1, err := shim.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "vt"})
				require.NoError(t, err)

				key2, err := shim.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "fiat"})
				require.NoError(t, err)

				mockIterator.HasNextReturnsOnCall(0, true)
				mockIterator.HasNextReturnsOnCall(1, true)
				mockIterator.HasNextReturnsOnCall(2, false)

				mockIterator.NextReturnsOnCall(0, &queryresult.KV{
					Key:   key1,
					Value: big.NewInt(100).Bytes(),
				}, nil)
				mockIterator.NextReturnsOnCall(1, &queryresult.KV{
					Key:   key2,
					Value: big.NewInt(200).Bytes(),
				}, nil)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mocks.NewMockStub(t)

			issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_secp256k1)
			require.NoError(t, err)

			config := makeBaseTokenConfig("CC Token", "CC", 8,
				issuer.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(&TestToken{})
			require.NoError(t, err)

			// preparing stub
			mockStub.GetStateReturnsOnCall(0, []byte(config), nil)

			if test.needACLAccess {
				mocks.ACLGetAccountInfo(t, mockStub, 0)
			}

			if test.prepareMockStubAdditional != nil {
				test.prepareMockStubAdditional(t, mockStub, issuer, user1)
			}

			mockStub.GetFunctionAndParametersReturns(test.functionName, test.prepareFunctionParameters(user1, user2))

			// invoking chaincode
			resp := cc.Invoke(mockStub)
			if test.resultMessage != "" {
				require.Equal(t, test.resultMessage, resp.GetMessage())
			} else {
				require.Empty(t, resp.GetMessage())
				require.Equal(t, test.preparePayloadEqual(t), resp.GetPayload())
				require.Equal(t, 0, mockStub.PutStateCallCount())
			}
		})
	}
}
