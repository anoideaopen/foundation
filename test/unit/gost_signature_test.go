package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
)

func TestGOSTSignatures(t *testing.T) {
	var (
		m                = mock.NewLedger(t)
		owner            = m.NewWallet()
		feeAddressSetter = m.NewWallet()
		feeSetter        = m.NewWallet()
		user1            = m.NewWallet()
		fiat             = NewFiatTestToken(token.BaseToken{})
	)

	owner.UseGOSTKey()

	config := makeBaseTokenConfig("Test Token", "TT", 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)

	m.NewCC(
		"fiat",
		fiat,
		config,
	)

	owner.SignedInvoke("fiat", "emit", user1.Address(), "1000")
	user1.BalanceShouldBe("fiat", 1000)
}
