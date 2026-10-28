package initiatives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrDisabled = errors.New("workspace initiatives are disabled")

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

type Query struct {
	ActorID, WorkspaceID, ID uuid.UUID
	WorkspaceSlug, Action    string
	Params                   map[string]string
	Present                  map[string]bool
	IncludeRevision          bool
}
type Command struct {
	Query
	Fields           map[string]any
	Add, Remove      []uuid.UUID
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Revalidate       func(context.Context) error
}

func token(workspace uuid.UUID, rev int64) string {
	return fmt.Sprintf("initiative:%s:%d", workspace, rev)
}
func scope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	role, err := projectmutations.WorkspaceRole(ctx, tx, q.ActorID, q.WorkspaceID, q.WorkspaceSlug, write)
	if err != nil {
		return err
	}
	if write && role != 20 {
		return projectmutations.ErrForbidden
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE"
	}
	var enabled bool
	if err = tx.QueryRow(ctx, `SELECT initiatives FROM workspace_features WHERE workspace_id=$1`+suffix, q.WorkspaceID).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return ErrDisabled
	}
	return nil
}
func rawRows(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]map[string]any, error) {
	return djangoapi.ReadJSONRows(ctx, tx, query, args...)
}
func record(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	rows, err := rawRows(ctx, tx, `SELECT to_jsonb(i) FROM initiatives i WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, q.ID, q.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, projectmutations.ErrNotFound
	}
	return projection(ctx, tx, q, rows[0])
}
func projection(ctx context.Context, tx pgx.Tx, q Query, row map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, key := range []string{"id", "name", "description", "description_html", "description_stripped", "description_binary", "start_date", "end_date", "logo_props", "state", "created_at", "updated_at", "deleted_at"} {
		out[key] = row[key]
	}
	out["workspace"] = row["workspace_id"]
	var lead *string
	err := tx.QueryRow(ctx, `SELECT (SELECT u.id::text FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) LIMIT 1)`, row["lead_id"], q.WorkspaceID).Scan(&lead)
	out["lead"] = lead
	return out, err
}
func (r *Repository) Read(ctx context.Context, q Query) (json.RawMessage, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = scope(ctx, tx, q, false); err != nil {
		return nil, err
	}
	var value any
	switch q.Action {
	case "list":
		var rows []map[string]any
		rows, err = rawRows(ctx, tx, `SELECT to_jsonb(i) FROM initiatives i WHERE workspace_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC,id`, q.WorkspaceID)
		values := []map[string]any{}
		if err == nil {
			for _, row := range rows {
				var projected map[string]any
				projected, err = projection(ctx, tx, q, row)
				if err != nil {
					break
				}
				values = append(values, projected)
			}
		}
		value = values
	case "retrieve":
		value, err = record(ctx, tx, q)
	case "list_projects", "list_workitems":
		if _, err = record(ctx, tx, q); err == nil {
			value, err = associationPage(ctx, tx, q)
		}
	default:
		return nil, projectmutations.ErrInvalid
	}
	if err != nil {
		return nil, err
	}
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM initiative_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		revision := token(q.WorkspaceID, rev)
		switch v := value.(type) {
		case map[string]any:
			v["revision"] = revision
		case []map[string]any:
			for _, row := range v {
				row["revision"] = revision
			}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, tx.Commit(ctx)
}
func associationPage(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	limit, page, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return nil, err
	}
	from := ` FROM initiative_projects rel JOIN projects p ON p.id=rel.project_id AND p.workspace_id=rel.workspace_id WHERE rel.initiative_id=$1 AND rel.workspace_id=$2 AND p.deleted_at IS NULL AND (p.network=2 OR EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=p.workspace_id AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20)))`
	selected, order := `jsonb_build_object('project_id',p.id)`, `p.created_at DESC,p.id`
	if q.Action == "list_workitems" {
		from = ` FROM initiative_work_items rel JOIN issues i ON i.id=rel.issue_id AND i.project_id=rel.project_id AND i.workspace_id=rel.workspace_id JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE rel.initiative_id=$1 AND rel.workspace_id=$2 AND p.archived_at IS NULL AND ` + djangoapi.IssueVisibleSQL("i", "$3::uuid", "$2::uuid")
		selected, order = `jsonb_build_object('id',i.id,'project_id',i.project_id)`, `i.created_at DESC,i.id`
	}
	var total int
	if err = tx.QueryRow(ctx, `SELECT count(*)`+from, q.ID, q.WorkspaceID, q.ActorID).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := rawRows(ctx, tx, `SELECT `+selected+from+` ORDER BY `+order+` LIMIT $4 OFFSET $5`, q.ID, q.WorkspaceID, q.ActorID, limit, offset)
	if err != nil {
		return nil, err
	}
	values := []any{}
	projectReader, err := workspaceprojectreads.NewRepository(tx)
	if err != nil {
		return nil, err
	}
	workspace := stduuid.UUID(q.WorkspaceID)
	for _, row := range rows {
		if q.Action == "list_projects" {
			project := stduuid.MustParse(row["project_id"].(string))
			raw, err := projectReader.APIProject(ctx, stduuid.UUID(q.ActorID), q.WorkspaceSlug, project, workspaceprojectreads.APIProjectDetailOptions{ExpectedWorkspaceID: &workspace})
			if err != nil {
				return nil, err
			}
			var value map[string]any
			if err = json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
			value["sort_order"] = nil
			values = append(values, value)
		} else {
			result, err := workitems.ReadItemInTransaction(ctx, tx, workitems.Request{ActorID: q.ActorID.String(), ExpectedWorkspaceID: &q.WorkspaceID, Slug: q.WorkspaceSlug, ProjectID: row["project_id"].(string), IssueID: row["id"].(string), API: true, Method: "GET", Operation: "retrieve", Params: map[string]string{}})
			if err != nil {
				return nil, err
			}
			values = append(values, result.Body)
		}
	}
	return djangoapi.Envelope(values, total, limit, page), nil
}
