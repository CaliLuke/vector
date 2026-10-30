package workspacewebhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound     = errors.New("webhook not found")
	ErrDuplicateURL = errors.New("webhook URL already exists")
	ErrIntegrity    = errors.New("webhook payload integrity error")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace webhooks database is required")
	}
	return &Repository{db: db}, nil
}

var webhookJSON = `json_build_object(
 'id',h.id,'workspace',h.workspace_id,'created_at',` + djangoapi.DjangoTimestampSQL("h.created_at") + `,'updated_at',` + djangoapi.DjangoTimestampSQL("h.updated_at") + `,
 'created_by',h.created_by_id,'updated_by',h.updated_by_id,'deleted_at',` + djangoapi.DjangoTimestampSQL("h.deleted_at") + `,
 'url',h.url,'is_active',h.is_active,'secret_key',h.secret_key,'project',h.project,
 'issue',h.issue,'module',h.module,'cycle',h.cycle,'milestone',h.milestone,'issue_comment',h.issue_comment,
 'is_internal',h.is_internal,'version',h.version)`

var webhookLogJSON = `json_build_object(
 'id',l.id,'workspace',l.workspace_id,'created_at',` + djangoapi.DjangoTimestampSQL("l.created_at") + `,'updated_at',` + djangoapi.DjangoTimestampSQL("l.updated_at") + `,
 'created_by',l.created_by_id,'updated_by',l.updated_by_id,'deleted_at',` + djangoapi.DjangoTimestampSQL("l.deleted_at") + `,
 'webhook',l.webhook,'event_type',l.event_type,'request_method',l.request_method,
 'request_headers',l.request_headers,'request_body',l.request_body,
 'response_status',l.response_status,'response_headers',l.response_headers,
 'response_body',l.response_body,'retry_count',l.retry_count)`

func (r *Repository) Authorized(ctx context.Context, actor uuid.UUID, slug string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM workspace_members wm
    JOIN workspaces w ON w.id=wm.workspace_id
    WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role=20
      AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL
)`, actor, slug).Scan(&allowed)
	return allowed, err
}

func (r *Repository) List(ctx context.Context, slug string) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
SELECT COALESCE(json_agg(item ORDER BY created_at DESC), '[]'::json)
FROM (
  SELECT `+webhookJSON+` AS item, h.created_at
  FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id
  WHERE w.slug=$1 AND w.deleted_at IS NULL AND h.deleted_at IS NULL
) rows`, slug).Scan(&data)
	return data, err
}

func (r *Repository) Get(ctx context.Context, slug string, id uuid.UUID) (json.RawMessage, error) {
	return r.webhookQuery(ctx, `WHERE h.id=$1 AND w.slug=$2 AND h.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug)
}

func (r *Repository) Create(ctx context.Context, p CreateParams) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
INSERT INTO webhooks AS h(
 id,created_at,updated_at,workspace_id,created_by_id,url,is_active,secret_key,project,issue,module,cycle,
 issue_comment,milestone,is_internal,version
)
SELECT @id::uuid,NOW(),NOW(),w.id,
 CASE WHEN @created_by_present THEN @created_by::uuid ELSE @actor_id::uuid END,
 @url,@is_active,@secret_key,@project,@issue,@module,@cycle,@issue_comment,@milestone,@is_internal,@version
FROM workspaces w WHERE w.slug=@slug AND w.deleted_at IS NULL
RETURNING `+webhookJSON, pgx.StrictNamedArgs{
		"id": uuid.New(), "actor_id": p.ActorID, "created_by_present": p.CreatedByPresent, "created_by": p.CreatedBy,
		"url": p.URL, "is_active": p.IsActive, "secret_key": p.SecretKey,
		"project": p.Project, "issue": p.Issue, "module": p.Module, "cycle": p.Cycle, "milestone": p.Milestone,
		"issue_comment": p.IssueComment, "is_internal": p.IsInternal, "version": p.Version, "slug": p.Slug,
	}).Scan(&data)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(strings.ToLower(pgErr.Message), "already exists") || pgErr.ConstraintName == "webhook_url_unique_url_when_deleted_at_null" {
				return nil, ErrDuplicateURL
			}
			return nil, fmt.Errorf("%w: %w", ErrIntegrity, err)
		}
		return nil, err
	}
	return data, nil
}

