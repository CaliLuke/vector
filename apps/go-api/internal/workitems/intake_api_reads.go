package workitems

import (
	"context"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

// ReadIntake keeps the queue, its item projections and pagination in the same
// authorized snapshot. The native HTTP reader retains its compatibility shape.
func (r *Repository) ReadIntake(ctx context.Context, q Request) (Result, error) {
	if q.ExpectedWorkspaceID == nil || q.ProjectID == "" || q.Method != "GET" || !q.API || q.Operation != "list" && q.Operation != "retrieve" {
		return Result{}, forbidden()
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return Result{}, err
	}
	reader := *r
	reader.db = tx
	if err = reader.authorizeWorkitemRead(ctx, q); err != nil {
		return Result{}, err
	}
	intake, err := reader.intakeID(ctx, q)
	if err != nil {
		return Result{}, err
	}
	where := `t.workspace_id=$1 AND t.project_id=$2 AND t.intake_id=$3 AND t.deleted_at IS NULL
 AND i.project_id=t.project_id AND i.workspace_id=t.workspace_id
 AND p.intake_view AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND (t.snoozed_till >= $4 OR t.snoozed_till IS NULL)
 AND ` + djangoapi.IssueVisibleSQL("i", "$5::uuid", "$1::uuid") + `
 AND (i.created_by_id=$5 OR EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=t.project_id AND pm.workspace_id=t.workspace_id AND pm.member_id=$5 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(15,20)))`
	args := []any{q.ExpectedWorkspaceID.String(), q.ProjectID, intake, r.now(), q.ActorID}
	from := ` FROM intake_issues t JOIN issues i ON i.id=t.issue_id JOIN projects p ON p.id=t.project_id AND p.workspace_id=t.workspace_id WHERE `
	var body any
	if q.Operation == "retrieve" {
		if q.IssueID == "" {
			return Result{}, requiredObject()
		}
		row, e := reader.row(ctx, `SELECT to_jsonb(t)`+from+where+` AND t.issue_id=$6`, append(args, q.IssueID)...)
		if e != nil {
			return Result{}, e
		}
		body, err = reader.intakeAPIRecord(ctx, q, row)
	} else {
		limit, page, offset, e := djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
		if e != nil {
			return Result{}, e
		}
		var total int
		if err = tx.QueryRow(ctx, `SELECT COUNT(*)`+from+where, args...).Scan(&total); err != nil {
			return Result{}, err
		}
		rows, e := reader.rows(ctx, `SELECT to_jsonb(t)`+from+where+` ORDER BY t.created_at DESC,t.id LIMIT $6 OFFSET $7`, append(args, limit, offset)...)
		if e != nil {
			return Result{}, e
		}
		values := []any{}
		for _, row := range rows {
			value, e := reader.intakeAPIRecord(ctx, q, row)
			if e != nil {
				return Result{}, e
			}
			values = append(values, value)
		}
		body = djangoapi.Envelope(values, total, limit, page)
	}
	if err != nil {
		return Result{}, err
	}
	return Result{Status: 200, Body: body}, tx.Commit(ctx)
}

func (r *Repository) intakeAPIRecord(ctx context.Context, q Request, row map[string]any) (map[string]any, error) {
	issue, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND i.workspace_id=$3`, row["issue_id"], q.ProjectID, q.ExpectedWorkspaceID.String())
	if err != nil {
		return nil, err
	}
	expanded, err := r.serializeVisibleIssue(ctx, issue, q, "", "state")
	if err != nil {
		return nil, err
	}
	expanded["description"] = issue["description_json"]
	result := map[string]any{"issue_detail": expanded, "inbox": row["intake_id"]}
	for _, key := range []string{"id", "created_at", "updated_at", "deleted_at", "status", "snoozed_till", "source", "source_email", "external_source", "external_id", "extra"} {
		result[key] = row[key]
	}
	for _, key := range []string{"created_by", "updated_by", "project", "workspace", "intake", "issue", "duplicate_to"} {
		result[key] = row[key+"_id"]
	}
	if duplicate := result["duplicate_to"]; duplicate != nil {
		var visible bool
		if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+`)`, duplicate, q.ActorID, q.ExpectedWorkspaceID.String()).Scan(&visible); err != nil {
			return nil, err
		}
		if !visible {
			result["duplicate_to"] = nil
		}
	}
	if q.Params["include_revision"] == "true" {
		var revision int64
		if err = r.db.QueryRow(ctx, `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, row["issue_id"]).Scan(&revision); err != nil {
			return nil, err
		}
		result["revision"] = fmt.Sprintf("intake:%s:%d", row["issue_id"], revision)
	}
	return result, nil
}
