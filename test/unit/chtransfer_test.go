package unit

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mock"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures_test"
	"github.com/anoideaopen/foundation/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestByCustomerForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	require.Error(t, err)

	user1.BalanceShouldBe("cc", 550)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 450)

	user1.GivenBalanceShouldBe("cc", "VT", 450)
	resp := user1.Invoke("cc", "givenBalancesWithPagination", "", "100")
	require.Equal(t, "{\"bookmark\":\"\",\"sum\":[{\"key\":\"\",\"value\":\"450\"}],\"records\":[{\"key\":\"\\u00002d\\u0000VT\\u0000\",\"value\":\"450\"}]}", resp)

	resp = user1.Invoke("cc", "tokenBalancesWithPagination", "", "100")
	require.Contains(t, resp, "{\"bookmark\":\"\",\"sum\":[{\"key\":\"\",\"value\":\"550\"}],\"records\":[{\"key\":\"\\u00002b\\u0000")

	resp = user1.Invoke("cc", "lockedTokenBalancesWithPagination", "", "100")
	require.Equal(t, "{\"bookmark\":\"\",\"sum\":[],\"records\":[]}", resp)

	resp = user1.Invoke("vt", "allowedBalancesWithPagination", "", "100")
	require.Contains(t, resp, "{\"bookmark\":\"\",\"sum\":[{\"key\":\"CC\",\"value\":\"450\"}],\"records\":[{\"key\":\"\\u00002c\\u0000")

	resp = user1.Invoke("vt", "lockedAllowedBalancesWithPagination", "", "100")
	require.Equal(t, "{\"bookmark\":\"\",\"sum\":[],\"records\":[]}", resp)
}

func TestByAdminForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeSetter := ledger.NewWallet()

	cc := token.BaseToken{}
	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg := ledger.NewCC("cc", &cc, ccConfig)
	require.Empty(t, initMsg)

	vt := token.BaseToken{}
	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg = ledger.NewCC("vt", &vt, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	err := owner.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin",
		id, "VT", user1.Address(), "CC", "450")
	require.NoError(t, err)
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("vt", id, time.Second*5)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	require.Error(t, err)

	user1.BalanceShouldBe("cc", 550)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 450)
}

func TestCancelForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	err := user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)

	user1.BalanceShouldBe("cc", 1000)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)
}

func TestByCustomerBackSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AddGivenBalance("vt", "CC", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "VT", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("vt", "VT", 0)
	user1.AllowedBalanceShouldBe("cc", "VT", 550)
	user1.BalanceShouldBe("vt", 450)
	user1.BalanceShouldBe("cc", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 550)
}

func TestByAdminBackSuccess(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &CustomToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("vt", &CustomToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AddGivenBalance("vt", "CC", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = owner.SignedInvoke("cc", "channelTransferByAdmin", id, "VT", user1.Address(), "VT", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("vt", "VT", 0)
	user1.AllowedBalanceShouldBe("cc", "VT", 550)
	user1.BalanceShouldBe("vt", 450)
	user1.BalanceShouldBe("cc", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 550)
}

func TestCancelBackSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeSetter := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)

	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "VT", "450")
	err := user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("cc", "VT", 1000)
}

func TestQueryAllTransfersFrom(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	ids := make(map[string]struct{})

	id := uuid.NewString()
	ids[id] = struct{}{}
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	ids[id] = struct{}{}
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	ids[id] = struct{}{}
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	ids[id] = struct{}{}
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	ids[id] = struct{}{}
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")

	b := ""
	for {
		resStr := user1.Invoke("cc", "channelTransfersFrom", "2", b)
		res := new(pb.CCTransfers)
		err := json.Unmarshal([]byte(resStr), &res)
		require.NoError(t, err)
		for _, tr := range res.Ccts {
			_, ok := ids[tr.Id]
			require.True(t, ok)
			delete(ids, tr.Id)
		}
		if res.Bookmark == "" {
			break
		}
		b = res.Bookmark
	}
}

