package mcpauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Workspace struct {
	ID   uuid.UUID
	Role int
}

type databaseReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type WorkspaceReader struct{ primary databaseReader }

func NewWorkspaceReader(primary databaseReader) (*WorkspaceReader, error) {
	if primary == nil {
		return nil, fmt.Errorf("MCP workspace primary database is required")
	}
	return &WorkspaceReader{primary: primary}, nil
}

// Membership reads current membership from the primary; it never trusts token
// workspace metadata or grants access based on a workspace slug alone.
func (r *WorkspaceReader) Membership(ctx context.Context, user uuid.UUID, slug string) (Workspace, error) {
	var workspace Workspace
	err := r.primary.QueryRow(ctx, `
SELECT w.id, wm.role
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE wm.member_id = $1 AND w.slug = $2
  AND wm.is_active = true AND wm.deleted_at IS NULL
  AND w.deleted_at IS NULL
LIMIT 1`, user, slug).Scan(&workspace.ID, &workspace.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, mcptransport.ErrAccessDenied
	}
	if err != nil {
		return Workspace{}, fmt.Errorf("read MCP workspace membership: %w", err)
	}
	return workspace, nil
}
