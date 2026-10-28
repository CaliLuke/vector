package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MutateResource binds project initialization, native field validation and
// durable effects to the same transaction as the optional retry receipt.
func (r *LifecycleRepository) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.WorkspaceSlug == "" || cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	var project *uuid.UUID
	switch cmd.Action {
	case "create":
		if cmd.ProjectID != uuid.Nil || cmd.ResourceID != uuid.Nil || cmd.ExpectedRevision != nil {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
	case "update", "delete":
		if cmd.ProjectID == uuid.Nil || cmd.ProjectID != cmd.ResourceID {
			return projectmutations.Result{}, projectmutations.ErrInvalid
		}
		project = &cmd.ProjectID
	default:
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	if cmd.Action == "delete" && len(cmd.Fields) != 0 {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	input, err := json.Marshal(map[string]any{"project_id": project, "fields": cmd.Fields, "expected_revision": cmd.ExpectedRevision})
	if err != nil {
		return projectmutations.Result{}, err
	}
	receipt, err := mutationreceipts.Execute(ctx, r.writer, mutationreceipts.Command{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: project, Tool: "project", Action: cmd.Action, RequestID: cmd.RequestID, NormalizedInput: input}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeProjectMutation(ctx, tx, cmd) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			id := cmd.ResourceID
			if cmd.Action != "create" {
				var revision int64
				scanErr := tx.QueryRow(ctx, `SELECT r.revision FROM projects p JOIN project_edit_revisions r ON r.project_id=p.id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL FOR UPDATE OF r`, id, cmd.WorkspaceID).Scan(&revision)
				if errors.Is(scanErr, pgx.ErrNoRows) {
					return nil, ErrNotFound
				}
				if scanErr != nil {
					return nil, scanErr
				}
				if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != projectRevision(id, revision) {
					return nil, ErrRevisionConflict
				}
			}
			db := projectMutationTx{Tx: tx}
			native, mutationErr := NewLifecycleRepository(db, projectmutations.Effects{Tx: tx}, r.baseHost, r.coverAssets)
			if mutationErr != nil {
				return nil, mutationErr
			}
			native.transactionalEffects = true
			body, mutationErr := json.Marshal(cmd.Fields)
			if mutationErr != nil {
				return nil, mutationErr
			}
			fields := djangoinput.Input{Fields: cmd.Fields, Raw: body}
			var record json.RawMessage
			switch cmd.Action {
			case "create":
				record, mutationErr = native.CreateProject(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, fields, false)
			case "update":
				record, mutationErr = native.UpdateProject(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, id.String(), fields, false, false)
			case "delete":
				mutationErr = native.DeleteProject(ctx, cmd.ActorID.String(), cmd.WorkspaceSlug, id.String(), false)
			}
			if mutationErr != nil {
				return nil, mutationErr
			}
			if cmd.Action == "create" {
				var created struct {
					ID uuid.UUID `json:"id"`
				}
				if decodeErr := json.Unmarshal(record, &created); decodeErr != nil {
					return nil, decodeErr
				}
				if created.ID == uuid.Nil {
					return nil, errors.New("created project has no identifier")
				}
				id = created.ID
			}
			var revision int64
			if revisionErr := tx.QueryRow(ctx, `SELECT revision FROM project_edit_revisions WHERE project_id=$1`, id).Scan(&revision); revisionErr != nil {
				return nil, revisionErr
			}
			return json.Marshal(projectmutations.Result{ID: id.String(), Revision: projectRevision(id, revision), Record: record})
		},
	})
	if err != nil {
		return projectmutations.Result{}, err
	}
	var result projectmutations.Result
	if err := json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if cmd.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = cmd.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}

func authorizeProjectMutation(ctx context.Context, tx pgx.Tx, cmd projectmutations.Command) error {
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, cmd.WorkspaceID); err != nil {
		return err
	}
	var workspaceRole int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm,w`, cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug).Scan(&workspaceRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if cmd.Action == "create" {
		if workspaceRole != 15 && workspaceRole != 20 {
			return ErrForbidden
		}
	} else {
		var id uuid.UUID
		// Retain authorization on a soft-deleted project for deletion replay;
		// the Apply callback still requires a live target for a new mutation.
		err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND workspace_id=$2 FOR UPDATE`, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var role int
		err = tx.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role IN(5,15,20) FOR SHARE`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&role)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if workspaceRole != 20 && role != 20 {
			return ErrForbidden
		}
	}
	if cmd.Revalidate != nil {
		return cmd.Revalidate(ctx)
	}
	return nil
}

type projectMutationTx struct{ pgx.Tx }

func (t projectMutationTx) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	if options.IsoLevel != "" && options.IsoLevel != pgx.ReadCommitted || options.AccessMode != "" || options.DeferrableMode != "" {
		return nil, errors.New("unsupported nested project transaction options")
	}
	return t.Begin(ctx)
}
