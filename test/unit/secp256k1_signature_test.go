package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
)

const (
	TestTokenName     = "Test Token"
	TestTokenSymbol   = "TT"
	TestTokenDecimals = 8
	TestTokenAdmin    = ""
)

const (
	FiatChaincodeName = "fiat"
	FiatChannelName   = "fiat"
	EmitFunctionName  = "emit"
	EmitAmount        = "1000"
	ExpectedAmount    = 1000
)

func Test_Secp256k1Signatures(t *testing.T) {
	var (
		m                = mock.NewLedger(t)
		owner            = m.NewWallet()
		feeAddressSetter = m.NewWallet()
		feeSetter        = m.NewWallet()
		user1            = m.NewWallet()
		fiat             = NewFiatTestToken(token.BaseToken{})
	)

	owner.UseSecp256k1Key()

	config := makeBaseTokenConfig(
		TestTokenName,
		TestTokenSymbol,
		TestTokenDecimals,
		owner.Address(),
		feeSetter.Address(),
		feeAddressSetter.Address(),
		TestTokenAdmin,
		nil,
	)

	m.NewCC(
		FiatChaincodeName,
		fiat,
		config,
	)

	owner.SignedInvoke(FiatChannelName, EmitFunctionName, user1.Address(), EmitAmount)
	user1.BalanceShouldBe(FiatChannelName, ExpectedAmount)
}