func (r *Repository) Update(ctx context.Context, p PatchParams) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
UPDATE webhooks h SET
 url=CASE WHEN @url_present THEN @url ELSE h.url END,
 is_active=CASE WHEN @is_active_present THEN @is_active ELSE h.is_active END,
 project=CASE WHEN @project_present THEN @project ELSE h.project END,
 issue=CASE WHEN @issue_present THEN @issue ELSE h.issue END,
 module=CASE WHEN @module_present THEN @module ELSE h.module END,
 cycle=CASE WHEN @cycle_present THEN @cycle ELSE h.cycle END,
 milestone=CASE WHEN @milestone_present THEN @milestone ELSE h.milestone END,
 issue_comment=CASE WHEN @issue_comment_present THEN @issue_comment ELSE h.issue_comment END,
 is_internal=CASE WHEN @is_internal_present THEN @is_internal ELSE h.is_internal END,
 version=CASE WHEN @version_present THEN @version ELSE h.version END,
 created_by_id=CASE WHEN @created_by_present THEN @created_by::uuid ELSE h.created_by_id END,
 updated_at=NOW(), updated_by_id=@actor_id
FROM workspaces w
WHERE h.id=@id AND h.workspace_id=w.id AND w.slug=@slug
 AND h.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING `+webhookJSON, pgx.StrictNamedArgs{
		"url_present": p.URLPresent, "url": p.URL,
		"is_active_present": p.IsActivePresent, "is_active": p.IsActive,
		"project_present": p.ProjectPresent, "project": p.Project,
		"issue_present": p.IssuePresent, "issue": p.Issue,
		"module_present": p.ModulePresent, "module": p.Module,
		"cycle_present": p.CyclePresent, "cycle": p.Cycle,
		"milestone_present": p.MilestonePresent, "milestone": p.Milestone,
		"issue_comment_present": p.IssueCommentPresent, "issue_comment": p.IssueComment,
		"is_internal_present": p.IsInternalPresent, "is_internal": p.IsInternal,
		"version_present": p.VersionPresent, "version": p.Version,
		"created_by_present": p.CreatedByPresent, "created_by": p.CreatedBy,
		"actor_id": p.ActorID, "id": p.ID, "slug": p.Slug,
	}).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("%w: %w", ErrIntegrity, err)
		}
	}
	return data, err
}

func (r *Repository) SoftDelete(ctx context.Context, slug string, id, actor uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
UPDATE webhooks h SET deleted_at=NOW(),updated_at=NOW(),updated_by_id=$3
FROM workspaces w WHERE h.id=$1 AND h.workspace_id=w.id AND w.slug=$2
 AND h.deleted_at IS NULL AND w.deleted_at IS NULL`, id, slug, actor)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *Repository) Regenerate(ctx context.Context, slug string, id, actor uuid.UUID, secret string) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
UPDATE webhooks h SET secret_key=$4,updated_at=NOW(),updated_by_id=$3
FROM workspaces w WHERE h.id=$1 AND h.workspace_id=w.id AND w.slug=$2
 AND h.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING `+webhookJSON, id, slug, actor, secret).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

func (r *Repository) Logs(ctx context.Context, slug string, webhookID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `
SELECT COALESCE(json_agg(item ORDER BY created_at DESC), '[]'::json)
FROM (
  SELECT `+webhookLogJSON+` AS item, l.created_at
  FROM webhook_logs l JOIN workspaces w ON w.id=l.workspace_id
  WHERE w.slug=$1 AND w.deleted_at IS NULL AND l.deleted_at IS NULL AND l.webhook=$2
) rows`, slug, webhookID).Scan(&data)
	return data, err
}

func (r *Repository) webhookQuery(ctx context.Context, where string, args ...any) (json.RawMessage, error) {
	var data []byte
	err := r.db.QueryRow(ctx, `SELECT `+webhookJSON+` FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id `+where, args...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

type CreateParams struct {
	ActorID          uuid.UUID
	Slug             string
	CreatedBy        *uuid.UUID
	CreatedByPresent bool
	URL              string
	IsActive         bool
	SecretKey        string
	Project          bool
	Issue            bool
	Module           bool
	Milestone        bool
	Cycle            bool
	IssueComment     bool
	IsInternal       bool
	Version          string
}

type PatchParams struct {
	ActorID             uuid.UUID
	Slug                string
	ID                  uuid.UUID
	URL                 string
	URLPresent          bool
	IsActive            bool
	IsActivePresent     bool
	Project             bool
	ProjectPresent      bool
	Issue               bool
	IssuePresent        bool
	Module              bool
	ModulePresent       bool
	Cycle               bool
	CyclePresent        bool
	Milestone           bool
	MilestonePresent    bool
	IssueComment        bool
	IssueCommentPresent bool
	IsInternal          bool
	IsInternalPresent   bool
	Version             string
	VersionPresent      bool
	CreatedBy           *uuid.UUID
	CreatedByPresent    bool
}
