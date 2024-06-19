package telemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/anoideaopen/foundation/proto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TracingCollectorEndpointEnv is publicly available to use before calling InstallTraceProvider
	// to be able to use the correct type of configuration either through environment variables
	// or chaincode initialization parameters
	TracingCollectorEndpointEnv = "CHAINCODE_TRACING_COLLECTOR_ENDPOINT"

	tracingCollectorAuthHeaderKey   = "CHAINCODE_TRACING_COLLECTOR_AUTH_HEADER_KEY"
	tracingCollectorAuthHeaderValue = "CHAINCODE_TRACING_COLLECTOR_AUTH_HEADER_VALUE"
	tracingCollectorCaPem           = "TRACING_COLLECTOR_CAPEM"
)

// InstallTraceProvider returns trace provider based on http otlp exporter .
func InstallTraceProvider(
	settings *proto.CollectorEndpoint,
	serviceName string,
	isTracingConfigFromEnv bool,
) {
	var tracerProvider trace.TracerProvider

	defer func() {
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	}()

	// If there is no endpoint, telemetry is disabled
	if settings == nil || len(settings.GetEndpoint()) == 0 {
		tracerProvider = trace.NewNoopTracerProvider()
		return
	}

	authHeaderKey := os.Getenv(tracingCollectorAuthHeaderKey)
	authHeaderValue := os.Getenv(tracingCollectorAuthHeaderValue)
	caCertsBase64 := os.Getenv(tracingCollectorCaPem)

	var client otlptrace.Client

	// If it is tracing config from chaincode init, use an insecure connection
	if !isTracingConfigFromEnv {
		client = otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(settings.GetEndpoint()),
			otlptracehttp.WithInsecure(),
		)
	}

	// If the environment variable with certificates is not empty, check if the authorization header exists
	// If the headers are missing, consider it an error
	if isTracingConfigFromEnv && isCACertsSet() && !isAuthHeaderSet() {
		fmt.Print("TLS CA environment is set, but auth header is wrong or empty")
		return
	}

	// If it is tracing config from environment and certificates are not provided but headers are, consider it an error
	if isTracingConfigFromEnv && !isCACertsSet() && isAuthHeaderSet() {
		fmt.Print("Auth header environment is set, but TLS CA is empty")
		return
	}

	// If it is tracing config from environment and both the environment variable with certificates and the header are set, get the TLS config
	if isTracingConfigFromEnv && isCACertsSet() && isAuthHeaderSet() {
		tlsConfig, err := getTLSConfig(caCertsBase64)
		if err != nil {
			fmt.Printf("Failed to load TLS configuration: %s", err)
			return
		}

		h := map[string]string{
			authHeaderKey: authHeaderValue,
		}
		client = otlptracehttp.NewClient(
			otlptracehttp.WithHeaders(h),
			otlptracehttp.WithEndpoint(settings.GetEndpoint()),
			otlptracehttp.WithTLSClientConfig(tlsConfig),
		)
	}

	// If it is tracing config from environment and certificates are not provided, use an insecure connection
	if isTracingConfigFromEnv && !isCACertsSet() && !isAuthHeaderSet() {
		client = otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(settings.GetEndpoint()),
			otlptracehttp.WithInsecure(),
		)
	}

	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		fmt.Printf("creating OTLP trace exporter: %v", err)
		return
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName)))
	if err != nil {
		fmt.Printf("creating resoure: %v", err)
		return
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r))
}

func isAuthHeaderSet() bool {
	authHeaderKey := os.Getenv(tracingCollectorAuthHeaderKey)
	authHeaderValue := os.Getenv(tracingCollectorAuthHeaderValue)
	if authHeaderKey != "" || authHeaderValue != "" {
		return true
	}
	return false
}

func isCACertsSet() bool {
	caCerts := os.Getenv(tracingCollectorCaPem)
	return caCerts != ""
}
