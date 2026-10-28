//go:build integration

package projectstatereads

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type stateScopeTransaction struct{ pgx.Tx }

func (tx *stateScopeTransaction) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return tx.Begin(ctx)
}

func TestStateMutationsAndDiscoveryRecheckCurrentScope(t *testing.T) {
	f := newStateMutationFixture(t)
	state := f.create(t, "Scoped")
	cmd := f.command("update", state)
	for _, test := range []struct {
		name, change string
		read, write  bool
		failure      error
	}{
		{"actor inactive", `UPDATE users SET is_active=false`, false, false, ErrForbidden},
		{"workspace deleted", `UPDATE workspaces SET deleted_at=now()`, false, false, ErrForbidden},
		{"workspace role unknown", `UPDATE workspace_members SET role=99`, false, false, ErrForbidden},
		{"workspace membership revoked", `UPDATE workspace_members SET is_active=false`, false, false, ErrForbidden},
		{"project archived", `UPDATE projects SET archived_at=now()`, true, false, ErrForbidden},
		{"project deleted", `UPDATE projects SET deleted_at=now()`, true, false, ErrForbidden},
		{"project guest", `UPDATE project_members SET role=5`, true, false, ErrForbidden},
		{"project membership revoked", `UPDATE project_members SET is_active=false`, true, false, ErrForbidden},
		{"state deleted", `UPDATE states SET deleted_at=now()`, true, true, ErrNotFound},
		{"triage group", `UPDATE states SET "group"='triage'`, true, true, ErrNotFound},
		{"triage flag", `UPDATE states SET is_triage=true`, true, true, ErrNotFound},
		{"workspace slug renamed", `UPDATE workspaces SET slug='renamed'`, false, false, ErrForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx, err := f.db.Begin(t.Context())
			require.NoError(t, err)
			defer func() { _ = tx.Rollback(context.WithoutCancel(t.Context())) }()
			_, err = tx.Exec(t.Context(), test.change)
			require.NoError(t, err)
			// A savepoint-capable wrapper lets both operations observe this
			// uncommitted fixture state without changing production admission.
			db := &stateScopeTransaction{Tx: tx}
			repository, err := NewWriteRepository(db)
			require.NoError(t, err)
			visible, err := repository.VisibleActions(t.Context(), cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug)
			require.NoError(t, err)
			require.Equal(t, StateCapabilities{Read: test.read, Write: test.write}, visible)
			attempt := cmd
			attempt.RequestID = stateValue(uuid.New())
			_, err = repository.MutateState(t.Context(), attempt)
			require.ErrorIs(t, err, test.failure)
		})
	}
}
