package unit

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
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
func (mt *FiatTestToken) TxEmit(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(mt.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := mt.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return mt.EmissionAdd(amount)
}

type MintableTestToken struct {
	token.BaseToken
}

func NewMintableTestToken(bt token.BaseToken) *MintableTestToken {
	return &MintableTestToken{bt}
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
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	feeAddressSetter := m.NewWallet()
	feeSetter := m.NewWallet()
	feeAggregator := m.NewWallet()
	fiat := NewFiatTestToken(token.BaseToken{
		Name:   "fiat token",
		Symbol: "FIAT",
	})
	m.NewChainCode("fiat", fiat, nil, nil, owner.Address(), feeSetter.Address(), feeAddressSetter.Address())

	user1 := m.NewWallet()
	user2 := m.NewWallet()

	owner.SignedInvoke("fiat", "emit", user1.Address(), "1000")
	user1.BalanceShouldBe("fiat", 1000)

	feeAddressSetter.SignedInvoke("fiat", "setFeeAddress", feeAggregator.Address())
	feeSetter.SignedInvoke("fiat", "setFee", "FIAT", "500000", "100", "100000")

	rawMD := feeSetter.Invoke("fiat", "metadata")
	md := &metadata{}

	assert.NoError(t, json.Unmarshal([]byte(rawMD), md))

	assert.Equal(t, "FIAT", md.Fee.Currency)
	assert.Equal(t, "500000", md.Fee.Fee.String())
	assert.Equal(t, "100000", md.Fee.Cap.String())
	assert.Equal(t, "100", md.Fee.Floor.String())
	assert.Equal(t, feeAggregator.Address(), md.Fee.Address)

	user1.SignedInvoke("fiat", "transfer", user2.Address(), "400", "")
	user1.BalanceShouldBe("fiat", 500)
	user2.BalanceShouldBe("fiat", 400)
}

func TestMultisigEmitTransfer(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewMultisigWallet(3)
	fiat := NewFiatTestToken(token.BaseToken{
		Name:   "fiat token",
		Symbol: "FIAT",
	})
	m.NewChainCode("fiat", fiat, nil, nil, owner.Address())

	user1 := m.NewWallet()

	_, res, _ := owner.RawSignedInvoke(2, "fiat", "emit", user1.Address(), "1000")
	assert.Equal(t, "", res.Error)
	user1.BalanceShouldBe("fiat", 1000)
}

func TestBuyLimit(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := NewMintableTestToken(
		token.BaseToken{
			Name:   "currency coin token",
			Symbol: "CC",
		})
	m.NewChainCode("cc", cc, nil, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddAllowedBalance("cc", "FIAT", 1000)

	owner.SignedInvoke("cc", "setRate", "buyToken", "FIAT", "50000000")

	user1.SignedInvoke("cc", "buyToken", "100", "FIAT")

	owner.SignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "200")

	_, resp, _ := user1.RawSignedInvoke("cc", "buyToken", "50", "FIAT")
	assert.Equal(t, "amount out of limits", resp.Error)

	_, resp, _ = user1.RawSignedInvoke("cc", "buyToken", "300", "FIAT")
	assert.Equal(t, "amount out of limits", resp.Error)

	user1.SignedInvoke("cc", "buyToken", "150", "FIAT")

	_, resp, _ = owner.RawSignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "0")
	assert.Equal(t, "", resp.Error)

	_, resp, _ = owner.RawSignedInvoke("cc", "setLimits", "buyToken", "FIAT", "100", "50")
	assert.Equal(t, "min limit is greater than max limit", resp.Error)
}
