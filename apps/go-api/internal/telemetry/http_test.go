package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
)

func TestConfigureHTTPTracingIsTransparentWithoutExporter(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "")

	wrap, shutdown, err := ConfigureHTTPTracing(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("ConfigureHTTPTracing() error = %v", err)
	}

	called := false
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	wrap(handler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if !called {
		t.Fatal("wrapped handler was not called")
	}
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestConfigureHTTPTracingHonorsDisabledSDK(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("OTEL_SDK_DISABLED", "TRUE")

	wrap, shutdown, err := ConfigureHTTPTracing(context.Background(), "test-service")
	if err != nil {
		t.Fatalf("ConfigureHTTPTracing() error = %v", err)
	}
	if wrap == nil || shutdown == nil {
		t.Fatal("disabled tracing did not return transparent lifecycle functions")
	}
}

type exportedHTTPSpans struct {
	names           []string
	clientAddresses []string
}

func TestConfigureHTTPTracingExportsStableServiceSpan(t *testing.T) {
	exported := make(chan exportedHTTPSpans, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/traces" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		export := new(collecttrace.ExportTraceServiceRequest)
		if err := proto.Unmarshal(body, export); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		spans := exportedHTTPSpans{}
		for _, resourceSpans := range export.ResourceSpans {
			for _, scopeSpans := range resourceSpans.ScopeSpans {
				for _, span := range scopeSpans.Spans {
					spans.names = append(spans.names, span.Name)
					for _, attribute := range span.Attributes {
						if attribute.Key == "client.address" {
							spans.clientAddresses = append(spans.clientAddresses, attribute.Value.GetStringValue())
						}
					}
				}
			}
		}
		exported <- spans
		writer.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", receiver.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_SDK_DISABLED", "")

	wrap, shutdown, err := ConfigureHTTPTracing(context.Background(), "plane-go-api-test")
	if err != nil {
		t.Fatalf("ConfigureHTTPTracing() error = %v", err)
	}
	policy, err := loomhttp.NewRequestMetadataPolicy(nil, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	if err != nil {
		t.Fatal(err)
	}
	inner := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request.Pattern = "GET /api/activities/{activity_id}/"
		writer.WriteHeader(http.StatusNoContent)
	})
	handler := loomhttp.RequestMetadataMiddleware(policy)(wrap(ClientAddressSpanMiddleware(inner)))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/activities/00000000-0000-4000-8000-000000000001/", nil)
	request.RemoteAddr = "10.0.0.2:4321"
	request.Header.Set("X-Forwarded-For", "203.0.113.66, 198.51.100.7, 10.0.0.1")
	handler.ServeHTTP(recorder, request)

	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	select {
	case spans := <-exported:
		if len(spans.names) != 1 || spans.names[0] != "plane-go-api-test" {
			t.Fatalf("exported span names = %#v", spans.names)
		}
		if len(spans.clientAddresses) != 1 || spans.clientAddresses[0] != "198.51.100.7" {
			t.Fatalf("exported client addresses = %#v", spans.clientAddresses)
		}
	default:
		t.Fatal("trace exporter did not receive a span")
	}
}
