package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateLink retains the native serializer, ownership and activity rules while
// putting the complete operation and its retry receipt in one transaction.
func (r *Repository) MutateLink(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ParentID == uuid.Nil {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	method := map[string]string{"create": "POST", "update": "PATCH", "delete": "DELETE"}[cmd.Action]
	if method == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	for field, value := range cmd.Fields {
		if field != "url" && field != "title" {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
		if _, ok := value.(string); !ok {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	if cmd.Action == "delete" && len(cmd.Fields) != 0 || cmd.Action == "update" && len(cmd.Fields) == 0 || cmd.Action == "create" && (cmd.Fields["url"] == nil || cmd.Fields["url"] == "") {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	q := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), IssueID: cmd.ParentID.String(), Method: method, API: true, Operation: "mcp_issue_links", Timezone: cmd.Timezone, Params: map[string]string{}}
	if cmd.Action != "create" {
		q.Params["pk"] = cmd.ResourceID.String()
	}
	resource := projectmutations.Resource{Name: "workitem_link", Table: "issue_links", RevisionTable: "issue_link_edit_revisions", RevisionColumn: "link_id", ParentColumn: "issue_id", SchemaLock: true}
	resource.Authorize = func(ctx context.Context, tx pgx.Tx) error {
		bound := q
		bound.ExpectedWorkspaceID = &cmd.WorkspaceID
		args := []any{}
		parent := collaborationVisibleParentSQL(bound, "link", &args)
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
		body, err := json.Marshal(cmd.Fields)
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
