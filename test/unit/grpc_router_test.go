package unit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/routing/grpc"
	"github.com/anoideaopen/foundation/core/routing/reflect"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	mockgrpc "github.com/anoideaopen/foundation/mocks/grpc"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/token/proto"
	"github.com/anoideaopen/foundation/test/unit/token/service"
	"github.com/anoideaopen/foundation/token"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGRPCRouter(t *testing.T) {
	mockStub := mockstub.NewMockStub(t)

	owner, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user1, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	mockStub.CreateAndSetConfig(
		"CC Token",
		"CC",
		8,
		owner.AddressBase58Check,
		"",
		"",
		owner.AddressBase58Check,
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
	cc, err := core.NewCC(balanceToken, core.WithRouters(reflectRouter, grpcRouter))
	require.NoError(t, err)

	// Prepare request.
	req := &proto.BalanceAdjustmentRequest{
		Address: &proto.Address{
			Base58Check: user1.AddressBase58Check,
		},
		Amount: &proto.BigInt{
			Value: "1000",
		},
		Reason: "Test reason",
	}

	// Add balance by admin with a client by URL.
	client := proto.NewBalanceServiceClient(mockgrpc.NewMockClientConn(t, mockStub, cc).SetCaller(owner))

	_, err = client.AddBalanceByAdmin(context.Background(), req)
	require.NoError(t, err)
	keyBalance, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user1.AddressBase58Check})
	require.NoError(t, err)
	for i := range mockStub.PutStateCallCount() {
		key, data := mockStub.PutStateArgsForCall(i)
		if key == keyBalance {
			require.Equal(t, big.NewInt(1000).Bytes(), data)
			break
		}
	}

	hello, err := client.HelloWorld(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "Hello World!", hello.GetMessage())

	resp := mockStub.QueryChaincode(cc, "metadata")

	var meta token.Metadata
	err = json.Unmarshal(resp.GetPayload(), &meta)
	require.NoError(t, err)

	require.Equal(t, "/foundation.token.BalanceService/AddBalanceByAdmin", meta.Methods[0])
	require.Equal(t, "/foundation.token.BalanceService/HelloWorld", meta.Methods[1])
}
