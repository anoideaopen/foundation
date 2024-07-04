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

// FiatToken - base struct
type FiatToken struct {
	token.BaseToken
	service.UnimplementedFiatServiceServer
}

// NewFiatToken creates fiat token
func NewFiatToken() *FiatToken {
	return &FiatToken{
		BaseToken: token.BaseToken{},
	}
}

// TxEmit - emits fiat token
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

func (ft *FiatToken) QueryAllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) (string, error) {
	return "Ok", ft.AllowedBalanceAdd(token, address, amount, reason)
}

func (ft *FiatToken) AddBalanceByAdmin(ctx context.Context, req *service.BalanceAdjustmentRequest) (*emptypb.Empty, error) {
	if grpc.SenderFromContext(ctx) == "" {
		return nil, errors.New("unauthorized")
	}

	if grpc.StubFromContext(ctx) == nil {
		return nil, errors.New("stub is nil")
	}

	value, _ := mbig.NewInt(0).SetString(req.GetAmount().GetValue(), 10)
	return &emptypb.Empty{}, balance.Add(
		grpc.StubFromContext(ctx),
		balance.BalanceTypeToken,
		req.GetAddress().GetBase58Check(),
		"",
		value,
	)
}

func (ft *FiatToken) AddBalanceByAdmin2(ctx context.Context, req *service.BalanceAdjustmentRequest) (*emptypb.Empty, error) {
	return ft.AddBalanceByAdmin(ctx, req)
}
