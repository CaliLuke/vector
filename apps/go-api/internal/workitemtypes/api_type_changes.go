package workitemtypes

import (
	"context"
	"strings"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func saveAPIType(ctx context.Context, tx pgx.Tx, in APIMutation, before Catalog) (uuid.UUID, error) {
	input := Input{Icon: "task", Color: "#8b5cf6"}
	if in.Action == "update" {
		found := false
		for _, item := range before.Types {
			if item.ID == in.TypeID.String() {
				input = Input{Name: item.Name, Description: item.Description, Icon: item.Icon, Color: item.Color, Active: item.Active, Mandatory: item.Mandatory}
				found = true
				break
			}
		}
		if !found {
			return uuid.Nil, projectmutations.ErrNotFound
		}
	}
	if in.Changes.Name != nil {
		input.Name = *in.Changes.Name
	}
	if in.Changes.Description != nil {
		input.Description = *in.Changes.Description
	}
	if in.Changes.Active != nil {
		input.Active = *in.Changes.Active
	}
	workspace, actor := in.WorkspaceID.String(), in.ActorID.String()
	if err := save(ctx, tx, workspace, actor, in.Action, in.TypeID.String(), input, before); err != nil {
		return uuid.Nil, err
	}
	id := in.TypeID
	if in.Action == "create" {
		if err := tx.QueryRow(ctx, `SELECT id FROM issue_types WHERE workspace_id=$1 AND name=$2 AND deleted_at IS NULL`, in.WorkspaceID, strings.TrimSpace(input.Name)).Scan(&id); err != nil {
			return uuid.Nil, err
		}
		// Native creation starts inactive. Explicit activation is a second native
		// operation in this same transaction, preserving its validation and rollout.
		if input.Active {
			current, err := loadCatalog(ctx, tx, workspace, true)
			if err != nil {
				return uuid.Nil, err
			}
			if err = save(ctx, tx, workspace, actor, "update", id.String(), input, current); err != nil {
				return uuid.Nil, err
			}
		}
	}
	if in.Changes.ExternalSource != nil || in.Changes.ExternalID != nil {
		_, err := tx.Exec(ctx, `UPDATE issue_types SET external_source=COALESCE($3,external_source),external_id=COALESCE($4,external_id),updated_at=clock_timestamp(),updated_by_id=$5 WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, in.WorkspaceID, in.Changes.ExternalSource, in.Changes.ExternalID, in.ActorID)
		if err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil
}
func applyAPITypeResolution(ctx context.Context, tx pgx.Tx, in APIMutation, plan apiTypeResolution, before Catalog) (uuid.UUID, error) {
	id := plan.id
	if plan.change != "" {
		active := true
		change := APIMutation{APIScope: in.APIScope, Action: plan.change, TypeID: id, Changes: APIChanges{Name: in.Changes.Name, Active: &active}}
		var err error
		id, err = saveAPIType(ctx, tx, change, before)
		if err != nil {
			return id, err
		}
	}
	if plan.adopt {
		return id, adoptAPIType(ctx, tx, in.APIScope, id)
	}
	return id, nil
}

func adoptAPIType(ctx context.Context, tx pgx.Tx, scope APIScope, id uuid.UUID) error {
	catalog, err := loadProjectCatalog(ctx, tx, scope.WorkspaceID.String(), scope.ProjectID.String(), true)
	if err != nil {
		return err
	}
	input := ProjectInput{Action: "import", TypeIDs: []string{id.String()}}
	for _, item := range catalog.Types {
		if item.ID == id.String() {
			if item.Available {
				return nil
			}
			if item.Imported {
				input.Action = "resume"
			}
			break
		}
	}
	if err = validateProjectMutation(catalog, input); err != nil {
		return err
	}
	return changeProjectType(ctx, tx, scope.WorkspaceID.String(), scope.ProjectID.String(), scope.ActorID.String(), id.String(), input.Action)
}
func replaceAPITypeProjects(ctx context.Context, tx pgx.Tx, scope APIScope, id uuid.UUID, wanted []uuid.UUID) error {
	existing := map[uuid.UUID]bool{}
	rows, err := tx.Query(ctx, `SELECT link.project_id FROM project_issue_types link JOIN projects p ON p.id=link.project_id AND p.workspace_id=link.workspace_id WHERE link.workspace_id=$1 AND link.issue_type_id=$2 AND link.deleted_at IS NULL AND p.deleted_at IS NULL`, scope.WorkspaceID, id)
	if err != nil {
		return err
	}
	for rows.Next() {
		var project uuid.UUID
		if err = rows.Scan(&project); err != nil {
			rows.Close()
			return err
		}
		existing[project] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	desired := map[uuid.UUID]bool{}
	all := append([]uuid.UUID{}, wanted...)
	for _, project := range wanted {
		desired[project] = true
	}
	for project := range existing {
		all = append(all, project)
	}
	for _, project := range canonicalAPIIDs(all) {
		if desired[project] && existing[project] {
			continue
		}
		currentScope := scope
		currentScope.ProjectID = project
		access, err := authorizeAPI(ctx, tx, currentScope, true)
		if err != nil {
			return err
		}
		if !access.projectAdmin {
			return projectmutations.ErrForbidden
		}
		if desired[project] {
			if err = adoptAPIType(ctx, tx, currentScope, id); err != nil {
				return err
			}
			continue
		}
		catalog, err := loadProjectCatalog(ctx, tx, scope.WorkspaceID.String(), project.String(), true)
		if err != nil {
			return err
		}
		input := ProjectInput{Action: "remove", TypeIDs: []string{id.String()}}
		if err = validateProjectMutation(catalog, input); err != nil {
			return err
		}
		if err = changeProjectType(ctx, tx, scope.WorkspaceID.String(), project.String(), scope.ActorID.String(), id.String(), "remove"); err != nil {
			return err
		}
	}
	return nil
}
