package mcpactions_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpactions"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

func TestStateMutationTelemetryExcludesInputsAndRawFailures(t *testing.T) {
	for _, outcome := range []string{"committed", "workspace_not_managed", "internal_error"} {
		t.Run(outcome, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			f := newFixture(t, mcpactions.WithLogContext(logger))
			f.states.write = func(_ context.Context, cmd states.MutationCommand) (states.MutationResult, error) {
				switch outcome {
				case "workspace_not_managed":
					return states.MutationResult{}, states.ErrWorkspaceNotManaged
				case "internal_error":
					return states.MutationResult{}, errors.New("private-sql-marker")
				}
				expires := time.Now().Add(time.Hour)
				return states.MutationResult{StateID: milestoneTestID, Revision: "private-revision", Result: json.RawMessage(`{"name":"private-name","color":"private-color"}`), RequestID: cmd.RequestID, ReplayExpiresAt: &expires, Replayed: new(true)}, nil
			}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", map[string]any{"name": "state", "arguments": map[string]any{"action": "create", "project_id": readProject, "name": "private-name", "color": "private-color", "external_id": "private-external", "request_id": "60000000-0000-4000-8000-000000000001"}})
			require.Equal(t, outcome != "committed", result.IsError)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &fields))
			require.Equal(t, "state", fields["mcp_tool"])
			require.Equal(t, "create", fields["mcp_action"])
			require.Equal(t, outcome, fields["outcome"])
			require.Contains(t, fields, "latency_ms")
			for _, secret := range []string{"private-", "test-one", "60000000-"} {
				require.NotContains(t, output.String(), secret)
			}
			if outcome == "committed" {
				require.Equal(t, milestoneTestID, fields["committed_state_id"])
				require.Equal(t, true, fields["replayed"])
			} else {
				require.NotContains(t, fields, "committed_state_id")
				require.NotContains(t, fields, "committed_project_id")
			}
		})
	}
}