func TestFailBeginTransfer(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	// TESTS

	// admin function sent by someone other than admin
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin",
		id, "VT", user1.Address(), "CC", "450")
	require.EqualError(t, err, cctransfer.ErrUnauthorisedNotAdmin.Error())

	// the admin sends the transfer to himself
	err = owner.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin",
		id, "VT", owner.Address(), "CC", "450")
	require.EqualError(t, err, cctransfer.ErrInvalidIDUser.Error())

	// CC-to-CC transfer
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "CC", "CC", "450")
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// transferring the wrong tokens
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "VT", "FIAT", "450")
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// insufficient funds
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "VT", "CC", "1100")
	require.EqualError(t, err, "failed to subtract token balance: insufficient balance")

	// such a transfer is already in place.
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "VT", "CC", "450")
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "VT", "CC", "450")
	require.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCreateTransferTo(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer",
		id, "VT", "CC", "450")
	require.NoError(t, err)

	cctRaw := user1.Invoke("cc", "channelTransferFrom", id)
	cct := new(pb.CCTransfer)
	err = json.Unmarshal([]byte(cctRaw), &cct)
	require.NoError(t, err)

	// TESTS

	// incorrect data format
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", "(09345345-0934]")
	require.Error(t, err)

	// the transfer went into the wrong channel
	tempTo := cct.To
	cct.To = "FIAT"
	b, err := json.Marshal(cct)
	require.NoError(t, err)
	cct.To = tempTo
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// From and To channels are equal
	tempFrom := cct.From
	cct.From = cct.To
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.From = tempFrom
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// token is not equal to one of the channels
	tempToken := cct.Token
	cct.Token = "FIAT"
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.Token = tempToken
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// misdirection of changes in balances
	tempDirect := cct.ForwardDirection
	cct.ForwardDirection = !tempDirect
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.ForwardDirection = tempDirect
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// The transfer is already in place
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	require.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCancelTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// transfer completed
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailCommitTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "commitCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailDeleteTransferFrom(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	cc := token.BaseToken{}
	config := makeBaseTokenConfig("CC Token", "CC", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &cc, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "deleteCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferNotCommit.Error())
}

func TestFailDeleteTransferTo(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	ccConfig := makeBaseTokenConfig("CC Token", "CC", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &token.BaseToken{}, ccConfig)
	require.Empty(t, initMsg)

	vtConfig := makeBaseTokenConfig("VT Token", "VT", 8,
		issuer.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("vt", &token.BaseToken{}, vtConfig)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()

	// TESTS

	// transfer not found
	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "deleteCCTransferTo", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())
}

func TestFailQueryAllTransfersFrom(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	cc := token.BaseToken{}
	config := makeBaseTokenConfig("CC Token", "CC", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("cc", &cc, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")
	id = uuid.NewString()
	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "100")

	b := ""
	resStr := user1.Invoke("cc", "channelTransfersFrom", "2", b)
	res := new(pb.CCTransfers)
	err := json.Unmarshal([]byte(resStr), &res)
	require.NoError(t, err)
	require.NotEmpty(t, res.Bookmark)

	b = "pfi" + res.Bookmark
	err = user1.InvokeWithError("cc", "channelTransfersFrom", "2", b)
	require.EqualError(t, err, cctransfer.ErrInvalidBookmark.Error())

	b = res.Bookmark
	err = user1.InvokeWithError("cc", "channelTransfersFrom", "-2", b)
	require.EqualError(t, err, cctransfer.ErrPageSizeLessOrEqZero.Error())
}

// TestFailForwardByAdmin tries to make channel transfer but
// admin in ContractConfig was not set.
func TestFailForwardByAdmin(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	cfg := pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   "CC",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: owner.Address()},
		},
		Token: &pb.TokenConfig{
			Name:     "CC Token",
			Decimals: 8,
			Issuer:   &pb.Wallet{Address: owner.Address()},
		},
	}

	cfgBytes, err := protojson.Marshal(&cfg)
	require.NoError(t, err)

	initMsg := ledger.NewCC("cc", &token.BaseToken{}, string(cfgBytes))
	require.Empty(t, initMsg)

	// unset admin and overwrite config data
	cfg.Contract.Admin = nil
	cfgBytes, err = protojson.Marshal(&cfg)
	require.NoError(t, err)
	ledger.GetStub("cc").State["__config"] = cfgBytes

	user1 := ledger.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err = owner.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin",
		id, "VT", fixtures_test.AdminAddr, "CC", "450")
	require.EqualError(t, err, cctransfer.ErrAdminNotSet.Error())
}

