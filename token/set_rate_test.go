package token

import (
	"testing"

	"github.com/anoideaopen/foundation/core/types/big"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

type serieSetRate struct {
	tokenName string
	dealType  string
	currency  string
	rate      string
	errorMsg  string
}

// TestSetRate - positive test with valid parameters
func TestSetRate(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateAllParametersAreEmpty - negative test with all parameters are empty
func TestSetRateAllParametersAreEmpty(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "",
		dealType:  "",
		currency:  "",
		rate:      "",
		errorMsg:  "channel undefined",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateToZero - negative test with invalid rate parameter set to zero
func TestSetRateToZero(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "",
		rate:      "0",
		errorMsg:  "trying to set rate = 0",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateToString - negative test with invalid rate parameter set to string
func TestSetRateToString(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "",
		rate:      "wonder",
		errorMsg:  "invalid argument value: 'wonder': for type '*big.Int'",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateMinusValue - negative test with invalid rate parameter set to minus value
func TestSetRateMinusValue(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "",
		rate:      "-3",
		errorMsg:  "validation failed: 'negative number'",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetTokenNameToWrongStringParameter - negative test with invalid token Name parameter set to wrong string
func TestSetRateSetTokenNameToWrongStringParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "wonder",
		dealType:  "distribute",
		currency:  "",
		rate:      "1",
		errorMsg:  "stub of [wonder] not found",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetTokenNameToNumericParameter - negative test with invalid token Name parameter set to numeric
func TestSetRateSetTokenNameToNumericParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "353",
		dealType:  "distribute",
		currency:  "",
		rate:      "1",
		errorMsg:  "stub of [353] not found",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetDealTypeToWrongstringParameter - negative test with invalid deal Type parameter set to wrong string
// ??????? err = nill, but test should be failed
func TestSetRateSetDealTypeToWrongstringParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "wonder",
		currency:  "",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetDealTypeToNumericParameter - negative test with invalid deal Type parameter set to numeric
// ??????? err = nill, but test should be failed
func TestSetRateSetDealTypeToNumericParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "353",
		currency:  "",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateCurrencyEqualToken - negative test with invalid currency parameter set to equals token
// wrong errorMSG, "is" unnecessary in this sentence.
func TestSetRateCurrencyEqualToken(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "TT",
		rate:      "3",
		errorMsg:  "currency is equals token: it is impossible",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetCurrencyToMinusValue - negative test with invalid currency parameter set to minus value
// ??????? err = nill, but test should be failed
func TestSetRateSetCurrencyToMinusValue(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "-10",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetCurrencyToWrongStringParameter - negative test with invalid currency parameter set to wrong string
// ??????? err = nill, but test should be failed
func TestSetRateSetCurrencyToWrongStringParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "wonder",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateSetCurrencyToNumericParameter - negative test with invalid currency parameter set to numeric
// ??????? err = nill, but test should be failed
func TestSetRateSetCurrencyToNumericParameter(t *testing.T) {
	t.Parallel()

	s := &serieSetRate{
		tokenName: "tt",
		dealType:  "distribute",
		currency:  "353",
		rate:      "1",
		errorMsg:  "",
	}

	BaseTokenSetRateTest(t, s)
}

// TestSetRateWrongAuthorized - negative test with invalid issuer
func TestSetRateWrongAuthorized(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	outsider := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	if err := outsider.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "1"); err != nil {
		require.Equal(t, ErrUnauthorized, err.Error())
	}
}

// TestSetRateWrongNumberParameters - negative test with incorrect number of parameters
func TestSetRateWrongNumberParameters(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	if err := issuer.RawSignedInvokeWithErrorReturned("tt", "setRate", "distribute", "", "", ""); err != nil {
		require.Contains(t, err.Error(), "incorrect number of keys or signs")
	}
}

// BaseTokenSetRateTest - base test for checking the SetRate API
func BaseTokenSetRateTest(t *testing.T, ser *serieSetRate) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	tt := &BaseToken{}
	config := makeBaseTokenConfig("Test Token", "TT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("tt", tt, config)

	if err := issuer.RawSignedInvokeWithErrorReturned(
		ser.tokenName,
		"setRate",
		ser.dealType,
		ser.currency,
		ser.rate,
	); err != nil {
		require.Contains(t, err.Error(), ser.errorMsg)
	} else {
		require.NoError(t, err)

		data, err1 := issuer.Ledger().GetStub("tt").GetState("tokenMetadata")
		require.NoError(t, err1)

		config := &proto.Token{}
		err2 := pb.Unmarshal(data, config)
		require.NoError(t, err2)

		rate := config.Rates[0]
		actualRate := new(big.Int).SetBytes(rate.Rate)

		stringRate := actualRate.String()
		if ser.rate != stringRate {
			t.Errorf("Invalid rate: %s", stringRate)
		}
	}
}
