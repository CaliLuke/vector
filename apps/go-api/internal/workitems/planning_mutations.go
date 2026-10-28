package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NormalizePlanningFields gives receipt hashing the same canonical IDs and set
// semantics used by the native operation. The MCP boundary supplies typed lists.
func NormalizePlanningFields(action string, fields map[string]any) (map[string]any, error) {
	result := map[string]any{}
	if action == "transfer_workitems" {
		value, ok := fields["new_cycle_id"].(string)
		id, err := uuid.Parse(value)
		if !ok || err != nil || id == uuid.Nil || len(fields) != 1 {
			return nil, projectmutations.ErrInvalid
		}
		result["new_cycle_id"] = id.String()
		return result, nil
	}
	if action != "manage_workitems" {
		return nil, projectmutations.ErrInvalid
	}
	for key := range fields {
		if key != "add_ids" && key != "remove_ids" {
			return nil, projectmutations.ErrInvalid
		}
	}
	total := 0
	for _, key := range []string{"add_ids", "remove_ids"} {
		values, ok := fields[key].([]string)
		if fields[key] != nil && !ok {
			return nil, projectmutations.ErrInvalid
		}
		ids := make([]string, 0, len(values))
		for _, value := range values {
			id, err := uuid.Parse(value)
			if err != nil || id == uuid.Nil {
				return nil, projectmutations.ErrInvalid
			}
			ids = append(ids, id.String())
		}
		slices.Sort(ids)
		ids = slices.Compact(ids)
		result[key] = ids
		total += len(ids)
	}
	if total == 0 {
		return nil, projectmutations.ErrInvalid
	}
	return result, nil
}

// ApplyPlanning runs under the caller's receipt transaction, schema lock and
// current project authorization. Native planning writes and effects use this
// same transaction; native nested transactions are savepoints only.
func ApplyPlanning(ctx context.Context, tx pgx.Tx, cmd projectmutations.Command, kind, origin string) error {
	if kind != "cycle" && kind != "module" {
		return projectmutations.ErrInvalid
	}
	fields, err := NormalizePlanningFields(cmd.Action, cmd.Fields)
	if err != nil {
		return err
	}
	r, err := NewRepository(tx, projectmutations.Effects{Tx: tx}, origin)
	if err != nil {
		return err
	}
	r.transactionalEffects = true
	q := Request{ActorID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), API: true, Method: "POST", Params: map[string]string{kind + "_id": cmd.ResourceID.String()}}
	if cmd.Action == "transfer_workitems" {
		if kind != "cycle" {
			return projectmutations.ErrInvalid
		}
		var id string
		scanErr := tx.QueryRow(ctx, `SELECT id::text FROM cycles WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR UPDATE`, fields["new_cycle_id"], cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if scanErr != nil {
			return scanErr
		}
		body, scanErr := json.Marshal(fields)
		if scanErr != nil {
			return scanErr
		}
		_, scanErr = r.transferCycle(ctx, q, bytes.NewReader(body), "application/json")
		return scanErr
	}
	add := fields["add_ids"].([]string)
	remove := fields["remove_ids"].([]string)
	if len(add) > 0 {
		body, marshalErr := json.Marshal(map[string]any{"issues": add})
		if marshalErr != nil {
			return marshalErr
		}
		if _, marshalErr = r.planningTransaction(ctx, q, bytes.NewReader(body), "application/json"); marshalErr != nil {
			return marshalErr
		}
	}
	for _, id := range remove {
		q.IssueID = id
		if _, err = r.removePlanning(ctx, q, kind, false); err != nil {
			return err
		}
	}
	return nil
}
