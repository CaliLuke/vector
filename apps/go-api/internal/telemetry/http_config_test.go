package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigureHTTPObservabilityNoopsWhenDisabledAndRejectsInvalidEndpoint(t *testing.T) {
	for _, test := range []struct {
		name      string
		endpoint  string
		disabled  string
		wantError bool
	}{
		{name: "endpoint unset"},
		{name: "sdk disabled before endpoint parsing", endpoint: "ftp://collector.invalid", disabled: "true"},
		{name: "invalid endpoint", endpoint: "ftp://collector.invalid", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", test.endpoint)
			t.Setenv("OTEL_SDK_DISABLED", test.disabled)
			t.Setenv("OTEL_SERVICE_NAME", "")

			wrap, output, shutdown, err := ConfigureHTTPObservability(context.Background(), "test-service")
			if test.wantError {
				if err == nil || !strings.Contains(err.Error(), "parse OTLP endpoint") {
					t.Fatalf("ConfigureHTTPObservability() error = %v, want endpoint parse error", err)
				}
				if wrap != nil || shutdown != nil {
					t.Fatal("invalid endpoint returned usable hooks")
				}
				return
			}
			if err != nil {
				t.Fatalf("ConfigureHTTPObservability() error = %v", err)
			}
			if wrap == nil || shutdown == nil || output.Writer != io.Discard || output.Format == nil {
				t.Fatalf("disabled observability returned incomplete no-op hooks: wrap nil=%v output=%#v shutdown nil=%v", wrap == nil, output, shutdown == nil)
			}

			called := false
			wrapped := wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
			wrapped.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			if !called {
				t.Fatal("no-op middleware did not invoke the handler")
			}
			if err := shutdown(context.Background()); err != nil {
				t.Fatalf("no-op shutdown() error = %v", err)
			}
		})
	}
}
