package unit

import (
	"context"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/token/proto"
	"github.com/anoideaopen/foundation/test/unit/token/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGRPCRouter(t *testing.T) {
	var (
		ledger = mock.NewLedger(t)
		owner  = ledger.NewWallet()
		user1  = ledger.NewWallet()
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

	balanceToken := &service.Balance{} // gRPC service.

	// Create gRPC router.
	grpcRouter := grpc.NewRouter(grpc.RouterConfig{
		Fallback: grpc.DefaultReflectxFallback(balanceToken),
		UseNames: true,
	})

	// Register gRPC service.
	proto.RegisterBalanceServiceServer(grpcRouter, balanceToken)

	// Init chaincode.
	initMsg := ledger.NewCC(
		"cc",
		balanceToken,
		ccConfig,
		core.WithRouter(grpcRouter),
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

	rawJSON, _ := protojson.Marshal(req)

	// Add balance by admin.
	owner.SignedInvoke("cc", "addBalanceByAdmin", string(rawJSON))
	user1.BalanceShouldBe("cc", 1000)

	// Add balance by admin with override function name.
	_, _ = owner.NbInvoke("cc", "CustomAddBalance", string(rawJSON))
	user1.BalanceShouldBe("cc", 2000)
}

func TestGRPCRouterWithURLs(t *testing.T) {
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

	balanceToken := &service.Balance{} // gRPC service.

	// Create gRPC router.
	grpcRouter := grpc.NewRouter(grpc.RouterConfig{
		Fallback: grpc.DefaultReflectxFallback(balanceToken),
	})

	// Register gRPC service.
	proto.RegisterBalanceServiceServer(grpcRouter, balanceToken)

	// Init chaincode.
	initMsg := ledger.NewCC(
		ch,
		balanceToken,
		ccConfig,
		core.WithRouter(grpcRouter),
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
	client := proto.NewBalanceServiceClient(mock.NewMockClientConn(ch).SetCaller(owner))

	_, err := client.AddBalanceByAdmin(context.Background(), req)
	require.NoError(t, err)
	user1.BalanceShouldBe(ch, 1000)

	hello, err := client.HelloWorld(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "Hello World!", hello.Message)

	// Add balance by admin with override function name.
	rawJSON, _ := protojson.Marshal(req)
	owner.NbInvoke(ch, "CustomAddBalance", string(rawJSON))
	user1.BalanceShouldBe(ch, 2000)
}
