package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"slices"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateWorkitem composes native writes with receipt replay and a revision which
// includes relationship changes made by every writer. Set edits read and merge
// under the same row/schema locks as the eventual native PATCH.
func (r *Repository) MutateWorkitem(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	fields, err := workitemMutationFields(cmd.Action, cmd.Fields)
	if err != nil {
		return projectmutations.Result{}, err
	}
	cmd.Fields = fields
	if cmd.ParentID != uuid.Nil {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	resource := projectmutations.Resource{Name: "workitem", Table: "issues", RevisionTable: "issue_edit_revisions", RevisionColumn: "issue_id", SchemaLock: true}
	return projectmutations.Execute(ctx, db, resource, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		writer := *r
		writer.db, writer.transactionalEffects, writer.publisher = tx, true, projectmutations.Effects{Tx: tx}
		q := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), IssueID: cmd.ResourceID.String(), API: true, Timezone: cmd.Timezone, Params: map[string]string{}, Present: map[string]bool{}, ExpectedWorkspaceID: &cmd.WorkspaceID}
		if cmd.Action == "archive" {
			q.Method = "POST"
			if fields["archive"] == false {
				q.Method = "DELETE"
			}
			q.Operation = "mcp_workitem_archive"
			if _, lifecycleTransactionErr := writer.lifecycleTransaction(ctx, q, nil, "application/json"); lifecycleTransactionErr != nil {
				return nil, lifecycleTransactionErr
			}
			return json.Marshal(map[string]any{"workitem_id": cmd.ResourceID.String(), "archived": q.Method == "POST"})
		}
		q.Method = map[string]string{"create": "POST", "update": "PATCH", "delete": "DELETE", "manage_assignee": "PATCH", "manage_label": "PATCH"}[cmd.Action]
		q.Operation = "mcp_projects_by_project_id_issues_by_pk"
		payload := maps.Clone(fields)
		if cmd.Action == "manage_assignee" || cmd.Action == "manage_label" {
			payload, err = writer.workitemSetChange(ctx, cmd)
			if err != nil {
				return nil, err
			}
		} else if err = writer.lockWorkitemSelections(ctx, cmd, payload); err != nil {
			return nil, err
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		value, err := writer.writeIssueTransaction(ctx, q, bytes.NewReader(raw), "application/json")
		if err != nil {
			return nil, err
		}
		if cmd.Action == "delete" {
			return nil, nil
		}
		return json.Marshal(value.Body)
	})
}
func workitemMutationFields(action string, fields map[string]any) (map[string]any, error) {
	out := maps.Clone(fields)
	if out == nil {
		out = map[string]any{}
	}
	allowed := []string{}
	switch action {
	case "create", "update":
		allowed = []string{"name", "assignees", "labels", "type_id", "point", "description_html", "priority", "start_date", "target_date", "sort_order", "is_draft", "parent", "state", "estimate_point", "external_source", "external_id"}
	case "delete":
	case "archive":
		allowed = []string{"archive"}
		if _, ok := out["archive"]; !ok {
			out["archive"] = true
		}
		if _, ok := out["archive"].(bool); !ok {
			return nil, projectmutations.ErrInvalid
		}
	case "manage_assignee", "manage_label":
		allowed = []string{"add_ids", "remove_ids"}
	default:
		return nil, projectmutations.ErrInvalid
	}
	for key := range out {
		if !slices.Contains(allowed, key) {
			return nil, projectmutations.ErrInvalid
		}
	}
	for _, key := range []string{"assignees", "labels", "add_ids", "remove_ids"} {
		if value, ok := out[key]; ok {
			ids, ok := value.([]string)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			normalized := make([]string, len(ids))
			for i, raw := range ids {
				id, err := uuid.Parse(raw)
				if err != nil || id == uuid.Nil {
					return nil, projectmutations.ErrInvalid
				}
				normalized[i] = id.String()
			}
			slices.Sort(normalized)
			out[key] = slices.Compact(normalized)
		}
	}
	for _, key := range []string{"type_id", "parent", "state", "estimate_point"} {
		if value, ok := out[key]; ok {
			raw, ok := value.(string)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			id, err := uuid.Parse(raw)
			if err != nil || id == uuid.Nil {
				return nil, projectmutations.ErrInvalid
			}
			out[key] = id.String()
		}
	}
	if action == "manage_assignee" || action == "manage_label" {
		total := 0
		for _, key := range []string{"add_ids", "remove_ids"} {
			ids, _ := out[key].([]string)
			if ids == nil {
				ids = []string{}
			}
			out[key] = ids
			total += len(ids)
		}
		if total == 0 {
			return nil, projectmutations.ErrInvalid
		}
	}
	return out, nil
}
func (r *Repository) workitemSetChange(ctx context.Context, cmd projectmutations.Command) (map[string]any, error) {
	table, column, field := "issue_labels", "label_id", "labels"
	if cmd.Action == "manage_assignee" {
		table, column, field = "issue_assignees", "assignee_id", "assignees"
	}
	adding, removing := cmd.Fields["add_ids"].([]string), cmd.Fields["remove_ids"].([]string)
	if err := r.lockWorkitemSelections(ctx, cmd, map[string]any{field: adding}); err != nil {
		return nil, err
	}
	ids, err := r.filteredIDs(ctx, "SELECT "+column+"::text FROM "+table+" WHERE issue_id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL ORDER BY created_at,id", cmd.ResourceID, cmd.ProjectID, cmd.WorkspaceID)
	if err != nil {
		return nil, err
	}
	values := []string{}
	for _, id := range ids {
		if !slices.Contains(removing, id) && !slices.Contains(values, id) {
			values = append(values, id)
		}
	}
	for _, id := range adding {
		if !slices.Contains(values, id) {
			values = append(values, id)
		}
	}
	return map[string]any{field: values}, nil
}
