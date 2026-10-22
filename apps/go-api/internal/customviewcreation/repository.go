// Package customviewcreation stores member-specific workspace and project view preferences.
package customviewcreation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound  = errors.New("custom view target not found")
	ErrForbidden = errors.New("custom view target is not an active member")
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("custom view database required")
	}
	return &Repository{db: db}, nil
}

// ProjectTarget preserves the endpoint's project lookup before its member check.
func (r *Repository) ProjectTarget(ctx context.Context, userID, slug, projectID string) error {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1::uuid AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, projectID, slug).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=$1::uuid AND pm.member_id=$2::uuid AND pm.is_active=true AND pm.deleted_at IS NULL)`, projectID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrForbidden
	}
	return nil
}

func (r *Repository) WorkspaceTarget(ctx context.Context, userID, slug string) error {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2::uuid AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, slug, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateWorkspace(ctx context.Context, userID, slug string, viewProps any) error {
	encoded, err := json.Marshal(viewProps)
	if err != nil {
		return fmt.Errorf("encode workspace view properties: %w", err)
	}
	command, err := r.db.Exec(ctx, `UPDATE workspace_members wm SET view_props=$3::jsonb,updated_at=NOW(),updated_by_id=$2::uuid FROM workspaces w WHERE wm.workspace_id=w.id AND w.slug=$1 AND wm.member_id=$2::uuid AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL`, slug, userID, encoded)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type ProjectViewChanges struct {
	ViewProps, DefaultProps, Preferences                        any
	SortOrder                                                   float64
	HasViewProps, HasDefaultProps, HasPreferences, HasSortOrder bool
}

func (r *Repository) UpdateProject(ctx context.Context, userID, slug, projectID string, changes ProjectViewChanges) error {
	viewProps, err := jsonParameter(changes.ViewProps, changes.HasViewProps)
	if err != nil {
		return err
	}
	defaultProps, err := jsonParameter(changes.DefaultProps, changes.HasDefaultProps)
	if err != nil {
		return err
	}
	preferences, err := jsonParameter(changes.Preferences, changes.HasPreferences)
	if err != nil {
		return err
	}
	command, err := r.db.Exec(ctx, `UPDATE project_members pm SET
view_props=CASE WHEN @has_view_props THEN @view_props::jsonb ELSE pm.view_props END,
default_props=CASE WHEN @has_default_props THEN @default_props::jsonb ELSE pm.default_props END,
preferences=CASE WHEN @has_preferences THEN @preferences::jsonb ELSE pm.preferences END,
sort_order=CASE WHEN @has_sort_order THEN @sort_order ELSE pm.sort_order END,
updated_at=NOW(),updated_by_id=@user_id::uuid
FROM projects p JOIN workspaces w ON w.id=p.workspace_id
WHERE pm.project_id=p.id AND pm.member_id=@user_id::uuid AND pm.project_id=@project_id::uuid AND w.slug=@slug
AND p.deleted_at IS NULL AND w.deleted_at IS NULL AND pm.is_active=true AND pm.deleted_at IS NULL`, pgx.StrictNamedArgs{
		"has_view_props": changes.HasViewProps, "view_props": viewProps,
		"has_default_props": changes.HasDefaultProps, "default_props": defaultProps,
		"has_preferences": changes.HasPreferences, "preferences": preferences,
		"has_sort_order": changes.HasSortOrder, "sort_order": changes.SortOrder,
		"user_id": userID, "project_id": projectID, "slug": slug,
	})
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

func jsonParameter(value any, present bool) ([]byte, error) {
	if !present || value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode project view property: %w", err)
	}
	return encoded, nil
}
