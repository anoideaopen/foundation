package service

import (
	"context"
	"errors"
	"math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/test/unit/token/proto"
	"github.com/anoideaopen/foundation/token"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Balance struct {
	token.BaseToken
	proto.UnimplementedBalanceServiceServer
}

func (b *Balance) AddBalanceByAdmin(
	ctx context.Context,
	req *proto.BalanceAdjustmentRequest,
) (*emptypb.Empty, error) {
	if grpc.SenderFromContext(ctx) == "" {
		return nil, errors.New("unauthorized")
	}

	if grpc.StubFromContext(ctx) == nil {
		return nil, errors.New("stub is nil")
	}

	value, _ := big.NewInt(0).SetString(req.GetAmount().GetValue(), 10)
	return &emptypb.Empty{}, balance.Add(
		grpc.StubFromContext(ctx),
		balance.BalanceTypeToken,
		req.GetAddress().GetBase58Check(),
		"",
		value,
	)
}

func (b *Balance) AddBalanceByAdmin2(
	ctx context.Context,
	req *proto.BalanceAdjustmentRequest,
) (*emptypb.Empty, error) {
	return b.AddBalanceByAdmin(ctx, req)
}

func (b *Balance) HelloWorld(context.Context, *emptypb.Empty) (*proto.HelloWorldResponse, error) {
	return &proto.HelloWorldResponse{
		Message: "Hello World!",
	}, nil
}
