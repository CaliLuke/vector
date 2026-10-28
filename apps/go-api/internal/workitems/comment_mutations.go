package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"maps"

	"github.com/CaliLuke/go-plane/internal/commentmentions"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateComment retains the native serializer, ownership and activity rules while
// putting the complete operation and its retry receipt in one transaction.
func (r *Repository) MutateComment(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ParentID == uuid.Nil {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	method := map[string]string{"create": "POST", "update": "PATCH", "delete": "DELETE"}[cmd.Action]
	if method == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	for field, value := range cmd.Fields {
		if field != "comment_html" && field != "access" && field != "external_source" && field != "external_id" {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
		if _, ok := value.(string); !ok {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	if cmd.Action == "delete" && len(cmd.Fields) != 0 || cmd.Action == "create" && (cmd.Fields["comment_html"] == nil || cmd.Fields["comment_html"] == "") {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	source, _ := cmd.Fields["comment_html"].(string)
	mentions, err := commentmentions.Parse(source)
	if err != nil {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	q := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), IssueID: cmd.ParentID.String(), Method: method, API: true, Operation: "mcp_issue_comments", Timezone: cmd.Timezone, Params: map[string]string{}}
	if cmd.Action != "create" {
		q.Params["pk"] = cmd.ResourceID.String()
	}
	resource := projectmutations.Resource{Name: "workitem_comment", Table: "issue_comments", RevisionTable: "issue_comment_edit_revisions", RevisionColumn: "comment_id", ParentColumn: "issue_id", SchemaLock: true, AllowGuests: true}
	resource.Authorize = func(ctx context.Context, tx pgx.Tx) error {
		bound := q
		bound.ExpectedWorkspaceID = &cmd.WorkspaceID
		args := []any{}
		parent := collaborationVisibleParentSQL(bound, "comment", &args)
		var id uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM (`+parent+` FOR SHARE OF visible_issue) permitted`, args...).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		return err
	}
	return projectmutations.Execute(ctx, db, resource, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		writer := *r
		writer.db, writer.transactionalEffects = tx, true
		writer.publisher = projectmutations.Effects{Tx: tx}
		writer.notifyCollaboration = true
		for _, member := range mentions.Users {
			var id uuid.UUID
			err := tx.QueryRow(ctx, `SELECT pm.member_id FROM project_members pm JOIN users u ON u.id=pm.member_id WHERE pm.project_id=$1 AND pm.workspace_id=$2 AND pm.member_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) AND u.is_active FOR SHARE OF pm,u`, cmd.ProjectID, cmd.WorkspaceID, member).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, projectmutations.ErrInvalid
			}
			if err != nil {
				return nil, err
			}
		}
		fields := maps.Clone(cmd.Fields)
		if source != "" {
			fields["comment_html"] = mentions.Render()
		}
		body, err := json.Marshal(fields)
		if err != nil {
			return nil, err
		}
		result, err := writer.ExecuteCollaboration(ctx, q, bytes.NewReader(body), "application/json")
		if err != nil {
			return nil, err
		}
		if cmd.Action == "delete" {
			return nil, nil
		}
		return json.Marshal(result.Body)
	})
}
