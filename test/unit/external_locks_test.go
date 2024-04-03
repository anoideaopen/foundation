package unit

import (
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/assert"
)

func TestExternalLockUnlock(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)
	user1.AddAllowedBalance("cc", "vt", 1000)

	request1 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "cc",
		Amount:  "600",
		Reason:  "test1",
		Docs:    nil,
		Payload: nil,
	}

	request2 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "vt",
		Amount:  "600",
		Reason:  "test2",
		Docs:    nil,
		Payload: nil,
	}

	data1, err := json.Marshal(request1)
	assert.NoError(t, err)

	data2, err := json.Marshal(request2)
	assert.NoError(t, err)

	idToken := owner.SignedInvoke("cc", "lockTokenBalance", string(data1))
	idAllowed := owner.SignedInvoke("cc", "lockAllowedBalance", string(data2))

	err = owner.InvokeWithError("cc", "getLockedTokenBalance", idToken)
	assert.NoError(t, err)
	err = owner.InvokeWithError("cc", "getLockedAllowedBalance", idAllowed)
	assert.NoError(t, err)

	user1.BalanceShouldBe("cc", 400)
	user1.AllowedBalanceShouldBe("cc", "vt", 400)

	request1.Id = idToken
	request1.Amount = "150"
	request2.Id = idAllowed
	request2.Amount = "150"

	data1, err = json.Marshal(request1)
	assert.NoError(t, err)

	data2, err = json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockTokenBalance", string(data1))
	assert.NoError(t, err)
	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockAllowedBalance", string(data2))
	assert.NoError(t, err)

	user1.BalanceShouldBe("cc", 550)
	user1.AllowedBalanceShouldBe("cc", "vt", 550)

	err = owner.InvokeWithError("cc", "getLockedTokenBalance", idToken)
	assert.NoError(t, err)
	err = owner.InvokeWithError("cc", "getLockedAllowedBalance", idAllowed)
	assert.NoError(t, err)

	request1.Amount = "450"
	request2.Amount = "450"

	data1, err = json.Marshal(request1)
	assert.NoError(t, err)

	data2, err = json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockTokenBalance", string(data1))
	assert.NoError(t, err)
	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockAllowedBalance", string(data2))
	assert.NoError(t, err)

	err = owner.InvokeWithError("cc", "getLockedTokenBalance", idToken)
	assert.ErrorContains(t, err, core.ErrLockNotExists.Error())
	err = owner.InvokeWithError("cc", "getLockedAllowedBalance", idAllowed)
	assert.ErrorContains(t, err, core.ErrLockNotExists.Error())

	user1.BalanceShouldBe("cc", 1000)
	user1.AllowedBalanceShouldBe("cc", "vt", 1000)
}

func TestNotAdminFailedLockUnlock(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)
	user1.AddAllowedBalance("cc", "vt", 1000)

	request1 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "cc",
		Amount:  "600",
		Reason:  "test1",
		Docs:    nil,
		Payload: nil,
	}

	request2 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "vt",
		Amount:  "600",
		Reason:  "test2",
		Docs:    nil,
		Payload: nil,
	}

	data1, err := json.Marshal(request1)
	assert.NoError(t, err)

	data2, err := json.Marshal(request2)
	assert.NoError(t, err)

	err = user1.RawSignedInvokeWithErrorReturned("cc", "lockTokenBalance", string(data1))
	assert.EqualError(t, err, core.ErrPlatformAdminOnly.Error())
	err = user1.RawSignedInvokeWithErrorReturned("cc", "lockAllowedBalance", string(data2))
	assert.EqualError(t, err, core.ErrPlatformAdminOnly.Error())

	err = user1.RawSignedInvokeWithErrorReturned("cc", "unlockTokenBalance", string(data1))
	assert.EqualError(t, err, core.ErrPlatformAdminOnly.Error())
	err = user1.RawSignedInvokeWithErrorReturned("cc", "unlockAllowedBalance", string(data2))
	assert.EqualError(t, err, core.ErrPlatformAdminOnly.Error())
}

