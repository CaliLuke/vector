package telemetry

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cluelog "github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	collectlogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/proto"
)

func TestConfigureHTTPObservabilityExportsStructuredLogsAndPreservesStdout(t *testing.T) {
	exported := make(chan *collectlogs.ExportLogsServiceRequest, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/logs" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		batch := new(collectlogs.ExportLogsServiceRequest)
		if err := proto.Unmarshal(body, batch); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		exported <- batch
		writer.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", receiver.URL)
	t.Setenv("OTEL_SERVICE_NAME", "plane-go-api-test")
	t.Setenv("OTEL_SDK_DISABLED", "")

	_, otelOutput, shutdown, err := ConfigureHTTPObservability(context.Background(), "plane-go-api")
	if err != nil {
		t.Fatalf("ConfigureHTTPObservability() error = %v", err)
	}
	var stdout bytes.Buffer
	traceID, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
	ctx = cluelog.Context(ctx,
		cluelog.WithOutputs(cluelog.Output{Writer: &stdout, Format: cluelog.FormatJSON}, otelOutput),
		cluelog.WithFunc(cluelog.Span),
		cluelog.WithFunc(func(context.Context) []cluelog.KV {
			return []cluelog.KV{{K: "request_id", V: "request-123"}}
		}),
	)
	cluelog.Error(ctx, io.ErrUnexpectedEOF,
		cluelog.KV{K: cluelog.MessageKey, V: `request failed token="private-value"`},
		cluelog.KV{K: "http.status", V: 503},
		cluelog.KV{K: "access_token", V: "private-value"},
	)
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"msg":"request failed token=\"private-value\""`) {
		t.Fatalf("stdout log changed: %s", stdout.String())
	}

	select {
	case batch := <-exported:
		if len(batch.ResourceLogs) != 1 || len(batch.ResourceLogs[0].ScopeLogs) != 1 || len(batch.ResourceLogs[0].ScopeLogs[0].LogRecords) != 1 {
			t.Fatalf("unexpected exported log structure: %#v", batch.ResourceLogs)
		}
		resource := batch.ResourceLogs[0].Resource
		if got := stringAttribute(resource.Attributes, "service.name"); got != "plane-go-api-test" {
			t.Fatalf("service.name = %q", got)
		}
		record := batch.ResourceLogs[0].ScopeLogs[0].LogRecords[0]
		if got := record.SeverityText; got != "error" {
			t.Fatalf("severity = %q, want error", got)
		}
		if got := record.Body.GetStringValue(); got != `request failed token="[redacted]"` {
			t.Fatalf("body = %q", got)
		}
		if string(record.TraceId) != string(traceID[:]) || string(record.SpanId) != string(spanID[:]) {
			t.Fatalf("native trace correlation fields = trace %x span %x", record.TraceId, record.SpanId)
		}
		attributes := record.Attributes
		for key, want := range map[string]string{
			"request_id":   "request-123",
			"trace_id":     traceID.String(),
			"span_id":      spanID.String(),
			"access_token": "[redacted]",
		} {
			if got := stringAttribute(attributes, key); got != want {
				t.Errorf("attribute %s = %q, want %q", key, got, want)
			}
		}
		if got := intAttribute(attributes, "http.status"); got != 503 {
			t.Errorf("http.status = %d, want 503", got)
		}
		if record.TimeUnixNano == 0 || time.Unix(0, int64(record.TimeUnixNano)).IsZero() {
			t.Error("exported log has no event timestamp")
		}
	case <-time.After(time.Second):
		t.Fatal("OTLP log exporter did not receive the record during shutdown")
	}
}

type recordingLogger struct {
	otellog.Logger
	ctx    context.Context
	record otellog.Record
}

func (logger *recordingLogger) Emit(ctx context.Context, record otellog.Record) {
	logger.ctx = ctx
	logger.record = record
}

func (*recordingLogger) Enabled(context.Context, otellog.EnabledParameters) bool { return true }

func TestClueLogFormatterPreservesTypedFieldsAndRedactsSecrets(t *testing.T) {
	logger := &recordingLogger{}
	formatter := newClueLogFormatter(logger)
	eventTime := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	traceID := "0123456789abcdef0123456789abcdef"
	spanID := "0123456789abcdef"
	entry := &cluelog.Entry{
		Time:     eventTime,
		Severity: cluelog.SeverityWarn,
		KeyVals: []cluelog.KV{
			{K: " ", V: "ignored"},
			{K: cluelog.TimestampKey, V: "ignored"},
			{K: cluelog.SeverityKey, V: "ignored"},
			{K: cluelog.MessageKey, V: `saving token="body-secret"`},
			{K: cluelog.TraceIDKey, V: traceID},
			{K: cluelog.SpanIDKey, V: spanID},
			{K: "request_id", V: "req-42"},
			{K: "http.status", V: 503},
			{K: "cache.hit", V: true},
			{K: "attempt", V: int32(3)},
			{K: "request.bytes", V: int64(64)},
			{K: "unsigned", V: uint(5)},
			{K: "unsigned32", V: uint32(6)},
			{K: "unsigned64", V: uint64(7)},
			{K: "ratio32", V: float32(1.5)},
			{K: "ratio64", V: float64(2.5)},
			{K: "access_token", V: "field-secret"},
			{K: "metadata", V: map[string]string{"token": "nested-secret", "kind": "label"}},
		},
	}

	if formatted := formatter(entry); len(formatted) != 0 {
		t.Fatalf("formatter output length = %d, want 0 because emission is structured", len(formatted))
	}
	if logger.record.Severity() != otellog.SeverityWarn || logger.record.SeverityText() != "warn" {
		t.Fatalf("record severity = %v (%q), want warn", logger.record.Severity(), logger.record.SeverityText())
	}
	if logger.record.Timestamp() != eventTime || logger.record.ObservedTimestamp().IsZero() {
		t.Fatalf("record timestamps = (%v, %v), want event time and observed time", logger.record.Timestamp(), logger.record.ObservedTimestamp())
	}
	if got := logger.record.Body().AsString(); got != `saving token="[redacted]"` {
		t.Fatalf("record body = %q", got)
	}
	spanContext := trace.SpanContextFromContext(logger.ctx)
	if !spanContext.IsValid() || spanContext.TraceID().String() != traceID || spanContext.SpanID().String() != spanID {
		t.Fatalf("emitted span context = %v, want trace %s span %s", spanContext, traceID, spanID)
	}

	attributes := make(map[string]attribute.Value)
	logger.record.WalkAttributes(func(attribute attribute.KeyValue) bool {
		attributes[string(attribute.Key)] = attribute.Value
		return true
	})
	if got := attributes["request_id"].AsString(); got != "req-42" {
		t.Errorf("request_id = %q", got)
	}
	if got := attributes["http.status"].AsInt64(); got != 503 || attributes["http.status"].Type() != attribute.INT64 {
		t.Errorf("http.status = %v (%v), want typed int64 503", got, attributes["http.status"].Type())
	}
	if got := attributes["cache.hit"].AsBool(); !got || attributes["cache.hit"].Type() != attribute.BOOL {
		t.Errorf("cache.hit = %v (%v), want typed bool true", got, attributes["cache.hit"].Type())
	}
	if got := attributes["attempt"].AsInt64(); got != 3 || attributes["attempt"].Type() != attribute.INT64 {
		t.Errorf("attempt = %v (%v), want typed int64 3", got, attributes["attempt"].Type())
	}
	for key, want := range map[string]int64{
		"request.bytes": 64,
		"unsigned":      5,
		"unsigned32":    6,
		"unsigned64":    7,
	} {
		if got := attributes[key].AsInt64(); got != want || attributes[key].Type() != attribute.INT64 {
			t.Errorf("%s = %v (%v), want typed int64 %d", key, got, attributes[key].Type(), want)
		}
	}
	for key, want := range map[string]float64{"ratio32": 1.5, "ratio64": 2.5} {
		if got := attributes[key].AsFloat64(); got != want || attributes[key].Type() != attribute.FLOAT64 {
			t.Errorf("%s = %v (%v), want typed float64 %g", key, got, attributes[key].Type(), want)
		}
	}
	if got := attributes["access_token"].AsString(); got != "[redacted]" {
		t.Errorf("access_token = %q, want redacted", got)
	}
	if got := attributes["metadata"].AsString(); got != `{"kind":"label","token":"[redacted]"}` {
		t.Errorf("metadata = %q", got)
	}
	if _, ok := attributes[" "]; ok {
		t.Error("empty key was exported")
	}
	if _, ok := attributes[cluelog.TimestampKey]; ok {
		t.Error("timestamp field was duplicated as an attribute")
	}
	if _, ok := attributes[cluelog.SeverityKey]; ok {
		t.Error("severity field was duplicated as an attribute")
	}
}

func TestClueSeverityMapsKnownAndUnknownValues(t *testing.T) {
	for _, test := range []struct {
		input cluelog.Severity
		want  otellog.Severity
	}{
		{input: cluelog.SeverityDebug, want: otellog.SeverityDebug},
		{input: cluelog.SeverityInfo, want: otellog.SeverityInfo},
		{input: cluelog.SeverityWarn, want: otellog.SeverityWarn},
		{input: cluelog.SeverityError, want: otellog.SeverityError},
		{input: cluelog.Severity(99), want: otellog.SeverityInfo},
	} {
		if got := clueSeverity(test.input); got != test.want {
			t.Errorf("clueSeverity(%v) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestClueLogFormatterUsesDefaultBodyAndOmitsInvalidTraceContext(t *testing.T) {
	logger := &recordingLogger{}
	newClueLogFormatter(logger)(&cluelog.Entry{
		Severity: cluelog.SeverityInfo,
		KeyVals: []cluelog.KV{
			{K: cluelog.TraceIDKey, V: "invalid"},
			{K: cluelog.SpanIDKey, V: "also-invalid"},
		},
	})
	if got := logger.record.Body().AsString(); got != "application log" {
		t.Fatalf("default body = %q", got)
	}
	if spanContext := trace.SpanContextFromContext(logger.ctx); spanContext.IsValid() {
		t.Fatalf("invalid correlation IDs produced a span context: %v", spanContext)
	}
}

func TestSafeLogAttributeSerializesValuesAndRedactsFallbackText(t *testing.T) {
	encoded := safeLogAttribute("values", []int{1, 2})
	if got := encoded.Value.AsString(); got != "[1,2]" {
		t.Fatalf("JSON attribute = %q", got)
	}
	fallback := safeLogAttribute("details", map[string]any{"token": make(chan int)})
	if got := fallback.Value.AsString(); !strings.Contains(got, "token:[redacted]") {
		t.Fatalf("marshal fallback = %q, want redacted token", got)
	}
	if got := safeLogAttribute("empty", nil).Value.AsString(); got != "null" {
		t.Fatalf("nil attribute = %q, want null", got)
	}
}

func stringAttribute(attributes []*commonpb.KeyValue, key string) string {
	for _, attribute := range attributes {
		if attribute.Key == key {
			return attribute.Value.GetStringValue()
		}
	}
	return ""
}

func intAttribute(attributes []*commonpb.KeyValue, key string) int64 {
	for _, attribute := range attributes {
		if attribute.Key == key {
			return attribute.Value.GetIntValue()
		}
	}
	return 0
}
