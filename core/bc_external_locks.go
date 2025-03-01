package core

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/proto"
)

var (
	// ErrBigIntFromString - error on big int from string
	ErrBigIntFromString = errors.New("big int from string")
	// ErrPlatformAdminOnly - error on platform admin only
	ErrPlatformAdminOnly = errors.New("platform admin only")
	// ErrEmptyLockID - error on empty lock id
	ErrEmptyLockID = errors.New("empty lock id")
	// ErrReason - error on reason
	ErrReason = errors.New("empty reason")
	// ErrLockNotExists - error on lock not exists
	ErrLockNotExists = errors.New("lock not exists")
	// ErrAddressRequired - error on address required
	ErrAddressRequired = errors.New("address required")
	// ErrAmountRequired - error on amount required
	ErrAmountRequired = errors.New("amount required")
	// ErrTokenTickerRequired - error on token ticker required
	ErrTokenTickerRequired = errors.New("token ticker required")
	// ErrAlreadyExist - error on already exist
	ErrAlreadyExist = errors.New("lock already exist")
	// ErrInsufficientFunds - error on insufficient funds
	ErrInsufficientFunds    = errors.New("insufficient balance")
	ErrAdminNotSet          = errors.New("admin is not set in contract config")
	ErrUnauthorisedNotAdmin = errors.New("unauthorised, sender is not an admin")
)

const (
	// BalanceTokenLockedEvent - event on token balance locked
	BalanceTokenLockedEvent = "BalanceTokenLocked"
	// BalanceTokenUnlockedEvent - event on token balance unlocked
	BalanceTokenUnlockedEvent = "BalanceTokenUnlocked"
	// BalanceAllowedLockedEvent - event on allowed balance locked
	BalanceAllowedLockedEvent = "BalanceAllowedLocked"
	// BalanceAllowedUnlockedEvent - event on allowed balance unlocked
	BalanceAllowedUnlockedEvent = "BalanceAllowedUnlocked"
)

// TxLockTokenBalance - blocks tokens on the user's token balance
// method is called by the chaincode admin, the input is BalanceLockRequest
func (bc *BaseContract) TxLockTokenBalance(
	sender *types.Sender,
	req *proto.BalanceLockRequest,
) error {
	if req.GetId() == "" {
		req.Id = bc.GetStub().GetTxID()
	}

	err := bc.verifyLockedArgs(sender, req)
	if err != nil {
		return err
	}

	// Check what's already there
	_, err = bc.getLockedTokenBalance(req.GetId())
	if err == nil {
		return ErrAlreadyExist
	}

	address, err := types.AddrFromBase58Check(req.GetAddress())
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	if err = bc.TokenBalanceLock(address, amount); err != nil {
		return err
	}

	// state record with balance lock details
	balanceLock := &proto.TokenBalanceLock{
		Id:            req.GetId(),
		Address:       req.GetAddress(),
		Token:         req.GetToken(),
		InitAmount:    req.GetAmount(),
		CurrentAmount: req.GetAmount(),
		Reason:        req.GetReason(),
		Docs:          req.GetDocs(),
		Payload:       req.GetPayload(),
	}

	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{balanceLock.GetId()})
	if err != nil {
		return fmt.Errorf("create key: %w", err)
	}

	data, err := json.Marshal(balanceLock)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	balanceLockedEvent := &proto.TokenBalanceLocked{
		Id:      balanceLock.GetId(),
		Address: balanceLock.GetAddress(),
		Token:   balanceLock.GetToken(),
		Amount:  balanceLock.GetCurrentAmount(),
		Reason:  balanceLock.GetReason(),
		Docs:    balanceLock.GetDocs(),
		Payload: balanceLock.GetPayload(),
	}
	event, err := json.Marshal(balanceLockedEvent)
	if err != nil {
		return err
	}

	if err = bc.GetStub().SetEvent(BalanceTokenLockedEvent, event); err != nil {
		return err
	}

	return bc.GetStub().PutState(key, data)
}

