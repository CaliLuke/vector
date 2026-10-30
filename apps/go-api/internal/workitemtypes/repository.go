// Package workitemtypes owns workspace type definitions and their rollout.
package workitemtypes

import (
	"context"
	"errors"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/jackc/pgx/v5"
)

type Database interface {
	Begin(context.Context) (pgx.Tx, error)
}
type Repository struct{ db Database }

func NewRepository(db Database) *Repository { return &Repository{db: db} }

type Error struct {
	Status      int
	Message     string
	ReviewAgain bool
}

func (e *Error) Error() string                { return e.Message }
func reject(status int, message string) error { return &Error{Status: status, Message: message} }

type Type struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	Active      bool   `json:"active"`
	Default     bool   `json:"default"`
	Mandatory   bool   `json:"mandatory"`
	Level       int    `json:"level"`
	Usage       int    `json:"usage"`
}
type Catalog struct {
	Enabled      bool   `json:"enabled"`
	CanManage    bool   `json:"can_manage"`
	Revision     int64  `json:"revision"`
	ProjectCount int    `json:"project_count"`
	Types        []Type `json:"types"`
}
type Input struct {
	Revision                       int64
	Name, Description, Icon, Color string
	Active, Mandatory              bool
}

func authorize(ctx context.Context, tx pgx.Tx, actor, slug string, write bool) (string, bool, error) {
	var workspace string
	var role int
	err := tx.QueryRow(ctx, `SELECT w.id::text,m.role FROM workspaces w JOIN workspace_members m ON m.workspace_id=w.id WHERE w.slug=$1 AND w.deleted_at IS NULL AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL`, slug, actor).Scan(&workspace, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, reject(403, "You do not have access to this workspace.")
	}
	if err != nil {
		return "", false, err
	}
	if write && role != 20 {
		return "", false, reject(403, "A workspace administrator is required to change work item types.")
	}
	return workspace, role == 20, nil
}
func (r *Repository) Catalog(ctx context.Context, actor, slug string) (Catalog, error) {
	return readSchemaCatalog(ctx, r, actor, slug, loadCatalog)
}

func loadCatalog(ctx context.Context, tx pgx.Tx, workspace string, admin bool) (Catalog, error) {
	out := Catalog{CanManage: admin, Types: []Type{}}
	err := tx.QueryRow(ctx, `SELECT COALESCE(s.enabled,false),COALESCE(s.revision,0),(SELECT count(*) FROM projects WHERE workspace_id=w.id AND deleted_at IS NULL) FROM workspaces w LEFT JOIN work_item_type_settings s ON s.workspace_id=w.id WHERE w.id=$1`, workspace).Scan(&out.Enabled, &out.Revision, &out.ProjectCount)
	if err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT t.id::text,t.name,t.description,COALESCE(t.logo_props->>'name','task'),COALESCE(t.logo_props->>'color','#8b5cf6'),t.is_active,t.is_default,t.mandatory,t.level::integer,
 (SELECT count(*) FROM issues WHERE type_id=t.id)+(SELECT count(*) FROM draft_issues WHERE type_id=t.id)
 FROM issue_types t WHERE t.workspace_id=$1 AND t.deleted_at IS NULL ORDER BY t.is_default DESC,t.created_at,t.id`, workspace)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Type
		if err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Icon, &item.Color, &item.Active, &item.Default, &item.Mandatory, &item.Level, &item.Usage); err != nil {
			return out, err
		}
		out.Types = append(out.Types, item)
	}
	return out, rows.Err()
}
func (r *Repository) Mutate(ctx context.Context, actor, slug, action, id string, input Input) (_ Catalog, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Catalog{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	workspace, admin, err := lockWorkspaceSchema(ctx, tx, actor, slug)
	if err != nil {
		return Catalog{}, err
	}
	before, err := loadCatalog(ctx, tx, workspace, admin)
	if err != nil {
		return before, err
	}
	if input.Revision != before.Revision {
		return before, reject(409, "Work item types changed. Reload and try again.")
	}
	if action != "enable" && !before.Enabled {
		return before, reject(409, "Enable work item types first.")
	}
	switch action {
	case "enable":
		err = enable(ctx, tx, workspace, actor, before)
	case "create", "update":
		err = save(ctx, tx, workspace, actor, action, id, input, before)
	case "delete":
		err = remove(ctx, tx, workspace, actor, id, before)
	default:
		err = reject(400, "Unknown work item type action.")
	}
	if err != nil {
		return Catalog{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, workspace); err != nil {
		return Catalog{}, err
	}
	out, err := loadCatalog(ctx, tx, workspace, admin)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
