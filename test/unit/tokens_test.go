package unit

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
	pb "google.golang.org/protobuf/proto"
)

type metadata struct {
	Fee struct {
		Address  string
		Currency string   `json:"currency"`
		Fee      *big.Int `json:"fee"`
		Floor    *big.Int `json:"floor"`
		Cap      *big.Int `json:"cap"`
	} `json:"fee"`
	Rates []metadataRate `json:"rates"`
}

type metadataRate struct {
	DealType string   `json:"deal_type"` //nolint:tagliatelle
	Currency string   `json:"currency"`
	Rate     *big.Int `json:"rate"`
	Min      *big.Int `json:"min"`
	Max      *big.Int `json:"max"`
}

// FiatToken - base struct
type FiatTestToken struct {
	token.BaseToken
}

// NewFiatToken creates fiat token
func NewFiatTestToken(bt token.BaseToken) *FiatTestToken {
	return &FiatTestToken{bt}
}

// TxEmit - emits fiat token
func (ft *FiatTestToken) TxEmit(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := ft.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return ft.EmissionAdd(amount)
}

// TxEmit - emits fiat token
func (ft *FiatTestToken) TxEmitIndustrial(sender *types.Sender, address *types.Address, amount *big.Int, token string) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	return ft.IndustrialBalanceAdd(token, address, amount, "txEmitIndustrial")
}

