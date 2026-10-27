package projectmembershipreads

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memberReaderRepository struct {
	authorizeErr, readErr error
	result                any
	reads                 int
}

func (r *memberReaderRepository) AuthorizeLifecycle(context.Context, LifecycleCommand) error {
	return r.authorizeErr
}

func (r *memberReaderRepository) ExecuteLifecycle(context.Context, LifecycleCommand) (any, error) {
	r.reads++
	return r.result, r.readErr
}

func TestMemberReaderRequiresActorAndProjectScopeBeforeReading(t *testing.T) {
	id := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	for _, tc := range []struct {
		name           string
		actor, project uuid.UUID
		slug           string
	}{
		{"missing actor", uuid.UUID{}, id, "workspace"},
		{"missing project", id, uuid.UUID{}, "workspace"},
		{"missing workspace", id, id, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &memberReaderRepository{}
			reader, err := NewMemberReader(repository)
			require.NoError(t, err)
			_, err = reader.ListProjectMembers(t.Context(), tc.actor, tc.slug, tc.project)
			require.Error(t, err)
			assert.Zero(t, repository.reads)
		})
	}
	_, err := NewMemberReader(nil)
	require.Error(t, err)
}

func TestMemberReaderPreservesRepositoryResultsAndFailures(t *testing.T) {
	id := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	for _, tc := range []struct {
		name       string
		repository memberReaderRepository
		want       string
		wantReads  int
	}{
		{"roster", memberReaderRepository{result: []any{map[string]any{"id": "member", "email": nil}}}, `[{"id":"member","email":null}]`, 1},
		{"empty roster", memberReaderRepository{result: []any{}}, `[]`, 1},
		{"permission denied", memberReaderRepository{authorizeErr: ErrForbidden}, "", 0},
		{"read failure", memberReaderRepository{readErr: errors.New("database unavailable")}, "", 1},
		{"invalid repository result", memberReaderRepository{result: make(chan int)}, "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewMemberReader(&tc.repository)
			require.NoError(t, err)
			result, err := reader.ListProjectMembers(t.Context(), id, "workspace", id)
			if tc.want == "" {
				require.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tc.want, string(result))
			}
			assert.Equal(t, tc.wantReads, tc.repository.reads)
		})
	}
}
