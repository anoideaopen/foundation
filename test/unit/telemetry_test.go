package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anoideaopen/foundation/mock"
	"github.com/anoideaopen/foundation/proto"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	collectorEndpoint = "172.23.0.6:4318"
)

// TestTelemetryWithInit does the next steps:
//   - instantiates chaincode with telemetry endpoint set;
//   - invokes chaincode Query, NoBatchTx, BatchedTx with parent span in transient map;
//   - invokes chaincode BatchedTx without parent span in transient map
func TestTelemetryWithInit(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	feeAddressSetter := ledgerMock.NewWallet()
	feeSetter := ledgerMock.NewWallet()

	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", &proto.CollectorEndpoint{Endpoint: collectorEndpoint})

	initMsg := ledgerMock.NewCC(
		testTokenCCName,
		&TestToken{},
		config,
	)
	require.Empty(t, initMsg)

	tracerProvider := sdktrace.NewTracerProvider()
	tr := tracerProvider.Tracer("test")
	ctx, _ := tr.Start(context.Background(), "top-test")

	spanContext := trace.SpanContextFromContext(ctx)
	fmt.Println("Start test TraceID: " + spanContext.TraceID().String())
	fmt.Println("Start test SpanID: " + spanContext.SpanID().String())

	var noTelemetryTxID, telemetryTx1ID, telemetryTx2ID, setEndpointTxId string
	t.Run("Query test", func(t *testing.T) {
		result := owner.InvokeTraced(ctx, testTokenCCName, "systemEnv")
		require.NotZero(t, result)
	})

	t.Run("NoBatchTx test", func(t *testing.T) {
		_, hash := owner.NbInvokeTraced(ctx, testTokenCCName, "healthCheckNb")
		require.NotZero(t, hash)
	})

	t.Run("Healthcheck checking", func(t *testing.T) {
		txID := owner.SignedInvokeTraced(ctx, testTokenCCName, "healthCheck")
		require.NotEmpty(t, txID)
		noTelemetryTxID = txID
	})

	t.Run("Healthcheck checking with telemetry #1", func(t *testing.T) {
		telemetryTx1ID = owner.SignedInvokeTraced(ctx, testTokenCCName, "healthCheck")
		require.NotEmpty(t, telemetryTx1ID)
	})

	t.Run("Health check without tracing", func(t *testing.T) {
		txID := owner.SignedInvoke(testTokenCCName, "healthCheck")
		require.NotEmpty(t, txID)
	})

	t.Run("failedTestCall telemetry #1", func(t *testing.T) {
		err := owner.RawSignedInvokeTracedWithErrorReturned(ctx, testTokenCCName, "failedTestCall")
		require.Error(t, err)
	})

	// t.Run("Fetch metadata with telemetry #1", func(t *testing.T) {
	//	err := owner.InvokeWithError(testTokenCCName, "metadata")
	//	require.NoError(t, err)
	// })

	t.Run("Healthcheck checking with telemetry #2", func(t *testing.T) {
		telemetryTx2ID = owner.SignedInvokeTraced(ctx, testTokenCCName, "healthCheck")
		require.NotEmpty(t, telemetryTx2ID)
	})

	// for {
	//	<-time.After(10 * time.Second)
	//	break
	// }

	time.Sleep(10 * time.Second)

	fmt.Printf(
		"  noTelemetryTxID: %s\n  setEndpointTxId: %s\n  telemetryTx1ID: %s\n  telemetryTx2ID: %s\n",
		noTelemetryTxID,
		setEndpointTxId,
		telemetryTx1ID, // should be visible in jaeger
		telemetryTx2ID, // should NOT be visible in jaeger
	)
}

// TestTelemetryWithoutInit  does the next steps:
//   - instantiates chaincode without telemetry endpoint set;
//   - invokes chaincode Query, NoBatchTx, BatchedTx with parent span in transient map;
//   - invokes chaincode BatchedTx without parent span in transient map
func TestTelemetryWithoutInit(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	feeAddressSetter := ledgerMock.NewWallet()
	feeSetter := ledgerMock.NewWallet()

	config := makeBaseTokenConfig(testTokenName, testTokenSymbol, 8,
		owner.Address(), feeSetter.Address(), feeAddressSetter.Address(), "", nil)

	initMsg := ledgerMock.NewCC(
		testTokenCCName,
		&TestToken{},
		config,
	)
	require.Empty(t, initMsg)

	tracerProvider := sdktrace.NewTracerProvider()
	tr := tracerProvider.Tracer("test")
	ctx, _ := tr.Start(context.Background(), "top-test")

	spanContext := trace.SpanContextFromContext(ctx)
	fmt.Println("Start test TraceID: " + spanContext.TraceID().String())
	fmt.Println("Start test SpanID: " + spanContext.SpanID().String())

	t.Run("Query test", func(t *testing.T) {
		result := owner.InvokeTraced(ctx, testTokenCCName, "systemEnv")
		require.NotZero(t, result)
	})

	t.Run("NoBatchTx test", func(t *testing.T) {
		_, hash := owner.NbInvokeTraced(ctx, testTokenCCName, "healthCheckNb")
		require.NotZero(t, hash)
	})

	t.Run("Healthcheck checking", func(t *testing.T) {
		txID := owner.SignedInvokeTraced(ctx, testTokenCCName, "healthCheck")
		require.NotEmpty(t, txID)
	})

	t.Run("Health check without tracing", func(t *testing.T) {
		txID := owner.SignedInvoke(testTokenCCName, "healthCheck")
		require.NotEmpty(t, txID)
	})
}