func (ft *FiatTestToken) TxAccountsTest(_ *types.Sender, addr string, pub string) error {
	args := make([][]byte, 0)
	args = append(args, []byte("getAccountsInfo"))
	for i := 0; i < 2; i++ {
		bytes, _ := json.Marshal([]string{"getAccountInfo", addr})
		args = append(args, bytes)
	}

	for i := 0; i < 5; i++ {
		bytes, _ := json.Marshal([]string{"checkKeys", pub})
		args = append(args, bytes)
	}

	for i := 0; i < 3; i++ {
		bytes, _ := json.Marshal([]string{"getAccountInfo", addr})
		args = append(args, bytes)
	}

	stub := ft.GetStub()

	_ = stub.InvokeChaincode("acl", args, "acl")

	return nil
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
// WARNING: DO NOT USE CODE LIKE THIS IN REAL TOKENS AS `map[string]string` IS NOT ORDERED
// AND WILL CAUSE ENDORSEMENT MISMATCH ON PEERS. THIS IS FOR TESTING PURPOSES ONLY.
// NOTE: THIS APPROACH IS USED DUE TO LEGACY CODE IN THE FOUNDATION LIBRARY.
// IMPLEMENTING A PROPER SOLUTION WOULD REQUIRE SIGNIFICANT CHANGES.
func (ft *FiatTestToken) QueryIndustrialBalanceOf(address *types.Address) (map[string]string, error) {
	return ft.IndustrialBalanceGet(address)
}

// QueryIndexCreated - returns true if index was created
func (ft *FiatTestToken) QueryIndexCreated(balanceTypeStr string) (bool, error) {
	balanceType, err := balance.StringToBalanceType(balanceTypeStr)
	if err != nil {
		return false, err
	}
	return balance.HasIndexCreatedFlag(ft.GetStub(), balanceType)
}

type MintableTestToken struct {
	token.BaseToken
}

func NewMintableTestToken() *MintableTestToken {
	return &MintableTestToken{token.BaseToken{}}
}

func (mt *MintableTestToken) TxBuyToken(sender *types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(mt.Issuer()) {
		return errors.New("impossible operation")
	}

	price, err := mt.CheckLimitsAndPrice("buyToken", amount, currency)
	if err != nil {
		return err
	}
	if err = mt.AllowedBalanceTransfer(currency, sender.Address(), mt.Issuer(), price, "buyToken"); err != nil {
		return err
	}
	if err = mt.TokenBalanceAdd(sender.Address(), amount, "buyToken"); err != nil {
		return err
	}

	return mt.EmissionAdd(amount)
}

func (mt *MintableTestToken) TxBuyBack(sender *types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(mt.Issuer()) {
		return errors.New("impossible operation")
	}

	price, err := mt.CheckLimitsAndPrice("buyBack", amount, currency)
	if err != nil {
		return err
	}
	if err = mt.AllowedBalanceTransfer(currency, mt.Issuer(), sender.Address(), price, "buyBack"); err != nil {
		return err
	}
	if err = mt.TokenBalanceSub(sender.Address(), amount, "buyBack"); err != nil {
		return err
	}
	return mt.EmissionSub(amount)
}

const keyMetadata = "tokenMetadata"

func TestEmitTransfer(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
			feeAggregator *mocks.UserFoundation,
		) []string
		funcInvokeChaincode func(
			cc *core.Chaincode,
			mockStub *mockstub.MockStub,
			functionName string,
			owner *mocks.UserFoundation,
			feeSetter *mocks.UserFoundation,
			feeAddressSetter *mocks.UserFoundation,
			user1 *mocks.UserFoundation,
			parameters ...string,
		) peer.Response
		funcCheckResponse func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
			feeAggregator *mocks.UserFoundation,
			resp peer.Response,
		)
	}{
		{
			name:         "Emit tokens",
			functionName: "emit",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				return []string{user1.AddressBase58Check, "1000"}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				feeSetter *mocks.UserFoundation,
				feeAddressSetter *mocks.UserFoundation,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == user1BalanceKey {
						require.Equal(t, big.NewInt(1000).Bytes(), value)
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "SetFeeAddress",
			functionName: "setFeeAddress",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				return []string{feeAggregator.AddressBase58Check}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				feeSetter *mocks.UserFoundation,
				feeAddressSetter *mocks.UserFoundation,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, feeAddressSetter, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				feeAddressHash := sha3.Sum256(feeAggregator.PublicKeyBytes)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						require.Equal(t, feeAddressHash[:], tokenMetadata.FeeAddress)
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "SetFee",
			functionName: "setFee",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				return []string{"FIAT", "500000", "100", "100000"}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				feeSetter *mocks.UserFoundation,
				feeAddressSetter *mocks.UserFoundation,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, feeSetter, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				metadata := &pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "FIAT",
						Fee:      big.NewInt(500000).Bytes(),
						Floor:    big.NewInt(100).Bytes(),
						Cap:      big.NewInt(100000).Bytes(),
					},
				}

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						require.True(t, proto.Equal(metadata.Fee, tokenMetadata.Fee))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Transfer tokens",
			functionName: "transfer",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				feeAddressHash := sha3.Sum256(feeAggregator.PublicKeyBytes)

				metadata := &pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "FIAT",
						Fee:      big.NewInt(500000).Bytes(),
						Floor:    big.NewInt(100).Bytes(),
						Cap:      big.NewInt(100000).Bytes(),
					},
					FeeAddress: feeAddressHash[:],
				}

				rawMetadata, err := proto.Marshal(metadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[user1BalanceKey] = big.NewInt(1000).Bytes()
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata
				return []string{user2.AddressBase58Check, "400", ""}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundation,
				feeSetter *mocks.UserFoundation,
				feeAddressSetter *mocks.UserFoundation,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user1, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				user2BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check})
				require.NoError(t, err)

				feeAggregatorBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{feeAggregator.AddressBase58Check})
				require.NoError(t, err)

				user1Checked := false
				user2Checked := false
				feeChecked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == user1BalanceKey {
						require.Equal(t, big.NewInt(500).Bytes(), value)
						user1Checked = true
					}
					if putStateKey == user2BalanceKey {
						require.Equal(t, big.NewInt(400).Bytes(), value)
						user2Checked = true
					}
					if putStateKey == feeAggregatorBalanceKey {
						require.Equal(t, big.NewInt(100).Bytes(), value)
						feeChecked = true
					}
				}
				require.True(t, user1Checked && user2Checked && feeChecked)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig("fiat", "FIAT", 8,
				owner.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check, "", nil)

			cc, err := core.NewCC(NewFiatTestToken(token.BaseToken{}))
			require.NoError(t, err)

			parameters := test.funcPrepareMockStub(t, mockStub, user1, user2, feeAggregator)
			resp := test.funcInvokeChaincode(cc, mockStub, test.functionName, owner, feeSetter, feeAddressSetter, user1, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())
			test.funcCheckResponse(t, mockStub, user1, user2, feeAggregator, resp)
		})
	}
}

