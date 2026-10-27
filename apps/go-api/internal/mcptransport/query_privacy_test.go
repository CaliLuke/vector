package mcptransport_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	"github.com/stretchr/testify/require"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestMCPRejectedQueriesAndCredentialsDoNotEnterHTTPTraces(t *testing.T) {
	exported := make(chan string, 4)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			w.WriteHeader(404)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		var batch collecttrace.ExportTraceServiceRequest
		if proto.Unmarshal(body, &batch) != nil {
			w.WriteHeader(400)
			return
		}
		exported <- protojson.Format(&batch)
		w.WriteHeader(200)
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", receiver.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_SDK_DISABLED", "")
	wrap, _, shutdown, err := telemetry.ConfigureHTTPObservability(t.Context(), "vector-mcp-privacy-test")
	require.NoError(t, err)
	h := mcptransport.RedactRejectedQueries(wrap(handler(t, config("oauth"), options())))
	r := httptest.NewRequest("POST", "https://vector.example/mcp?access_token=private-query-marker", strings.NewReader(`{"private-body-marker":true}`))
	r.Header.Set("Authorization", "Bearer private-bearer-marker")
	r.Header.Set("Cookie", "session-id=private-cookie-marker")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 400, w.Code)
	require.NotEmpty(t, w.Header().Get("X-Request-Id"))
	require.NoError(t, shutdown(context.Background()))
	select {
	case spans := <-exported:
		require.NotContains(t, spans, "private-")
		require.Contains(t, spans, "traceId")
		require.Contains(t, spans, "400")
	default:
		t.Fatal("no HTTP trace exported")
	}
}

func TestMCPQueryPrivacyLeavesOtherApplicationQueriesIntact(t *testing.T) {
	var target string
	h := mcptransport.RedactRejectedQueries(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { target = r.URL.RawQuery }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "https://vector.example/api/example/?filter=kept", nil))
	require.Equal(t, "filter=kept", target)
}
