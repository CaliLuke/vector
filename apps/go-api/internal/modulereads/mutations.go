package modulereads

import (
	"context"
	"encoding/json"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5"
)

func (r *LifecycleRepository) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.Action != "archive" && cmd.Action != "unarchive" && cmd.Action != "create" && cmd.Action != "update" && cmd.Action != "delete" && cmd.Action != "manage_workitems" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	if cmd.Action == "manage_workitems" {
		var err error
		cmd.Fields, err = workitems.NormalizePlanningFields(cmd.Action, cmd.Fields)
		if err != nil {
			return projectmutations.Result{}, err
		}
	} else if cmd.Action != "create" && cmd.Action != "update" && len(cmd.Fields) > 0 {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	db, ok := r.database.(mutationreceipts.Database)
	if !ok {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	return projectmutations.Execute(ctx, db, projectmutations.Resource{Name: "module", Table: "modules", RevisionTable: "module_edit_revisions", RevisionColumn: "module_id", SchemaLock: true}, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		transaction := &LifecycleRepository{database: tx, reads: &Repository{database: tx}, effects: projectmutations.Effects{Tx: tx}, now: r.now, baseHost: r.baseHost}
		switch cmd.Action {
		case "manage_workitems":
			return nil, workitems.ApplyPlanning(ctx, tx, cmd, "module", r.baseHost)
		case "create":
			return transaction.CreateModuleV1(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.Fields, nil, r.baseHost)
		case "update":
			return transaction.UpdateModuleV1(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.Fields, nil, r.baseHost)
		case "delete":
			if err := transaction.CheckDeleteModule(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), true); err != nil {
				return nil, err
			}
			return nil, transaction.DeleteModule(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), true, r.baseHost)
		}
		_, err := transaction.SetModuleArchived(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, cmd.ProjectID.String(), cmd.ResourceID.String(), cmd.Action == "archive")
		return nil, err
	})
}
