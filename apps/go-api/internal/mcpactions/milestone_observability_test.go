package mcpactions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMilestoneMutationsLogOnlySanitizedCommittedIdentities(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(previous); require.NoError(t, provider.Shutdown(context.Background())) })
	for _, denied := range []bool{false, true} {
		t.Run(fmt.Sprint(denied), func(t *testing.T) {
			exporter.Reset()
			var output bytes.Buffer
			logger := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithFunc(log.Span), log.WithDisableBuffering(func(context.Context) bool { return true }))
			f := newFixture(t, mcpactions.WithLogContext(logger))
			f.milestones.write = func(_ context.Context, _ string, opts milestones.WriteOptions, _ uuid.UUID, _ milestones.Fields, _, _ []uuid.UUID) (milestones.MutationResult, error) {
				if denied {
					return milestones.MutationResult{}, fmt.Errorf("private-database-marker: %w", milestones.ErrConflict)
				}
				expires := time.Now().Add(48 * time.Hour)
				return milestones.MutationResult{MilestoneID: milestoneTestID, Revision: "private-revision-marker", Result: &milestones.Record{ID: milestoneTestID, Title: "private-title-marker", CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-02T03:04:05Z"}, RequestID: opts.RequestID, ReplayExpiresAt: &expires, Replayed: new(true)}, nil
			}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(`{"action":"create","project_id":"`+readProject+`","title":"private-title-marker","external_id":"private-external-marker","request_id":"60000000-0000-4000-8000-000000000001"}`))
			require.Equal(t, denied, result.IsError)
			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, "milestone", record["mcp_tool"])
			require.Equal(t, "create", record["mcp_action"])
			require.Equal(t, "personal_token", record["credential_class"])
			require.Contains(t, record, "latency_ms")
			require.NotContains(t, output.String(), "private-")
			require.NotContains(t, output.String(), "test-one")
			require.NotContains(t, output.String(), "60000000-")
			if denied {
				require.Equal(t, "conflict", record["outcome"])
				require.NotContains(t, record, "committed_milestone_id")
				require.NotContains(t, record, "committed_project_id")
			} else {
				require.Equal(t, "committed", record["outcome"])
				require.Equal(t, milestoneTestID, record["committed_milestone_id"])
				require.Equal(t, readProject, record["committed_project_id"])
				require.Equal(t, true, record["replayed"])
			}
			var found bool
			for _, span := range exporter.GetSpans() {
				if span.Name == "mcp.milestone.create" {
					found = true
					require.Contains(t, output.String(), span.SpanContext.TraceID().String())
					attrs := map[string]any{}
					for _, attr := range span.Attributes {
						attrs[string(attr.Key)] = attr.Value.AsInterface()
					}
					require.Equal(t, record["outcome"], attrs["outcome"])
					encoded, err := json.Marshal(attrs)
					require.NoError(t, err)
					require.NotContains(t, string(encoded), "private-")
				}
			}
			require.True(t, found)
		})
	}
}
