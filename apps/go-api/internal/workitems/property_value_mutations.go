package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutatePropertyValue reuses native item validation, history, subscriptions and
// effects. The API identity table is synchronized by all canonical item writers.
func (r *Repository) MutatePropertyValue(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ResourceID == uuid.Nil || cmd.ParentID == uuid.Nil || cmd.Action != "set_value" && cmd.Action != "delete_value" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	for key := range cmd.Fields {
		if key != "value" && key != "description_html" && key != "external_source" && key != "external_id" {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	if cmd.Action == "delete_value" && len(cmd.Fields) > 0 {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	for _, key := range []string{"external_source", "external_id"} {
		if value, exists := cmd.Fields[key]; exists {
			v, ok := value.(string)
			if !ok || len(v) > 255 || strings.ContainsRune(v, 0) {
				return projectmutations.Result{}, projectmutations.ErrInvalid
			}
		}
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	q := workitemtypes.PropertyValueAPIQuery{APIScope: workitemtypes.APIScope{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, WorkspaceSlug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID}, IssueID: cmd.ResourceID, PropertyID: cmd.ParentID}
	var scope workitemtypes.PropertyValueScope
	resource := projectmutations.Resource{Name: "workitem_property_value", Table: "issues", RevisionTable: "issue_edit_revisions", RevisionColumn: "issue_id", SchemaLock: true, Authorize: func(ctx context.Context, tx pgx.Tx) error {
		var err error
		scope, err = workitemtypes.LoadPropertyValueScope(ctx, tx, q, true)
		return err
	}}
	return projectmutations.Execute(ctx, db, resource, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		if cmd.Action == "delete_value" && !scope.Present {
			return nil, projectmutations.ErrNotFound
		}
		value, exists := cmd.Fields["value"]
		html, hasHTML := cmd.Fields["description_html"]
		if cmd.Action == "set_value" {
			if exists == hasHTML {
				return nil, projectmutations.ErrInvalid
			}
			if scope.Definition.Kind == property.RichText {
				if !hasHTML {
					return nil, failure(400, "error", "Send rich text as description_html.")
				}
				if _, ok := html.(string); !ok {
					return nil, projectmutations.ErrInvalid
				}
				value = html
			} else if hasHTML {
				return nil, projectmutations.ErrInvalid
			}
			if scope.Configuration.Multiple {
				if id, ok := value.(string); ok {
					value = []string{id}
				}
			}
		} else {
			value = nil
		}
		writer := *r
		writer.propertyValueMutation = true
		writer.db, writer.transactionalEffects, writer.publisher = tx, true, projectmutations.Effects{Tx: tx}
		if cmd.Action == "delete_value" {
			writer.clearedPropertyIDs = []string{cmd.ParentID.String()}
		}
		body, err := json.Marshal(map[string]any{"custom_properties": map[string]any{cmd.ParentID.String(): value}})
		if err != nil {
			return nil, err
		}
		request := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), IssueID: cmd.ResourceID.String(), Method: "PATCH", API: true, Operation: "mcp_workitem_property_value", Timezone: cmd.Timezone, ExpectedWorkspaceID: &cmd.WorkspaceID, Params: map[string]string{}, Present: map[string]bool{}}
		if _, err = writer.writeIssueTransaction(ctx, request, bytes.NewReader(body), "application/json"); err != nil {
			return nil, err
		}
		if cmd.Action == "delete_value" {
			return json.RawMessage("null"), nil
		}
		if _, err = tx.Exec(ctx, `UPDATE work_item_property_value_identities SET external_source=COALESCE($3,external_source),external_id=COALESCE($4,external_id),updated_at=clock_timestamp() WHERE issue_id=$1 AND property_id=$2 AND deleted_at IS NULL AND (external_source IS DISTINCT FROM COALESCE($3,external_source) OR external_id IS DISTINCT FROM COALESCE($4,external_id))`, cmd.ResourceID, cmd.ParentID, cmd.Fields["external_source"], cmd.Fields["external_id"]); err != nil {
			return nil, err
		}
		raw, err := workitemtypes.ProjectPropertyValueAPI(ctx, tx, q)
		if err != nil {
			return nil, err
		}
		return djangoapi.JSONTimestamps(raw, cmd.Timezone)
	})
}
