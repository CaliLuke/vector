package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type milestoneStub struct {
	// Tests only install operations they exercise; an unexpected invocation fails.
	visible milestones.Capabilities
	err     error
	read    func(milestones.Scope, uuid.UUID, bool) (milestones.Record, error)
	list    func(milestones.Scope, djangoapi.Query) (milestones.Page[milestones.Record], error)
	links   func(milestones.Scope, uuid.UUID, djangoapi.Query) (milestones.Page[milestones.Link], error)
	write   func(context.Context, string, milestones.WriteOptions, uuid.UUID, milestones.Fields, []uuid.UUID, []uuid.UUID) (milestones.MutationResult, error)
}

func (s *milestoneStub) VisibleActions(context.Context, uuid.UUID, uuid.UUID, string) (milestones.Capabilities, error) {
	return s.visible, s.err
}
func (s *milestoneStub) Retrieve(_ context.Context, scope milestones.Scope, id uuid.UUID, revision bool) (milestones.Record, error) {
	return s.read(scope, id, revision)
}
func (s *milestoneStub) List(_ context.Context, scope milestones.Scope, q djangoapi.Query) (milestones.Page[milestones.Record], error) {
	return s.list(scope, q)
}
func (s *milestoneStub) ListWorkItems(_ context.Context, scope milestones.Scope, id uuid.UUID, q djangoapi.Query) (milestones.Page[milestones.Link], error) {
	return s.links(scope, id, q)
}
func (s *milestoneStub) Create(ctx context.Context, o milestones.WriteOptions, f milestones.Fields) (milestones.MutationResult, error) {
	return s.write(ctx, "create", o, uuid.Nil, f, nil, nil)
}
func (s *milestoneStub) Update(ctx context.Context, o milestones.WriteOptions, id uuid.UUID, f milestones.Fields) (milestones.MutationResult, error) {
	return s.write(ctx, "update", o, id, f, nil, nil)
}
func (s *milestoneStub) Delete(ctx context.Context, o milestones.WriteOptions, id uuid.UUID) (milestones.MutationResult, error) {
	return s.write(ctx, "delete", o, id, milestones.Fields{}, nil, nil)
}
func (s *milestoneStub) ManageWorkItems(ctx context.Context, o milestones.WriteOptions, id uuid.UUID, a, r []uuid.UUID) (milestones.MutationResult, error) {
	return s.write(ctx, "manage_workitems", o, id, milestones.Fields{}, a, r)
}

func TestMilestoneDiscoveryUsesNativeCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name string
		cap  milestones.Capabilities
		want []string
	}{
		{"none", milestones.Capabilities{}, nil},
		{"read", milestones.Capabilities{Read: true}, []string{"list", "retrieve", "list_workitems"}},
		{"write", milestones.Capabilities{Read: true, Write: true}, []string{"list", "retrieve", "list_workitems", "create", "update", "delete", "manage_workitems"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.milestones.visible = tc.cap
			result := rpc(t, f.handler(t, nil), "test-one", "tools/list", map[string]any{})
			if len(tc.want) == 0 {
				for _, tool := range result.Tools {
					require.NotEqual(t, "milestone", tool.Name)
				}
				return
			}
			require.ElementsMatch(t, tc.want, discoveredMilestoneActions(t, result))
			if tc.cap.Write {
				require.Len(t, result.Tools, 9)
				tool := discoveredTool(t, result, "milestone")
				require.Equal(t, "object", tool.InputSchema.Type)
				require.False(t, tool.Annotations["readOnlyHint"])
				require.True(t, tool.Annotations["destructiveHint"])
				require.False(t, tool.Annotations["openWorldHint"])
				total := 0
				for _, registered := range result.Tools {
					for _, raw := range registered.InputSchema.OneOf {
						var branch struct {
							Properties struct{ Action struct{ Enum []string } }
						}
						require.NoError(t, json.Unmarshal(raw, &branch))
						total += len(branch.Properties.Action.Enum)
					}
				}
				require.Equal(t, 28, total)
			}
		})
	}
	f := newFixture(t)
	f.milestones.err = errors.New("private database failure")
	_, err := f.service.DiscoverActions(milestoneContext(t, f))
	require.Error(t, err)
}

func discoveredMilestoneActions(t *testing.T, result rpcResult) []string {
	t.Helper()
	var actions []string
	for _, tool := range result.Tools {
		if tool.Name != "milestone" {
			continue
		}
		for _, branch := range tool.InputSchema.OneOf {
			var schema struct {
				Properties struct{ Action struct{ Enum []string } }
			}
			require.NoError(t, json.Unmarshal(branch, &schema))
			actions = append(actions, schema.Properties.Action.Enum...)
		}
	}
	return actions
}
func milestoneCall(input string) map[string]any {
	return map[string]any{"name": "milestone", "arguments": json.RawMessage(input)}
}
