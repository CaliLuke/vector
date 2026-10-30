package cyclereads

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5"
)

func (r *LifecycleRepository) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.Action != "archive" && cmd.Action != "unarchive" && cmd.Action != "complete" && cmd.Action != "create" && cmd.Action != "update" && cmd.Action != "delete" && cmd.Action != "manage_workitems" && cmd.Action != "transfer_workitems" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	var create CycleCreate
	var validation map[string]any
	switch cmd.Action {
	case "create":
		create, validation = ValidateCreateInput(cmd.Fields, nil, true)
	case "update":
		cmd.Fields, validation = ValidatePatchInput(cmd.Fields, nil, true)
	case "manage_workitems", "transfer_workitems":
		var err error
		cmd.Fields, err = workitems.NormalizePlanningFields(cmd.Action, cmd.Fields)
		if err != nil {
			return projectmutations.Result{}, err
		}
	default:
		if len(cmd.Fields) != 0 {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	}
	if len(validation) > 0 {
		return projectmutations.Result{}, &djangoapi.HTTPError{Status: 400, Body: validation}
	}
	db, ok := r.database.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	return projectmutations.Execute(ctx, db, projectmutations.Resource{Name: "cycle", Table: "cycles", RevisionTable: "cycle_edit_revisions", RevisionColumn: "cycle_id", SchemaLock: true, DeleteOwnerColumn: "owned_by_id"}, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		transaction := &LifecycleRepository{database: tx, effects: projectmutations.Effects{Tx: tx}, now: r.now, origin: r.origin}
		switch cmd.Action {
		case "manage_workitems", "transfer_workitems":
			return nil, workitems.ApplyPlanning(ctx, tx, cmd, "cycle", r.origin)
		case "create":
			result, err := transaction.Create(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), create, true, cmd.Fields)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		case "update":
			result, err := transaction.Patch(ctx, cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.ActorID.String(), true, cmd.Fields, cmd.Fields)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		case "delete":
			return nil, transaction.Delete(ctx, cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.ActorID.String(), true)
		}
		cycle, err := transaction.Cycle(ctx, cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String())
		if err != nil {
			return nil, err
		}
		now := r.now()
		if cmd.Action == "complete" || cmd.Action == "archive" && (cycle.EndDate == nil || !cycle.EndDate.Before(now)) {
			day, dateErr := transaction.completionDay(ctx, cmd.ActorID.String(), now)
			if dateErr != nil {
				return nil, dateErr
			}
			fields := map[string]any{"end_date": day}
			result, patchErr := transaction.Patch(ctx, cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.ActorID.String(), true, fields, fields)
			if patchErr != nil {
				return nil, patchErr
			}
			if cmd.Action == "complete" {
				return json.Marshal(result)
			}
		}
		_, err = transaction.SetArchived(ctx, cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.ActorID.String(), cmd.Action == "archive")
		return nil, err
	})
}

// completionDay uses the same actor timezone that Patch uses to parse date-only
// input. A UTC day interpreted in another zone can place completion in the future.
func (r *LifecycleRepository) completionDay(ctx context.Context, actor string, now time.Time) (string, error) {
	var zone string
	if err := r.database.QueryRow(ctx, `SELECT COALESCE(NULLIF(user_timezone,''),'UTC') FROM users WHERE id=$1`, actor).Scan(&zone); err != nil {
		return "", err
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return "", err
	}
	return now.In(location).Format("2006-01-02"), nil
}
