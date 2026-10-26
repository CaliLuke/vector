package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	cluelog "github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loomotel "github.com/CaliLuke/loom/observability/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ConfigureHTTPObservability enables request traces and structured application
// logs when an OTLP endpoint is configured. The returned log output can be
// added alongside the application's existing stdout output.
func ConfigureHTTPObservability(ctx context.Context, serviceName string) (func(http.Handler) http.Handler, cluelog.Output, func(context.Context) error, error) {
	noopWrap := func(handler http.Handler) http.Handler { return handler }
	noopShutdown := func(context.Context) error { return nil }
	noopOutput := cluelog.Output{Writer: io.Discard, Format: cluelog.FormatJSON}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" || strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return noopWrap, noopOutput, noopShutdown, nil
	}

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Host == "" || parsedEndpoint.Scheme != "http" && parsedEndpoint.Scheme != "https" {
		return nil, cluelog.Output{}, nil, fmt.Errorf("parse OTLP endpoint %q", endpoint)
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
		Logs: loomotel.LogConfig{
			Enabled:  true,
			Endpoint: parsedEndpoint.Host,
			Insecure: parsedEndpoint.Scheme == "http",
		},
	})
	if err != nil {
		return nil, cluelog.Output{}, nil, err
	}

	wrap := loomotel.HTTPMiddleware(loomotel.HTTPConfig{
		ServiceName:    serviceName,
		MetricMode:     loomotel.HTTPMetricModeNone,
		TracerProvider: runtime.TracerProvider,
		Propagators:    runtime.Propagators,
	})
	output := noopOutput
	if runtime.LoggerProvider != nil {
		output = openTelemetryOutput(runtime.LoggerProvider.Logger(serviceName))
	}
	return wrap, output, runtime.Shutdown, nil
}
func ClientAddressSpanMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if address := loomhttp.EffectiveClientAddress(request); address != "" {
			trace.SpanFromContext(request.Context()).SetAttributes(attribute.String("client.address", address))
		}
		next.ServeHTTP(response, request)
	})
}
