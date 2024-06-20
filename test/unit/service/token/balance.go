package token

import (
	"context"
	"errors"
	"math/big"

	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/grpc/svccontext"
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
	if svccontext.Sender(ctx) == "" {
		return nil, errors.New("unauthorized")
	}

	if svccontext.Stub(ctx) == nil {
		return nil, errors.New("stub is nil")
	}

	value, _ := big.NewInt(0).SetString(req.Amount.Value, 10)
	return &emptypb.Empty{}, balance.Add(
		svccontext.Stub(ctx),
		balance.BalanceTypeToken,
		req.Address.Address,
		"",
		value,
	)
}