// TxUnlockTokenBalance - unblocks (fully or partially) tokens on the user's token balance
// method is called by the chaincode admin, the input is BalanceLockRequest
func (bc *BaseContract) TxUnlockTokenBalance( //nolint:funlen
	sender *types.Sender,
	req *proto.BalanceLockRequest,
) error {
	err := bc.verifyLockedArgs(sender, req)
	if err != nil {
		return err
	}

	// Check what's already there
	balanceLock, err := bc.getLockedTokenBalance(req.GetId())
	if err != nil {
		return err
	}

	address, err := types.AddrFromBase58Check(req.GetAddress())
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	cur, ok := new(big.Int).SetString(balanceLock.GetCurrentAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	isDelete := false
	c := cur.Cmp(amount)
	switch {
	case c < 0:
		return ErrInsufficientFunds
	case c == 0:
		isDelete = true
	}

	if err = bc.TokenBalanceUnlock(address, amount); err != nil {
		return err
	}

	// state record with balance lock details
	balanceLock.CurrentAmount = new(big.Int).Sub(cur, amount).String()

	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{balanceLock.GetId()})
	if err != nil {
		return fmt.Errorf("create key: %w", err)
	}

	data, err := json.Marshal(balanceLock)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	balanceLockedEvent := &proto.TokenBalanceUnlocked{
		Id:                balanceLock.GetId(),
		Address:           balanceLock.GetAddress(),
		Token:             balanceLock.GetToken(),
		Amount:            balanceLock.GetCurrentAmount(),
		Reason:            balanceLock.GetReason(),
		Docs:              balanceLock.GetDocs(),
		Payload:           balanceLock.GetPayload(),
		CompleteOperation: isDelete,
	}
	event, err := json.Marshal(balanceLockedEvent)
	if err != nil {
		return err
	}

	if err = bc.GetStub().SetEvent(BalanceTokenUnlockedEvent, event); err != nil {
		return err
	}

	if isDelete {
		return bc.GetStub().DelState(key)
	}

	return bc.GetStub().PutState(key, data)
}

// QueryGetLockedTokenBalance - returns an existing balance token lock TokenBalanceLock
func (bc *BaseContract) QueryGetLockedTokenBalance(
	lockID string,
) (*proto.TokenBalanceLock, error) {
	return bc.getLockedTokenBalance(lockID)
}

// TxLockAllowedBalance - blocks tokens on the user's allowedbalance
// method calls the chaincode admin, the input is a BalanceLockRequest
func (bc *BaseContract) TxLockAllowedBalance(
	sender *types.Sender,
	req *proto.BalanceLockRequest,
) error {
	if req.GetId() == "" {
		req.Id = bc.GetStub().GetTxID()
	}

	err := bc.verifyLockedArgs(sender, req)
	if err != nil {
		return err
	}

	// Check what's already there
	_, err = bc.getLockedAllowedBalance(req.GetId())
	if err == nil {
		return ErrAlreadyExist
	}

	address, err := types.AddrFromBase58Check(req.GetAddress())
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	if err = bc.AllowedBalanceLock(req.GetToken(), address, amount); err != nil {
		return err
	}

	// state record with balance lock details
	balanceLock := &proto.AllowedBalanceLock{
		Id:            req.GetId(),
		Address:       req.GetAddress(),
		Token:         req.GetToken(),
		InitAmount:    req.GetAmount(),
		CurrentAmount: req.GetAmount(),
		Reason:        req.GetReason(),
		Docs:          req.GetDocs(),
		Payload:       req.GetPayload(),
	}

	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{balanceLock.GetId()})
	if err != nil {
		return fmt.Errorf("create key: %w", err)
	}

	data, err := json.Marshal(balanceLock)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	balanceLockedEvent := &proto.AllowedBalanceLocked{
		Id:      balanceLock.GetId(),
		Address: balanceLock.GetAddress(),
		Token:   balanceLock.GetToken(),
		Amount:  balanceLock.GetCurrentAmount(),
		Reason:  balanceLock.GetReason(),
		Docs:    balanceLock.GetDocs(),
		Payload: balanceLock.GetPayload(),
	}
	event, err := json.Marshal(balanceLockedEvent)
	if err != nil {
		return err
	}

	if err = bc.GetStub().SetEvent(BalanceAllowedLockedEvent, event); err != nil {
		return err
	}

	return bc.GetStub().PutState(key, data)
}

