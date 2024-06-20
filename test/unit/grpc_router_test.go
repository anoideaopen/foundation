package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/grpc"
	"github.com/anoideaopen/foundation/core/reflectx"
	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/test/unit/service"
	"github.com/anoideaopen/foundation/test/unit/service/token"
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

	balanceToken := &token.Balance{} // gRPC service.

	// Create gRPC router.
	grpcRouter, err := grpc.NewRouter(grpc.RouterConfig{
		Fallback: grpc.DefaultReflectxFallback(
			balanceToken,
			reflectx.RouterConfig{},
		),
	})
	require.NoError(t, err)

	// Register gRPC service.
	service.RegisterBalanceServiceServer(grpcRouter, balanceToken)

	// Init chaincode.
	initMsg := ledger.NewCC(
		"cc",
		balanceToken,
		ccConfig,
		core.WithRouter(grpcRouter),
	)
	require.Empty(t, initMsg)

	// Prepare request.
	req := &service.BalanceAdjustmentRequest{
		Suffix: "",
		Address: &service.Address{
			Address: user1.Address(),
		},
		Amount: &service.BigInt{
			Value: "1000",
		},
		Reason: "Test reason",
	}

	rawJSON, _ := protojson.Marshal(req)

	// Add balance by admin.
	owner.SignedInvoke("cc", "addBalanceByAdmin", string(rawJSON))
	user1.BalanceShouldBe("cc", 1000)
}
