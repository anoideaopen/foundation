package token

import (
	"context"
	"errors"
	"math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/grpc/grpcctx"
	"github.com/anoideaopen/foundation/test/unit/service"
	"github.com/anoideaopen/foundation/token"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Balance struct {
	token.BaseToken
	service.UnimplementedBalanceServiceServer
}

func (b *Balance) AddBalanceByAdmin(
	ctx context.Context,
	req *service.BalanceAdjustmentRequest,
) (*emptypb.Empty, error) {
	if grpcctx.Sender(ctx) == "" {
		return nil, errors.New("unauthorized")
	}

	if grpcctx.Stub(ctx) == nil {
		return nil, errors.New("stub is nil")
	}

	value, _ := big.NewInt(0).SetString(req.GetAmount().GetValue(), 10)
	return &emptypb.Empty{}, balance.Add(
		grpcctx.Stub(ctx),
		balance.BalanceTypeToken,
		req.GetAddress().GetAddress(),
		"",
		value,
	)
}
