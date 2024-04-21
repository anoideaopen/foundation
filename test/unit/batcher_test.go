package unit

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

func TestBatcherEmitTransfer(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAggregator := ledger.NewWallet()

	fiat := NewFiatTestToken(token.BaseToken{})
	fiatConfig := makeBaseTokenConfig("fiat", "FIAT", 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)
	initMsg := ledger.NewCC("fiat", fiat, fiatConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()

	peerResp, err := owner.BatcherSignedInvoke("fiat", "emit", user1.Address(), "1000")
	require.NoError(t, err)
	require.NotNil(t, peerResp)
	require.Equal(t, peerResp.Status, int32(200))
	require.Empty(t, peerResp.Message)

	user1.BalanceShouldBe("fiat", 1000)

	_, err = feeAddressSetter.BatcherSignedInvoke("fiat", "setFeeAddress", feeAggregator.Address())
	require.NoError(t, err)
	_, err = feeSetter.BatcherSignedInvoke("fiat", "setFee", "FIAT", "500000", "100", "100000")
	require.NoError(t, err)

	rawMD := feeSetter.Invoke("fiat", "metadata")
	md := &metadata{}
	require.NoError(t, json.Unmarshal([]byte(rawMD), md))

	require.Equal(t, "FIAT", md.Fee.Currency)
	require.Equal(t, "500000", md.Fee.Fee.String())
	require.Equal(t, "100000", md.Fee.Cap.String())
	require.Equal(t, "100", md.Fee.Floor.String())
	require.Equal(t, feeAggregator.Address(), md.Fee.Address)

	user2 := ledger.NewWallet()
	_, err = user1.BatcherSignedInvoke("fiat", "transfer", user2.Address(), "400", "")
	require.NoError(t, err)
	user1.BalanceShouldBe("fiat", 500)
	user2.BalanceShouldBe("fiat", 400)
}
