package workitemtypes

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

type ProjectType struct {
	Type
	Imported      bool `json:"imported"`
	ProjectActive bool `json:"project_active"`
	Available     bool `json:"available"`
}
type ProjectCatalog struct {
	DefaultTypeID string        `json:"default_type_id"`
	Enabled       bool          `json:"enabled"`
	CanManage     bool          `json:"can_manage"`
	Revision      int64         `json:"revision"`
	ProjectName   string        `json:"project_name"`
	Types         []ProjectType `json:"types"`
}
type ProjectInput struct {
	Revision int64
	Action   string
	TypeIDs  []string
}

func authorizeProject(ctx context.Context, tx pgx.Tx, actor, slug, project string, write bool) (string, bool, error) {
	workspace, _, err := authorize(ctx, tx, actor, slug, false)
	if err != nil {
		return "", false, err
	}
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, tx, actor, slug, project, false)
	if err != nil {
		return "", false, err
	}
	if !allowed {
		return "", false, reject(403, "You do not have access to this project.")
	}
	var admin bool
	err = tx.QueryRow(ctx, `SELECT m.role=20 FROM projects p JOIN project_members m ON m.project_id=p.id AND m.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL`, project, workspace, actor).Scan(&admin)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, reject(404, "Project not found.")
	}
	if err != nil {
		return "", false, err
	}
	if write && !admin {
		return "", false, reject(403, "A project administrator is required to manage project configuration.")
	}
	return workspace, admin, nil
}

func (r *Repository) ProjectCatalog(ctx context.Context, actor, slug, project string) (ProjectCatalog, error) {
	return readProjectSchemaCatalog(ctx, r, actor, slug, project, loadProjectCatalog)
}

func loadProjectCatalog(ctx context.Context, tx pgx.Tx, workspace, project string, admin bool) (ProjectCatalog, error) {
	out := ProjectCatalog{CanManage: admin, Types: []ProjectType{}}
	err := tx.QueryRow(ctx, `SELECT COALESCE(s.enabled,false),COALESCE(s.revision,0),p.name,COALESCE((SELECT preference.type_id::text FROM work_item_project_defaults preference WHERE preference.project_id=p.id AND preference.workspace_id=p.workspace_id),'') FROM projects p LEFT JOIN work_item_type_settings s ON s.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL`, project, workspace).Scan(&out.Enabled, &out.Revision, &out.ProjectName, &out.DefaultTypeID)
	if err != nil {
		return out, err
	}
	// Project configuration never exposes workspace-wide issue or private draft usage.
	rows, err := tx.Query(ctx, `SELECT t.id::text,t.name,t.description,COALESCE(t.logo_props->>'name','task'),COALESCE(t.logo_props->>'color','#8b5cf6'),t.is_active,t.is_default,t.mandatory,t.level::integer,
 link.id IS NOT NULL,COALESCE(link.is_active,false)
 FROM issue_types t LEFT JOIN project_issue_types link ON link.issue_type_id=t.id AND link.project_id=$2 AND link.workspace_id=t.workspace_id AND link.deleted_at IS NULL
 WHERE t.workspace_id=$1 AND t.deleted_at IS NULL ORDER BY t.is_default DESC,t.created_at,t.id`, workspace, project)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ProjectType
		if err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Icon, &item.Color, &item.Active, &item.Default, &item.Mandatory, &item.Level, &item.Imported, &item.ProjectActive); err != nil {
			return out, err
		}
		item.Available = out.Enabled && item.Active && (item.Default || item.Mandatory || (item.Imported && item.ProjectActive))
		out.Types = append(out.Types, item)
	}
	return out, rows.Err()
}
