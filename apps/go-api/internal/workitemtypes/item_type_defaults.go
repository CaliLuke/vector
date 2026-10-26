package workitemtypes

import (
	"context"
	"errors"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

type itemTypeChoices struct {
	IDs       []string
	Suggested string
	Reason    string
}

// Explicit standalone choices may use any level. Only suggestions use the
// workspace default level; children always use the parent's adjacent level.
func availableItemTypes(ctx context.Context, tx pgx.Tx, in ItemValueInput) (itemTypeChoices, error) {
	out := itemTypeChoices{IDs: []string{}}
	rows, err := tx.Query(ctx, `SELECT t.id::text,t.is_default,COALESCE(preference.type_id=t.id,false),
 NOT COALESCE(settings.hierarchy_enabled,false) OR $3<>'' OR t.level=settings.default_level AS suggested_level
 FROM issue_types t
 LEFT JOIN work_item_project_defaults preference ON preference.project_id=NULLIF($2,'')::uuid AND preference.workspace_id=t.workspace_id
 LEFT JOIN work_item_type_settings settings ON settings.workspace_id=t.workspace_id
 WHERE t.workspace_id=$1 AND t.deleted_at IS NULL AND t.is_active
 AND ($2='' OR t.is_default OR t.mandatory OR EXISTS(SELECT 1 FROM project_issue_types imported WHERE imported.project_id=NULLIF($2,'')::uuid AND imported.workspace_id=t.workspace_id AND imported.issue_type_id=t.id AND imported.deleted_at IS NULL AND imported.is_active))
 AND (NOT COALESCE(settings.hierarchy_enabled,false) OR $3='' OR t.level=(SELECT COALESCE(parent_type.level,0)-1 FROM issues parent LEFT JOIN issue_types parent_type ON parent_type.id=parent.type_id AND parent_type.workspace_id=parent.workspace_id WHERE parent.id=NULLIF($3,'')::uuid AND parent.workspace_id=t.workspace_id))
 ORDER BY t.id`, in.WorkspaceID, in.ProjectID, in.ParentID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	candidates := []string{}
	task, preferred := "", ""
	taskCount := 0
	for rows.Next() {
		var id string
		var protected, projectDefault, suggestedLevel bool
		if err = rows.Scan(&id, &protected, &projectDefault, &suggestedLevel); err != nil {
			return out, err
		}
		out.IDs = append(out.IDs, id)
		if !suggestedLevel {
			continue
		}
		candidates = append(candidates, id)
		if protected {
			task = id
			taskCount++
		}
		if projectDefault {
			preferred = id
		}
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	switch {
	case preferred != "":
		out.Suggested = preferred
	case taskCount == 1:
		out.Suggested = task
	case len(candidates) == 1:
		out.Suggested = candidates[0]
	case len(candidates) == 0:
		out.Reason = "No available type matches the suggested hierarchy level. Choose a type or update the project’s types."
	default:
		out.Reason = "Choose a work item type. More than one type matches the suggested hierarchy level."
	}
	if in.ParentID != "" && len(out.IDs) == 0 {
		out.Reason = "This parent has no available child type. Choose another parent or update the project’s types."
	}
	return out, nil
}

// Omission preserves an existing parent, including one the actor can no longer
// see. A newly selected parent must be readable; its identity never leaves here.
func schemaParent(ctx context.Context, tx pgx.Tx, workspace, actor string, input ItemSchemaInput, identity schemaIdentity) (string, error) {
	if input.ParentID == nil {
		return identity.ParentID, nil
	}
	id := strings.ToLower(*input.ParentID)
	if id == "" {
		return "", nil
	}
	if _, err := uuid.Parse(id); err != nil {
		return "", reject(422, "Choose an available parent work item.")
	}
	var visible bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i WHERE i.id=$1 AND i.workspace_id=$2 AND i.deleted_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$2::uuid")+`)`, id, workspace, actor).Scan(&visible)
	if err != nil {
		return "", err
	}
	if !visible {
		return "", reject(422, "Choose an available parent work item.")
	}
	return id, nil
}

func resolveSchemaType(ctx context.Context, tx pgx.Tx, in ItemValueInput, out *ItemSchema) error {
	choices, err := availableItemTypes(ctx, tx, in)
	if err != nil {
		return err
	}
	out.EligibleTypeIDs = choices.IDs
	selected := in.TypeID
	if selected == "" && !in.Existing && in.ProjectID != "" {
		selected = choices.Suggested
	}
	if selected == "" {
		out.SelectionReason = choices.Reason
		if out.SelectionReason == "" {
			out.SelectionReason = "Choose a work item type."
		}
		return nil
	}
	in.TypeID = selected
	out.TypeID, err = resolveItemType(ctx, tx, in)
	var invalid *Error
	if errors.As(err, &invalid) && invalid.Status == 422 {
		out.SelectionReason = invalid.Message
		return nil
	}
	return err
}

// A missing selection is valid for a schema/draft, but never for publication.
func suggestedItemType(ctx context.Context, tx pgx.Tx, in ItemValueInput) (string, error) {
	if in.Existing || in.TypeSpecified || !in.Creating || in.ProjectID == "" {
		return "", nil
	}
	choices, err := availableItemTypes(ctx, tx, in)
	if err != nil {
		return "", err
	}
	return choices.Suggested, nil
}
