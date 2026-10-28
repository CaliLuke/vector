package workitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/jackc/pgx/v5"
)

// ReadWorkitem binds API queries, pagination and expanded relations to one
// primary-data snapshot and the credential's immutable workspace identity.
func (r *Repository) ReadWorkitem(ctx context.Context, q Request) (Result, error) {
	if q.ExpectedWorkspaceID == nil || q.Method != "GET" || !q.API {
		return Result{}, forbidden()
	}
	propertyPQL, err := issuequery.PQLUsesProperties(q.Params["pql"])
	if err != nil {
		return Result{}, pqlRequestError(err)
	}
	read := func(tx pgx.Tx, snapshot *workitemtypes.PropertyDiscoverySnapshot) (Result, error) {
		reader := *r
		reader.db = tx
		if authorizeWorkitemReadErr := reader.authorizeWorkitemRead(ctx, q); authorizeWorkitemReadErr != nil {
			return Result{}, authorizeWorkitemReadErr
		}
		if snapshot != nil {
			reader.propertyDiscovery, reader.typeDiscovery = snapshot, &snapshot.Types
			if preparePropertyPQLErr := reader.preparePropertyPQL(ctx, q); preparePropertyPQLErr != nil {
				return Result{}, preparePropertyPQLErr
			}
		}
		switch q.Operation {
		case "list", "list_archived", "cycle_items", "module_items", "count":
			if q.Operation == "list_archived" {
				q.Operation = "get_api_v1_archived_issues"
			}
			return reader.listIssueResults(ctx, q, nil)
		case "retrieve", "retrieve_by_identifier":
			return reader.boundWorkitemDetail(ctx, q)
		case "search":
			return reader.search(ctx, q)
		default:
			return Result{}, failure(400, "error", "Invalid work item read action.")
		}
	}
	if propertyPQL {
		result, withPropertyDiscoveryReadErr := workitemtypes.WithPropertyDiscoveryRead(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, read)
		return result, pqlDiscoveryReadError(withPropertyDiscoveryReadErr)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return Result{}, err
	}
	result, err := read(tx, nil)
	if err != nil {
		return Result{}, err
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) authorizeWorkitemRead(ctx context.Context, q Request) error {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w
 JOIN workspace_members wm ON wm.workspace_id=w.id
 JOIN users u ON u.id=wm.member_id
 WHERE w.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND u.id=$3 AND u.is_active
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN (5,15,20)
 AND ($4='' OR EXISTS(SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id
 WHERE p.id::text=$4 AND p.workspace_id=w.id AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND pm.member_id=u.id AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (5,15,20))))`, q.ExpectedWorkspaceID.String(), q.Slug, q.ActorID, q.ProjectID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return forbidden()
	}
	table := ""
	switch q.Operation {
	case "cycle_items":
		table = "cycles"
	case "module_items":
		table = "modules"
	}
	if table != "" {
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL)`, q.Params["parent_id"], q.ProjectID, q.ExpectedWorkspaceID.String()).Scan(&allowed)
		if err != nil {
			return err
		}
		if !allowed {
			return requiredObject()
		}
	}
	return nil
}

func (r *Repository) boundWorkitemDetail(ctx context.Context, q Request) (Result, error) {
	args := []any{q.ExpectedWorkspaceID.String(), q.ActorID}
	where := `i.workspace_id=$1 AND ` + liveIssue + ` AND ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "$1::uuid")
	if q.Operation == "retrieve_by_identifier" {
		head, sequence, ok := splitIdentifier(q.Params["identifier"])
		if !ok || head == "" || sequence == "" || strings.Trim(sequence, "0123456789") != "" {
			return Result{}, failure(400, "error", "Expected PROJECT-N work item identifier.")
		}
		args = append(args, head, sequence)
		where += ` AND p.identifier=$3 AND i.sequence_id::numeric=$4::numeric`
	} else {
		args = append(args, q.ProjectID, q.IssueID)
		where += ` AND i.project_id=$3 AND i.id=$4`
	}
	row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE `+where, args...)
	if err != nil {
		return Result{}, err
	}
	body, err := r.serializeVisibleIssue(ctx, row, q, q.Params["fields"], q.Params["expand"])
	if err != nil {
		return Result{}, err
	}
	if q.Params["include_revision"] == "true" {
		var revision int64
		if err = r.db.QueryRow(ctx, `SELECT revision FROM issue_edit_revisions WHERE issue_id=$1`, row["id"]).Scan(&revision); err != nil {
			return Result{}, err
		}
		body["revision"] = fmt.Sprintf("workitem:%s:%d", row["id"], revision)
	}
	return Result{200, body}, nil
}

func appendBoundWorkitemScope(q Request, where []string, args []any) ([]string, []any) {
	if q.ExpectedWorkspaceID == nil {
		return where, args
	}
	args = append(args, q.ExpectedWorkspaceID.String())
	where = append(where, fmt.Sprintf("issue.workspace_id=$%d", len(args)), "project.workspace_id=issue.workspace_id", "project.archived_at IS NULL", djangoapi.IssueVisibleSQL("issue", "$1::uuid", "issue.workspace_id"))
	if q.Operation == "cycle_items" || q.Operation == "module_items" {
		kind := "cycle"
		if q.Operation == "module_items" {
			kind = "module"
		}
		args = append(args, q.Params["parent_id"])
		where = append(where, fmt.Sprintf(`EXISTS(SELECT 1 FROM %s_issues link WHERE link.issue_id=issue.id AND link.project_id=issue.project_id AND link.workspace_id=issue.workspace_id AND link.deleted_at IS NULL AND link.%s_id=$%d)`, kind, kind, len(args)))
	}
	return where, args
}
