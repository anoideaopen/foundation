package unit

import (
	"encoding/json"
	"errors"
	"golang.org/x/crypto/sha3"
	"net/http"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/require"
)

const BatchRobotCert = "0a0a61746f6d797a654d535012d7062d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943536a434341664367417749424167495241496b514e37444f456b6836686f52425057633157495577436759494b6f5a497a6a304541774977675963780a437a414a42674e5642415954416c56544d524d77455159445651514945777044595778705a6d3979626d6c684d525977464159445651514845773154595734670a526e4a68626d4e7063324e764d534d77495159445651514b45787068644739746558706c4c6e56686443356b624851755958527662586c365a53356a6144456d0a4d4351474131554541784d64593245755958527662586c365a533531595851755a4778304c6d463062323135656d5575593267774868634e4d6a41784d44457a0a4d4467314e6a41775768634e4d7a41784d4445784d4467314e6a4177576a42324d517377435159445651514745774a56557a45544d4245474131554543424d4b0a5132467361575a76636d3570595445574d4251474131554542784d4e5532467549455a795957356a61584e6a627a45504d4130474131554543784d47593278700a5a5735304d536b774a7759445651514444434256633256794d554268644739746558706c4c6e56686443356b624851755958527662586c365a53356a6144425a0a4d424d4742797147534d34394167454743437147534d3439417745484130494142427266315057484d51674d736e786263465a346f3579774b476e677830594e0a504b6270494335423761446f6a46747932576e4871416b5656723270697853502b4668497634434c634935633162473963365a375738616a5454424c4d4134470a41315564447745422f775145417749486744414d42674e5648524d4241663845416a41414d437347413155644977516b4d434b4149464b2f5335356c6f4865700a6137384441363173364e6f7433727a4367436f435356386f71462b37585172344d416f4743437147534d343942414d43413067414d4555434951436e6870476d0a58515664754b632b634266554d6b31494a6835354444726b3335436d436c4d657041533353674967596b634d6e5a6b385a42727179796953544d6466526248740a5a32506837364e656d536b62345651706230553d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a"

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

func TestEmitTransfer(t *testing.T) {
	t.Parallel()

	keyMetadata := "tokenMetadata"

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

			fiatConfig := makeBaseTokenConfig("fiat", "FIAT", 8,
				owner.AddressBase58Check, feeSetter.AddressBase58Check, feeAddressSetter.AddressBase58Check, "", nil)

			cc, err := core.NewCC(NewFiatTestToken(token.BaseToken{}))
			require.NoError(t, err)

			mockStub.SetConfig(fiatConfig)

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

	ledger := mock.NewLedger(t)
	owner := ledger.NewMultisigWallet(3)

	fiat := NewFiatTestToken(token.BaseToken{})
	fiatConfig := makeBaseTokenConfig("fiat token", "FIAT", 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC("fiat", fiat, fiatConfig)

	user1 := ledger.NewWallet()

	_, res, _ := owner.RawSignedInvoke(2, "fiat", "emit", user1.Address(), "1000")
	require.Equal(t, "", res.Error)
	user1.BalanceShouldBe("fiat", 1000)
}

func TestBuyLimit(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	cc := NewMintableTestToken()
	ccConfig := makeBaseTokenConfig("currency coin token", "CC", 8,
		owner.Address(), "", "", "", nil)
	ledger.NewCC("cc", cc, ccConfig)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("cc", "FIAT", 1000)

	owner.SignedInvoke("cc", "setRate", "buyToken", "FIAT", "50000000")

	user1.SignedInvoke("cc", "buyToken", "100", "FIAT")

	owner.SignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "200")

	_, resp, _ := user1.RawSignedInvoke("cc", "buyToken", "50", "FIAT")
	require.Equal(t, "amount out of limits", resp.Error)

	_, resp, _ = user1.RawSignedInvoke("cc", "buyToken", "300", "FIAT")
	require.Equal(t, "amount out of limits", resp.Error)

	user1.SignedInvoke("cc", "buyToken", "150", "FIAT")

	_, resp, _ = owner.RawSignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "0")
	require.Equal(t, "", resp.Error)

	_, resp, _ = owner.RawSignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "50")
	require.Equal(t, "min limit is greater than max limit", resp.Error)
}
