package unit

import (
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/balance"
	"github.com/anoideaopen/foundation/core/types/big"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/token"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestTaskExecutor(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAddressSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeSetter, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	feeAggregator, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user2, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description         string
		errorMsg            string
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []*mockstub.ExecutorRequest
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.BatchResponse)
	}{
		{
			description: "group TxExecutor emit and transfer",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []*mockstub.ExecutorRequest {
				userBalanceKey, err := mockStub.CreateCompositeKey(balance.BalanceTypeToken.String(), []string{user.AddressBase58Check})
				require.NoError(t, err)
				mockStub.GetStateCallsMap[userBalanceKey] = new(big.Int).SetUint64(1000).Bytes()

				return []*mockstub.ExecutorRequest{
					{User: issuer, Task: &pbfound.Task{Method: "emit", Args: []string{user.AddressBase58Check, "1000"}}},
					{User: feeAddressSetter, Task: &pbfound.Task{Method: "setFeeAddress", Args: []string{feeAggregator.AddressBase58Check}}},
					{User: feeSetter, Task: &pbfound.Task{Method: "setFee", Args: []string{"FIAT", "500000", "100", "100000"}}},
					{User: user, Task: &pbfound.Task{Method: "transfer", Args: []string{user2.AddressBase58Check, "400", ""}}},
					{User: user2, Task: &pbfound.Task{Method: "accountsTest", Args: []string{user.AddressBase58Check, user.PublicKeyBase58}}},
				}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.BatchResponse) {
				require.Equal(t, mockStub.PutStateCallCount(), 9)
				require.Len(t, resp.GetTxResponses(), 5)
				for _, r := range resp.GetTxResponses() {
					require.Nil(t, r.GetError())
				}
			},
		},
		{
			description: "group TxExecutor Healthcheck",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []*mockstub.ExecutorRequest {
				return []*mockstub.ExecutorRequest{
					{User: issuer, Task: &pbfound.Task{Method: "healthCheck", Args: []string{}}},
				}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.BatchResponse) {
				require.Equal(t, mockStub.PutStateCallCount(), 1)
				require.Len(t, resp.GetTxResponses(), 1)
				for _, r := range resp.GetTxResponses() {
					require.Nil(t, r.GetError())
				}
			},
		},
		{
			description: "group TxExecutor HealthCheckNb",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []*mockstub.ExecutorRequest {
				return []*mockstub.ExecutorRequest{
					{User: issuer, Task: &pbfound.Task{Method: "healthCheckNb", Args: []string{}}},
				}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.BatchResponse) {
				require.Equal(t, mockStub.PutStateCallCount(), 1)
				require.Len(t, resp.GetTxResponses(), 1)
				for _, r := range resp.GetTxResponses() {
					require.Nil(t, r.GetError())
				}
			},
		},
		{
			description: "group TxExecutor 50 - Healthcheck, 50 - HealthCheckNb",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []*mockstub.ExecutorRequest {
				reqs := make([]*mockstub.ExecutorRequest, 0, 100)
				for i := 0; i < 50; i++ {
					reqs = append(reqs, &mockstub.ExecutorRequest{User: issuer, Task: &pbfound.Task{Method: "healthCheck", Args: []string{}}})
				}
				for i := 0; i < 50; i++ {
					reqs = append(reqs, &mockstub.ExecutorRequest{User: issuer, Task: &pbfound.Task{Method: "healthCheckNb", Args: []string{}}})
				}
				return reqs
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.BatchResponse) {
				require.Equal(t, 1, mockStub.PutStateCallCount())
				require.Len(t, resp.GetTxResponses(), 100)
				for _, r := range resp.GetTxResponses() {
					require.Nil(t, r.GetError())
				}
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				"fiat",
				"FIAT",
				8,
				issuer.AddressBase58Check,
				feeSetter.AddressBase58Check,
				feeAddressSetter.AddressBase58Check,
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(NewFiatTestToken(&token.BaseToken{}))
			require.NoError(t, err)

			tasksReq := testCase.funcPrepareMockStub(t, mockStub)

			_, resp := mockStub.TxInvokeTaskExecutor(cc, "", "", "", tasksReq)

			require.Equal(t, resp.GetStatus(), int32(shim.OK))
			require.Empty(t, resp.GetMessage())

			bResp := &pbfound.BatchResponse{}
			err = pb.Unmarshal(resp.GetPayload(), bResp)
			require.NoError(t, err)

			if testCase.funcCheckResponse != nil {
				testCase.funcCheckResponse(t, mockStub, bResp)
			}
		})
	}
}
