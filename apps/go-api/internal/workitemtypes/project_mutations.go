package workitemtypes

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) MutateProject(ctx context.Context, actor, slug, project string, input ProjectInput) (ProjectCatalog, error) {
	return withProjectSchemaWrite(ctx, r, actor, slug, project, func(tx pgx.Tx, workspace string, admin bool) (ProjectCatalog, error) {
		before, err := loadProjectCatalog(ctx, tx, workspace, project, admin)
		if err != nil {
			return before, err
		}
		if !before.Enabled {
			return before, reject(409, "Enable work item types in workspace settings first.")
		}
		if before.Revision != input.Revision {
			return before, reject(409, "Work item types changed. Review the refreshed catalog and try again.")
		}
		if err = validateProjectMutation(before, input); err != nil {
			return before, err
		}
		if input.Action == "set_default" {
			if err = changeProjectDefault(ctx, tx, workspace, project, input.TypeIDs); err != nil {
				return ProjectCatalog{}, err
			}
		}
		if input.Action != "set_default" {
			for _, id := range input.TypeIDs {
				if err = changeProjectType(ctx, tx, workspace, project, actor, id, input.Action); err != nil {
					return ProjectCatalog{}, err
				}
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, workspace); err != nil {
			return ProjectCatalog{}, err
		}
		out, err := loadProjectCatalog(ctx, tx, workspace, project, admin)
		if err != nil {
			return out, err
		}
		return out, nil
	})
}

func validateProjectMutation(catalog ProjectCatalog, input ProjectInput) error {
	if input.Action == "set_default" {
		return validateProjectDefault(catalog, input.TypeIDs)
	}
	switch input.Action {
	case "import", "pause", "resume", "remove":
	default:
		return reject(422, "Unknown project type action.")
	}
	if len(input.TypeIDs) == 0 || len(input.TypeIDs) > 100 || (input.Action != "import" && len(input.TypeIDs) != 1) {
		return reject(422, "Choose up to 100 types to import, or one imported type to change.")
	}
	types := make(map[string]ProjectType, len(catalog.Types))
	for _, item := range catalog.Types {
		types[item.ID] = item
	}
	seen := make(map[string]bool, len(input.TypeIDs))
	for _, id := range input.TypeIDs {
		item, found := types[id]
		if !found {
			return reject(404, "Work item type not found in this workspace.")
		}
		if seen[id] {
			return reject(422, "Choose each type only once.")
		}
		seen[id] = true
		if item.Default || item.Mandatory {
			return reject(422, "Default and mandatory types are managed by the workspace.")
		}
		if input.Action == "import" && item.Imported {
			return reject(409, "This type is already imported. Resume it from project settings.")
		}
		if input.Action != "import" && !item.Imported {
			return reject(404, "This type has not been imported into the project.")
		}
		if (input.Action == "import" || input.Action == "resume") && !item.Active {
			return reject(422, "Activate this type in the workspace library first.")
		}
	}
	return nil
}

func changeProjectType(ctx context.Context, tx pgx.Tx, workspace, project, actor, id, action string) error {
	if action == "import" {
		_, err := tx.Exec(ctx, `INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,level,is_default,is_active,created_at,updated_at,created_by_id,updated_by_id)
 SELECT gen_random_uuid(),$1,$2,t.id,t.level::integer,false,true,now(),now(),$3,$3 FROM issue_types t WHERE t.workspace_id=$1 AND t.id=$4 AND t.deleted_at IS NULL`, workspace, project, actor, id)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE project_issue_types SET is_active=$5,deleted_at=CASE WHEN $6 THEN now() ELSE NULL END,updated_at=now(),updated_by_id=$3
 WHERE workspace_id=$1 AND project_id=$2 AND issue_type_id=$4 AND deleted_at IS NULL`, workspace, project, actor, id, action == "resume", action == "remove")
	return err
}
