package workspacefavorites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("workspace favorite not found")
var ErrAlreadyExists = errors.New("workspace favorite already exists")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace favorites database is required")
	}
	return &Repository{db: db}, nil
}

// CreateInput mirrors the writable fields of UserFavorite. A nil pointer means
// that the corresponding nullable field should be NULL (or use its default).
type CreateInput struct {
	EntityType       string
	EntityIdentifier *uuid.UUID
	Name             *string
	IsFolder         bool
	Sequence         *float64
	Parent           *uuid.UUID
	ProjectID        *uuid.UUID
}

// UpdateInput carries PATCH presence separately from values. This is needed
// because nil is a valid value for entity_identifier, name, parent, and
// project_id.
type UpdateInput struct {
	EntityTypePresent       bool
	EntityType              *string
	EntityIdentifierPresent bool
	EntityIdentifier        *uuid.UUID
	NamePresent             bool
	Name                    *string
	IsFolderPresent         bool
	IsFolder                *bool
	SequencePresent         bool
	Sequence                *float64
	ParentPresent           bool
	Parent                  *uuid.UUID
	ProjectIDPresent        bool
	ProjectID               *uuid.UUID
}

const favoriteEntityData = `CASE f.entity_type
WHEN 'project' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props) FROM projects e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'page' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',(SELECT pp.project_id FROM project_pages pp JOIN projects ep ON ep.id=pp.project_id AND ep.deleted_at IS NULL WHERE pp.page_id=e.id AND pp.deleted_at IS NULL ORDER BY pp.created_at DESC LIMIT 1)) FROM pages e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'cycle' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM cycles e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'module' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM modules e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
WHEN 'view' THEN (SELECT json_build_object('id',e.id,'name',e.name,'logo_props',e.logo_props,'project_id',e.project_id) FROM issue_views e WHERE e.id=f.entity_identifier AND e.deleted_at IS NULL)
ELSE NULL END`

const favoriteJSON = `json_build_object('id',f.id,'entity_type',f.entity_type,'entity_identifier',f.entity_identifier,'entity_data',` + favoriteEntityData + `,'name',f.name,'is_folder',f.is_folder,'sequence',f.sequence,'parent',f.parent_id,'workspace_id',f.workspace_id,'project_id',f.project_id)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(favorite ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+favoriteJSON+` AS favorite,f.created_at FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.user_id=$1 AND w.slug=$2 AND f.parent_id IS NULL AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND ((f.project_id IS NULL AND f.entity_type <> 'page') OR EXISTS (SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id WHERE pm.project_id=f.project_id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL AND p.deleted_at IS NULL))) x`, user, slug).Scan(&body)
	return body, err
}

func (r *Repository) Group(ctx context.Context, user uuid.UUID, slug string, parent uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(favorite ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+favoriteJSON+` AS favorite,f.created_at FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.user_id=$1 AND w.slug=$2 AND f.parent_id=$3 AND f.deleted_at IS NULL AND w.deleted_at IS NULL AND (f.project_id IS NULL OR EXISTS (SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id WHERE pm.project_id=f.project_id AND pm.member_id=$1 AND pm.is_active=true AND pm.deleted_at IS NULL AND p.deleted_at IS NULL))) x`, user, slug, parent).Scan(&body)
	return body, err
}

func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT `+favoriteJSON+` FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE f.id=$3 AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL`, user, slug, id).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug string, input CreateInput) (json.RawMessage, error) {
	var body []byte
	// Django returns the existing favorite when an entity identifier is given.
	// The CTE also makes the check and insert one statement, while the partial
	// unique constraint remains the final protection against races.
	err := r.db.QueryRow(ctx, `WITH existing AS (SELECT f.id FROM user_favorites f JOIN workspaces w ON w.id=f.workspace_id WHERE $4 IS NOT NULL AND f.user_id=$1 AND w.slug=$2 AND f.entity_type=$3 AND f.entity_identifier=$4 AND f.deleted_at IS NULL AND w.deleted_at IS NULL ORDER BY f.created_at ASC LIMIT 1), inserted AS (INSERT INTO user_favorites AS f(workspace_id,user_id,entity_type,entity_identifier,name,is_folder,sequence,parent_id,project_id,created_by_id,id,created_at,updated_at) SELECT w.id,$1,$3,$4,$5,$6,CASE WHEN EXISTS (SELECT 1 FROM user_favorites previous WHERE previous.workspace_id=COALESCE(p.workspace_id,w.id) AND previous.deleted_at IS NULL) THEN COALESCE((SELECT MAX(previous.sequence) FROM user_favorites previous WHERE previous.workspace_id=COALESCE(p.workspace_id,w.id) AND previous.deleted_at IS NULL),65535)+10000 ELSE COALESCE($7,65535) END,$8,$9,$1,$10,NOW(),NOW() FROM workspaces w LEFT JOIN projects p ON p.id=$9 AND p.deleted_at IS NULL WHERE w.slug=$2 AND w.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM existing) RETURNING f.id), selected AS (SELECT id FROM existing UNION ALL SELECT id FROM inserted LIMIT 1) SELECT `+favoriteJSON+` FROM user_favorites f JOIN selected s ON s.id=f.id`, user, slug, input.EntityType, input.EntityIdentifier, input.Name, input.IsFolder, input.Sequence, input.Parent, input.ProjectID, uuid.NewV7()).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrAlreadyExists
	}
	return body, err
}

func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, input UpdateInput) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `UPDATE user_favorites f SET entity_type=CASE WHEN $4 THEN $5 ELSE f.entity_type END,entity_identifier=CASE WHEN $6 THEN $7 ELSE f.entity_identifier END,name=CASE WHEN $8 THEN $9 ELSE f.name END,is_folder=CASE WHEN $10 THEN $11 ELSE f.is_folder END,sequence=CASE WHEN $12 THEN $13 ELSE f.sequence END,parent_id=CASE WHEN $14 THEN $15 ELSE f.parent_id END,project_id=CASE WHEN $16 THEN $17 ELSE f.project_id END,updated_at=NOW(),updated_by_id=$1 FROM workspaces w WHERE f.id=$3 AND f.workspace_id=w.id AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING `+favoriteJSON, user, slug, id, input.EntityTypePresent, input.EntityType, input.EntityIdentifierPresent, input.EntityIdentifier, input.NamePresent, input.Name, input.IsFolderPresent, input.IsFolder, input.SequencePresent, input.Sequence, input.ParentPresent, input.Parent, input.ProjectIDPresent, input.ProjectID).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, ErrAlreadyExists
	}
	return body, err
}

func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_favorites f USING workspaces w WHERE f.id=$3 AND f.workspace_id=w.id AND f.user_id=$1 AND w.slug=$2 AND f.deleted_at IS NULL AND w.deleted_at IS NULL`, user, slug, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
