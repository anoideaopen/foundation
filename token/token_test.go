package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	"github.com/stretchr/testify/require"
)

const (
	testTokenSymbol = "TT"
	testTokenCCName = "tt"

	testEmitAmount    = 1000
	testEmitSubAmount = 100
	testFee           = 500000 // commission amount in percent, calculated according to the formula ds
	testFloor         = 100    // minimum commission amount in tokens
	testCap           = 100000 // maximum commission amount in tokens

	testTokenGetIssuerFnName           = "getIssuer"
	testTokenGetFeeSetterFnName        = "getFeeSetter"
	testTokenGetFeeAddressSetterFnName = "getFeeAddressSetter"

	testEmissionAddFnName   = "emissionAdd"
	testEmissionSubFnName   = "emissionSub"
	testSetFeeSubFnName     = "setFee"
	testSetFeeAddressFnName = "setFeeAddress"
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

// TestToken helps to test base token roles.
type TestToken struct {
	BaseToken
}

func (tt *TestToken) QueryGetIssuer() (string, error) {
	addr := tt.Issuer().String()
	return addr, nil
}

func (tt *TestToken) QueryGetFeeSetter() (string, error) {
	addr := tt.FeeSetter().String()
	return addr, nil
}

func (tt *TestToken) QueryGetFeeAddressSetter() (string, error) {
	addr := tt.FeeAddressSetter().String()
	return addr, nil
}

func (tt *TestToken) TxEmissionAdd(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New(ErrUnauthorized)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(ErrAmountEqualZero)
	}
	if err := tt.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return tt.EmissionAdd(amount)
}

func (tt *TestToken) TxEmissionSub(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(tt.Issuer()) {
		return errors.New(ErrUnauthorized)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(ErrAmountEqualZero)
	}
	if err := tt.TokenBalanceSub(address, amount, "txEmitSub"); err != nil {
		return err
	}
	return tt.EmissionSub(amount)
}

// TestBaseTokenRoles - Checking the base token roles
func TestBaseTokenRoles(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenCCName, testTokenSymbol, 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	t.Run("Issuer address check", func(t *testing.T) {
		actualIssuerAddr := issuer.Invoke(testTokenCCName, testTokenGetIssuerFnName)
		actualIssuerAddr = trimStartEndQuotes(actualIssuerAddr)
		require.Equal(t, issuer.Address(), actualIssuerAddr)
	})

	t.Run("FeeSetter address check", func(t *testing.T) {
		actualFeeSetterAddr := issuer.Invoke(testTokenCCName, testTokenGetFeeSetterFnName)
		actualFeeSetterAddr = trimStartEndQuotes(actualFeeSetterAddr)
		require.Equal(t, feeSetter.Address(), actualFeeSetterAddr)
	})

	t.Run("FeeAddressSetter address check", func(t *testing.T) {
		actualFeeAddressSetterAddr := issuer.Invoke(testTokenCCName, testTokenGetFeeAddressSetterFnName)
		actualFeeAddressSetterAddr = trimStartEndQuotes(actualFeeAddressSetterAddr)
		require.Equal(t, feeAddressSetter.Address(), actualFeeAddressSetterAddr)
	})
}

// TestEmitToken - Checking that emission is working
func TestEmitToken(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	user := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenCCName, testTokenSymbol, 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	t.Run("Test emitSub token", func(t *testing.T) {
		issuer.SignedInvoke(
			testTokenCCName,
			testEmissionAddFnName,
			user.Address(),
			fmt.Sprint(testEmitAmount),
		)
		user.BalanceShouldBe(testTokenCCName, testEmitAmount)
	})
}

// TestEmissionSub - Checking that emission sub is working
func TestEmissionSub(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenCCName, testTokenSymbol, 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	user := ledger.NewWallet()

	issuer.SignedInvoke(testTokenCCName, testEmissionAddFnName, user.Address(), fmt.Sprint(testEmitAmount))
	user.BalanceShouldBe(testTokenCCName, testEmitAmount)

	t.Run("Test emitSub token", func(t *testing.T) {
		issuer.SignedInvoke(testTokenCCName, testEmissionSubFnName, user.Address(), fmt.Sprint(testEmitSubAmount))
		user.BalanceShouldBe(testTokenCCName, testEmitAmount-testEmitSubAmount)
	})
}

// TestEmissionSub - Checking that setting fee is working
func TestSetFee(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeAggregator := ledger.NewWallet()

	tt := &TestToken{}
	config := makeBaseTokenConfig(testTokenCCName, testTokenSymbol, 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	t.Run("Test emit token", func(t *testing.T) {
		feeAddressSetter.SignedInvoke(testTokenCCName, testSetFeeAddressFnName, feeAggregator.Address())
		feeSetter.SignedInvoke(testTokenCCName, testSetFeeSubFnName, testTokenSymbol, fmt.Sprint(testFee), fmt.Sprint(testFloor), fmt.Sprint(testCap))

		rawMD := feeSetter.Invoke(testTokenCCName, "metadata")
		md := &metadata{}

		require.NoError(t, json.Unmarshal([]byte(rawMD), md))
		require.Equal(t, testTokenSymbol, md.Fee.Currency)
		require.Equal(t, fmt.Sprint(testFee), md.Fee.Fee.String())
		require.Equal(t, fmt.Sprint(testFloor), md.Fee.Floor.String())
		require.Equal(t, fmt.Sprint(testCap), md.Fee.Cap.String())
		require.Equal(t, feeAggregator.Address(), md.Fee.Address)
	})
}

func trimStartEndQuotes(s string) string {
	const quoteSign = "\""
	res := strings.TrimPrefix(s, quoteSign)
	res = strings.TrimSuffix(res, quoteSign)
	return res
}
