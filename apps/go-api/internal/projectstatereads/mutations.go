package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type MutationResult struct {
	StateID         string          `json:"state_id"`
	Revision        string          `json:"revision"`
	Result          json.RawMessage `json:"result"`
	RequestID       *uuid.UUID      `json:"request_id,omitempty"`
	ReplayExpiresAt *time.Time      `json:"replay_expires_at,omitempty"`
	Replayed        *bool           `json:"replayed,omitempty"`
}

// MutateState shares the native state rules and SQL helpers with HTTP while
// binding fresh authorization, optional CAS/receipts and effects to one commit.
func (r *WriteRepository) MutateState(ctx context.Context, cmd MutationCommand) (MutationResult, error) {
	fields, input, err := normalizeStateMutation(cmd)
	if err != nil {
		return MutationResult{}, err
	}
	var project *uuid.UUID
	if cmd.ProjectID != uuid.Nil {
		project = &cmd.ProjectID
	}
	result, err := mutationreceipts.Execute(ctx, r.database, mutationreceipts.Command{
		ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: project, Tool: "state", Action: cmd.Action,
		RequestID: cmd.RequestID, NormalizedInput: input,
	}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeStateMutation(ctx, tx, cmd) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			return applyStateMutation(ctx, tx, cmd, fields)
		},
	})
	if err != nil {
		return MutationResult{}, err
	}
	var value MutationResult
	if err = json.Unmarshal(result.Value, &value); err != nil {
		return MutationResult{}, err
	}
	id, err := uuid.Parse(value.StateID)
	if err != nil || id == uuid.Nil || value.Revision == "" {
		return MutationResult{}, errors.New("invalid persisted state mutation result")
	}
	if cmd.RequestID != nil {
		value.RequestID, value.ReplayExpiresAt, value.Replayed = cmd.RequestID, result.ReplayExpiresAt, &result.Replayed
	}
	if r.cache != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		r.InvalidateAppCache(cleanup, cmd.WorkspaceSlug)
	}
	return value, nil
}

func authorizeStateMutation(ctx context.Context, tx pgx.Tx, cmd MutationCommand) error {
	var admitted uuid.UUID
	err := tx.QueryRow(ctx, `SELECT w.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id
 JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active
 AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,w,wm`, cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug).Scan(&admitted)
	if err != nil {
		return stateScopeError(err)
	}
	if cmd.ProjectID != uuid.Nil {
		err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND workspace_id=$2
 AND deleted_at IS NULL AND archived_at IS NULL FOR UPDATE`, cmd.ProjectID, cmd.WorkspaceID).Scan(&admitted)
		if err != nil {
			return stateScopeError(err)
		}
		err = tx.QueryRow(ctx, `SELECT member_id FROM project_members WHERE project_id=$1 AND workspace_id=$2
 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role IN(15,20) FOR SHARE`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&admitted)
		if err != nil {
			return stateScopeError(err)
		}
	}
	if cmd.Revalidate != nil {
		if err = cmd.Revalidate(ctx); err != nil {
			return err
		}
	}
	if cmd.ProjectID == uuid.Nil {
		return ErrWorkspaceNotManaged
	}
	return nil
}

func stateScopeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	return err
}

func applyStateMutation(ctx context.Context, tx pgx.Tx, cmd MutationCommand, fields map[string]any) (json.RawMessage, error) {
	if cmd.Action != "create" {
		if err := lockStateMutationTarget(ctx, tx, cmd); err != nil {
			return nil, err
		}
	}
	input := StateWriteInput{UserID: cmd.ActorID.String(), Slug: cmd.WorkspaceSlug, ProjectID: cmd.ProjectID.String(), StateID: cmd.StateID.String(), Fields: fields}
	var raw json.RawMessage
	var err error
	switch cmd.Action {
	case "create":
		raw, err = createState(ctx, tx, input, fields)
	case "update":
		raw, err = updateState(ctx, tx, input, fields)
	case "delete":
		err = deleteState(ctx, tx, input)
	}
	if err != nil {
		return nil, err
	}
	id := cmd.StateID
	if cmd.Action == "create" {
		var record struct {
			ID uuid.UUID `json:"id"`
		}
		if err = json.Unmarshal(raw, &record); err != nil {
			return nil, err
		}
		id = record.ID
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1`, id).Scan(&revision); err != nil {
		return nil, err
	}
	return json.Marshal(MutationResult{StateID: id.String(), Revision: stateRevision(id.String(), revision), Result: raw})
}

func lockStateMutationTarget(ctx context.Context, tx pgx.Tx, cmd MutationCommand) error {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM states WHERE id=$1 AND project_id=$2 AND workspace_id=$3
 AND deleted_at IS NULL AND "group"<>'triage' AND NOT is_triage FOR UPDATE`, cmd.StateID, cmd.ProjectID, cmd.WorkspaceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM state_edit_revisions WHERE state_id=$1 FOR UPDATE`, id).Scan(&revision); err != nil {
		return err
	}
	if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != stateRevision(id.String(), revision) {
		return ErrRevisionConflict
	}
	return nil
}

func stateRevision(id string, revision int64) string {
	return fmt.Sprintf("state:%s:%d", id, revision)
}