func TestMultiTransferByCustomerItemsLen(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	items := make([]core.TransferItem, 0)
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", uuid.NewString(), "IT2", string(itemsJSON))
	require.EqualError(t, err, "invalid argument transfer items count found 0 but expected from 1 to 100")

	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", uuid.NewString(), "IT2", string(itemsJSON))
	require.NoError(t, err)

	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", uuid.NewString(), "IT2", string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrInvalidTokenAlreadyExists.Error())

	items = make([]core.TransferItem, 0, 100)
	for i := 0; i < 100; i++ {
		itemToken := fmt.Sprintf("IT1_%d", i)
		items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
		user1.AddTokenBalance("it1", itemToken, 1)
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", uuid.NewString(), "IT2", string(itemsJSON))
	require.NoError(t, err)

	items = make([]core.TransferItem, 0, 100)
	for i := 0; i < 101; i++ {
		itemToken := fmt.Sprintf("IT1_%d", i)
		items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
		user1.AddTokenBalance("it1", itemToken, 1)
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", uuid.NewString(), "IT2", string(itemsJSON))
	require.EqualError(t, err, "invalid argument transfer items count found 101 but expected from 1 to 100")
}

func TestMultiTransferByAdminItemsLen(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeSetter := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	items := make([]core.TransferItem, 0)
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", uuid.NewString(), "IT2", user1.Address(), string(itemsJSON))
	require.EqualError(t, err, "invalid argument transfer items count found 0 but expected from 1 to 100")

	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", uuid.NewString(), "IT2", user1.Address(), string(itemsJSON))
	require.NoError(t, err)

	items = make([]core.TransferItem, 0, 100)
	for i := 0; i < 100; i++ {
		itemToken := fmt.Sprintf("IT1_%d", i)
		items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
		user1.AddTokenBalance("it1", itemToken, 1)
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", uuid.NewString(), "IT2", user1.Address(), string(itemsJSON))
	require.NoError(t, err)

	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", uuid.NewString(), "IT2", user1.Address(), string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrInvalidTokenAlreadyExists.Error())

	items = make([]core.TransferItem, 0, 100)
	for i := 0; i < 101; i++ {
		itemToken := fmt.Sprintf("IT1_%d", i)
		items = append(items, core.TransferItem{Token: itemToken, Amount: new(big.Int).SetInt64(1)})
		user1.AddTokenBalance("it1", itemToken, 1)
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", uuid.NewString(), "IT2", user1.Address(), string(itemsJSON))
	require.EqualError(t, err, "invalid argument transfer items count found 101 but expected from 1 to 100")
}

func TestMultiTransferByCustomerForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()

	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	itt := user1.Invoke("it1", "channelTransferFrom", id)

	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", itt)
	require.NoError(t, err)
	ledger.WaitChTransferTo("it2", id, time.Second*5)
	_ = user1.Invoke("it2", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it2", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it1", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("it2", "channelTransferTo", id)
	require.Error(t, err)

	balanceResponse := owner.Invoke("it1", "industrialBalanceGet", user1.Address())
	balanceIT1group1, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "1")
	require.NoError(t, err)
	require.Equal(t, "550", balanceIT1group1)
	balanceIT2, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "2")
	require.NoError(t, err)
	require.Equal(t, "1100", balanceIT2)
	user1.AllowedBalanceShouldBe("it2", "IT1_1", 450)
	user1.AllowedBalanceShouldBe("it2", "IT1_2", 900)
	user1.CheckGivenBalanceShouldBe("it2", "IT2", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2", 1350)
}

func TestMultiTransferByAdminForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeSetter := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", id, "IT2", user1.Address(), string(itemsJSON))
	require.NoError(t, err)
	cct := user1.Invoke("it1", "channelTransferFrom", id)

	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("it2", id, time.Second*5)
	err = user1.InvokeWithError("it2", "channelTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it2", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it1", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("it2", "channelTransferTo", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("it2", "IT1_1", 450)
	user1.AllowedBalanceShouldBe("it2", "IT1_2", 900)
}

func TestMultiTransferCancelForwardSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()

	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "cancelCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)

	balanceResponse := owner.Invoke("it1", "industrialBalanceGet", user1.Address())
	balanceIT1group1, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "1")
	require.NoError(t, err)
	require.Equal(t, "1000", balanceIT1group1)
	balanceIT1group2, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "2")
	require.NoError(t, err)
	require.Equal(t, "2000", balanceIT1group2)
	user1.CheckGivenBalanceShouldBe("it1", "IT1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2", 0)
}

func TestMultiTransferByCustomerBackSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", owner.Address(), nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("it1", "IT2_1", 1000)
	user1.AddAllowedBalance("it1", "IT2_2", 2000)
	user1.AddGivenBalance("it2", "IT1", 3000)
	user1.AllowedBalanceShouldBe("it1", "IT2_1", 1000)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 2000)

	id := uuid.NewString()

	items := []core.TransferItem{
		{Token: "IT2_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT2_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	cct := user1.Invoke("it1", "channelTransferFrom", id)

	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("it2", id, time.Second*5)
	_ = user1.Invoke("it2", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it2", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it1", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("it2", "channelTransferTo", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("it2", "IT2", 0)
	user1.AllowedBalanceShouldBe("it1", "IT2_1", 550)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 1100)

	balanceResponse := owner.Invoke("it2", "industrialBalanceGet", user1.Address())
	balanceIT1group1, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "1")
	require.NoError(t, err)
	require.Equal(t, "450", balanceIT1group1)
	balanceIT1group2, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "2")
	require.NoError(t, err)
	require.Equal(t, "900", balanceIT1group2)

	user1.CheckGivenBalanceShouldBe("it1", "IT1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2_1", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT2_1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2_2", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT2_2", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT1", 1650)
}

func TestMultiTransferByAdminBackSuccess(t *testing.T) {
	t.Parallel()

	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("it1", "IT2_1", 1000)
	user1.AddAllowedBalance("it1", "IT2_2", 2000)
	user1.AddGivenBalance("it2", "IT1", 3000)
	user1.AllowedBalanceShouldBe("it1", "IT2_1", 1000)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT2_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT2_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", id, "IT2", user1.Address(), string(itemsJSON))
	require.NoError(t, err)
	cct := user1.Invoke("it1", "channelTransferFrom", id)

	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", cct)
	require.NoError(t, err)
	ledger.WaitChTransferTo("it2", id, time.Second*5)
	_ = user1.Invoke("it2", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it2", "deleteCCTransferTo", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("it1", "deleteCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)
	err = user1.InvokeWithError("it2", "channelTransferTo", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("it2", "IT2", 0)
	user1.AllowedBalanceShouldBe("it1", "IT2_1", 550)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 1100)

	balanceResponse := owner.Invoke("it2", "industrialBalanceGet", user1.Address())
	balanceIT1group1, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "1")
	require.NoError(t, err)
	require.Equal(t, "450", balanceIT1group1)
	balanceIT1group2, err := GetIndustrialBalanceFromResponseByGroup(balanceResponse, "2")
	require.NoError(t, err)
	require.Equal(t, "900", balanceIT1group2)

	user1.CheckGivenBalanceShouldBe("it1", "IT1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2_1", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT2_1", 0)
	user1.CheckGivenBalanceShouldBe("it1", "IT2_2", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT2_2", 0)
	user1.CheckGivenBalanceShouldBe("it2", "IT1", 1650)
}

func TestMultiTransferCancelBackSuccess(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()
	feeSetter := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), feeSetter.Address(), "", owner.Address(), nil)

	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddAllowedBalance("it1", "IT2_1", 1000)
	user1.AddAllowedBalance("it1", "IT2_2", 2000)
	user1.AllowedBalanceShouldBe("it1", "IT2_1", 1000)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT2_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT2_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "cancelCCTransferFrom", id)
	require.NoError(t, err)

	err = user1.InvokeWithError("it1", "channelTransferFrom", id)
	require.Error(t, err)

	user1.AllowedBalanceShouldBe("it1", "IT2_1", 1000)
	user1.AllowedBalanceShouldBe("it1", "IT2_2", 2000)
}

