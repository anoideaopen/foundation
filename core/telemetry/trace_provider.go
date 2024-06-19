package telemetry

import (
	"context"
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
	"os"
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

	var client otlptrace.Client
	if isTracingConfigFromEnv {
		authHeaderKey := os.Getenv(tracingCollectorAuthHeaderKey)
		authHeaderValue := os.Getenv(tracingCollectorAuthHeaderValue)
		caCertsBase64 := os.Getenv(tracingCollectorCaPem)

		// If the environment variable with certificates is not empty, check if the authorization header exists
		// If the headers are missing, consider it an error
		if caCertsBase64 != "" {
			if authHeaderKey == "" || authHeaderValue == "" {
				fmt.Print("TLS CA environment is set, but auth header is wrong or empty")
				return
			}
			// If both the environment variable with certificates and the header are set, get the TLS config
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
		} else {
			// If certificates are not provided but headers are, consider it an error
			if authHeaderKey != "" || authHeaderValue != "" {
				fmt.Print("Auth headers are set, but TLS CA environment is missing or empty")
				return
			}
			// If certificates are not provided, use an insecure connection
			client = otlptracehttp.NewClient(
				otlptracehttp.WithEndpoint(settings.GetEndpoint()),
				otlptracehttp.WithInsecure(),
			)
		}
	} else {
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
