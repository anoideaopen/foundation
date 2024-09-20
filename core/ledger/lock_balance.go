package ledger

import (
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"strings"
)

const (
	DefaultAllowedBalanceLockReason   = "allowed balance lock"
	DefaultAllowedBalanceUnlockReason = "allowed balance unlock"

	DefaultIndustrialBalanceLockReason   = "industrial balance lock"
	DefaultIndustrialBalanceUnlockReason = "industrial balance unlock"
)

type lockOpt struct {
	reason string
}

type LockOpt func(*lockOpt)

func WithLockReason(reason string) LockOpt {
	return func(bo *lockOpt) {
		bo.reason = reason
	}
}

func applyLockOpts(opts []LockOpt) *lockOpt {
	o := &lockOpt{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func AllowedBalanceLock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	opts ...LockOpt,
) error {
	opt := applyLockOpts(opts)
	if opt.reason == `` {
		opt.reason = DefaultAllowedBalanceLockReason
	}

	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol,
			address,
			address,
			amount,
			opt.reason,
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowed,
		address.String(),
		balance.BalanceTypeAllowedLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func AllowedBalanceUnlock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	opts ...LockOpt,
) error {
	opt := applyLockOpts(opts)
	if opt.reason == `` {
		opt.reason = DefaultAllowedBalanceUnlockReason
	}

	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol,
			address,
			address,
			amount,
			opt.reason,
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeAllowedLocked,
		address.String(),
		balance.BalanceTypeAllowed,
		address.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceLock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	opts ...LockOpt,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]

	opt := applyLockOpts(opts)
	if opt.reason == `` {
		opt.reason = DefaultIndustrialBalanceLockReason
	}

	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			address,
			amount,
			opt.reason,
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeToken,
		address.String(),
		balance.BalanceTypeTokenLocked,
		address.String(),
		token,
		&amount.Int,
	)
}

func IndustrialBalanceUnlock(
	stub shim.ChaincodeStubInterface,
	symbol string,
	token string,
	address *types.Address,
	amount *big.Int,
	opts ...LockOpt,
) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]

	opt := applyLockOpts(opts)
	if opt.reason == `` {
		opt.reason = DefaultIndustrialBalanceUnlockReason
	}
	if stub, ok := stub.(Accounting); ok {
		stub.AddAccountingRecord(
			symbol+"_"+token,
			address,
			address,
			amount,
			opt.reason,
		)
	}
	return balance.Move(
		stub,
		balance.BalanceTypeTokenLocked,
		address.String(),
		balance.BalanceTypeToken,
		address.String(),
		token,
		&amount.Int,
	)
}
