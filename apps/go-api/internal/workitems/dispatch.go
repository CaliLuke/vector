package workitems

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

// AuthorizeInitial runs permission_classes before throttle and timezone. App
// action decorators intentionally run later, inside their individual handlers.
func (r *Repository) AuthorizeInitial(ctx context.Context, q Request) error {
	if !q.API {
		if strings.Contains(q.Operation, "_issue_relation") || strings.HasSuffix(q.Operation, "_remove_relation") || strings.HasSuffix(q.Operation, "_sub_issues") || strings.HasSuffix(q.Operation, "_bulk_archive_issues") {
			allowed, err := djangoapi.ProjectEntityAllowed(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, q.Method != "GET" && q.Method != "HEAD" && q.Method != "OPTIONS")
			if err != nil {
				return err
			}
			if !allowed {
				return failure(403, "detail", "You do not have permission to perform this action.")
			}
		}
		if strings.Contains(q.Operation, "_user_issues_by_user_id") {
			return r.allowWorkspace(ctx, q, []int{5, 15, 20})
		}
		return nil
	}
	if strings.HasSuffix(q.Operation, "_search") {
		return nil
	}
	var allowed bool
	var err error
	if strings.Contains(q.Operation, "_intake_issues") {
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.project_id=$2 AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL)`, q.Slug, q.ProjectID, q.ActorID).Scan(&allowed)
	} else if composite := q.Params["identifier"]; composite != "" {
		// ProjectEntityPermission resolves project_identifier only for safe
		// methods. Its unsafe branch uses this view's absent project_id,
		// which cannot match a non-null project membership.
		if q.Method != "GET" && q.Method != "HEAD" && q.Method != "OPTIONS" {
			return failure(403, "detail", "You do not have permission to perform this action.")
		}
		project, _, ok := splitIdentifier(composite)
		if !ok {
			return failure(404, "error", "Page not found.")
		}
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members m JOIN projects p ON p.id=m.project_id JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND p.identifier=$2 AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL)`, q.Slug, project, q.ActorID).Scan(&allowed)
	} else {
		allowed, err = djangoapi.ProjectEntityAllowed(ctx, r.db, q.ActorID, q.Slug, q.ProjectID, q.Method != "GET" && q.Method != "HEAD" && q.Method != "OPTIONS")
	}
	if err != nil {
		return err
	}
	if !allowed {
		return failure(403, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

func (r *Repository) Execute(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	op := strings.ReplaceAll(q.Operation, "_work_items", "_issues")
	if strings.Contains(op, "_intake_issues") || strings.Contains(op, "_inbox_issues") {
		return r.intake(ctx, q, body, contentType)
	}
	if strings.Contains(op, "_draft_issues") || strings.Contains(op, "_draft_to_issue") {
		return r.drafts(ctx, q, body, contentType)
	}
	if q.Params["identifier"] != "" {
		return r.identifier(ctx, q)
	}
	if strings.HasSuffix(op, "_search") || strings.HasSuffix(op, "_search_issues") {
		return r.search(ctx, q)
	}
	if strings.Contains(op, "_cycles_by_cycle_id_") || strings.Contains(op, "_modules_by_module_id_") || strings.HasSuffix(op, "_issues_by_issue_id_modules") {
		return r.planning(ctx, q, body, contentType)
	}
	if strings.HasSuffix(op, "_archive") || strings.HasSuffix(op, "_bulk_archive_issues") || strings.HasSuffix(op, "_bulk_delete_issues") || strings.HasSuffix(op, "_deleted_issues") || strings.HasSuffix(op, "_issue_dates") {
		return r.lifecycle(ctx, q, body, contentType)
	}
	if strings.HasSuffix(op, "_issue_relation") || strings.HasSuffix(op, "_relations") || strings.HasSuffix(op, "_remove_relation") {
		return r.relations(ctx, q, body, contentType)
	}
	if strings.HasSuffix(op, "_sub_issues") {
		return r.subissues(ctx, q, body, contentType)
	}
	if q.Method == "GET" && (strings.HasSuffix(op, "_issues_list") || strings.HasSuffix(op, "_issues_detail") || strings.HasSuffix(op, "_v2_issues") || strings.HasSuffix(op, "_archived_issues") || strings.Contains(op, "_user_issues_by_user_id") || strings.HasSuffix(op, "_by_slug_issues")) {
		return r.listIssues(ctx, q)
	}
	if strings.HasSuffix(op, "_projects_by_project_id_issues") || strings.HasSuffix(op, "_projects_by_project_id_issues_by_pk") {
		if q.Method != "GET" {
			return r.writeIssue(ctx, q, body, contentType)
		}
		if q.IssueID != "" {
			return r.readIssue(ctx, q)
		}
		return r.listIssues(ctx, q)
	}
	return Result{}, fmt.Errorf("work-item operation %s is not implemented", q.Operation)
}

// Django's first str capture is greedy but must leave both sides nonempty.
func splitIdentifier(value string) (string, string, bool) {
	for i := len(value) - 2; i > 0; i-- {
		if value[i] == '-' {
			return value[:i], value[i+1:], true
		}
	}
	return "", "", false
}
