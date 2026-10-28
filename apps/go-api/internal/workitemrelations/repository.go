// Package workitemrelations owns dependency and custom relation operations.
package workitemrelations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db mutationreceipts.Database }

func NewRepository(db mutationreceipts.Database) *Repository { return &Repository{db: db} }

var DependencyTypes = []string{"blocking", "blocked_by", "start_before", "start_after", "finish_before", "finish_after"}

type Query struct {
	ActorID, WorkspaceID, ProjectID, IssueID uuid.UUID
	WorkspaceSlug, Action                    string
	IsDefault, IsActive                      *bool
	IncludeRevision                          bool
}
type Changes struct {
	Name, Outward, Inward, Color *string
	Active                       *bool
}
type Command struct {
	Query
	DefinitionID, RelatedID       uuid.UUID
	Targets                       []uuid.UUID
	RelationType, DefinitionLabel string
	IsDependency                  bool
	Changes                       Changes
	RequestID                     *uuid.UUID
	ExpectedRevision              *string
	Revalidate                    func(context.Context) error
}

func revisionToken(workspace uuid.UUID, revision int64) string {
	return fmt.Sprintf("workitem_relation:%s:%d", workspace, revision)
}
func workspaceScope(ctx context.Context, tx pgx.Tx, q Query, write bool) error {
	if q.ActorID == uuid.Nil || q.WorkspaceID == uuid.Nil || q.WorkspaceSlug == "" {
		return projectmutations.ErrInvalid
	}
	suffix := ""
	if write {
		suffix = " FOR SHARE OF u,w,wm"
	}
	var role int
	err := tx.QueryRow(ctx, `SELECT wm.role FROM users u JOIN workspace_members wm ON wm.member_id=u.id JOIN workspaces w ON w.id=wm.workspace_id WHERE u.id=$1 AND u.is_active AND w.id=$2 AND w.slug=$3 AND w.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`+suffix, q.ActorID, q.WorkspaceID, q.WorkspaceSlug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return projectmutations.ErrForbidden
	}
	if err != nil {
		return err
	}
	if write && strings.HasSuffix(q.Action, "_definition") && role != 20 {
		return projectmutations.ErrForbidden
	}
	return nil
}
func itemScope(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID, write bool) (uuid.UUID, error) {
	suffix := ""
	if write {
		suffix = " FOR UPDATE OF i"
	}
	var project uuid.UUID
	err := tx.QueryRow(ctx, `SELECT i.project_id FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE i.id=$1 AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$2::uuid", "$3::uuid")+suffix, id, q.ActorID, q.WorkspaceID).Scan(&project)
	if errors.Is(err, pgx.ErrNoRows) {
		err = projectmutations.ErrNotFound
	}
	return project, err
}
func authorizeMutation(ctx context.Context, tx pgx.Tx, in Command) error {
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return err
	}
	if err := workspaceScope(ctx, tx, in.Query, true); err != nil {
		return err
	}
	if !strings.HasSuffix(in.Action, "_definition") {
		ids := append([]uuid.UUID{in.IssueID}, in.Targets...)
		if in.Action == "delete" {
			ids = append(ids, in.RelatedID)
		}
		projects := []uuid.UUID{}
		for _, id := range ids {
			p, err := itemScope(ctx, tx, in.Query, id, false)
			if err != nil {
				return err
			}
			if id == in.IssueID && p != in.ProjectID {
				return projectmutations.ErrNotFound
			}
			projects = append(projects, p)
		}
		slices.SortFunc(projects, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		for _, p := range slices.Compact(projects) {
			if err := projectmutations.Authorize(ctx, tx, projectmutations.Resource{}, projectmutations.Command{ActorID: in.ActorID, WorkspaceID: in.WorkspaceID, WorkspaceSlug: in.WorkspaceSlug, ProjectID: p}); err != nil {
				return err
			}
		}
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		for _, id := range slices.Compact(ids) {
			currentProject, err := itemScope(ctx, tx, in.Query, id, true)
			if err != nil {
				return err
			}
			if !slices.Contains(projects, currentProject) || id == in.IssueID && currentProject != in.ProjectID {
				return projectmutations.ErrForbidden
			}
		}
	}
	if in.Revalidate != nil {
		return in.Revalidate(ctx)
	}
	return nil
}
func (r *Repository) Read(ctx context.Context, q Query) (json.RawMessage, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err = workspaceScope(ctx, tx, q, false); err != nil {
		return nil, err
	}
	var value map[string]any
	switch q.Action {
	case "list_definitions":
		rows, definitionRowsErr := definitionRows(ctx, tx, q, uuid.Nil)
		if definitionRowsErr != nil {
			return nil, definitionRowsErr
		}
		value = map[string]any{"built_in_dependencies": DependencyTypes, "custom_definitions": rows}
	case "list":
		project, itemScopeErr := itemScope(ctx, tx, q, q.IssueID, false)
		if itemScopeErr != nil {
			return nil, itemScopeErr
		}
		if project != q.ProjectID {
			return nil, projectmutations.ErrNotFound
		}
		value, itemScopeErr = relationRows(ctx, tx, q)
		if itemScopeErr != nil {
			return nil, itemScopeErr
		}
	default:
		return nil, projectmutations.ErrInvalid
	}
	if q.IncludeRevision {
		var rev int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM work_item_relation_revisions WHERE workspace_id=$1`, q.WorkspaceID).Scan(&rev); err != nil {
			return nil, err
		}
		value["revision"] = revisionToken(q.WorkspaceID, rev)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
