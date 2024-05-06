package unit

import (
	"errors"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
)

const (
	testTokenName      = "Testing Token"
	testTokenSymbol    = "TT"
	testTokenCCName    = "tt"
	testTokenWithGroup = "tt_testGroup"
	testGroup          = "testGroup"

	testMessageEmptyNonce = "\"0\""

	testGetNonceFnName = "getNonce"
)

type TestToken struct {
	token.BaseToken
}

func (tt *TestToken) TxTestCall() error {
	traceCtx := tt.GetTraceContext()
	_, span := tt.TracingHandler().StartNewSpan(traceCtx, "TxTestCall()")
	defer span.End()

	return nil
}

func (tt *TestToken) TxFailedTestCall() error {
	traceCtx := tt.GetTraceContext()
	_, span := tt.TracingHandler().StartNewSpan(traceCtx, "TxTestCall()")
	defer span.End()

	return errors.New("ALARM")
}

func (tt *TestToken) TxEmissionAdd(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}
	if err := tt.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return tt.EmissionAdd(amount)
}

// TestGetEmptyNonce - Checking that new wallet have empty nonce
func TestGetEmptyNonce(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledgerMock.NewCC(testTokenCCName, tt, config)
	require.Empty(t, initMsg)

	t.Run("Get nonce with new wallet", func(t *testing.T) {
		nonce := owner.Invoke(testTokenCCName, testGetNonceFnName, owner.Address())
		require.Equal(t, nonce, testMessageEmptyNonce)
	})
}

// TestGetNonce - Checking that the nonce after some operation is not null
func TestGetNonce(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC(testTokenCCName, tt, config)
	require.Empty(t, initMsg)

	owner.SignedInvoke(testTokenCCName, "emissionAdd", owner.Address(), "1000")
	owner.BalanceShouldBe(testTokenCCName, 1000)

	t.Run("Get nonce with new wallet", func(t *testing.T) {
		nonce := owner.Invoke(testTokenCCName, testGetNonceFnName, owner.Address())
		require.NotEqual(t, nonce, testMessageEmptyNonce)
	})
}

// TestInit - Checking that init with right mspId working
func TestInit(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		issuer.Address(), "", "", "", nil)

	t.Run("Init new chaincode", func(t *testing.T) {
		message := ledger.NewCC(testTokenCCName, tt, config)
		require.Empty(t, message)
	})
}

// TestTxHealthCheck - Checking healthcheck method.
func TestTxHealthCheck(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledgerMock.NewCC(testTokenCCName, tt, config)
	require.Empty(t, initMsg)

	t.Run("Healthcheck checking", func(t *testing.T) {
		txID := owner.SignedInvoke(testTokenCCName, "healthCheck")
		require.NotEmpty(t, txID)
	})
}