func TestFailedMoreLockThenBalance(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)
	user1.AddAllowedBalance("cc", "vt", 1000)

	request1 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "cc",
		Amount:  "1100",
		Reason:  "test1",
		Docs:    nil,
		Payload: nil,
	}

	request2 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "vt",
		Amount:  "1100",
		Reason:  "test2",
		Docs:    nil,
		Payload: nil,
	}

	data1, err := json.Marshal(request1)
	assert.NoError(t, err)

	data2, err := json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "lockTokenBalance", string(data1))
	assert.EqualError(t, err, core.ErrInsufficientFunds.Error())
	err = owner.RawSignedInvokeWithErrorReturned("cc", "lockAllowedBalance", string(data2))
	assert.EqualError(t, err, core.ErrInsufficientFunds.Error())
}

func TestFailedCreateTwiceLock(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)
	user1.AddAllowedBalance("cc", "vt", 1000)

	request1 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "cc",
		Amount:  "600",
		Reason:  "test1",
		Docs:    nil,
		Payload: nil,
	}

	request2 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "vt",
		Amount:  "600",
		Reason:  "test2",
		Docs:    nil,
		Payload: nil,
	}

	data1, err := json.Marshal(request1)
	assert.NoError(t, err)

	data2, err := json.Marshal(request2)
	assert.NoError(t, err)

	idToken := owner.SignedInvoke("cc", "lockTokenBalance", string(data1))
	idAllowed := owner.SignedInvoke("cc", "lockAllowedBalance", string(data2))

	request1.Id = idToken
	request2.Id = idAllowed

	data1, err = json.Marshal(request1)
	assert.NoError(t, err)

	data2, err = json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "lockTokenBalance", string(data1))
	assert.EqualError(t, err, core.ErrAlredyExist.Error())
	err = owner.RawSignedInvokeWithErrorReturned("cc", "lockAllowedBalance", string(data2))
	assert.EqualError(t, err, core.ErrAlredyExist.Error())
}

func TestFailedUnlock(t *testing.T) {
	m := mock.NewLedger(t)
	owner := m.NewWallet()
	cc := token.BaseToken{
		Symbol: "CC",
	}
	m.NewChainCode("cc", &cc, &core.ContractOptions{NonceTTL: 50}, nil, owner.Address())

	user1 := m.NewWallet()
	user1.AddBalance("cc", 1000)
	user1.AddAllowedBalance("cc", "vt", 1000)

	request1 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "cc",
		Amount:  "600",
		Reason:  "test1",
		Docs:    nil,
		Payload: nil,
	}

	request2 := &proto.BalanceLockRequest{
		Address: user1.Address(),
		Token:   "vt",
		Amount:  "600",
		Reason:  "test2",
		Docs:    nil,
		Payload: nil,
	}

	data1, err := json.Marshal(request1)
	assert.NoError(t, err)

	data2, err := json.Marshal(request2)
	assert.NoError(t, err)

	idToken := owner.SignedInvoke("cc", "lockTokenBalance", string(data1))
	idAllowed := owner.SignedInvoke("cc", "lockAllowedBalance", string(data2))

	request1.Id = idToken
	request1.Amount = "610"
	request2.Id = idAllowed
	request2.Amount = "610"

	data1, err = json.Marshal(request1)
	assert.NoError(t, err)
	data2, err = json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockTokenBalance", string(data1))
	assert.EqualError(t, err, core.ErrInsufficientFunds.Error())
	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockAllowedBalance", string(data2))
	assert.EqualError(t, err, core.ErrInsufficientFunds.Error())

	request1.Amount = "-100"
	request2.Amount = "-100"

	data1, err = json.Marshal(request1)
	assert.NoError(t, err)
	data2, err = json.Marshal(request2)
	assert.NoError(t, err)

	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockTokenBalance", string(data1))
	assert.EqualError(t, err, "amount should be positive")
	err = owner.RawSignedInvokeWithErrorReturned("cc", "unlockAllowedBalance", string(data2))
	assert.EqualError(t, err, "amount should be positive")
}
