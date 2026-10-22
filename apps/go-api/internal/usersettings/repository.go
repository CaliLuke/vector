package usersettings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"uuid"
)

var ErrNotFound = errors.New("user settings record not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("user settings database is required")
	}
	return &Repository{db: db}, nil
}

// Get mirrors UserMeSettingsSerializer. A last workspace is used only when
// it is an active membership; otherwise the earliest active workspace is the
// fallback. A workspace asset takes precedence over the legacy logo field,
// and no asset intentionally serializes as an empty string.
func (r *Repository) Get(ctx context.Context, user uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `WITH user_profile AS (
  SELECT u.id,u.email,p.last_workspace_id
  FROM users u JOIN profiles p ON p.user_id=u.id
  WHERE u.id=$1
), invite_count AS (
  SELECT count(*)::int AS invites
  FROM workspace_member_invites i
  JOIN user_profile up ON up.email=i.email
  WHERE i.deleted_at IS NULL
), selected_workspace AS (
  SELECT w.id,w.slug,w.name,w.logo_asset_id,up.last_workspace_id
  FROM user_profile up
  JOIN workspaces w ON w.deleted_at IS NULL
  JOIN workspace_members wm ON wm.workspace_id=w.id
    AND wm.member_id=up.id AND wm.is_active=true AND wm.deleted_at IS NULL
  ORDER BY CASE WHEN w.id=up.last_workspace_id THEN 0 ELSE 1 END,w.created_at ASC
  LIMIT 1
)
SELECT json_build_object(
  'id',up.id,
  'email',up.email,

  'workspace',(jsonb_build_object(
	'last_workspace_id',CASE WHEN sw.id=up.last_workspace_id THEN sw.id ELSE NULL END,
	'last_workspace_slug',CASE WHEN sw.id=up.last_workspace_id THEN sw.slug ELSE NULL END,
    'fallback_workspace_id',sw.id,
    'fallback_workspace_slug',sw.slug,
    'invites',ic.invites
  ) || CASE WHEN sw.id=up.last_workspace_id THEN jsonb_build_object(
    'last_workspace_name',sw.name,
    'last_workspace_logo',CASE
      WHEN sw.logo_asset_id IS NULL THEN ''
      WHEN fa.entity_type='WORKSPACE_LOGO' THEN '/api/assets/v2/static/' || sw.logo_asset_id::text || '/'
      ELSE NULL
    END
  ) ELSE '{}'::jsonb END)::json
)
FROM user_profile up
CROSS JOIN invite_count ic
LEFT JOIN selected_workspace sw ON true
LEFT JOIN file_assets fa ON fa.id=sw.logo_asset_id`, user).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}
