package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/cctransfer"
	"github.com/anoideaopen/foundation/mock"
	pb "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestByCustomerForwardSuccess(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	assert.NoError(t, err)
	m.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	assert.Error(t, err)

	user1.BalanceShouldBe("cc", 550)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 450)
}

func TestByAdminForwardSuccess(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	_ = owner.SignedInvoke("cc", "channelTransferByAdmin", id, "VT", user1.Address(), "CC", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	assert.NoError(t, err)
	m.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	assert.Error(t, err)

	user1.BalanceShouldBe("cc", 550)
	user1.AllowedBalanceShouldBe("vt", "CC", 450)
	user1.CheckGivenBalanceShouldBe("vt", "VT", 0)
	user1.CheckGivenBalanceShouldBe("vt", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 450)
}

func TestCancelForwardSuccess(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	err := user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)

	user1.BalanceShouldBe("cc", 1000)
	user1.CheckGivenBalanceShouldBe("cc", "CC", 0)
	user1.CheckGivenBalanceShouldBe("cc", "VT", 0)
}

func TestByCustomerBackSuccess(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AddGivenBalance("vt", "CC", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "VT", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	assert.NoError(t, err)
	m.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	assert.Error(t, err)

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
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AddGivenBalance("vt", "CC", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = owner.SignedInvoke("cc", "channelTransferByAdmin", id, "VT", user1.Address(), "VT", "450")
	cct := user1.Invoke("cc", "channelTransferFrom", id)

	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cct)
	assert.NoError(t, err)
	m.WaitChTransferTo("vt", id, time.Second*5)
	_ = user1.Invoke("vt", "channelTransferTo", id)

	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("vt", "deleteCCTransferTo", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)
	err = user1.InvokeWithError("vt", "channelTransferTo", id)
	assert.Error(t, err)

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
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddAllowedBalance("cc", "VT", 1000)
	user1.AllowedBalanceShouldBe("cc", "VT", 1000)

	id := uuid.NewString()

	_ = user1.SignedInvoke("cc", "channelTransferByCustomer", id, "VT", "VT", "450")
	err := user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.NoError(t, err)

	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	assert.NoError(t, err)

	err = user1.InvokeWithError("cc", "channelTransferFrom", id)
	assert.Error(t, err)

	user1.AllowedBalanceShouldBe("cc", "VT", 1000)
}

func TestQueryAllTransfersFrom(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
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
		assert.NoError(t, err)
		for _, tr := range res.Ccts {
			_, ok := ids[tr.Id]
			assert.True(t, ok)
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
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()

	// TESTS

	// admin function sent by someone other than admin
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin", id, "VT", user1.Address(), "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrNotFoundAdminKey.Error())

	// the admin sends the transfer to himself
	err = owner.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin", id, "VT", owner.Address(), "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidIDUser.Error())

	// CC-to-CC transfer
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "CC", "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// transferring the wrong tokens
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "FIAT", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// insufficient funds
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "1100")
	assert.EqualError(t, err, "insufficient funds to process")

	// such a transfer is already in place.
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCreateTransferTo(t *testing.T) {
	// preparation
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)
	cctRaw := user1.Invoke("cc", "channelTransferFrom", id)
	cct := new(pb.CCTransfer)
	err = json.Unmarshal([]byte(cctRaw), &cct)
	assert.NoError(t, err)

	// TESTS

	// incorrect data format
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", "(09345345-0934]")
	assert.Error(t, err)

	// the transfer went into the wrong channel
	tempTo := cct.To
	cct.To = "FIAT"
	b, err := json.Marshal(cct)
	assert.NoError(t, err)
	cct.To = tempTo
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// From and To channels are equal
	tempFrom := cct.From
	cct.From = cct.To
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.From = tempFrom
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// token is not equal to one of the channels
	tempToken := cct.Token
	cct.Token = "FIAT"
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.Token = tempToken
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// misdirection of changes in balances
	tempDirect := cct.ForwardDirection
	cct.ForwardDirection = !tempDirect
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.ForwardDirection = tempDirect
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// The transfer is already in place
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	assert.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCancelTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// transfer completed
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailCommitTransferFrom(t *testing.T) { //nolint:dupl
	// preparation
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "commitCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailDeleteTransferFrom(t *testing.T) {
	// preparation
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)

	id := uuid.NewString()
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)

	// TESTS

	// transfer not found
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "deleteCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// the transfer is already committed
	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferNotCommit.Error())
}

func TestFailDeleteTransferTo(t *testing.T) {
	// preparation
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()

	// TESTS

	// transfer not found
	_, _, err := user1.RawChTransferInvokeWithBatch("vt", "deleteCCTransferTo", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())
}

func TestFailQueryAllTransfersFrom(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
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
	assert.NoError(t, err)
	assert.NotEmpty(t, res.Bookmark)

	b = "pfi" + res.Bookmark
	err = user1.InvokeWithError("cc", "channelTransfersFrom", "2", b)
	assert.EqualError(t, err, cctransfer.ErrInvalidBookmark.Error())

	b = res.Bookmark
	err = user1.InvokeWithError("cc", "channelTransfersFrom", "-2", b)
	assert.EqualError(t, err, cctransfer.ErrPageSizeLessOrEqZero.Error())
}