func TestMultiTransferQueryAllTransfersFrom(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	ids := make(map[string]struct{})

	id := uuid.NewString()
	ids[id] = struct{}{}
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(100)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(100)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	ids[id] = struct{}{}
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	ids[id] = struct{}{}
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	ids[id] = struct{}{}
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	ids[id] = struct{}{}
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	b := ""
	for {
		resStr := user1.Invoke("it1", "channelTransfersFrom", "2", b)
		res := new(pb.CCTransfers)
		err := json.Unmarshal([]byte(resStr), &res)
		require.NoError(t, err)
		for _, tr := range res.Ccts {
			_, ok := ids[tr.Id]
			require.True(t, ok)
			delete(ids, tr.Id)
		}
		if res.Bookmark == "" {
			break
		}
		b = res.Bookmark
	}
}

func TestMultiTransferFailBeginTransfer(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	// TESTS

	// admin function sent by someone other than admin
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", id, "IT2", user1.Address(), string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrUnauthorisedNotAdmin.Error())

	// the admin sends the transfer to himself
	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", id, "IT2", owner.Address(), string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrInvalidIDUser.Error())

	// IT1-to-IT1 transfer
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT1", string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// transferring the wrong tokens
	items = []core.TransferItem{
		{Token: "FIAT_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "FIAT_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// insufficient funds
	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(2450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(2900)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.EqualError(t, err, "failed to subtract token balance: insufficient balance")

	// such a transfer is already in place.
	items = []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err = json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestMultiTransferFailCreateTransferTo(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		owner.Address(), "", "", owner.Address(), nil)

	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	cctRaw := user1.Invoke("it1", "channelTransferFrom", id)
	cct := new(pb.CCTransfer)
	err = json.Unmarshal([]byte(cctRaw), &cct)
	require.NoError(t, err)

	// TESTS

	// incorrect data format
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", "(09345345-0934]")
	require.Error(t, err)

	// the transfer went into the wrong channel
	tempTo := cct.To
	cct.To = "FIAT"
	b, err := json.Marshal(cct)
	require.NoError(t, err)
	cct.To = tempTo
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// From and To channels are equal
	tempFrom := cct.From
	cct.From = cct.To
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.From = tempFrom
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// token is not equal to one of the channels
	tempToken := cct.Token
	cct.GetItems()[0].Token = "FIAT"
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.Token = tempToken
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// misdirection of changes in balances
	tempDirect := cct.ForwardDirection
	cct.ForwardDirection = !tempDirect
	b, err = json.Marshal(cct)
	require.NoError(t, err)
	cct.ForwardDirection = tempDirect
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", string(b))
	require.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// The transfer is already in place
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", cctRaw)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("it2", "createCCTransferTo", cctRaw)
	require.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestMultiTransferFailCancelTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "cancelCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// transfer completed
	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "cancelCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestMultiTransferFailCommitTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		owner.Address(), "", "", "", nil)

	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "commitCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.NoError(t, err)
	_, _, err = user1.RawChTransferInvoke("it1", "commitCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestMultiTransferFailDeleteTransferFrom(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}
	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("it1", "deleteCCTransferFrom", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("it1", "deleteCCTransferFrom", id)
	require.EqualError(t, err, cctransfer.ErrTransferNotCommit.Error())
}

func TestMultiTransferFailDeleteTransferTo(t *testing.T) {
	// preparation
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	it1Config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, it1Config)
	require.Empty(t, initMsg)

	it2Config := makeBaseTokenConfig("IT2 Token", "IT2", 8,
		issuer.Address(), "", "", "", nil)
	initMsg = ledger.NewCC("it2", &TestToken{}, it2Config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()

	// TESTS

	// transfer not found
	_, _, err := user1.RawChTransferInvokeWithBatch("it2", "deleteCCTransferTo", uuid.NewString())
	require.EqualError(t, err, cctransfer.ErrNotFound.Error())
}

func TestMultiTransferFailQueryAllTransfersFrom(t *testing.T) {
	ledger := mock.NewLedger(t)
	issuer := ledger.NewWallet()

	config := makeBaseTokenConfig("IT1 Token", "IT1", 8,
		issuer.Address(), "", "", "", nil)
	initMsg := ledger.NewCC("it1", &TestToken{}, config)
	require.Empty(t, initMsg)

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()

	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(100)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(100)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)
	id = uuid.NewString()
	err = user1.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByCustomer", id, "IT2", string(itemsJSON))
	require.NoError(t, err)

	b := ""
	resStr := user1.Invoke("it1", "channelTransfersFrom", "2", b)
	res := new(pb.CCTransfers)
	err = json.Unmarshal([]byte(resStr), &res)
	require.NoError(t, err)
	require.NotEmpty(t, res.Bookmark)

	b = "pfi" + res.Bookmark
	err = user1.InvokeWithError("it1", "channelTransfersFrom", "2", b)
	require.EqualError(t, err, cctransfer.ErrInvalidBookmark.Error())

	b = res.Bookmark
	err = user1.InvokeWithError("it1", "channelTransfersFrom", "-2", b)
	require.EqualError(t, err, cctransfer.ErrPageSizeLessOrEqZero.Error())
}

// TestFailForwardByAdmin tries to make channel transfer but
// admin in ContractConfig was not set.
func TestMultiTransferFailForwardByAdmin(t *testing.T) {
	ledger := mock.NewLedger(t)
	owner := ledger.NewWallet()

	cfg := pb.Config{
		Contract: &pb.ContractConfig{
			Symbol:   "IT1",
			RobotSKI: fixtures_test.RobotHashedCert,
			Admin:    &pb.Wallet{Address: owner.Address()},
		},
		Token: &pb.TokenConfig{
			Name:     "IT1 Token",
			Decimals: 8,
			Issuer:   &pb.Wallet{Address: owner.Address()},
		},
	}

	cfgBytes, err := protojson.Marshal(&cfg)
	require.NoError(t, err)

	initMsg := ledger.NewCC("it1", &TestToken{}, string(cfgBytes))
	require.Empty(t, initMsg)

	// unset admin and overwrite config data
	cfg.Contract.Admin = nil
	cfgBytes, err = protojson.Marshal(&cfg)
	require.NoError(t, err)
	ledger.GetStub("it1").State["__config"] = cfgBytes

	user1 := ledger.NewWallet()
	user1.AddTokenBalance("it1", "IT1_1", 1000)
	user1.AddTokenBalance("it1", "IT1_2", 2000)

	id := uuid.NewString()
	items := []core.TransferItem{
		{Token: "IT1_1", Amount: new(big.Int).SetInt64(450)},
		{Token: "IT1_2", Amount: new(big.Int).SetInt64(900)},
	}

	itemsJSON, err := json.Marshal(items)
	require.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("it1", "channelMultiTransferByAdmin", id, "IT2",
		fixtures_test.AdminAddr, string(itemsJSON))
	require.EqualError(t, err, cctransfer.ErrAdminNotSet.Error())
}
