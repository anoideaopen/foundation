package telemetry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/anoideaopen/foundation/proto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	// TracingCollectorEndpointEnv is publicly available to use before calling InstallTraceProvider
	// to be able to use the correct type of configuration either through environment variables
	// or chaincode initialization parameters
	TracingCollectorEndpointEnv = "CHAINCODE_TRACING_COLLECTOR_ENDPOINT"

	TracingCollectorAuthHeaderKey   = "CHAINCODE_TRACING_COLLECTOR_AUTH_HEADER_KEY"
	TracingCollectorAuthHeaderValue = "CHAINCODE_TRACING_COLLECTOR_AUTH_HEADER_VALUE"
	TracingCollectorCaPem           = "CHAINCODE_TRACING_COLLECTOR_CAPEM"
)

// InstallTraceProvider returns trace provider based on http otlp exporter .
func InstallTraceProvider(
	settings *proto.CollectorEndpoint,
	serviceName string,
) {
	var tracerProvider trace.TracerProvider
	tracerProvider = noop.NewTracerProvider()

	defer func() {
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	}()

	// If there is no endpoint, telemetry is disabled
	if settings == nil || len(settings.GetEndpoint()) == 0 {
		return
	}

	err := checkSettings(settings)
	if err != nil {
		fmt.Printf("failed to check collector settings: %s", err)
		return
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(settings.GetEndpoint()),
		otlptracehttp.WithInsecure(),
	)

	if isSecure(settings) {
		tlsConfig, err := getTLSConfig(settings.GetTlsCa())
		if err != nil {
			fmt.Printf("failed to load TLS configuration: %s", err)
			return
		}
		client = getSecureClient(settings, tlsConfig)
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
		fmt.Printf("creating resource: %v", err)
		return
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r))
}

func getSecureClient(settings *proto.CollectorEndpoint, tlsConfig *tls.Config) otlptrace.Client {
	h := map[string]string{
		settings.GetAuthorizationHeaderKey(): settings.GetAuthorizationHeaderValue(),
	}
	client := otlptracehttp.NewClient(
		otlptracehttp.WithHeaders(h),
		otlptracehttp.WithEndpoint(settings.GetEndpoint()),
		otlptracehttp.WithTLSClientConfig(tlsConfig),
	)
	return client
}

// checkAuthEnvironments checks for possible erroneous combinations in case the user forgot to specify some variables
func checkSettings(settings *proto.CollectorEndpoint) error {
	// If the environment variable with certificates is not empty, check if the authorization header exists
	// If the headers are missing, consider it an error
	if isCACertsSet(settings.GetTlsCa()) && !isAuthHeaderSet(settings.GetAuthorizationHeaderKey(), settings.GetAuthorizationHeaderValue()) {
		return errors.New("TLS CA environment is set, but auth header is wrong or empty")
	}

	// If the header is not empty but there are no certificates, consider it an error
	if !isCACertsSet(settings.GetTlsCa()) && isAuthHeaderSet(settings.GetAuthorizationHeaderKey(), settings.GetAuthorizationHeaderValue()) {
		return errors.New("auth header environment is set, but TLS CA is empty")
	}
	return nil
}

// isSecure checks if both the header and certificates are received, creating a client with their use
// such a client will be considered secure
func isSecure(settings *proto.CollectorEndpoint) bool {
	if isAuthHeaderSet(settings.GetAuthorizationHeaderKey(), settings.GetAuthorizationHeaderValue()) && isCACertsSet(settings.GetTlsCa()) {
		return true
	}
	return false
}

func isAuthHeaderSet(authHeaderKey string, authHeaderValue string) bool {
	if authHeaderKey != "" && authHeaderValue != "" {
		return true
	}
	return false
}

func isCACertsSet(caCerts string) bool {
	return caCerts != ""
}
