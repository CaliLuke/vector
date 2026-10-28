package projectconfiguration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateEstimate changes the native catalogue in a receipt transaction. Selected
// estimate resolution occurs after replay, so retries remain valid after unlinking
// or deleting the selected estimate. Revision tokens cover the whole catalogue.
func (r *Repository) MutateEstimate(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.ProjectID == uuid.Nil || cmd.WorkspaceSlug == "" || cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	fields, err := estimateAPIFields(cmd.Action, cmd.Fields)
	if err != nil {
		return projectmutations.Result{}, err
	}
	cmd.Fields = fields
	switch cmd.Action {
	case "create":
		if cmd.ResourceID != uuid.Nil || cmd.ParentID != uuid.Nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "update", "delete":
		if cmd.ResourceID != cmd.ProjectID || cmd.ParentID != uuid.Nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "create_points":
		if cmd.ResourceID == uuid.Nil || cmd.ParentID != uuid.Nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "update_point", "delete_point":
		if cmd.ResourceID == uuid.Nil || cmd.ParentID == uuid.Nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	db, ok := r.db.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, errors.New("estimate transaction database unavailable")
	}
	input, err := json.Marshal(map[string]any{"estimate_id": cmd.ResourceID, "point_id": cmd.ParentID, "fields": fields, "expected_revision": cmd.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: &cmd.ProjectID, Tool: "project_estimate", Action: cmd.Action, RequestID: cmd.RequestID, NormalizedInput: input}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			return projectmutations.Authorize(ctx, tx, projectmutations.Resource{SchemaLock: true}, cmd)
		},
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			var revision int64
			if scanErr := tx.QueryRow(ctx, `SELECT revision FROM project_estimate_revisions WHERE project_id=$1 FOR UPDATE`, cmd.ProjectID).Scan(&revision); scanErr != nil {
				return nil, scanErr
			}
			token := func() string { return fmt.Sprintf("project_estimate:%s:%d", cmd.ProjectID, revision) }
			if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != token() {
				return nil, projectmutations.ErrRevision
			}
			raw, applyEstimateAPIErr := applyEstimateAPI(ctx, tx, cmd)
			if applyEstimateAPIErr != nil {
				return nil, applyEstimateAPIErr
			}
			if applyEstimateAPIErr = tx.QueryRow(ctx, `SELECT revision FROM project_estimate_revisions WHERE project_id=$1`, cmd.ProjectID).Scan(&revision); applyEstimateAPIErr != nil {
				return nil, applyEstimateAPIErr
			}
			return json.Marshal(projectmutations.Result{ID: cmd.ProjectID.String(), Revision: token(), Record: raw})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if cmd.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = cmd.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
func applyEstimateAPI(ctx context.Context, tx pgx.Tx, cmd projectmutations.Command) (json.RawMessage, error) {
	id := cmd.ResourceID
	if cmd.Action == "update" || cmd.Action == "delete" {
		err := tx.QueryRow(ctx, `SELECT e.id FROM projects p JOIN estimates e ON e.id=p.estimate_id AND e.project_id=p.id AND e.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND e.deleted_at IS NULL FOR UPDATE OF e`, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectmutations.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	} else if cmd.Action != "create" {
		err := tx.QueryRow(ctx, `SELECT id FROM estimates WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR UPDATE`, id, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectmutations.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	if cmd.Action == "update_point" || cmd.Action == "delete_point" {
		var point uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM estimate_points WHERE id=$1 AND estimate_id=$2 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL FOR UPDATE`, cmd.ParentID, id, cmd.ProjectID, cmd.WorkspaceID).Scan(&point)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, projectmutations.ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	native := &Repository{db: tx}
	switch cmd.Action {
	case "create":
		id = uuid.New()
		f := cmd.Fields
		_, err := tx.Exec(ctx, `INSERT INTO estimates(id,project_id,workspace_id,name,description,type,last_used,external_source,external_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULL,clock_timestamp(),clock_timestamp())`, id, cmd.ProjectID, cmd.WorkspaceID, f["name"], f["description"], f["type"], f["last_used"], f["external_source"], f["external_id"], cmd.ActorID)
		if err != nil {
			return nil, mapWriteError(err)
		}
	case "update":
		if err := updateEstimateAPIFields(ctx, tx, "estimates", id, cmd); err != nil {
			return nil, err
		}
	case "delete":
		if err := native.DeleteEstimate(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), id.String()); err != nil {
			return nil, err
		}
		return nil, nil
	case "create_points":
		points := cmd.Fields["points"].([]map[string]any)
		records := make([]json.RawMessage, 0, len(points))
		for _, f := range points {
			point := uuid.New()
			_, err := tx.Exec(ctx, `INSERT INTO estimate_points(id,estimate_id,project_id,workspace_id,key,value,description,external_source,external_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULL,clock_timestamp(),clock_timestamp())`, point, id, cmd.ProjectID, cmd.WorkspaceID, f["key"], f["value"], f["description"], f["external_source"], f["external_id"], cmd.ActorID)
			if err != nil {
				return nil, mapWriteError(err)
			}
			raw, err := scanJSON(ctx, tx, `SELECT `+estimatePointAPIJSON+` FROM estimate_points ep WHERE ep.id=$1`, point.String())
			if err != nil {
				return nil, err
			}
			records = append(records, raw)
		}
		return json.Marshal(records)
	case "update_point":
		if err := updateEstimateAPIFields(ctx, tx, "estimate_points", cmd.ParentID, cmd); err != nil {
			return nil, err
		}
		return scanJSON(ctx, tx, `SELECT `+estimatePointAPIJSON+` FROM estimate_points ep WHERE ep.id=$1`, cmd.ParentID.String())
	case "delete_point":
		_, err := native.DeleteEstimatePoint(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), id.String(), cmd.ParentID.String(), strings.NewReader("{}"), "application/json")
		return nil, err
	}
	return scanJSON(ctx, tx, `SELECT `+estimateAPIJSON+` FROM estimates e WHERE e.id=$1`, id.String())
}
func updateEstimateAPIFields(ctx context.Context, tx pgx.Tx, table string, id uuid.UUID, cmd projectmutations.Command) error {
	sets := []string{"updated_at=clock_timestamp()", "updated_by_id=$2"}
	args := []any{id, cmd.ActorID, cmd.ProjectID, cmd.WorkspaceID}
	// Column identifiers come only from estimateAPIFields' closed validation.
	for _, key := range []string{"name", "description", "external_source", "external_id", "value", "key"} {
		if value, ok := cmd.Fields[key]; ok {
			args = append(args, value)
			sets = append(sets, fmt.Sprintf("%s=$%d", pgx.Identifier{key}.Sanitize(), len(args)))
		}
	}
	_, err := tx.Exec(ctx, "UPDATE "+pgx.Identifier{table}.Sanitize()+" SET "+strings.Join(sets, ",")+" WHERE id=$1 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL", args...)
	return mapWriteError(err)
}
