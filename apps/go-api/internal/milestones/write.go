package milestones

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrConflict         = errors.New("milestone conflicts with an existing resource")
	ErrRevisionConflict = errors.New("milestone revision has changed")
)

type WriteOptions struct {
	Scope
	RequestID        *uuid.UUID
	ExpectedRevision *string
	Origin           string
	// Transports revalidate their credential here, after receipt and domain lock
	// waits and before replay. Database membership locks remain held to commit.
	Revalidate func(context.Context) error
}

func (opts WriteOptions) revalidate(ctx context.Context) error {
	if opts.Revalidate != nil {
		return opts.Revalidate(ctx)
	}
	return nil
}

type MutationResult struct {
	MilestoneID     string     `json:"milestone_id"`
	Revision        string     `json:"revision"`
	Result          *Record    `json:"result"`
	RequestID       *uuid.UUID `json:"request_id,omitempty"`
	ReplayExpiresAt *time.Time `json:"replay_expires_at,omitempty"`
	Replayed        *bool      `json:"replayed,omitempty"`
}

func (r *Repository) write(ctx context.Context, opts WriteOptions, action string, input map[string]any, apply func(pgx.Tx) (MutationResult, error)) (MutationResult, error) {
	if !opts.valid() || opts.ExpectedRevision != nil && *opts.ExpectedRevision == "" {
		return MutationResult{}, ErrInvalidInput
	}
	if opts.ExpectedRevision != nil {
		input["expected_revision"] = *opts.ExpectedRevision
	}
	normalized, err := json.Marshal(input)
	if err != nil {
		return MutationResult{}, err
	}
	result, err := mutationreceipts.Execute(ctx, r.database, mutationreceipts.Command{
		ActorID: opts.ActorID, WorkspaceID: opts.WorkspaceID, ProjectID: &opts.ProjectID, Tool: "milestone", Action: action, RequestID: opts.RequestID, NormalizedInput: normalized,
	}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return authorizeWrite(ctx, tx, opts) },
		Apply: func(_ context.Context, tx pgx.Tx) (json.RawMessage, error) {
			value, applyErr := apply(tx)
			if applyErr != nil {
				return nil, applyErr
			}
			return json.Marshal(value)
		},
	})
	if err != nil {
		var failure *pgconn.PgError
		if errors.As(err, &failure) && failure.Code == "23505" && (failure.ConstraintName == "milestones_live_title" || failure.ConstraintName == "milestone_issues_live_item") {
			return MutationResult{}, ErrConflict
		}
		return MutationResult{}, err
	}
	var value MutationResult
	if err = json.Unmarshal(result.Value, &value); err != nil {
		return MutationResult{}, err
	}
	if _, err = uuid.Parse(value.MilestoneID); err != nil || value.Revision == "" {
		return MutationResult{}, errors.New("invalid persisted milestone result")
	}
	if opts.RequestID != nil {
		value.RequestID, value.ReplayExpiresAt, value.Replayed = opts.RequestID, result.ReplayExpiresAt, &result.Replayed
	}
	return value, nil
}

func authorizeWrite(ctx context.Context, tx pgx.Tx, opts WriteOptions) error {
	var admitted uuid.UUID
	err := tx.QueryRow(ctx, `SELECT w.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id
 JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active
 AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,w,wm`, opts.ActorID, opts.WorkspaceID, opts.WorkspaceSlug).Scan(&admitted)
	if err != nil {
		return scopeError(err)
	}
	// Project serialization precedes milestone, revision and item locks.
	err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND archived_at IS NULL FOR UPDATE`, opts.ProjectID, opts.WorkspaceID).Scan(&admitted)
	if err != nil {
		return scopeError(err)
	}
	err = tx.QueryRow(ctx, `SELECT member_id FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role IN(15,20) FOR SHARE`, opts.ProjectID, opts.WorkspaceID, opts.ActorID).Scan(&admitted)
	if err != nil {
		return scopeError(err)
	}
	return opts.revalidate(ctx)
}

func scopeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	return err
}

func lockMilestone(ctx context.Context, tx pgx.Tx, opts WriteOptions, id uuid.UUID) error {
	var locked uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM milestones WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL FOR UPDATE`, id, opts.ProjectID, opts.WorkspaceID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM milestone_edit_revisions WHERE milestone_id=$1 FOR UPDATE`, id).Scan(&revision); err != nil {
		return err
	}
	if opts.ExpectedRevision != nil && *opts.ExpectedRevision != revisionToken(id, revision) {
		return ErrRevisionConflict
	}
	return nil
}

func mutationResult(ctx context.Context, tx pgx.Tx, id uuid.UUID, record *Record) (MutationResult, error) {
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM milestone_edit_revisions WHERE milestone_id=$1`, id).Scan(&revision); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{MilestoneID: id.String(), Revision: revisionToken(id, revision), Result: record}, nil
}
