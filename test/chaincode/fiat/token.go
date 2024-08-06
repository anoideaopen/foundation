package main

import (
	"context"
	"errors"
	mbig "math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/core/types"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/test/chaincode/fiat/service"
	"github.com/anoideaopen/foundation/token"
	"google.golang.org/protobuf/types/known/emptypb"
)

// FiatToken represents a custom smart contract for handling fiat tokens.
// It implements both reflect and gRPC-based methods for interacting with the blockchain.
type FiatToken struct {
	token.BaseToken
	service.UnimplementedFiatServiceServer
}

// NewFiatToken creates a new instance of the FiatToken contract.
func NewFiatToken() *FiatToken {
	return &FiatToken{
		BaseToken: token.BaseToken{},
	}
}

// TxEmit emits fiat tokens to a specified address.
// This method is routed using the reflect router, and the first argument is the sender
// who must be authorized to emit tokens.
func (ft *FiatToken) TxEmit(sender *types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New("unauthorized")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	if err := ft.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return ft.EmissionAdd(amount)
}

// QueryAllowedBalanceAdd allows querying the balance addition permission for a given token and address.
// This method is routed using the reflect router.
func (ft *FiatToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", ft.AllowedBalanceAdd(token, address, amount, reason)
}

// AddBalanceByAdmin adjusts the balance of a specified address by an admin.
// This method is routed using the gRPC router, and requires proper context with sender and stub information.
func (ft *FiatToken) AddBalanceByAdmin(ctx context.Context, req *service.BalanceAdjustmentRequest) (*emptypb.Empty, error) {
	// Check if the sender is authorized
	if grpc.SenderFromContext(ctx) == "" {
		return nil, errors.New("unauthorized")
	}

	// Ensure the stub is not nil
	if grpc.StubFromContext(ctx) == nil {
		return nil, errors.New("stub is nil")
	}

	// Convert the amount from the request
	value, _ := mbig.NewInt(0).SetString(req.GetAmount().GetValue(), 10)
	// Adjust the balance
	return &emptypb.Empty{}, balance.Add(
		grpc.StubFromContext(ctx),
		balance.BalanceTypeToken,
		req.GetAddress().GetBase58Check(),
		"",
		value,
	)
}
