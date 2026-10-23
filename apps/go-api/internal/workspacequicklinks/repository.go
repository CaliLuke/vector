package workspacequicklinks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("workspace quick link not found")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace quick links database is required")
	}
	return &Repository{db: db}, nil
}

const quickLinkJSON = `json_build_object('id',q.id,'created_at',q.created_at,'updated_at',q.updated_at,'created_by',q.created_by_id,'updated_by',q.updated_by_id,'deleted_at',q.deleted_at,'workspace',q.workspace_id,'project',q.project_id,'owner',q.owner_id,'title',q.title,'url',q.url,'metadata',q.metadata)`

func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (5,15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT COALESCE(json_agg(link ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+quickLinkJSON+` AS link,q.created_at FROM workspace_user_links q JOIN workspaces w ON w.id=q.workspace_id WHERE w.slug=$1 AND q.owner_id=$2 AND q.deleted_at IS NULL AND w.deleted_at IS NULL) x`, slug, user).Scan(&body)
	return body, err
}

func (r *Repository) Get(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT `+quickLinkJSON+` FROM workspace_user_links q JOIN workspaces w ON w.id=q.workspace_id WHERE q.id=$1 AND w.slug=$2 AND q.owner_id=$3 AND q.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, user).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) Create(ctx context.Context, user uuid.UUID, slug, url string, title *string, metadata json.RawMessage) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `INSERT INTO workspace_user_links AS q(workspace_id,owner_id,title,url,metadata,created_by_id) SELECT w.id,$1,$2,$3,$4,$1 FROM workspaces w WHERE w.slug=$5 AND w.deleted_at IS NULL RETURNING `+quickLinkJSON, user, title, url, metadata, slug).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) Update(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID, titlePresent bool, title *string, url *string, metadataPresent bool, metadata json.RawMessage) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `UPDATE workspace_user_links q SET title=CASE WHEN $4 THEN $5::varchar ELSE q.title END,url=COALESCE($6,q.url),metadata=CASE WHEN $7 THEN $8::jsonb ELSE q.metadata END,updated_at=NOW(),updated_by_id=$3 FROM workspaces w WHERE q.id=$1 AND q.workspace_id=w.id AND w.slug=$2 AND q.owner_id=$3 AND q.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING `+quickLinkJSON, id, slug, user, titlePresent, title, url, metadataPresent, metadata).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return body, err
}

func (r *Repository) Delete(ctx context.Context, user uuid.UUID, slug string, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE workspace_user_links q SET deleted_at=NOW(),updated_at=NOW(),updated_by_id=$3 FROM workspaces w WHERE q.id=$1 AND q.workspace_id=w.id AND w.slug=$2 AND q.owner_id=$3 AND q.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, user)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
