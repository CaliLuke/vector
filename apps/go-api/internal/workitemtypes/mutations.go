package workitemtypes

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// Icon names are presentation identifiers from the client catalog, including legacy lowercase names.
var iconPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,63}$`)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validate(input Input) error {
	if name := strings.TrimSpace(input.Name); name == "" || utf8.RuneCountInString(name) > 255 {
		return reject(422, "Enter a type name between 1 and 255 characters.")
	}
	if utf8.RuneCountInString(input.Description) > 10000 {
		return reject(422, "The description must be 10,000 characters or fewer.")
	}
	if !iconPattern.MatchString(input.Icon) {
		return reject(422, "Choose a valid type icon.")
	}
	if !colorPattern.MatchString(input.Color) {
		return reject(422, "Choose a valid type color.")
	}
	return nil
}
func save(ctx context.Context, tx pgx.Tx, workspace, actor, action, id string, input Input, before Catalog) error {
	if err := validate(input); err != nil {
		return err
	}
	input.Name = strings.TrimSpace(input.Name)
	var existing *Type
	for i := range before.Types {
		item := &before.Types[i]
		if item.ID == id {
			existing = item
		}
		if item.ID != id && strings.EqualFold(strings.TrimSpace(item.Name), input.Name) {
			return reject(409, "A work item type with this name already exists.")
		}
	}
	if action == "update" && existing == nil {
		return reject(404, "Work item type not found.")
	}
	if existing != nil && existing.Default && (!input.Active || !input.Mandatory) {
		return reject(422, "The default type must stay active and mandatory.")
	}
	if input.Mandatory && !input.Active {
		return reject(422, "Mandatory types must stay active. Change rollout to Import before deactivating this type.")
	}
	logo, err := json.Marshal(map[string]string{"name": input.Icon, "color": input.Color})
	if err != nil {
		return err
	}
	if action == "create" {
		id = uuid.New().String()
		// Creation always begins inactive, at level zero and with explicit project import.
		_, err = tx.Exec(ctx, `INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,false,false,false,0,false,now(),now(),$6,$6)`, id, workspace, input.Name, input.Description, logo, actor)
	} else {
		_, err = tx.Exec(ctx, `UPDATE issue_types SET name=$3,description=$4,logo_props=$5,is_active=$6,mandatory=$7,updated_by_id=$8,updated_at=now() WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL`, workspace, id, input.Name, input.Description, logo, input.Active, input.Mandatory, actor)
	}
	if err != nil {
		return err
	}
	if action == "update" && input.Mandatory {
		return importEverywhere(ctx, tx, workspace, actor, id)
	}
	return nil
}
func importEverywhere(ctx context.Context, tx pgx.Tx, workspace, actor, id string) error {
	_, err := tx.Exec(ctx, `INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,level,is_default,is_active,created_at,updated_at,created_by_id,updated_by_id)
 SELECT gen_random_uuid(),p.workspace_id,p.id,t.id,t.level::integer,t.is_default,true,now(),now(),$3,$3 FROM projects p JOIN issue_types t ON t.id=$2 AND t.workspace_id=p.workspace_id WHERE p.workspace_id=$1 AND p.deleted_at IS NULL
 ON CONFLICT(project_id,issue_type_id) WHERE deleted_at IS NULL DO UPDATE SET is_active=true,is_default=excluded.is_default,updated_at=now(),updated_by_id=excluded.updated_by_id`, workspace, id, actor)
	return err
}
func enable(ctx context.Context, tx pgx.Tx, workspace, actor string, before Catalog) error {
	if before.Enabled {
		return nil
	}
	var defaultID string
	for _, item := range before.Types {
		if item.Default {
			if defaultID != "" {
				return reject(409, "This workspace has multiple default types. Resolve them before enabling work item types.")
			}
			defaultID = item.ID
		}
	}
	if defaultID == "" {
		// Adopt an existing Task rather than creating a duplicate definition.
		for _, item := range before.Types {
			if strings.EqualFold(strings.TrimSpace(item.Name), "Task") {
				if defaultID != "" {
					return reject(409, "This workspace has duplicate Task types. Resolve them before enabling work item types.")
				}
				defaultID = item.ID
			}
		}
	}
	if defaultID == "" {
		defaultID = uuid.New().String()
		if _, err := tx.Exec(ctx, `INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,'Task','A unit of work.','{"name":"task","color":"#8b5cf6"}',true,true,true,0,false,now(),now(),$3,$3)`, defaultID, workspace, actor); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE issue_types SET is_default=true,is_active=true,mandatory=true,updated_at=now(),updated_by_id=$3 WHERE workspace_id=$1 AND id=$2`, workspace, defaultID, actor); err != nil {
			return err
		}
	}
	if err := importEverywhere(ctx, tx, workspace, actor, defaultID); err != nil {
		return err
	}
	// Assign a type to published legacy work. Private drafts retain unfinished selection.
	if _, err := tx.Exec(ctx, `UPDATE issues SET type_id=$2 WHERE workspace_id=$1 AND type_id IS NULL`, workspace, defaultID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO work_item_type_settings(workspace_id,enabled) VALUES($1,true) ON CONFLICT(workspace_id) DO UPDATE SET enabled=true`, workspace)
	return err
}
func remove(ctx context.Context, tx pgx.Tx, workspace, actor, id string, before Catalog) error {
	var target *Type
	for i := range before.Types {
		if before.Types[i].ID == id {
			target = &before.Types[i]
		}
	}
	if target == nil {
		return reject(404, "Work item type not found.")
	}
	if target.Default {
		return reject(422, "The default type cannot be deleted.")
	}
	// Keep archived, deleted and draft work recoverable. Replacement migration is a separate operation.
	if target.Usage > 0 {
		return reject(409, "This type is used by work items or drafts. Reassign those items before deleting it.")
	}
	if _, err := tx.Exec(ctx, `UPDATE project_issue_types SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE workspace_id=$1 AND issue_type_id=$2 AND deleted_at IS NULL`, workspace, id, actor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM work_item_project_defaults WHERE workspace_id=$1 AND type_id=$2`, workspace, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE issue_types SET deleted_at=now(),updated_at=now(),updated_by_id=$3 WHERE workspace_id=$1 AND id=$2`, workspace, id, actor); err != nil {
		return err
	}
	if err := removeTypeConfiguration(ctx, tx, workspace, actor, id); err != nil {
		return err
	}
	return compactTypeRemovalHierarchy(ctx, tx, workspace, actor)
}
