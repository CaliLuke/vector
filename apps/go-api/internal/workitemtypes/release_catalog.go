package workitemtypes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type ReleaseValues struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetDate  string `json:"target_date"`
}
type Release struct {
	ReleaseValues
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}
type ReleaseCatalog struct {
	Revision    int64     `json:"revision"`
	ProjectName string    `json:"project_name"`
	CanManage   bool      `json:"can_manage"`
	Releases    []Release `json:"releases"`
}
type ReleaseInput struct {
	Revision int64
	Action   string
	ID       string
	Values   ReleaseValues
}

func (r *Repository) ReleaseCatalog(ctx context.Context, actor, slug, project string) (ReleaseCatalog, error) {
	return readProjectSchemaCatalog(ctx, r, actor, slug, project, loadReleaseCatalog)
}
func loadReleaseCatalog(ctx context.Context, tx pgx.Tx, workspace, project string, admin bool) (ReleaseCatalog, error) {
	out := ReleaseCatalog{CanManage: admin, Releases: []Release{}}
	err := tx.QueryRow(ctx, `SELECT p.name,COALESCE(s.revision,0) FROM projects p LEFT JOIN work_item_type_settings s ON s.workspace_id=p.workspace_id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL`, project, workspace).Scan(&out.ProjectName, &out.Revision)
	if err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,name,description,COALESCE(target_date::text,''),created_at FROM work_item_releases WHERE workspace_id=$1 AND project_id=$2 ORDER BY name,id`, workspace, project)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Release
		var created time.Time
		if err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.TargetDate, &created); err != nil {
			return out, err
		}
		item.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out.Releases = append(out.Releases, item)
	}
	return out, rows.Err()
}
