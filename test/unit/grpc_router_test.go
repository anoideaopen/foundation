package unit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/core/routing/reflect"
	"github.com/anoideaopen/foundation/mock"
	mockgrpc "github.com/anoideaopen/foundation/mock/grpc"
	"github.com/anoideaopen/foundation/test/unit/token/proto"
	"github.com/anoideaopen/foundation/test/unit/token/service"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGRPCRouter(t *testing.T) {
	var (
		ledger = mock.NewLedger(t)
		owner  = ledger.NewWallet()
		user1  = ledger.NewWallet()
		ch     = "cc"
	)

	ccConfig := makeBaseTokenConfig(
		"CC Token",
		"CC",
		8,
		owner.Address(),
		"",
		"",
		owner.Address(),
		nil,
	)

	var (
		balanceToken  = &service.Balance{} // gRPC service.
		grpcRouter    = grpc.NewRouter()
		reflectRouter = reflect.MustNewRouter(balanceToken)
	)

	// Register gRPC service.
	proto.RegisterBalanceServiceServer(grpcRouter, balanceToken)

	// Init chaincode.
	initMsg := ledger.NewCC(
		ch,
		balanceToken,
		ccConfig,
		core.WithRouters(reflectRouter, grpcRouter),
	)
	require.Empty(t, initMsg)

	// Prepare request.
	req := &proto.BalanceAdjustmentRequest{
		Address: &proto.Address{
			Base58Check: user1.Address(),
		},
		Amount: &proto.BigInt{
			Value: "1000",
		},
		Reason: "Test reason",
	}

	// Add balance by admin with a client by URL.
	client := proto.NewBalanceServiceClient(mockgrpc.NewMockClientConn(ch).SetCaller(owner))

	_, err := client.AddBalanceByAdmin(context.Background(), req)
	require.NoError(t, err)
	user1.BalanceShouldBe(ch, 1000)

	hello, err := client.HelloWorld(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "Hello World!", hello.Message)

	resp := user1.Invoke(ch, "metadata")

	var meta token.Metadata
	err = json.Unmarshal([]byte(resp), &meta)
	require.NoError(t, err)

	require.Equal(t, meta.Methods[0], "/foundation.token.BalanceService/AddBalanceByAdmin")
	require.Equal(t, meta.Methods[1], "/foundation.token.BalanceService/HelloWorld")
}
