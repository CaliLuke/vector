package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5/pgconn"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestPropertyFailureDiagnosticsPreserveCorrelationAndPrivacy(t *testing.T) {
	const private = "private property value and database detail"
	cases := []struct {
		name                   string
		failure                error
		kind, status, sqlstate string
	}{
		{"database", fmt.Errorf("%s: %w", private, &pgconn.PgError{Code: "23505", Message: private, Detail: private, Where: private}), "postgres", "status_500", "23505"},
		{"canceled", fmt.Errorf("%s: %w", private, context.Canceled), "context_canceled", "status_500", ""},
		{"conflict", &store.Error{Status: 409, Message: "Catalog changed", ReviewAgain: true}, "", "status_409", ""},
		{"validation", &store.Error{Status: 422, Message: "Invalid definition"}, "", "status_422", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewWorkItemTypes(&typeServiceRepository{failure: tc.failure}, typeServiceSessions{})
			if err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
			spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
			ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled}))
			ctx = log.Context(ctx, log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}), log.WithFunc(log.Span), log.WithDisableBuffering(func(context.Context) bool { return true }))
			ctx, err = service.(api.Authorizer).APIKeyAuth(ctx, "valid", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.MutateProperty(ctx, &api.MutatePropertyPayload{Slug: "private-workspace", Data: &api.PropertyInput{Action: "create", Definition: &api.PropertyDefinition{Name: private, Kind: "text"}}})
			var response *api.TypeError
			if !errors.As(err, &response) || response.Kind != tc.status {
				t.Fatalf("response=%v", err)
			}
			if strings.Contains(response.Message, private) || strings.Contains(logs.String(), private) || strings.Contains(logs.String(), "private-workspace") {
				t.Fatal("diagnostics leaked private data")
			}
			if tc.kind == "" {
				if logs.Len() != 0 {
					t.Fatalf("expected rejection logged as infrastructure failure: %s", logs.String())
				}
				if tc.status == "status_409" && (response.ReviewAgain == nil || !*response.ReviewAgain) {
					t.Fatal("review-again contract lost")
				}
				return
			}
			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("missing diagnostic: %v, %s", err, logs.String())
			}
			if entry["operation"] != "work_item_types.properties.create" || entry["error_kind"] != tc.kind || entry["trace_id"] != traceID.String() || entry["span_id"] != spanID.String() {
				t.Fatalf("incomplete diagnostic: %#v", entry)
			}
			if tc.sqlstate != "" && entry["sqlstate"] != tc.sqlstate {
				t.Fatalf("SQLSTATE lost: %#v", entry)
			}
		})
	}
}

func TestPropertyCommandsAnnotateSuccessAndRejectionSpans(t *testing.T) {
	for _, tc := range []struct {
		action, want string
		failure      error
	}{
		{"create", "create", nil},
		{"create", "create", &store.Error{Status: 409, Message: "Catalog changed"}},
		{"private arbitrary input", "unknown", &store.Error{Status: 400, Message: "Unknown action"}},
	} {
		t.Run(tc.want+fmt.Sprint(tc.failure != nil), func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			defer func() { _ = provider.Shutdown(context.Background()) }()
			ctx, span := provider.Tracer("test").Start(context.Background(), "property command")
			service, err := NewWorkItemTypes(&typeServiceRepository{failure: tc.failure}, typeServiceSessions{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, err = service.(api.Authorizer).APIKeyAuth(ctx, "valid", nil)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = service.MutateProperty(ctx, &api.MutatePropertyPayload{Slug: "workspace", Data: &api.PropertyInput{Action: tc.action, Definition: &api.PropertyDefinition{Name: "Task code", Kind: "text"}}})
			span.End()
			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("span count=%d", len(spans))
			}
			attributes := spans[0].Attributes()
			if len(attributes) != 1 || string(attributes[0].Key) != "work_item_types.property.action" || attributes[0].Value.AsString() != tc.want {
				t.Fatalf("unsafe/missing command attribute: %v", attributes)
			}
		})
	}
}
