package mcpauth

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceStoreOutageIsNotReportedAsAbsentMembership(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgresql://localhost/unused")
	require.NoError(t, err)
	pool.Close()
	reader, err := NewWorkspaceReader(pool)
	require.NoError(t, err)

	workspace, err := reader.Membership(t.Context(), uuid.New(), "workspace-one")

	require.Error(t, err)
	require.NotErrorIs(t, err, mcptransport.ErrAccessDenied)
	require.Equal(t, uuid.Nil, workspace.ID)
}
