package workitems

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func collaborationKind(q Request) (string, string) {
	switch {
	case collaborationComment(q):
		return "comment", "issue_comments"
	case collaborationLink(q):
		return "link", "issue_links"
	case strings.Contains(q.Operation, "_reactions") && q.Params["comment_id"] != "":
		return "commentReaction", "comment_reactions"
	case strings.Contains(q.Operation, "_reactions"):
		return "issueReaction", "issue_reactions"
	default:
		return "subscriber", "issue_subscribers"
	}
}

func (r *Repository) ExecuteCollaboration(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	kind, table := collaborationKind(q)
	if collaborationPublic(q) {
		board, err := r.collaborationBoard(ctx, q)
		if err != nil {
			if q.Method == "GET" && q.Params["pk"] == "" {
				return Result{200, []any{}}, nil
			}
			return Result{}, err
		}
		q.ProjectID = board["project_id"].(string)
		if err = r.db.QueryRow(ctx, `SELECT slug FROM workspaces WHERE id=$1`, board["workspace_id"]).Scan(&q.Slug); err != nil {
			return Result{}, err
		}
		flag, message := "is_comments_enabled", "Comments are not enabled for this project"
		if kind != "comment" {
			flag = "is_reactions_enabled"
			message = "Reactions are not enabled for this project board"
			if kind == "commentReaction" {
				message = "Reactions are not enabled for this board"
			}
		}
		if board[flag] != true {
			if q.Method == "GET" {
				if q.Params["pk"] != "" {
					return Result{}, failure(404, "detail", "No IssueComment matches the given query.")
				}
				return Result{200, []any{}}, nil
			}
			return Result{}, failure(400, "error", message)
		}
	}
	if q.Method == "GET" || q.Method == "HEAD" {
		return r.readCollaboration(ctx, q, kind, table)
	}
	if !q.API && !collaborationPublic(q) && !collaborationLink(q) && kind != "subscriber" && q.Method != "PUT" {
		if kind == "comment" && q.Method != "POST" {
			if err := r.allowCollaborationCommentOwner(ctx, q); err != nil {
				return Result{}, err
			}
		} else if err := r.allowProject(ctx, q, []int{5, 15, 20}, false); err != nil {
			return Result{}, err
		}
	}
	if err := r.authorizeCollaborationParent(ctx, q, kind); err != nil {
		return Result{}, err
	}
	if kind == "subscriber" {
		return r.writeCollaborationSubscriber(ctx, q, body, contentType)
	}
	if q.Method == "DELETE" {
		return r.destroyCollaboration(ctx, q, kind, table)
	}
	var old map[string]any
	var err error
	if q.Method != "POST" {
		old, err = r.collaborationObject(ctx, q, kind, table, q.Method == "PUT")
		if err != nil {
			return Result{}, err
		}
	}
	// App guest checks precede parsing the body and serializer validation.
	if kind == "comment" && !q.API && !collaborationPublic(q) && q.Method == "POST" {
		project, err := r.project(ctx, q.ProjectID)
		if err != nil {
			return Result{}, err
		}
		issue, err := r.row(ctx, `SELECT to_jsonb(i) FROM issues i WHERE id=$1 AND deleted_at IS NULL`, q.IssueID)
		if err != nil {
			return Result{}, err
		}
		var guest bool
		if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND member_id=$2 AND role=5 AND is_active AND deleted_at IS NULL)`, q.ProjectID, q.ActorID).Scan(&guest); err != nil {
			return Result{}, err
		}
		if guest && project["guest_view_all_features"] == false && issue["created_by_id"] != q.ActorID {
			return Result{}, failure(400, "error", "You are not allowed to comment on the issue")
		}
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return Result{}, err
	}
	if err = r.collaborationExternalConflict(ctx, q, input, old); err != nil {
		return Result{}, err
	}
	changes, err := r.validateCollaboration(ctx, q, input, table)
	if err != nil {
		return Result{}, err
	}
	return r.writeCollaboration(ctx, q, kind, table, input, changes, old)
}

func (r *Repository) allowCollaborationCommentOwner(ctx context.Context, q Request) error {
	var allowed bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL)`, q.Slug, q.ActorID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return forbidden()
	}
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_comments WHERE id=$1 AND created_by_id=$2 AND deleted_at IS NULL)`, q.Params["pk"], q.ActorID).Scan(&allowed); err != nil {
		return err
	}
	if allowed {
		return nil
	}
	return r.allowProject(ctx, q, []int{20}, false)
}

func (r *Repository) collaborationObject(ctx context.Context, q Request, kind, table string, filtered bool) (map[string]any, error) {
	args := []any{q.Slug, q.ProjectID}
	where := `w.slug=$1 AND t.project_id=$2 AND t.deleted_at IS NULL`
	if kind == "commentReaction" {
		args = append(args, q.Params["comment_id"])
		where += ` AND t.comment_id=$3`
	} else {
		args = append(args, q.IssueID)
		where += ` AND t.issue_id=$3`
	}
	switch kind {
	case "issueReaction", "commentReaction":
		args = append(args, q.Params["reaction_code"], q.ActorID)
		where += ` AND t.reaction=$4 AND t.actor_id=$5`
	case "subscriber":
		subscriber := q.Params["subscriber_id"]
		if subscriber == "" {
			subscriber = q.ActorID
		}
		args = append(args, subscriber)
		where += ` AND t.subscriber_id=$4`
	default:
		args = append(args, q.Params["pk"])
		where += ` AND t.id=$4`
	}
	if collaborationPublic(q) && kind == "comment" {
		args = append(args, q.ActorID)
		where += ` AND t.access='EXTERNAL' AND t.actor_id=$5`
	}
	if filtered && !q.API && !collaborationPublic(q) {
		args = append(args, q.ActorID)
		where += ` AND p.archived_at IS NULL AND EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=t.project_id AND m.member_id=$5 AND m.is_active AND m.deleted_at IS NULL)`
	}
	if filtered && kind == "comment" {
		extra, e := collaborationFilters(q, &args)
		if e != nil {
			return nil, e
		}
		where += extra
	}
	if !collaborationPublic(q) {
		where += ` AND EXISTS(` + collaborationVisibleParentSQL(q, kind, &args) + `)`
	}
	row, err := r.row(ctx, `SELECT to_jsonb(t) FROM `+table+` t JOIN workspaces w ON w.id=t.workspace_id JOIN projects p ON p.id=t.project_id WHERE `+where, args...)
	var missing *djangoapi.HTTPError
	if errors.As(err, &missing) && missing.Status == 404 {
		if filtered && !q.API {
			model := "IssueComment"
			if kind == "link" {
				model = "IssueLink"
			}
			return nil, failure(404, "detail", "No "+model+" matches the given query.")
		}
		return nil, collaborationMissing(q)
	}
	return row, err
}

func collaborationFilters(q Request, args *[]any) (string, error) {
	where := ""
	for _, filter := range [][2]string{{"issue__id", "t.issue_id"}, {"workspace__id", "t.workspace_id"}} {
		if !q.Present[filter[0]] || q.Params[filter[0]] == "" {
			continue
		}
		id, err := djangoinput.UUID(q.Params[filter[0]])
		if err != nil {
			return "", invalid(map[string]any{filter[0]: []string{"Enter a valid UUID."}})
		}
		*args = append(*args, id.String())
		where += " AND " + filter[1] + "=$" + strconv.Itoa(len(*args))
	}
	return where, nil
}

func (r *Repository) readCollaboration(ctx context.Context, q Request, kind, table string) (Result, error) {
	if kind == "subscriber" {
		if strings.Contains(q.Operation, "_subscribe") && !strings.Contains(q.Operation, "_subscribers") {
			_, err := r.collaborationObject(ctx, q, kind, table, false)
			var missing *djangoapi.HTTPError
			if errors.As(err, &missing) && missing.Status == 404 {
				return Result{200, map[string]any{"subscribed": false}}, nil
			}
			if err != nil {
				return Result{}, err
			}
			return Result{200, map[string]any{"subscribed": true}}, nil
		}
		rows, err := r.visibleCollaborationRows(ctx, q, kind, `SELECT to_jsonb(m) FROM project_members m JOIN workspaces w ON w.id=m.workspace_id WHERE w.slug=$1 AND m.project_id=$2 AND m.is_active AND m.deleted_at IS NULL ORDER BY m.created_at DESC`, q.Slug, q.ProjectID)
		if err != nil {
			return Result{}, err
		}
		out := []any{}
		for _, row := range rows {
			member, err := r.collaborationUser(ctx, row["member_id"])
			if err != nil {
				return Result{}, err
			}
			out = append(out, map[string]any{"id": row["id"], "member": member})
		}
		return Result{200, out}, nil
	}
	args := []any{q.Slug, q.ProjectID}
	where := `w.slug=$1 AND t.project_id=$2 AND t.deleted_at IS NULL`
	if kind == "commentReaction" {
		args = append(args, q.Params["comment_id"])
		where += ` AND t.comment_id=$3`
	} else {
		args = append(args, q.IssueID)
		where += ` AND t.issue_id=$3`
	}
	if !collaborationPublic(q) {
		args = append(args, q.ActorID)
		where += ` AND p.archived_at IS NULL AND EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=t.project_id AND m.member_id=$4 AND m.is_active AND m.deleted_at IS NULL)`
	}
	if collaborationPublic(q) && kind == "comment" {
		where += ` AND t.access='EXTERNAL'`
	}
	// Existing denormalized rows must not expose another project's issue or comment.
	if kind == "commentReaction" {
		where += ` AND EXISTS(SELECT 1 FROM issue_comments ref WHERE ref.id=t.comment_id AND ref.project_id=t.project_id AND ref.workspace_id=t.workspace_id`
		if collaborationPublic(q) {
			where += ` AND ref.access='EXTERNAL'`
		}
		where += `)`
	} else {
		where += ` AND EXISTS(SELECT 1 FROM issues ref WHERE ref.id=t.issue_id AND ref.project_id=t.project_id AND ref.workspace_id=t.workspace_id)`
	}

	if kind == "comment" && !q.API {
		extra, err := collaborationFilters(q, &args)
		if err != nil {
			return Result{}, err
		}
		where += extra
	}
	if q.Params["pk"] != "" {
		args = append(args, q.Params["pk"])
		where += ` AND t.id=$` + strconv.Itoa(len(args))
	}
	ordering := "DESC"
	if collaborationPublic(q) && kind == "comment" {
		ordering = "ASC"
	}
	rows, err := r.visibleCollaborationRows(ctx, q, kind, `SELECT to_jsonb(t) FROM `+table+` t JOIN workspaces w ON w.id=t.workspace_id JOIN projects p ON p.id=t.project_id WHERE `+where+` ORDER BY t.created_at `+ordering, args...)
	if err != nil {
		return Result{}, err
	}
	if q.Params["pk"] != "" {
		if len(rows) == 0 {
			if q.API {
				return Result{}, collaborationMissing(q)
			}
			model := "IssueComment"
			if kind == "link" {
				model = "IssueLink"
			}
			return Result{}, failure(404, "detail", "No "+model+" matches the given query.")
		}
		result, err := r.collaborationProjection(ctx, q, rows[0], kind, true)
		return Result{200, result}, err
	}
	total := len(rows)
	limit, page, offset := total, 0, 0
	if q.API {
		limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: q.Params, Present: q.Present})
		if err != nil {
			return Result{}, err
		}
		end := min(len(rows), offset+limit)
		if offset > len(rows) {
			offset = len(rows)
		}
		rows = rows[offset:end]
	}
	result := []any{}
	for _, row := range rows {
		projected, err := r.collaborationProjection(ctx, q, row, kind, true)
		if err != nil {
			return Result{}, err
		}
		result = append(result, projected)
	}
	if q.API {
		return Result{200, djangoapi.Envelope(result, total, limit, page)}, nil
	}
	return Result{200, result}, nil
}

func (r *Repository) ProtocolCollaborationPutAction(ctx context.Context, q Request) (bool, error) {
	if q.Params["pk"] == "" || collaborationPublic(q) {
		return true, nil
	}
	q.Method = "PUT"
	if err := r.AuthorizeCollaborationInitial(ctx, q); err != nil {
		var handled *djangoapi.HTTPError
		if errors.As(err, &handled) {
			return false, nil
		}
		return false, err
	}
	kind, table := collaborationKind(q)
	_, err := r.collaborationObject(ctx, q, kind, table, true)
	var handled *djangoapi.HTTPError
	if errors.As(err, &handled) {
		return false, nil
	}
	return err == nil, err
}
