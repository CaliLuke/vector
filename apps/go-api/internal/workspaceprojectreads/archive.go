package workspaceprojectreads

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

var ErrRevisionConflict = errors.New("project revision has changed")

// ArchiveCommand is a credential-neutral application command. App retains the
// browser route's member policy; false selects the v1 API policy used by the SDK.
// WorkspaceID may be omitted by HTTP, but MCP supplies its verified immutable ID.
// Revalidate runs before domain authorization and again after its lock waits,
// before any receipt is inspected.
type ArchiveCommand struct {
	ActorID, WorkspaceID, ProjectID uuid.UUID
	WorkspaceSlug                   string
	Archive, App                    bool
	RequestID                       *uuid.UUID
	ExpectedRevision                *string
	Revalidate                      func(context.Context) error
}

type ArchiveResult struct {
	At       time.Time               `json:"at"`
	Revision string                  `json:"revision"`
	Receipt  mutationreceipts.Result `json:"-"`
}

// ArchiveProject owns authorization, the project/favorite changes, revision
// comparison and the optional receipt in one primary-database transaction.
func (r *LifecycleRepository) ArchiveProject(ctx context.Context, cmd ArchiveCommand) (ArchiveResult, error) {
	if cmd.WorkspaceID == uuid.Nil {
		err := r.writer.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, cmd.WorkspaceSlug).Scan(&cmd.WorkspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ArchiveResult{}, ErrNotFound
		}
		if err != nil {
			return ArchiveResult{}, err
		}
	}
	input, err := json.Marshal(struct {
		ProjectID        uuid.UUID `json:"project_id"`
		ExpectedRevision *string   `json:"expected_revision,omitempty"`
	}{cmd.ProjectID, cmd.ExpectedRevision})
	if err != nil {
		return ArchiveResult{}, err
	}
	action := "unarchive"
	if cmd.Archive {
		action = "archive"
	}
	result, err := mutationreceipts.Execute(ctx, r.writer, mutationreceipts.Command{
		ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: &cmd.ProjectID, Tool: "project", Action: action,
		RequestID: cmd.RequestID, NormalizedInput: input,
	}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error {
			if authorizeErr := authorizeArchive(ctx, tx, cmd); authorizeErr != nil {
				return authorizeErr
			}
			// A project or membership lock can delay this call after the first
			// credential check. Revalidate before inspecting or creating receipts.
			if cmd.Revalidate != nil {
				return cmd.Revalidate(ctx)
			}
			return nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) { return applyArchive(ctx, tx, cmd) },
	})
	if err != nil {
		return ArchiveResult{}, err
	}
	var archived ArchiveResult
	if err = json.Unmarshal(result.Value, &archived); err != nil {
		return ArchiveResult{}, err
	}
	archived.Receipt = result
	return archived, nil
}

func authorizeArchive(ctx context.Context, tx pgx.Tx, cmd ArchiveCommand) error {
	if cmd.Revalidate != nil {
		if err := cmd.Revalidate(ctx); err != nil {
			return err
		}
	}
	var workspaceRole int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u
 JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id
 WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 FOR SHARE OF u,w,wm`, cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug).Scan(&workspaceRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	// Lock the project before revision reads. A waiting update must see the latest
	// counter in the next READ COMMITTED statement, not a pre-lock join snapshot.
	var project uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR UPDATE`, cmd.ProjectID, cmd.WorkspaceID).Scan(&project)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !cmd.App && cmd.Archive {
		if workspaceRole == 15 || workspaceRole == 20 {
			return nil
		}
		return ErrForbidden
	}
	var role int
	err = tx.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3
 AND is_active AND deleted_at IS NULL AND role IN(5,15,20) FOR SHARE`, cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if workspaceRole == 20 || role == 20 || cmd.App && role == 15 {
		return nil
	}
	return ErrForbidden
}

func applyArchive(ctx context.Context, tx pgx.Tx, cmd ArchiveCommand) (json.RawMessage, error) {
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM project_edit_revisions WHERE project_id=$1`, cmd.ProjectID).Scan(&revision); err != nil {
		return nil, err
	}
	if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != projectRevision(cmd.ProjectID, revision) {
		return nil, ErrRevisionConflict
	}
	var result ArchiveResult
	err := tx.QueryRow(ctx, `UPDATE projects SET archived_at=CASE WHEN $3 THEN clock_timestamp() ELSE NULL END,
 updated_at=clock_timestamp(),updated_by_id=$1 WHERE id=$2 RETURNING COALESCE(archived_at,updated_at)`, cmd.ActorID, cmd.ProjectID, cmd.Archive).Scan(&result.At)
	if err != nil {
		return nil, err
	}
	if cmd.Archive {
		if _, err = tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=clock_timestamp() WHERE project_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, cmd.ProjectID, cmd.WorkspaceID); err != nil {
			return nil, err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT revision FROM project_edit_revisions WHERE project_id=$1`, cmd.ProjectID).Scan(&revision); err != nil {
		return nil, err
	}
	result.Revision = projectRevision(cmd.ProjectID, revision)
	return json.Marshal(result)
}

func projectRevision(id uuid.UUID, revision int64) string {
	return fmt.Sprintf("project:%s:%d", id, revision)
}
