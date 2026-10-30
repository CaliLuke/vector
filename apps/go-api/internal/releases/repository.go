// Package releases owns workspace delivery catalogues and item associations.
package releases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ID, ReleaseID uuid.UUID
	WorkspaceSlug, Tool, Action         string
	Params                              map[string]string
	Present                             map[string]bool
	IncludeRevision                     bool
}
type Command struct {
	Query
	Fields           map[string]any
	Add, Remove      []uuid.UUID
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

func token(workspace uuid.UUID, revision int64) string {
	return fmt.Sprintf("release:%s:%d", workspace, revision)
}
func table(tool string) string {
	switch tool {
	case "release":
		return "work_item_releases"
	case "release_label":
		return "release_labels"
	case "release_tag":
		return "release_tags"
	}
	return ""
}
func workspaceScope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	if q.ActorID == uuid.Nil || q.WorkspaceID == uuid.Nil || q.WorkspaceSlug == "" || table(q.Tool) == "" {
		return projectmutations.ErrInvalid
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE OF u,wm,w"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, q.ActorID, q.WorkspaceID, q.WorkspaceSlug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrForbidden
	}
	if err != nil {
		return err
	}
	if write && role != 20 {
		return projectmutations.ErrForbidden
	}
	return nil
}
func releaseExists(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID) error {
	var found uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM work_item_releases WHERE id=$1 AND workspace_id=$2 AND project_id IS NULL AND deleted_at IS NULL`, id, q.WorkspaceID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrNotFound
	}
	return err
}
func (r *Repository) Read(ctx context.Context, q Query) (_ json.RawMessage, resultErr error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err = workspaceScope(ctx, tx, q, false); err != nil {
		return nil, err
	}
	var value any
	switch q.Action {
	case "retrieve":
		value, err = record(ctx, tx, q, q.ID)
	case "get_changelog":
		if q.Tool != "release" {
			return nil, projectmutations.ErrInvalid
		}
		if err = releaseExists(ctx, tx, q, q.ID); err == nil {
			value, err = changelog(ctx, tx, q, q.ID)
		}
	case "list", "list_workitems":
		value, err = page(ctx, tx, q)
	default:
		return nil, projectmutations.ErrInvalid
	}
	if err != nil {
		return nil, err
	}
	if q.IncludeRevision {
		var revision int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM release_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&revision); err != nil {
			return nil, err
		}
		value.(map[string]any)["revision"] = token(q.WorkspaceID, revision)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func rawRows(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]map[string]any, error) {
	return djangoapi.ReadJSONRows(ctx, tx, query, args...)
}
func record(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID) (map[string]any, error) {
	suffix := ""
	if q.Tool == "release" {
		suffix = " AND t.project_id IS NULL"
	}
	rows, err := rawRows(ctx, tx, `SELECT to_jsonb(t) FROM `+table(q.Tool)+` t WHERE t.id=$1 AND t.workspace_id=$2 AND t.deleted_at IS NULL`+suffix, id, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, projectmutations.ErrNotFound
	}
	return project(ctx, tx, q, rows[0])
}
func page(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	limit, p, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return nil, err
	}
	from := ` FROM ` + table(q.Tool) + ` t WHERE t.workspace_id=$1 AND t.deleted_at IS NULL`
	args := []any{q.WorkspaceID}
	order := "t.created_at DESC,t.id"
	if q.Tool == "release" {
		from += " AND t.project_id IS NULL"
	}
	if q.Tool == "release_label" {
		order = "t.sort_order,t.name,t.id"
		if q.ReleaseID != uuid.Nil {
			if err = releaseExists(ctx, tx, q, q.ReleaseID); err != nil {
				return nil, err
			}
			args = append(args, q.ReleaseID)
			from += ` AND EXISTS(SELECT 1 FROM release_item_labels rel WHERE rel.label_id=t.id AND rel.workspace_id=t.workspace_id AND rel.release_id=$2)`
		}
	}
	rowSQL := "to_jsonb(t)"
	if q.Action == "list_workitems" {
		if q.Tool != "release" {
			return nil, projectmutations.ErrInvalid
		}
		if err = releaseExists(ctx, tx, q, q.ID); err != nil {
			return nil, err
		}
		args = []any{q.WorkspaceID, q.ID, q.ActorID}
		from = ` FROM release_work_items rel JOIN issues t ON t.id=rel.issue_id AND t.project_id=rel.project_id AND t.workspace_id=rel.workspace_id WHERE rel.workspace_id=$1 AND rel.release_id=$2 AND ` + djangoapi.IssueVisibleSQL("t", "$3::uuid", "$1::uuid")
		rowSQL = `jsonb_build_object('id',t.id,'name',t.name,'project_id',t.project_id)`
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, err
	}
	n := len(args)
	args = append(args, limit, offset)
	rows, err := rawRows(ctx, tx, `SELECT `+rowSQL+from+` ORDER BY `+order+fmt.Sprintf(" LIMIT $%d OFFSET $%d", n+1, n+2), args...)
	if err != nil {
		return nil, err
	}
	values := []any{}
	for _, row := range rows {
		if q.Action == "list_workitems" {
			values = append(values, row)
		} else {
			value, err := project(ctx, tx, q, row)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
	}
	return djangoapi.Envelope(values, total, limit, p), nil
}
func project(ctx context.Context, tx pgx.Tx, q Query, row map[string]any) (map[string]any, error) {
	keys := []string{"id", "name", "color", "sort_order"}
	if q.Tool == "release_tag" {
		keys = []string{"id", "version", "description", "commit_hash", "git_tag", "created_at", "updated_at", "name", "color", "sort_order"}
	}
	if q.Tool == "release" {
		keys = []string{"id", "name", "status", "target_date", "release_date", "is_prerelease", "external_source", "external_id", "created_at", "updated_at", "start_date", "logo_props"}
	}
	out := map[string]any{"workspace": row["workspace_id"]}
	for _, key := range keys {
		out[key] = row[key]
	}
	if q.Tool != "release" {
		return out, nil
	}
	out["created_by"], out["updated_by"] = row["created_by_id"], row["updated_by_id"]
	out["description"] = map[string]any{"id": row["description_id"], "description_html": row["description_html"], "description_json": row["description_json"], "description_stripped": row["description"]}
	var lead, tag *string
	var latest bool
	err := tx.QueryRow(ctx, `SELECT
 (SELECT u.id::text FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) LIMIT 1),
 (SELECT id::text FROM release_tags WHERE id=$3 AND workspace_id=$2 AND deleted_at IS NULL),
 COALESCE($4::uuid=(SELECT id FROM work_item_releases WHERE workspace_id=$2 AND project_id IS NULL AND deleted_at IS NULL AND status='released' AND NOT is_prerelease ORDER BY release_date DESC NULLS LAST,created_at DESC,id DESC LIMIT 1),false)`, row["lead_id"], q.WorkspaceID, row["tag_id"], row["id"]).Scan(&lead, &tag, &latest)
	out["lead"], out["tag"], out["is_latest"] = lead, tag, latest
	return out, err
}
func changelog(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID) (map[string]any, error) {
	rows, err := rawRows(ctx, tx, `SELECT jsonb_build_object('id',c.id,'changelog',jsonb_build_object('id',c.id,'description_html',c.description_html,'description_json',c.description_json,'description_stripped',c.description_stripped),'created_at',c.created_at,'updated_at',c.updated_at,'created_by',c.created_by_id,'updated_by',c.updated_by_id,'workspace',c.workspace_id,'release',c.release_id) FROM release_changelogs c WHERE c.release_id=$1 AND c.workspace_id=$2`, id, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, projectmutations.ErrNotFound
	}
	return rows[0], nil
}
