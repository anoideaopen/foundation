package telemetry

import (
	"context"
	"crypto/tls"
	"errors"
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
	caCerts := os.Getenv(tracingCollectorCaPem)

	var client otlptrace.Client

	// If the endpoint is obtained from the config, we create an insecure client.
	if !isTracingConfigFromEnv { //nolint:nestif // TODO For pipeline completion, clarify during review.
		client = getUnsecureClient(settings.GetEndpoint())
	} else {
		// If the endpoint is obtained from the env, we check that there are no errors in passing certificates and the header,
		// either they should not be at all, or they should all be filled in.
		if err := checkAuthEnvironments(authHeaderKey, authHeaderValue, caCerts); err != nil {
			fmt.Printf("Failed to load auth environments: %s", err)
			return
		}

		// If none of the variables are filled, create an insecure client
		if !isSecure(authHeaderKey, authHeaderValue, caCerts) {
			client = getUnsecureClient(settings.GetEndpoint())
		} else {
			// If all variables are filled, we try to get valid CA certificates from the env
			// and create tlsConfig.
			tlsConfig, err := getTLSConfig(caCerts)
			if err != nil {
				fmt.Printf("failed to load TLS configuration: %s", err)
				return
			}
			// After creating tlsConfig, create a secure client
			client = getSecureClient(authHeaderKey, authHeaderValue, settings.GetEndpoint(), tlsConfig)
		}
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

func getSecureClient(authHeaderKey string, authHeaderValue string, endpoint string, tlsConfig *tls.Config) otlptrace.Client {
	h := map[string]string{
		authHeaderKey: authHeaderValue,
	}
	client := otlptracehttp.NewClient(
		otlptracehttp.WithHeaders(h),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithTLSClientConfig(tlsConfig),
	)
	return client
}

func getUnsecureClient(endpoint string) otlptrace.Client {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	return client
}

// checkAuthEnvironments checks for possible erroneous combinations in case the user forgot to specify some variables
func checkAuthEnvironments(authHeaderKey string, authHeaderValue string, caCerts string) error {
	// If the environment variable with certificates is not empty, check if the authorization header exists
	// If the headers are missing, consider it an error
	if isCACertsSet(caCerts) && !isAuthHeaderSet(authHeaderKey, authHeaderValue) {
		return errors.New("TLS CA environment is set, but auth header is wrong or empty")
	}

	// If the header is not empty but there are no certificates, consider it an error
	if !isCACertsSet(caCerts) && isAuthHeaderSet(authHeaderKey, authHeaderValue) {
		return errors.New("auth header environment is set, but TLS CA is empty")
	}
	return nil
}

// isSecure checks if both the header and certificates are received, creating a client with their use
// such a client will be considered secure
func isSecure(authHeaderKey string, authHeaderValue string, caCerts string) bool {
	if isAuthHeaderSet(authHeaderKey, authHeaderValue) && isCACertsSet(caCerts) {
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
