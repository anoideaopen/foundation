package unit

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/anoideaopen/foundation/core"
	"github.com/anoideaopen/foundation/core/config"
	"github.com/anoideaopen/foundation/core/telemetry"
	"github.com/anoideaopen/foundation/mocks"
	"github.com/anoideaopen/foundation/mocks/mockstub"
	"github.com/anoideaopen/foundation/proto"
	pbfound "github.com/anoideaopen/foundation/proto"
	"github.com/anoideaopen/foundation/test/unit/fixtures"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

const (
	collectorEndpoint = "172.23.0.6:4318"
)

func TestTelemetry(t *testing.T) {
	t.Parallel()

	issuer, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	user, err := mocks.NewUserFoundation(pbfound.KeyType_ed25519)
	require.NoError(t, err)

	tracerProvider := sdktrace.NewTracerProvider()
	tr := tracerProvider.Tracer("test")
	ctx, _ := tr.Start(context.Background(), "top-test")

	spanContext := trace.SpanContextFromContext(ctx)
	fmt.Println("Start test TraceID: " + spanContext.TraceID().String())
	fmt.Println("Start test SpanID: " + spanContext.SpanID().String())

	carrier := propagation.MapCarrier{}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	transientDataMap, err := telemetry.PackToTransientMap(carrier)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description         string
		functionName        string
		isQuery             bool
		noBatch             bool
		errorMsg            string
		signUser            *mocks.UserFoundation
		codeResp            int32
		funcPrepareMockStub func(t *testing.T, mockStub *mockstub.MockStub) []string
		funcCheckResponse   func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse)
		funcCheckQuery      func(t *testing.T, mockStub *mockstub.MockStub, payload []byte)
	}{
		{
			description:  "BatchTx with init Collector Endpoint and with telemetry",
			functionName: "healthCheck",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:                   testTokenSymbol,
						RobotSKI:                 fixtures.RobotHashedCert,
						Admin:                    &pbfound.Wallet{Address: issuer.AddressBase58Check},
						TracingCollectorEndpoint: &proto.CollectorEndpoint{Endpoint: collectorEndpoint},
					},
					Token: &pbfound.TokenConfig{
						Name:     testTokenName,
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if strings.Contains(k, config.BatchPrefix) {
						pending := new(proto.PendingTx)
						err = pb.Unmarshal(v, pending)
						require.NoError(t, err)
						require.Equal(t, carrier.Keys()[0], pending.GetPairs()[0].GetKey())
						require.Equal(t, carrier.Get(carrier.Keys()[0]), pending.GetPairs()[0].GetValue())

						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "BatchTx with init Collector Endpoint and without tracing",
			functionName: "healthCheck",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:                   testTokenSymbol,
						RobotSKI:                 fixtures.RobotHashedCert,
						Admin:                    &pbfound.Wallet{Address: issuer.AddressBase58Check},
						TracingCollectorEndpoint: &proto.CollectorEndpoint{Endpoint: collectorEndpoint},
					},
					Token: &pbfound.TokenConfig{
						Name:     testTokenName,
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes

				return []string{}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if strings.Contains(k, config.BatchPrefix) {
						pending := new(proto.PendingTx)
						err = pb.Unmarshal(v, pending)
						require.NoError(t, err)
						require.Nil(t, pending.GetPairs())

						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "failedTestCall with init Collector Endpoint and with telemetry",
			functionName: "failedTestCall",
			signUser:     user,
			codeResp:     int32(shim.ERROR),
			errorMsg:     "incorrect number of arguments: found 6 but expected 0: validate TxFailedTestCall",
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:                   testTokenSymbol,
						RobotSKI:                 fixtures.RobotHashedCert,
						Admin:                    &pbfound.Wallet{Address: issuer.AddressBase58Check},
						TracingCollectorEndpoint: &proto.CollectorEndpoint{Endpoint: collectorEndpoint},
					},
					Token: &pbfound.TokenConfig{
						Name:     testTokenName,
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
		},
		{
			description:  "NoBatchTx with init Collector Endpoint and with telemetry",
			functionName: "healthCheckNb",
			signUser:     user,
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:                   testTokenSymbol,
						RobotSKI:                 fixtures.RobotHashedCert,
						Admin:                    &pbfound.Wallet{Address: issuer.AddressBase58Check},
						TracingCollectorEndpoint: &proto.CollectorEndpoint{Endpoint: collectorEndpoint},
					},
					Token: &pbfound.TokenConfig{
						Name:     testTokenName,
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
		},
		{
			description:  "Query with init Collector Endpoint and with telemetry",
			functionName: "systemEnv",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				cfg := &pbfound.Config{
					Contract: &pbfound.ContractConfig{
						Symbol:                   testTokenSymbol,
						RobotSKI:                 fixtures.RobotHashedCert,
						Admin:                    &pbfound.Wallet{Address: issuer.AddressBase58Check},
						TracingCollectorEndpoint: &proto.CollectorEndpoint{Endpoint: collectorEndpoint},
					},
					Token: &pbfound.TokenConfig{
						Name:     testTokenName,
						Decimals: 8,
						Issuer:   &pbfound.Wallet{Address: issuer.AddressBase58Check},
					},
				}
				cfgBytes, _ := protojson.Marshal(cfg)
				mockStub.GetStateCallsMap["__config"] = cfgBytes
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.NotEmpty(t, payload)
			},
		},
		{
			description:  "BatchTx without init Collector Endpoint and with telemetry",
			functionName: "healthCheck",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if strings.Contains(k, config.BatchPrefix) {
						pending := new(proto.PendingTx)
						err = pb.Unmarshal(v, pending)
						require.NoError(t, err)
						require.Equal(t, carrier.Keys()[0], pending.GetPairs()[0].GetKey())
						require.Equal(t, carrier.Get(carrier.Keys()[0]), pending.GetPairs()[0].GetValue())

						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "BatchTx without init Collector Endpoint and without tracing",
			functionName: "healthCheck",
			signUser:     user,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				return []string{}
			},
			funcCheckResponse: func(t *testing.T, mockStub *mockstub.MockStub, resp *pbfound.TxResponse) {
				var j int
				for i := 0; i < mockStub.PutStateCallCount(); i++ {
					k, v := mockStub.PutStateArgsForCall(i)
					if strings.Contains(k, config.BatchPrefix) {
						pending := new(proto.PendingTx)
						err = pb.Unmarshal(v, pending)
						require.NoError(t, err)
						require.Nil(t, pending.GetPairs())

						j++
					}

					if j == 1 {
						return
					}
				}
				require.Fail(t, "not found checking data")
			},
		},
		{
			description:  "NoBatchTx without init Collector Endpoint and with telemetry",
			functionName: "healthCheckNb",
			signUser:     user,
			noBatch:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
		},
		{
			description:  "Query without init Collector Endpoint and with telemetry",
			functionName: "systemEnv",
			isQuery:      true,
			funcPrepareMockStub: func(t *testing.T, mockStub *mockstub.MockStub) []string {
				mockStub.GetTransientReturns(transientDataMap, nil)

				return []string{}
			},
			funcCheckQuery: func(t *testing.T, mockStub *mockstub.MockStub, payload []byte) {
				require.NotEmpty(t, payload)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			mockStub := mockstub.NewMockStub(t)

			mockStub.CreateAndSetConfig(
				testTokenName,
				testTokenSymbol,
				8,
				issuer.AddressBase58Check,
				"",
				"",
				issuer.AddressBase58Check,
				nil,
			)

			cc, err := core.NewCC(&TestToken{})
			require.NoError(t, err)

			parameters := testCase.funcPrepareMockStub(t, mockStub)

			var (
				txId string
				resp *peer.Response
			)
			if testCase.isQuery {
				resp = mockStub.QueryChaincode(cc, testCase.functionName, parameters...)
			} else if testCase.noBatch {
				resp = mockStub.NbTxInvokeChaincodeSigned(cc, testCase.functionName, testCase.signUser, "", "", "", parameters...)
			} else {
				txId, resp = mockStub.TxInvokeChaincodeSigned(cc, testCase.functionName, testCase.signUser, "", "", "", parameters...)
			}

			// check result
			if testCase.codeResp == int32(shim.ERROR) {
				require.Equal(t, resp.GetStatus(), testCase.codeResp)
				require.Contains(t, resp.GetMessage(), testCase.errorMsg)
				require.Empty(t, resp.GetPayload())
				return
			}

			require.Equal(t, resp.GetStatus(), int32(shim.OK))
			require.Empty(t, resp.GetMessage())

			if testCase.isQuery {
				if testCase.funcCheckQuery != nil {
					testCase.funcCheckQuery(t, mockStub, resp.GetPayload())
				}
				return
			}

			bResp := &pbfound.BatchResponse{}
			if string(resp.GetPayload()) != "null" {
				err = pb.Unmarshal(resp.GetPayload(), bResp)
				require.NoError(t, err)
			}

			var respb *pbfound.TxResponse
			for _, r := range bResp.GetTxResponses() {
				if hex.EncodeToString(r.GetId()) == txId {
					respb = r
					break
				}
			}

			if len(testCase.errorMsg) != 0 {
				require.Contains(t, respb.GetError().GetError(), testCase.errorMsg)
				return
			}

			if testCase.funcCheckResponse != nil {
				testCase.funcCheckResponse(t, mockStub, respb)
			}
		})
	}
}
