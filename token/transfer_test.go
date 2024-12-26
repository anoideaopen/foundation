package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/anoideaopen/foundation/core/types"
	ma "github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	pb "github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

const vtName = "Validation Token"

func TestBaseTokenTxTransfer(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	buyer := ledger.NewWallet()
	seller := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig("vt", "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	issuer.SignedInvoke("vt", "emitToken", "10")
	issuer.SignedInvoke("vt", "setRate", "buyToken", "usd", "100000000")
	issuer.SignedInvoke("vt", "setLimits", "buyToken", "usd", "1", "10")

	seller.AddAllowedBalance("vt", "usd", 5)

	err := seller.RawSignedInvokeWithErrorReturned("vt", "buyToken", "5", "usd")
	require.NoError(t, err)

	if err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "0", ""); err != nil {
		require.ErrorContains(t, err, ErrAmountEqualZero)
	}
	if err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "100", ""); err != nil {
		require.ErrorContains(t, err, "insufficient balance")
	}
	err = seller.RawSignedInvokeWithErrorReturned("vt", "transfer", buyer.Address(), "5", "")
	require.NoError(t, err)
}

func TestTransferWithFee(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	feeAggregator := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig("vt token", "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	issuer.SignedInvoke("vt", "emitToken", "101")

	currency := "VT"
	feeAmount := "500000"

	t.Run("[negative] trying to set empty fee", func(t *testing.T) {
		err := feeSetter.RawSignedInvokeWithErrorReturned("vt", "setFee", currency, "", "1", "0")
		require.EqualError(t, err, "invalid argument value: '': for type '*big.Int': 'math/big: cannot unmarshal \"\" into a *big.Int': validate TxSetFee, argument 2")
	})

	t.Run("[negative] trying to set negative fee", func(t *testing.T) {
		err := feeSetter.RawSignedInvokeWithErrorReturned("vt", "setFee", "", "-1", "1", "0")
		require.EqualError(t, err, "invalid argument value: '-1': validation failed: 'negative number': validate TxSetFee, argument 2")
	})

	feeSetter.SignedInvoke("vt", "setFee", "VT", feeAmount, "1", "0")

	var err error
	t.Run("[negative] trying to transfer when sender equals address to", func(t *testing.T) {
		err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", issuer.Address(), "100", "")
		require.ErrorContains(t, err, "sender and recipient are same users")
	})

	t.Run("[negative] trying to transfer negative amount", func(t *testing.T) {
		err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "-100", "")
		require.EqualError(t, err, "invalid argument value: '-100': validation failed: 'negative number': validate TxTransfer, argument 2")
	})

	t.Run("[negative] trying to transfer zero amount", func(t *testing.T) {
		err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "0", "")
		require.ErrorContains(t, err, ErrAmountEqualZero)
	})

	t.Run("[negative] trying to transfer when fee address is not set", func(t *testing.T) {
		err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
		require.ErrorContains(t, err, ErrFeeAddressNotConfigured.Error())
	})

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	require.ErrorContains(t, err, ErrFeeAddressNotConfigured.Error())

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())
	issuer.SignedInvoke("vt", "transfer", user.Address(), "100", "")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")
	err = json.Unmarshal([]byte(rawResp), &predict)
	require.NoError(t, err)
	fmt.Println("Invoke predictFee response: ", predict.Fee)

	issuer.BalanceShouldBe("vt", 0)
	user.BalanceShouldBe("vt", 100)
	feeAggregator.BalanceShouldBe("vt", 1)
}