func TestMultisigEmitTransfer(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
			feeAggregator *mocks.UserFoundation,
		) []string
		funcInvokeChaincode func(
			cc *core.Chaincode,
			mockStub *mockstub.MockStub,
			functionName string,
			owner *mocks.UserFoundationMultisigned,
			user1 *mocks.UserFoundation,
			parameters ...string,
		) peer.Response
		funcCheckResponse func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			user1 *mocks.UserFoundation,
			user2 *mocks.UserFoundation,
			feeAggregator *mocks.UserFoundation,
			resp peer.Response,
		)
	}{
		{
			name:         "Multisigned emission",
			functionName: "emit",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				return []string{user1.AddressBase58Check, "1000"}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundationMultisigned,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeMultisigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == user1BalanceKey {
						require.Equal(t, big.NewInt(1000).Bytes(), value)
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Multisigned transfer",
			functionName: "transfer",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
			) []string {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				feeAddressHash := sha3.Sum256(feeAggregator.PublicKeyBytes)

				metadata := &pbfound.Token{
					Fee: &pbfound.TokenFee{
						Currency: "FIAT",
						Fee:      big.NewInt(500000).Bytes(),
						Floor:    big.NewInt(100).Bytes(),
						Cap:      big.NewInt(100000).Bytes(),
					},
					FeeAddress: feeAddressHash[:],
				}

				rawMetadata, err := proto.Marshal(metadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[user1BalanceKey] = big.NewInt(1000).Bytes()
				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata
				return []string{user2.AddressBase58Check, "400", ""}
			},
			funcInvokeChaincode: func(
				cc *core.Chaincode,
				mockStub *mockstub.MockStub,
				functionName string,
				owner *mocks.UserFoundationMultisigned,
				user1 *mocks.UserFoundation,
				parameters ...string,
			) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user1, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				user1 *mocks.UserFoundation,
				user2 *mocks.UserFoundation,
				feeAggregator *mocks.UserFoundation,
				resp peer.Response,
			) {
				user1BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
				require.NoError(t, err)

				user2BalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user2.AddressBase58Check})
				require.NoError(t, err)

				feeAggregatorBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{feeAggregator.AddressBase58Check})
				require.NoError(t, err)

				user1Checked := false
				user2Checked := false
				feeChecked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == user1BalanceKey {
						require.Equal(t, big.NewInt(500).Bytes(), value)
						user1Checked = true
					}
					if putStateKey == user2BalanceKey {
						require.Equal(t, big.NewInt(400).Bytes(), value)
						user2Checked = true
					}
					if putStateKey == feeAggregatorBalanceKey {
						require.Equal(t, big.NewInt(100).Bytes(), value)
						feeChecked = true
					}
				}
				require.True(t, user1Checked && user2Checked && feeChecked)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {

			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundationMultisigned(pbfound.KeyType_ed25519, 3)
			require.NoError(t, err)

			feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig("fiat token", "FIAT", 8,
				owner.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check, "", nil)

			cc, err := core.NewCC(NewFiatTestToken(token.BaseToken{}))
			require.NoError(t, err)

			parameters := test.funcPrepareMockStub(t, mockStub, user1, user2, feeAggregator)
			resp := test.funcInvokeChaincode(cc, mockStub, test.functionName, owner, user1, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())
			test.funcCheckResponse(t, mockStub, user1, user2, feeAggregator, resp)
		})
	}
}

