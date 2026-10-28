package templates

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func row(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(t) FROM resource_templates t WHERE id=$1 AND workspace_id=$2 AND project_id IS NOT DISTINCT FROM $3::uuid AND kind=$4 AND deleted_at IS NULL`, q.ID, q.WorkspaceID, projectID(q.ProjectID), q.Kind).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, projectmutations.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var value map[string]any
	err = djangoapi.DecodeJSON(raw, &value)
	return value, err
}
func (r *Repository) Mutate(ctx context.Context, in Command) (projectmutations.Result, error) {
	if err := normalize(&in); err != nil {
		return projectmutations.Result{}, err
	}
	normalized, err := json.Marshal(struct {
		Query             Query
		Action            string
		Name, Description *string
		Data              map[string]any
		Revision          *string
	}{in.Query, in.Action, in.Name, in.Description, in.Data, in.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var project *uuid.UUID
	if in.ProjectID != uuid.Nil {
		project = &in.ProjectID
	}
	receipt, err := mutationreceipts.Execute(ctx, r.db, mutationreceipts.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, ProjectID: project, Tool: "template", Action: in.Action, RequestID: in.RequestID, NormalizedInput: normalized}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			if _, execErr := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); execErr != nil {
				return execErr
			}
			if scopeErr := scope(ctx, tx, in.Query, true); scopeErr != nil {
				return scopeErr
			}
			if in.Revalidate != nil {
				return in.Revalidate(ctx)
			}
			return nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM workspace_template_settings WHERE workspace_id=$1 FOR UPDATE`, in.WorkspaceID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			if in.ExpectedRevision != nil && *in.ExpectedRevision != token(in.WorkspaceID, revision) {
				return nil, projectmutations.ErrRevision
			}
			id := in.ID
			var before, after map[string]any
			if in.Action != "create" {
				var operationErr error
				before, operationErr = row(ctx, tx, in.Query)
				if operationErr != nil {
					return nil, operationErr
				}
			}
			var operationErr2 error
			switch in.Action {
			case "create":
				id = uuid.New()
				summary := ""
				if in.Description != nil {
					summary = *in.Description
				}
				data, _ := json.Marshal(in.Data)
				_, operationErr2 = tx.Exec(ctx, `INSERT INTO resource_templates(id,workspace_id,project_id,kind,name,summary,template_data,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$8)`, id, in.WorkspaceID, projectID(in.ProjectID), in.Kind, *in.Name, summary, data, in.ActorID)
			case "update":
				var data any
				if in.Data != nil {
					data, _ = json.Marshal(in.Data)
				}
				_, operationErr2 = tx.Exec(ctx, `UPDATE resource_templates SET name=COALESCE($3,name),summary=COALESCE($4,summary),template_data=template_data || COALESCE($5::jsonb,'{}'),updated_at=now(),updated_by_id=$6 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.Name, in.Description, data, in.ActorID)
			case "delete":
				_, operationErr2 = tx.Exec(ctx, `UPDATE resource_templates SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE id=$1 AND workspace_id=$2`, id, in.WorkspaceID, in.ActorID)
			}
			if operationErr2 != nil {
				return nil, operationErr2
			}
			if in.Action != "delete" {
				q := in.Query
				q.ID = id
				after, operationErr2 = row(ctx, tx, q)
				if operationErr2 != nil {
					return nil, operationErr2
				}
			}
			old, _ := json.Marshal(before)
			current, _ := json.Marshal(after)
			if _, operationErr2 = tx.Exec(ctx, `INSERT INTO template_events(workspace_id,template_id,actor_id,action,before_value,after_value) VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb)`, in.WorkspaceID, id, in.ActorID, in.Action, old, current); operationErr2 != nil {
				return nil, operationErr2
			}
			if operationErr2 = tx.QueryRow(ctx, `SELECT revision FROM workspace_template_settings WHERE workspace_id=$1`, in.WorkspaceID).Scan(&revision); operationErr2 != nil {
				return nil, operationErr2
			}
			var value any
			if after != nil {
				value = projection(after)
			}
			raw, operationErr2 := json.Marshal(value)
			if operationErr2 != nil {
				return nil, operationErr2
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: token(in.WorkspaceID, revision), Record: raw})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if in.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = in.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
