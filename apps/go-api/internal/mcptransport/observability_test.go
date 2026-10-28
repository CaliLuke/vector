package mcptransport_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
	collectlog "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collecttrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestMCPAuthenticationFailureProducesSanitizedCorrelatedLogAndTrace(t *testing.T) {
	for _, tc := range []struct {
		name     string
		failure  error
		status   int
		severity logsv1.SeverityNumber
	}{
		{"invalid credential", mcptransport.ErrInvalidCredential, http.StatusUnauthorized, logsv1.SeverityNumber_SEVERITY_NUMBER_WARN},
		{"provider outage", errors.New("private-provider-outage-marker"), http.StatusServiceUnavailable, logsv1.SeverityNumber_SEVERITY_NUMBER_ERROR},
	} {
		t.Run(tc.name, func(t *testing.T) { assertAuthenticationFailureTelemetry(t, tc.failure, tc.status, tc.severity) })
	}
}

func assertAuthenticationFailureTelemetry(t *testing.T, failure error, status int, severity logsv1.SeverityNumber) {
	t.Helper()
	logs, traces := make(chan string, 4), make(chan string, 4)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/v1/logs":
			var batch collectlog.ExportLogsServiceRequest
			if proto.Unmarshal(body, &batch) != nil {
				w.WriteHeader(400)
				return
			}
			logs <- protojson.Format(&batch)
		case "/v1/traces":
			var batch collecttrace.ExportTraceServiceRequest
			if proto.Unmarshal(body, &batch) != nil {
				w.WriteHeader(400)
				return
			}
			traces <- protojson.Format(&batch)
		default:
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", receiver.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_SDK_DISABLED", "")
	wrap, output, shutdown, err := telemetry.ConfigureHTTPObservability(t.Context(), "vector-mcp-observer-test")
	require.NoError(t, err)
	var stdout bytes.Buffer
	logger := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &stdout, Format: log.FormatJSON}, output), log.WithFunc(log.Span), log.WithDisableBuffering(func(context.Context) bool { return true }))
	opts := options()
	opts.Observer = mcptransport.LogObserver(logger)
	opts.OAuthAuth = func(context.Context, *http.Request) (context.Context, error) {
		return nil, failure
	}
	h := mcptransport.RedactRejectedQueries(wrap(handler(t, config("oauth"), opts)))
	r := httptest.NewRequest("POST", "https://vector.example/mcp", strings.NewReader(`{"private-body-marker":true}`))
	r.Header.Set("Authorization", "Bearer private-bearer-marker")
	r.Header.Set("Cookie", "session-id=private-cookie-marker")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, status, w.Code)
	requestID := w.Header().Get("X-Request-Id")
	require.NotEmpty(t, requestID)
	require.NoError(t, shutdown(context.Background()))
	var logged, traced string
	select {
	case logged = <-logs:
	default:
		t.Fatal("no failure log exported")
	}
	select {
	case traced = <-traces:
	default:
		t.Fatal("no failure trace exported")
	}
	for _, text := range []string{logged, traced, stdout.String()} {
		require.NotContains(t, text, "private-")
	}
	require.Contains(t, logged, requestID)
	require.Contains(t, stdout.String(), requestID)
	require.Contains(t, logged, "request_rejected")
	var logExport collectlog.ExportLogsServiceRequest
	var traceExport collecttrace.ExportTraceServiceRequest
	require.NoError(t, protojson.Unmarshal([]byte(logged), &logExport))
	require.NoError(t, protojson.Unmarshal([]byte(traced), &traceExport))
	logRecord := logExport.ResourceLogs[0].ScopeLogs[0].LogRecords[0]
	require.Equal(t, severity, logRecord.SeverityNumber)
	span := traceExport.ResourceSpans[0].ScopeSpans[0].Spans[0]
	require.NotEmpty(t, logRecord.TraceId)
	require.Equal(t, span.TraceId, logRecord.TraceId)
	require.Equal(t, span.SpanId, logRecord.SpanId)
}

func TestMCPGeneratedValidationFailureDoesNotLogArgumentsOrRPCID(t *testing.T) {
	var output bytes.Buffer
	logger := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
	opts := options()
	opts.Observer = mcptransport.LogObserver(logger)
	h := handler(t, config("header"), opts)
	response := call(h, "/mcp/api-key", `{"jsonrpc":"2.0","id":"private-rpc-id-marker","method":"tools/call","params":{"name":"member","arguments":{"action":"private-action-marker"}}}`, http.Header{"Mcp-Protocol-Version": {"2026-07-28"}})
	require.Equal(t, 400, response.Code)
	require.Contains(t, response.Body.String(), "error")
	require.NotEmpty(t, output.String())
	require.Contains(t, output.String(), response.Header().Get("X-Request-Id"))
	require.NotContains(t, output.String(), "private-")
}
