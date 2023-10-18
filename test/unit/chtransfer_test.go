package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/atomyze-foundation/foundation/core"
	"github.com/atomyze-foundation/foundation/core/cctransfer"
	"github.com/atomyze-foundation/foundation/mock"
	pb "github.com/atomyze-foundation/foundation/proto"
	"github.com/atomyze-foundation/foundation/token"
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
	// подготовка
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

	// ТЕСТЫ

	// админская функция отправлена не админом
	err := user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin", id, "VT", user1.Address(), "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrNotFoundAdminKey.Error())

	// админ отправляет перевод на себя
	err = owner.RawSignedInvokeWithErrorReturned("cc", "channelTransferByAdmin", id, "VT", owner.Address(), "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidIDUser.Error())

	// перевод из канала СС в канал СС
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "CC", "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// перевод не тех токенов
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "FIAT", "450")
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// недостаточно средств
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "1100")
	assert.EqualError(t, err, "insufficient funds to process")

	// такой трансфер уже есть
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.NoError(t, err)
	err = user1.RawSignedInvokeWithErrorReturned("cc", "channelTransferByCustomer", id, "VT", "CC", "450")
	assert.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCreateTransferTo(t *testing.T) {
	// подготовка
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

	// ТЕСТЫ

	// неверный формат данных
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", "(09345345-0934]")
	assert.Error(t, err)

	// трансфер кинули не в тот канал
	tempTo := cct.To
	cct.To = "FIAT"
	b, err := json.Marshal(cct)
	assert.NoError(t, err)
	cct.To = tempTo
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// каналы фром и ту равны
	tempFrom := cct.From
	cct.From = cct.To
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.From = tempFrom
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidChannel.Error())

	// токен не равен одному из каналов
	tempToken := cct.Token
	cct.Token = "FIAT"
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.Token = tempToken
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// неверное направление именения балансов
	tempDirect := cct.ForwardDirection
	cct.ForwardDirection = !tempDirect
	b, err = json.Marshal(cct)
	assert.NoError(t, err)
	cct.ForwardDirection = tempDirect
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", string(b))
	assert.EqualError(t, err, cctransfer.ErrInvalidToken.Error())

	// трансфер уже есть
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("vt", "createCCTransferTo", cctRaw)
	assert.EqualError(t, err, cctransfer.ErrIDTransferExist.Error())
}

func TestFailCancelTransferFrom(t *testing.T) { //nolint:dupl
	// подготовка
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

	// ТЕСТЫ

	// трансфер не найден
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// трансфер закомичен
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "cancelCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailCommitTransferFrom(t *testing.T) { //nolint:dupl
	// подготовка
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

	// ТЕСТЫ

	// трансфер не найден
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "commitCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// трансфер уже закомичен
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.NoError(t, err)
	_, _, err = user1.RawChTransferInvoke("cc", "commitCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferCommit.Error())
}

func TestFailDeleteTransferFrom(t *testing.T) {
	// подготовка
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

	// ТЕСТЫ

	// трансфер не найден
	_, _, err = user1.RawChTransferInvokeWithBatch("cc", "deleteCCTransferFrom", uuid.NewString())
	assert.EqualError(t, err, cctransfer.ErrNotFound.Error())

	// трансфер уже закомичен
	_, _, err = user1.RawChTransferInvoke("cc", "deleteCCTransferFrom", id)
	assert.EqualError(t, err, cctransfer.ErrTransferNotCommit.Error())
}

func TestFailDeleteTransferTo(t *testing.T) {
	// подготовка
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	vt := token.BaseToken{
		Symbol: "VT",
	}
	m.NewChainCode("vt", &vt, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()

	// ТЕСТЫ

	// трансфер не найден
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
