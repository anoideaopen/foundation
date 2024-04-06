package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestIndexer(t *testing.T) {
	var (
		m                = mock.NewLedger(t)
		owner            = m.NewWallet()
		feeAddressSetter = m.NewWallet()
		feeSetter        = m.NewWallet()
		user1            = m.NewWallet()
		user2            = m.NewWallet()
		user3            = m.NewWallet()
		user4            = m.NewWallet()
		fiat             = NewFiatTestToken(token.BaseToken{})

		blockchain = "HLF"
		usdt       = "USDT"
		tokenID    = blockchain + "_" + usdt
	)
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)

	m.NewCC("fiat", fiat, config)

	// Accrual of balance.
	owner.SignedInvoke("fiat", "emit", user1.Address(), "1000")
	user1.BalanceShouldBe("fiat", 1000)

	// Accrual of toknow directly to the State.
	user1.AddTokenBalance("fiat", tokenID, 1000)
	user1.IndustrialBalanceShouldBe("fiat", usdt, 1000)

	user2.AddTokenBalance("fiat", tokenID, 1000)
	user2.IndustrialBalanceShouldBe("fiat", usdt, 1000)

	user3.AddTokenBalance("fiat", tokenID, 1000)
	user3.IndustrialBalanceShouldBe("fiat", usdt, 1000)

	stub := m.GetStub("fiat")

	// Checking for the presence of an index.
	index, err := balance.HasIndexCreatedFlag(stub, balance.BalanceTypeToken)
	require.Nil(t, err)
	require.False(t, index)

	// Checking that there are no token holders because the index is not constructed.
	ownersBeforeIndexing, err := balance.ListOwnersByToken(stub, balance.BalanceTypeToken, usdt)
	require.Nil(t, err)
	require.Len(t, ownersBeforeIndexing, 0)

	// Index construction.
	stub.MockTransactionStart("index")
	err = balance.CreateIndex(stub, balance.BalanceTypeToken)
	require.Nil(t, err)
	stub.MockTransactionEnd("index")

	// Checking that the index is constructed.
	index, err = balance.HasIndexCreatedFlag(stub, balance.BalanceTypeToken)
	require.Nil(t, err)
	require.True(t, index)

	ownersAfterIndexing, err := balance.ListOwnersByToken(stub, balance.BalanceTypeToken, usdt)
	require.Nil(t, err)
	require.Len(t, ownersAfterIndexing, 3)

	// Accrual of tokens through issuance.
	owner.SignedInvoke("fiat", "emitIndustrial", user4.Address(), "1000", usdt)
	user4.IndustrialBalanceShouldBe("fiat", usdt, 1000)

	// Checking that there are more token holders and the index has been built.
	ownersAutoIndexed, err := balance.ListOwnersByToken(stub, balance.BalanceTypeToken, usdt)
	require.Nil(t, err)
	require.Len(t, ownersAutoIndexed, 4)
}