// TxUnlockAllowedBalance - unblocks (fully or partially) tokens on the user's allowedbalance
// method calls the chaincode admin, the input is a BalanceLockRequest
func (bc *BaseContract) TxUnlockAllowedBalance( //nolint:funlen
	sender *types.Sender,
	req *proto.BalanceLockRequest,
) error {
	err := bc.verifyLockedArgs(sender, req)
	if err != nil {
		return err
	}

	// Check what's already there
	balanceLock, err := bc.getLockedAllowedBalance(req.GetId())
	if err != nil {
		return err
	}

	address, err := types.AddrFromBase58Check(req.GetAddress())
	if err != nil {
		return fmt.Errorf("address: %w", err)
	}

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	cur, ok := new(big.Int).SetString(balanceLock.GetCurrentAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	isDelete := false
	c := cur.Cmp(amount)
	switch {
	case c < 0:
		return ErrInsufficientFunds
	case c == 0:
		isDelete = true
	}

	if err = bc.AllowedBalanceUnLock(balanceLock.GetToken(), address, amount); err != nil {
		return err
	}

	// state record with balance lock details
	balanceLock.CurrentAmount = new(big.Int).Sub(cur, amount).String()

	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{balanceLock.GetId()})
	if err != nil {
		return fmt.Errorf("create key: %w", err)
	}

	data, err := json.Marshal(balanceLock)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	balanceLockedEvent := &proto.AllowedBalanceUnlocked{
		Id:                balanceLock.GetId(),
		Address:           balanceLock.GetAddress(),
		Token:             balanceLock.GetToken(),
		Amount:            balanceLock.GetCurrentAmount(),
		Reason:            balanceLock.GetReason(),
		Docs:              balanceLock.GetDocs(),
		Payload:           balanceLock.GetPayload(),
		CompleteOperation: isDelete,
	}
	event, err := json.Marshal(balanceLockedEvent)
	if err != nil {
		return err
	}

	if err = bc.GetStub().SetEvent(BalanceAllowedUnlockedEvent, event); err != nil {
		return err
	}

	if isDelete {
		return bc.GetStub().DelState(key)
	}
	return bc.GetStub().PutState(key, data)
}

// QueryGetLockedAllowedBalance - returns the existing blocking of the allowedbalance AllowedBalanceLock
func (bc *BaseContract) QueryGetLockedAllowedBalance(
	lockID string,
) (*proto.AllowedBalanceLock, error) {
	return bc.getLockedAllowedBalance(lockID)
}

func (bc *BaseContract) getLockedTokenBalance(lockID string) (*proto.TokenBalanceLock, error) {
	if lockID == "" {
		return nil, ErrEmptyLockID
	}
	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeTokenExternalLocked.String(), []string{lockID})
	if err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}

	data, err := bc.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("get token balance lock from state: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("lock id=%s: %w", lockID, ErrLockNotExists)
	}

	balanceLock := &proto.TokenBalanceLock{}
	if err = json.Unmarshal(data, balanceLock); err != nil {
		return nil, fmt.Errorf("unmarshal token balance lock state: %w", err)
	}

	return balanceLock, nil
}

func (bc *BaseContract) getLockedAllowedBalance(lockID string) (*proto.AllowedBalanceLock, error) {
	if lockID == "" {
		return nil, ErrEmptyLockID
	}
	key, err := bc.GetStub().CreateCompositeKey(balance.BalanceTypeAllowedExternalLocked.String(), []string{lockID})
	if err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}

	data, err := bc.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("get allowed balance lock from state: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("lock id=%s: %w", lockID, ErrLockNotExists)
	}

	balanceLock := &proto.AllowedBalanceLock{}
	if err = json.Unmarshal(data, balanceLock); err != nil {
		return nil, fmt.Errorf("unmarshal allowed balance lock state: %w", err)
	}

	return balanceLock, nil
}

func (bc *BaseContract) verifyLockedArgs(
	sender *types.Sender,
	req *proto.BalanceLockRequest,
) error {
	// Sender verification
	if !bc.ContractConfig().IsAdminSet() {
		return ErrAdminNotSet
	}

	if admin, err := types.AddrFromBase58Check(bc.ContractConfig().GetAdmin().GetAddress()); err == nil {
		if !sender.Equal(admin) {
			return ErrUnauthorisedNotAdmin
		}
	} else {
		return fmt.Errorf("creating admin address: %w", err)
	}

	// Request verification
	if req.GetId() == "" {
		return ErrEmptyLockID
	}

	if req.GetAddress() == "" {
		return ErrAddressRequired
	}

	if req.GetAmount() == "" {
		return ErrAmountRequired
	}

	if req.GetToken() == "" {
		return ErrTokenTickerRequired
	}

	if req.GetReason() == "" {
		return ErrReason
	}

	return nil
}
