package mcpactions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stateStub struct {
	visible states.StateCapabilities
	read    func(context.Context, string, djangoapi.ProjectReadRequest) (any, error)
	write   func(context.Context, states.MutationCommand) (states.MutationResult, error)
}

func (s *stateStub) VisibleActions(context.Context, uuid.UUID, uuid.UUID, string) (states.StateCapabilities, error) {
	return s.visible, nil
}
func (s *stateStub) ReadAPIV1(ctx context.Context, actor string, request djangoapi.ProjectReadRequest) (any, error) {
	if s.read == nil {
		return nil, errors.New("unexpected state read")
	}
	return s.read(ctx, actor, request)
}
func (s *stateStub) MutateState(ctx context.Context, cmd states.MutationCommand) (states.MutationResult, error) {
	if s.write == nil {
		return states.MutationResult{}, errors.New("unexpected state write")
	}
	return s.write(ctx, cmd)
}
func TestStateDiscoveryFiltersCurrentCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cap   states.StateCapabilities
		count int
	}{
		{"none", states.StateCapabilities{}, 0}, {"read", states.StateCapabilities{Read: true}, 2}, {"write", states.StateCapabilities{Read: true, Write: true}, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.states.visible = tc.cap
			result := rpc(t, f.handler(t, nil), "test-one", "tools/list", map[string]any{})
			found := false
			for _, tool := range result.Tools {
				if tool.Name == "state" {
					found = true
					require.Len(t, tool.InputSchema.OneOf, tc.count)
				}
			}
			require.Equal(t, tc.count != 0, found)
		})
	}
}
