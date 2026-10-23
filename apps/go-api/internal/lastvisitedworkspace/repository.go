package lastvisitedworkspace

import (
	"context"
	"encoding/json"
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
		return nil, fmt.Errorf("last visited workspace database is required")
	}
	return &Repository{db}, nil
}

// Get reads the migrated Profile field and explicitly mirrors the two Django
// serializers. In particular, do not reference users.last_workspace_id: that
// legacy column is absent in current installations.
func (r *Repository) Get(ctx context.Context, user uuid.UUID) (json.RawMessage, error) {
	var body []byte
	q := `WITH profile AS (
    SELECT last_workspace_id AS workspace_id FROM profiles WHERE user_id=$1
), ws AS (
    SELECT json_build_object(
        'background_color',w.background_color,'created_at',w.created_at,'created_by',w.created_by_id,
        'deleted_at',w.deleted_at,'id',w.id,'logo',w.logo,'logo_asset',w.logo_asset_id,
        'logo_url',CASE WHEN w.logo_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || w.logo_asset_id::text || '/' ELSE NULLIF(w.logo,'') END,
        'name',w.name,'organization_size',w.organization_size,'owner',w.owner_id,'slug',w.slug,
        'timezone',w.timezone,'updated_at',w.updated_at,'updated_by',w.updated_by_id
    ) AS details
    FROM workspaces w JOIN profile p ON p.workspace_id=w.id WHERE w.deleted_at IS NULL
), members AS (
    SELECT COALESCE(json_agg(json_build_object(
        'comment',pm.comment,'created_at',pm.created_at,'created_by',pm.created_by_id,
        'default_props',pm.default_props,'deleted_at',pm.deleted_at,'id',pm.id,'is_active',pm.is_active,
        'member',json_build_object(
            'avatar',u.avatar,
            'avatar_url',CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || u.avatar_asset_id::text || '/' ELSE NULLIF(u.avatar,'') END,
            'display_name',u.display_name,'first_name',u.first_name,'id',u.id,'is_bot',u.is_bot,'last_name',u.last_name
        ),
        'preferences',pm.preferences,
        'project',json_build_object(
            'cover_image',p.cover_image,
            'cover_image_url',CASE WHEN p.cover_image_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || p.cover_image_asset_id::text || '/' ELSE NULLIF(p.cover_image,'') END,
            'description',p.description,'id',p.id,'identifier',p.identifier,'logo_props',p.logo_props,'name',p.name
        ),
        'role',pm.role,'sort_order',pm.sort_order,'updated_at',pm.updated_at,'updated_by',pm.updated_by_id,
        'view_props',pm.view_props,
        'workspace',json_build_object(
            'id',w.id,
            'logo_url',CASE WHEN w.logo_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || w.logo_asset_id::text || '/' ELSE NULLIF(w.logo,'') END,
            'name',w.name,'slug',w.slug
        )
    ) ORDER BY pm.created_at DESC),'[]'::json) AS details
    FROM project_members pm
    JOIN projects p ON p.id=pm.project_id AND p.deleted_at IS NULL
    JOIN workspaces w ON w.id=p.workspace_id AND w.deleted_at IS NULL
    JOIN users u ON u.id=pm.member_id
    WHERE pm.member_id=$1 AND pm.deleted_at IS NULL AND pm.workspace_id=(SELECT workspace_id FROM profile)
), result AS (
    SELECT COALESCE((SELECT details FROM ws),'{}'::json) AS workspace_details,
           (SELECT details FROM members) AS project_details
)
SELECT json_build_object('project_details',project_details,'workspace_details',workspace_details) FROM result`
	err := r.db.QueryRow(ctx, q, user).Scan(&body)
	return body, err
}