func TestBuyLimit(t *testing.T) {
	t.Parallel()

	testCollection := []struct {
		name                string
		functionName        string
		funcPrepareMockStub func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			owner *mocks.UserFoundation,
			user *mocks.UserFoundation,
		) []string
		funcInvokeChaincode func(
			cc *core.Chaincode,
			mockStub *mockstub.MockStub,
			functionName string,
			owner *mocks.UserFoundation,
			user1 *mocks.UserFoundation,
			parameters ...string,
		) peer.Response
		funcCheckResponse func(
			t *testing.T,
			mockStub *mockstub.MockStub,
			owner *mocks.UserFoundation,
			user *mocks.UserFoundation,
			resp peer.Response,
		)
	}{
		{
			name:         "Setting rate",
			functionName: "setRate",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				return []string{"buyToken", "FIAT", "50000000"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						expectedRate := &pbfound.TokenRate{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
						}

						require.True(t, proto.Equal(expectedRate, tokenMetadata.Rates[0]))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Buying token",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyBalance] = big.NewInt(1000).Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"100", "FIAT"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				keyAllowedBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				checkedBalance := false
				checkedAllowed := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyBalance {
						require.Equal(t, big.NewInt(100).Bytes(), value)
						checkedBalance = true
					}
					if putStateKey == keyAllowedBalance {
						require.Equal(t, big.NewInt(950).Bytes(), value)
						checkedAllowed = true
					}
				}
				require.True(t, checkedBalance && checkedAllowed)
			},
		},
		{
			name:         "Setting limits",
			functionName: "setLimits",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation) []string {
				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"buyToken", "FIAT", "100", "200"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						expectedRate := &pbfound.TokenRate{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
							Min:      big.NewInt(100).Bytes(),
							Max:      big.NewInt(200).Bytes(),
						}

						require.True(t, proto.Equal(expectedRate, tokenMetadata.Rates[0]))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "[negative] Error setting limits",
			functionName: "setLimits",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation) []string {
				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"buyToken", "FIAT", "100", "50"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, token.ErrMinLimitGreaterThanMax, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "Setting limit without Max value",
			functionName: "setLimits",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation) []string {
				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"buyToken", "FIAT", "100", "0"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, owner, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				checked := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyMetadata {
						tokenMetadata := &pbfound.Token{}

						err := proto.Unmarshal(value, tokenMetadata)
						require.NoError(t, err)

						expectedRate := &pbfound.TokenRate{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
							Min:      big.NewInt(100).Bytes(),
							Max:      big.NewInt(0).Bytes(),
						}

						require.True(t, proto.Equal(expectedRate, tokenMetadata.Rates[0]))
						checked = true
					}
				}
				require.True(t, checked)
			},
		},
		{
			name:         "Buying token with limits",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyBalance] = big.NewInt(1000).Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
							Min:      big.NewInt(100).Bytes(),
							Max:      big.NewInt(200).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"100", "FIAT"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)

				keyAllowedBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				checkedBalance := false
				checkedAllowed := false
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					putStateKey, value := mockStub.PutStateArgsForCall(i)
					if putStateKey == keyBalance {
						require.Equal(t, big.NewInt(100).Bytes(), value)
						checkedBalance = true
					}
					if putStateKey == keyAllowedBalance {
						require.Equal(t, big.NewInt(950).Bytes(), value)
						checkedAllowed = true
					}
				}
				require.True(t, checkedBalance && checkedAllowed)
			},
		},
		{
			name:         "[negative] Error buying token below limits",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyBalance] = big.NewInt(1000).Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
							Min:      big.NewInt(100).Bytes(),
							Max:      big.NewInt(200).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"50", "FIAT"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, token.ErrAmountOutOfLimits, payload.TxResponses[0].Error.Error)
			},
		},
		{
			name:         "[negative] Error buying token above limits",
			functionName: "buyToken",
			funcPrepareMockStub: func(
				t *testing.T,
				mockStub *mockstub.MockStub,
				owner *mocks.UserFoundation,
				user *mocks.UserFoundation,
			) []string {
				keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeAllowed.String(), []string{user.AddressBase58Check, "FIAT"})
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyBalance] = big.NewInt(1000).Bytes()

				tokenMetadata := &pbfound.Token{
					Rates: []*pbfound.TokenRate{
						{
							DealType: "buyToken",
							Currency: "FIAT",
							Rate:     big.NewInt(50000000).Bytes(),
							Min:      big.NewInt(100).Bytes(),
							Max:      big.NewInt(200).Bytes(),
						},
					},
				}

				rawMetadata, err := proto.Marshal(tokenMetadata)
				require.NoError(t, err)

				mockStub.GetStateCallsMap[keyMetadata] = rawMetadata

				return []string{"300", "FIAT"}
			},
			funcInvokeChaincode: func(cc *core.Chaincode, mockStub *mockstub.MockStub, functionName string, owner *mocks.UserFoundation, user *mocks.UserFoundation, parameters ...string) peer.Response {
				_, resp := mockStub.TxInvokeChaincodeSigned(cc, functionName, user, "", "", "", parameters...)
				return resp
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, owner *mocks.UserFoundation, user *mocks.UserFoundation, resp peer.Response) {
				payload := &pbfound.BatchResponse{}

				err := pb.Unmarshal(resp.Payload, payload)
				require.NoError(t, err)
				require.Equal(t, token.ErrAmountOutOfLimits, payload.TxResponses[0].Error.Error)
			},
		},
	}

	for _, test := range testCollection {
		t.Run(test.name, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			mockStub.CreateAndSetConfig("currency coin token", "CC", 8,
				owner.AddressBase58Check, "", "", "", nil)

			cc, err := core.NewCC(NewMintableTestToken())
			require.NoError(t, err)

			user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
			require.NoError(t, err)

			parameters := test.funcPrepareMockStub(t, mockStub, owner, user)

			resp := test.funcInvokeChaincode(cc, mockStub, test.functionName, owner, user, parameters...)
			require.Equal(t, int32(http.StatusOK), resp.GetStatus())
			require.Empty(t, resp.GetMessage())
			test.funcCheckResponse(t, mockStub, owner, user, resp)
		})
	}
}
