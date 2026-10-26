package workitemtypes

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) MutateReleaseCatalog(ctx context.Context, actor, slug, project string, input ReleaseInput) (ReleaseCatalog, error) {
	return withProjectSchemaWrite(ctx, r, actor, slug, project, func(tx pgx.Tx, workspace string, admin bool) (ReleaseCatalog, error) {
		before, err := loadReleaseCatalog(ctx, tx, workspace, project, admin)
		if err != nil {
			return before, err
		}
		if err = validateReleaseInput(&input); err != nil {
			return before, err
		}
		// Repeating an accepted create with the same stable client ID and values is a
		// no-op. Never infer a successful retry from a matching name alone.
		for _, item := range before.Releases {
			if input.Action == "create" && item.ID == input.ID && item.ReleaseValues == input.Values {
				return before, nil
			}
		}
		if before.Revision != input.Revision {
			return before, reject(409, "The release catalog changed. Reload current values before saving.")
		}
		if err = applyReleaseMutation(ctx, tx, workspace, project, input, before); err != nil {
			return ReleaseCatalog{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO work_item_type_settings(workspace_id,revision) VALUES($1,1) ON CONFLICT(workspace_id) DO UPDATE SET revision=work_item_type_settings.revision+1`, workspace); err != nil {
			return ReleaseCatalog{}, err
		}
		return loadReleaseCatalog(ctx, tx, workspace, project, admin)
	})
}

func validateReleaseInput(input *ReleaseInput) error {
	if input.Action != "create" && input.Action != "update" && input.Action != "delete" {
		return reject(422, "Choose a valid release action.")
	}
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return reject(422, "Choose a valid release.")
	}
	input.ID = id.String()
	if input.Revision < 0 {
		return reject(422, "Choose a valid catalog revision.")
	}
	if input.Action == "delete" {
		return nil
	}
	input.Values.Name = strings.TrimSpace(input.Values.Name)
	if input.Values.Name == "" || utf8.RuneCountInString(input.Values.Name) > 255 || strings.ContainsFunc(input.Values.Name, unicode.IsControl) {
		return reject(422, "Enter a release name between 1 and 255 characters without control characters.")
	}
	if utf8.RuneCountInString(input.Values.Description) > 10000 || strings.ContainsRune(input.Values.Description, 0) {
		return reject(422, "The description must be 10,000 characters or fewer and cannot contain null characters.")
	}
	if input.Values.TargetDate != "" {
		date, err := time.Parse(time.DateOnly, input.Values.TargetDate)
		if err != nil || date.Year() < 1 || date.Format(time.DateOnly) != input.Values.TargetDate {
			return reject(422, "Enter a valid target date in YYYY-MM-DD format.")
		}
	}
	return nil
}
func applyReleaseMutation(ctx context.Context, tx pgx.Tx, workspace, project string, input ReleaseInput, catalog ReleaseCatalog) error {
	found := false
	for _, item := range catalog.Releases {
		if item.ID == input.ID {
			found = true
		}
		if input.Action != "delete" && item.ID != input.ID && item.Name == input.Values.Name {
			return reject(422, "A release with this name already exists in the project.")
		}
	}
	if input.Action == "create" {
		if found {
			return reject(409, "This release already exists with different values. Reload the catalog.")
		}
		_, err := tx.Exec(ctx, `INSERT INTO work_item_releases(id,workspace_id,project_id,name,description,target_date) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::date)`, input.ID, workspace, project, input.Values.Name, input.Values.Description, input.Values.TargetDate)
		var constraint *pgconn.PgError
		if errors.As(err, &constraint) && constraint.Code == "23505" {
			return reject(409, "This release could not be created. Reload the catalog before trying again.")
		}
		return err
	}
	if !found {
		return reject(404, "Release not found in this project.")
	}
	if input.Action == "delete" {
		return deleteUnusedRelease(ctx, tx, workspace, project, input.ID)
	}
	_, err := tx.Exec(ctx, `UPDATE work_item_releases SET name=$4,description=$5,target_date=NULLIF($6,'')::date WHERE id=$1 AND workspace_id=$2 AND project_id=$3`, input.ID, workspace, project, input.Values.Name, input.Values.Description, input.Values.TargetDate)
	return err
}
func deleteUnusedRelease(ctx context.Context, tx pgx.Tx, workspace, project, id string) error {
	var used bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM (SELECT custom_properties FROM issues WHERE workspace_id=$1 UNION ALL SELECT custom_properties FROM draft_issues WHERE workspace_id=$1) item
 JOIN work_item_properties property ON property.workspace_id=$1 AND property.definition->>'kind'='Release picker'
 WHERE item.custom_properties->property.id::text=to_jsonb($2::text)
 OR item.custom_properties->property.id::text @> jsonb_build_array($2::text))`, workspace, id).Scan(&used)
	if err != nil {
		return err
	}
	if used {
		return reject(422, "This release is still referenced by a work item or draft. Clear those references before deleting it.")
	}
	_, err = tx.Exec(ctx, `DELETE FROM work_item_releases WHERE id=$1 AND workspace_id=$2 AND project_id=$3`, id, workspace, project)
	return err
}
