package mcpactions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	projects "github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestProjectMutationsEmitSanitizedCorrelatedOutcomes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous); require.NoError(t, provider.Shutdown(context.Background())) })
	for _, tc := range []struct {
		name, outcome string
		err           error
		replay        bool
	}{
		{name: "commit", outcome: "committed"},
		{name: "replay", outcome: "committed", replay: true},
		{name: "stale", outcome: "revision_conflict", err: projects.ErrRevisionConflict},
		{name: "changed input", outcome: "request_conflict", err: mutationreceipts.ErrConflict},
		{name: "uncertain", outcome: "indeterminate", err: mutationreceipts.ErrIndeterminate},
		{name: "denied", outcome: "forbidden", err: projects.ErrForbidden},
		{name: "failure", outcome: "internal_error", err: errors.New("private-database-marker")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exporter.Reset()
			var output bytes.Buffer
			logger := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithFunc(log.Span), log.WithDisableBuffering(func(context.Context) bool { return true }))
			f := newFixture(t, mcpactions.WithLogContext(logger))
			f.projectWrites.err = tc.err
			f.projectWrites.result.Receipt.Replayed = tc.replay
			expiry := time.Now().Add(48 * time.Hour)
			f.projectWrites.result.Receipt.ReplayExpiresAt = &expiry
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", projectCall(`{"action":"archive","project_id":"30000000-0000-4000-8000-000000000001","request_id":"40000000-0000-4000-8000-000000000001","expected_revision":"private-revision-marker"}`))
			require.Equal(t, tc.err != nil, result.IsError)
			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record), output.String())
			require.Equal(t, tc.outcome, record["outcome"])
			require.Equal(t, "project", record["mcp_tool"])
			require.Equal(t, "archive", record["mcp_action"])
			require.Equal(t, "10000000-0000-4000-8000-000000000001", record["actor_id"])
			require.Equal(t, "20000000-0000-4000-8000-000000000001", record["workspace_id"])
			require.Equal(t, "personal_token", record["credential_class"])
			require.Equal(t, tc.replay, record["replayed"])
			require.Contains(t, record, "latency_ms")
			require.NotEmpty(t, record[log.RequestIDKey])
			require.NotContains(t, output.String(), "private-")
			require.NotContains(t, output.String(), "test-one")
			require.NotContains(t, output.String(), "40000000-")
			if tc.err == nil {
				require.Equal(t, "30000000-0000-4000-8000-000000000001", record["committed_project_id"])
			} else {
				require.NotContains(t, record, "committed_project_id")
			}
			spans := exporter.GetSpans()
			var found bool
			for _, span := range spans {
				if span.Name != "mcp.project.archive" {
					continue
				}
				found = true
				require.Contains(t, output.String(), span.SpanContext.TraceID().String())
				require.Contains(t, output.String(), span.SpanContext.SpanID().String())
				if tc.err != nil {
					require.Equal(t, codes.Error, span.Status.Code)
				}
				attrs := map[string]any{}
				for _, attr := range span.Attributes {
					attrs[string(attr.Key)] = attr.Value.AsInterface()
				}
				require.Equal(t, tc.outcome, attrs["outcome"])
				require.Equal(t, record[log.RequestIDKey], attrs[log.RequestIDKey])
				encoded, err := json.Marshal(attrs)
				require.NoError(t, err)
				require.NotContains(t, string(encoded), "private-")
			}
			require.True(t, found, "missing mutation span")
		})
	}
}
