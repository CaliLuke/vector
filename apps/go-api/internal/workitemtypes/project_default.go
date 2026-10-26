package workitemtypes

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func validateProjectDefault(catalog ProjectCatalog, ids []string) error {
	if len(ids) > 1 {
		return reject(422, "Choose one default type, or use automatic selection.")
	}
	if len(ids) == 0 {
		return nil
	}
	for _, item := range catalog.Types {
		if item.ID == ids[0] {
			if !item.Available {
				return reject(422, "Choose an available type as the project default.")
			}
			return nil
		}
	}
	return reject(404, "Work item type not found in this workspace.")
}

func changeProjectDefault(ctx context.Context, tx pgx.Tx, workspace, project string, ids []string) error {
	if len(ids) == 0 {
		_, err := tx.Exec(ctx, `DELETE FROM work_item_project_defaults WHERE project_id=$1 AND workspace_id=$2`, project, workspace)
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO work_item_project_defaults(project_id,workspace_id,type_id) VALUES($1,$2,$3) ON CONFLICT(project_id) DO UPDATE SET type_id=EXCLUDED.type_id WHERE work_item_project_defaults.workspace_id=EXCLUDED.workspace_id`, project, workspace, ids[0])
	return err
}
