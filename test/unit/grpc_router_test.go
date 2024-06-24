package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/grpc"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/token/proto"
	"github.com/anoideaopen/foundation/test/unit/token/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
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
		UseURLs:  true,
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

	// Add balance by admin with URL.
	owner.SignedInvoke("cc", "/foundationtoken.BalanceService/AddBalanceByAdmin", string(rawJSON))
	user1.BalanceShouldBe("cc", 1000)

	// Add balance by admin with override function name.
	owner.NbInvoke("cc", "CustomAddBalance", string(rawJSON))
	user1.BalanceShouldBe("cc", 2000)
}
