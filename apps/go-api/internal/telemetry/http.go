package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
	loomotel "github.com/CaliLuke/loom/observability/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ConfigureHTTPTracing enables OTLP/HTTP request traces when an exporter
// endpoint is configured. Production behavior is unchanged when it is absent.
func ConfigureHTTPTracing(ctx context.Context, serviceName string) (func(http.Handler) http.Handler, func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" || strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return func(handler http.Handler) http.Handler { return handler }, func(context.Context) error { return nil }, nil
	}

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Host == "" || parsedEndpoint.Scheme != "http" && parsedEndpoint.Scheme != "https" {
		return nil, nil, fmt.Errorf("parse OTLP endpoint %q", endpoint)
	}
	if configuredName := os.Getenv("OTEL_SERVICE_NAME"); configuredName != "" {
		serviceName = configuredName
	}
	runtime, err := loomotel.New(ctx, loomotel.Config{
		ServiceName: serviceName,
		ResourceAttributes: []attribute.KeyValue{
			attribute.String("app.component", "http"),
		},
		Traces: loomotel.TraceConfig{
			Enabled:     true,
			Endpoint:    parsedEndpoint.Host,
			Insecure:    parsedEndpoint.Scheme == "http",
			SampleRatio: 1,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	wrap := loomotel.HTTPMiddleware(loomotel.HTTPConfig{
		ServiceName:    serviceName,
		MetricMode:     loomotel.HTTPMetricModeNone,
		TracerProvider: runtime.TracerProvider,
		Propagators:    runtime.Propagators,
	})
	return wrap, runtime.Shutdown, nil
}
func ClientAddressSpanMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if address := loomhttp.EffectiveClientAddress(request); address != "" {
			trace.SpanFromContext(request.Context()).SetAttributes(attribute.String("client.address", address))
		}
		next.ServeHTTP(response, request)
	})
}
