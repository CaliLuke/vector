// Package projectmutations binds native project-resource changes to current
// membership, revisions and optional retry receipts in one transaction.
package projectmutations

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

var ErrForbidden = errors.New("project mutation access denied")
var ErrNotFound = errors.New("project resource unavailable")
var ErrRevision = errors.New("resource revision changed")
var ErrInvalid = errors.New("invalid project mutation")

type Command struct {
	ActorID, WorkspaceID, ProjectID, ResourceID uuid.UUID
	ParentID                                    uuid.UUID
	Timezone                                    string
	WorkspaceSlug, Action                       string
	RequestID                                   *uuid.UUID
	ExpectedRevision                            *string
	Fields                                      map[string]any
	Revalidate                                  func(context.Context) error
}
type Result struct {
	ID              string          `json:"id"`
	Revision        string          `json:"revision"`
	Record          json.RawMessage `json:"record"`
	RequestID       *uuid.UUID      `json:"request_id,omitempty"`
	ReplayExpiresAt *time.Time      `json:"replay_expires_at,omitempty"`
	Replayed        *bool           `json:"replayed,omitempty"`
}

// Resource identifiers are authored by native domain callers, never by requests.
type Resource struct {
	Name, Table, RevisionTable, RevisionColumn string
	SchemaLock                                 bool
	AllowGuests                                bool
	ParentColumn                               string
	Authorize                                  func(context.Context, pgx.Tx) error
}

func Execute(ctx context.Context, db mutationreceipts.Database, resource Resource, cmd Command, apply func(context.Context, pgx.Tx) (json.RawMessage, error)) (Result, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.ProjectID == uuid.Nil || (cmd.Action != "create" && cmd.ResourceID == uuid.Nil) || (cmd.Action == "create" && (cmd.ResourceID != uuid.Nil || cmd.ExpectedRevision != nil)) || cmd.WorkspaceSlug == "" || cmd.Action == "" || cmd.ExpectedRevision != nil && *cmd.ExpectedRevision == "" {
		return Result{}, ErrInvalid
	}
	input, err := json.Marshal(map[string]any{"id": cmd.ResourceID, "parent_id": cmd.ParentID, "expected_revision": cmd.ExpectedRevision, "fields": cmd.Fields})
	if err != nil {
		return Result{}, err
	}
	receipt, err := mutationreceipts.Execute(ctx, db, mutationreceipts.Command{ActorID: cmd.ActorID, WorkspaceID: cmd.WorkspaceID, ProjectID: &cmd.ProjectID, Tool: resource.Name, Action: cmd.Action, RequestID: cmd.RequestID, NormalizedInput: input}, mutationreceipts.Operation{
		Authorize: func(ctx context.Context, tx pgx.Tx) error { return Authorize(ctx, tx, resource, cmd) },
		Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
			id := cmd.ResourceID
			revisionSQL := "SELECT revision FROM " + pgx.Identifier{resource.RevisionTable}.Sanitize() + " WHERE " + pgx.Identifier{resource.RevisionColumn}.Sanitize() + "=$1 FOR UPDATE"
			var revision int64
			token := func() string { return fmt.Sprintf("%s:%s:%d", resource.Name, id, revision) }
			if cmd.Action != "create" {
				query := "SELECT id FROM " + pgx.Identifier{resource.Table}.Sanitize() + " WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL"
				args := []any{id, cmd.ProjectID, cmd.WorkspaceID}
				if resource.ParentColumn != "" {
					query += " AND " + pgx.Identifier{resource.ParentColumn}.Sanitize() + "=$4"
					args = append(args, cmd.ParentID)
				}
				scanErr := tx.QueryRow(ctx, query+" FOR UPDATE", args...).Scan(&id)
				if errors.Is(scanErr, pgx.ErrNoRows) {
					return nil, ErrNotFound
				}
				if scanErr != nil {
					return nil, scanErr
				}
				if scanErr = tx.QueryRow(ctx, revisionSQL, id).Scan(&revision); scanErr != nil {
					return nil, scanErr
				}
				if cmd.ExpectedRevision != nil && *cmd.ExpectedRevision != token() {
					return nil, ErrRevision
				}
			}
			raw, applyErr := apply(ctx, tx)
			if applyErr != nil {
				return nil, applyErr
			}
			if cmd.Action == "create" {
				var created struct {
					ID uuid.UUID `json:"id"`
				}
				if applyErr = json.Unmarshal(raw, &created); applyErr != nil {
					return nil, applyErr
				}
				if created.ID == uuid.Nil {
					return nil, errors.New("created resource has no identifier")
				}
				id = created.ID
			}
			if applyErr = tx.QueryRow(ctx, revisionSQL, id).Scan(&revision); applyErr != nil {
				return nil, applyErr
			}
			return json.Marshal(Result{ID: id.String(), Revision: token(), Record: raw})
		},
	})
	if err != nil {
		return Result{}, err
	}
	var result Result
	if err = json.Unmarshal(receipt.Value, &result); err != nil {
		return result, err
	}
	if cmd.RequestID != nil {
		result.RequestID, result.ReplayExpiresAt, result.Replayed = cmd.RequestID, receipt.ReplayExpiresAt, &receipt.Replayed
	}
	return result, nil
}
func lockScope(ctx context.Context, tx pgx.Tx, cmd Command, allowGuests bool) error {
	roles := []int{15, 20}
	if allowGuests {
		roles = []int{5, 15, 20}
	}
	var id uuid.UUID
	queries := []struct {
		sql  string
		args []any
	}{
		{`SELECT w.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id
 WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL
 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) FOR SHARE OF u,wm,w`, []any{cmd.ActorID, cmd.WorkspaceID, cmd.WorkspaceSlug}},
		{`SELECT id FROM projects WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND archived_at IS NULL FOR UPDATE`, []any{cmd.ProjectID, cmd.WorkspaceID}},
		{`SELECT member_id FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND is_active AND deleted_at IS NULL AND role=ANY($4::int[]) FOR SHARE`, []any{cmd.ProjectID, cmd.WorkspaceID, cmd.ActorID, roles}},
	}
	for _, query := range queries {
		err := tx.QueryRow(ctx, query.sql, query.args...).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Authorize holds the common project mutation scope through the caller's transaction.
// Domain-specific multi-record operations use this before receipt replay as well.
func Authorize(ctx context.Context, tx pgx.Tx, resource Resource, cmd Command) error {
	if resource.SchemaLock {
		if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, cmd.WorkspaceID); err != nil {
			return err
		}
	}
	if err := lockScope(ctx, tx, cmd, resource.AllowGuests); err != nil {
		return err
	}
	if resource.Authorize != nil {
		if err := resource.Authorize(ctx, tx); err != nil {
			return err
		}
	}
	if cmd.Revalidate != nil {
		return cmd.Revalidate(ctx)
	}
	return nil
}
