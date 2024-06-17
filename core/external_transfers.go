package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/proto"
)

var (
	ErrAmountMustBeGreaterThanZero = errors.New("amount must be greater than zero")
	ErrSameAddresses               = errors.New("from and to addresses must be different")
)

// TxTransferBalance - transfer balance from one address to another address
// by the chaincode admin, the input is TransferRequest.
func (bc *BaseContract) TxTransferBalance(
	sender *types.Sender,
	req *proto.TransferRequest,
) error {
	if !bc.config.IsAdminSet() {
		return ErrAdminNotSet
	}

	if admin, err := types.AddrFromBase58Check(bc.config.GetAdmin().GetAddress()); err == nil {
		if !sender.Equal(admin) {
			return ErrUnauthorisedNotAdmin
		}
	} else {
		return fmt.Errorf("creating admin address: %w", err)
	}

	if req.GetRequestId() == "" {
		req.RequestId = bc.stub.GetTxID()
	}

	fromAddress, err := types.AddrFromBase58Check(req.GetFromAddress())
	if err != nil {
		return fmt.Errorf("from address: %w", err)
	}

	toAddress, err := types.AddrFromBase58Check(req.GetToAddress())
	if err != nil {
		return fmt.Errorf("to address: %w", err)
	}

	if fromAddress.Equal(toAddress) {
		return ErrSameAddresses
	}

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		return ErrBigIntFromString
	}

	if amount.Sign() <= 0 {
		return ErrAmountMustBeGreaterThanZero
	}

	return balance.Move(
		bc.GetStub(),
		balance.BalanceType(req.GetBalanceType()),
		fromAddress.String(),
		balance.BalanceType(req.GetBalanceType()),
		toAddress.String(),
		req.GetToken(),
		amount,
	)
}
