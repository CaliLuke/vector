package workspacememberme

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("database required")
	}
	return &Repository{db}, nil
}
func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var b []byte
	err := r.db.QueryRow(ctx, `
SELECT
    jsonb_build_object(
        'company_role', CASE WHEN wm.id IS NULL THEN '' ELSE wm.company_role END,
        'created_by', wm.created_by_id,
        'default_props', wm.default_props,
        'deleted_at', wm.deleted_at,
        'explored_features', wm.explored_features,
        'getting_started_checklist', wm.getting_started_checklist,
        'is_active', COALESCE(wm.is_active, false),
        'issue_props', wm.issue_props,
        'member', wm.member_id,
        'role', wm.role,
        'tips', wm.tips,
        'updated_by', wm.updated_by_id,
        'view_props', wm.view_props,
        'workspace', wm.workspace_id
    ) || CASE WHEN wm.id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object(
        'created_at', wm.created_at,
        'draft_issue_count', COALESCE((
            SELECT count(*)
            FROM draft_issues d
            WHERE d.created_by_id = $1 AND d.workspace_id = wm.workspace_id
        ), 0),
        'id', wm.id,
        'updated_at', wm.updated_at
    ) END
FROM workspaces w
LEFT JOIN workspace_members wm
    ON wm.workspace_id = w.id
    AND wm.member_id = $1
    AND wm.is_active = true
    AND wm.deleted_at IS NULL
WHERE w.slug = $2 AND w.deleted_at IS NULL`, user, slug).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return json.RawMessage(`{"company_role":"","created_by":null,"default_props":null,"deleted_at":null,"explored_features":null,"getting_started_checklist":null,"is_active":false,"issue_props":null,"member":null,"role":null,"tips":null,"updated_by":null,"view_props":null,"workspace":null}`), nil
	}
	return b, err
}