func TestTransferWithFeeWithEmptyCurrency(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	feeAggregator := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig(vtName, "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	issuer.SignedInvoke("vt", "emitToken", "101")

	currency := ""
	feeAmount := "500000"
	floor := "1"
	cap := "0"

	feeSetter.SignedInvoke("vt", "setFee", "VT", feeAmount, "1", "0")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	require.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	cfg := &proto.Token{
		TotalEmission: nil,
		Fee: &proto.TokenFee{
			Currency: currency,
			Fee:      []byte(feeAmount),
			Floor:    []byte(floor),
			Cap:      []byte(cap),
		},
		Rates:      nil,
		FeeAddress: []byte(feeAggregator.Address()),
	}
	bytes, err := pb.Marshal(cfg)
	require.NoError(t, err)

	stub := ledger.GetStub("vt")
	stub.MockTransactionStart("test")
	err = stub.PutState("tokenMetadata", bytes)
	require.NoError(t, err)

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	require.ErrorContains(t, err, "config fee currency can't be empty")

	issuer.BalanceShouldBe("vt", 101)
	user.BalanceShouldBe("vt", 0)
	feeAggregator.BalanceShouldBe("vt", 0)
}

func TestTransferWithFeeWithNilTokenFee(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	feeAggregator := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig(vtName, "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	issuer.SignedInvoke("vt", "emitToken", "101")

	feeSetter.SignedInvoke("vt", "setFee", "VT", "500000", "1", "0")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	require.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	cfg := &proto.Token{
		TotalEmission: nil,
		Fee:           nil,
		Rates:         nil,
		FeeAddress:    []byte(feeAggregator.Address()),
	}
	bytes, err := pb.Marshal(cfg)
	require.NoError(t, err)

	stub := ledger.GetStub("vt")
	stub.MockTransactionStart("test")
	err = stub.PutState("tokenMetadata", bytes)
	require.NoError(t, err)

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	require.NoError(t, err)

	issuer.BalanceShouldBe("vt", 1)
	user.BalanceShouldBe("vt", 100)
	feeAggregator.BalanceShouldBe("vt", 0)
}

func TestTransferWithFeeWithWrongAddress(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()

	feeAggregator := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig(vtName, "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	currency := "VT"
	feeAmount := "500000"
	floor := "1"
	cap := "0"

	issuer.SignedInvoke("vt", "emitToken", "101")

	feeSetter.SignedInvoke("vt", "setFee", "VT", feeAmount, "1", "0")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	require.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())

	cfg := &proto.Token{
		TotalEmission: nil,
		Fee: &proto.TokenFee{
			Currency: currency,
			Fee:      []byte(feeAmount),
			Floor:    []byte(floor),
			Cap:      []byte(cap),
		},
		Rates:      nil,
		FeeAddress: []byte("1111"),
	}
	bytes, err := pb.Marshal(cfg)
	require.NoError(t, err)

	stub := ledger.GetStub("vt")
	stub.MockTransactionStart("test")
	err = stub.PutState("tokenMetadata", bytes)
	require.NoError(t, err)

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	require.ErrorContains(t, err, "config fee address has a wrong len. actual 4 but expected 32")

	issuer.BalanceShouldBe("vt", 101)
	user.BalanceShouldBe("vt", 0)
	feeAggregator.BalanceShouldBe("vt", 0)
}

func TestTransferWithFeeWithWrongSymbol(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	feeAggregator := ledger.NewWallet()
	user := ledger.NewWallet()

	currency := "asd"
	feeAmount := "500000"
	floor := "1"
	cap := "0"

	vt := &VT{}
	config := makeBaseTokenConfig(vtName, "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	issuer.SignedInvoke("vt", "emitToken", "101")

	feeSetter.SignedInvoke("vt", "setFee", "VT", feeAmount, "1", "0")

	predict := &Predict{}
	rawResp := issuer.Invoke("vt", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	require.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	cfg := &proto.Token{
		TotalEmission: nil,
		Fee: &proto.TokenFee{
			Currency: currency,
			Fee:      []byte(feeAmount),
			Floor:    []byte(floor),
			Cap:      []byte(cap),
		},
		Rates:      nil,
		FeeAddress: []byte(feeAggregator.Address()),
	}
	bytes, err := pb.Marshal(cfg)
	require.NoError(t, err)

	stub := ledger.GetStub("vt")
	stub.MockTransactionStart("test")
	err = stub.PutState("tokenMetadata", bytes)
	require.NoError(t, err)

	feeAddressSetter.SignedInvoke("vt", "setFeeAddress", feeAggregator.Address())

	err = issuer.RawSignedInvokeWithErrorReturned("vt", "transfer", user.Address(), "100", "")
	require.ErrorContains(t, err, "incorrect fee currency")

	issuer.BalanceShouldBe("vt", 101)
	user.BalanceShouldBe("vt", 0)
	feeAggregator.BalanceShouldBe("vt", 0)
}

func TestAllowedIndustrialBalanceTransfer(t *testing.T) {
	ledger := ma.NewLedger(t)
	issuer := ledger.NewWallet()
	feeAddressSetter := ledger.NewWallet()
	feeSetter := ledger.NewWallet()
	user := ledger.NewWallet()

	vt := &VT{}
	config := makeBaseTokenConfig(vtName, "VT", 8,
		issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())
	ledger.NewCC("vt", vt, config)

	const (
		ba1 = "BA02_GOLDBARLONDON.01"
		ba2 = "BA02_GOLDBARLONDON.02"
	)

	issuer.AddAllowedBalance("vt", ba1, 100000000)
	issuer.AddAllowedBalance("vt", ba2, 100000000)
	issuer.AllowedBalanceShouldBe("vt", ba1, 100000000)
	issuer.AllowedBalanceShouldBe("vt", ba2, 100000000)

	industrialAssets := []*types.MultiSwapAsset{
		{
			Group:  ba1,
			Amount: "50000000",
		},
		{
			Group:  ba2,
			Amount: "100000000",
		},
	}

	rawGA, err := json.Marshal(industrialAssets)
	require.NoError(t, err)

	issuer.SignedInvoke("vt", "allowedIndustrialBalanceTransfer", user.Address(), string(rawGA), "ref")
	issuer.AllowedBalanceShouldBe("vt", ba1, 50000000)
	issuer.AllowedBalanceShouldBe("vt", ba2, 0)
	user.AllowedBalanceShouldBe("vt", ba1, 50000000)
	user.AllowedBalanceShouldBe("vt", ba2, 100000000)
}
