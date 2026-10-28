package workspaceprojectreads

import (
	"context"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
)

var projectRelationFields = []string{"project_lead", "default_assignee", "default_state", "estimate"}

func (r *LifecycleRepository) validateProjectReferences(ctx context.Context, v *projectFieldValidation, app, create bool) error {
	for _, key := range projectRelationFields {
		if create && (key == "default_state" || key == "estimate") && !app {
			continue
		}
		query := `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`
		switch key {
		case "default_state":
			query = `SELECT EXISTS(SELECT 1 FROM states WHERE id=$1 AND deleted_at IS NULL AND "group"<>'triage')`
		case "estimate":
			query = `SELECT EXISTS(SELECT 1 FROM estimates WHERE id=$1 AND deleted_at IS NULL)`
		}
		if err := r.validateProjectReference(ctx, v, key, query); err != nil {
			return err
		}
	}
	return nil
}

// validateProjectReference checks syntax and existence before scope validation.
// Nullable references retain a supplied null; omitted or invalid references add no value.
func (r *LifecycleRepository) validateProjectReference(ctx context.Context, v *projectFieldValidation, key, query string) error {
	raw, present := v.input.FieldValue(key, false, false, true)
	if !present {
		return nil
	}
	if raw == nil || raw == "" {
		v.values[key+"_id"] = nil
		return nil
	}
	if _, boolean := raw.(bool); boolean {
		v.failures[key] = []string{"Incorrect type. Expected pk value, received bool."}
		return nil
	}
	id, err := djangoinput.UUID(raw)
	if err != nil {
		value, stringErr := projectFieldErrorString(v.input, key, raw)
		if stringErr != nil {
			return stringErr
		}
		v.failures[key] = []string{"“" + value + "” is not a valid UUID."}
		return nil
	}
	var exists bool
	if err := r.writer.QueryRow(ctx, query, id.String()).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		value, stringErr := projectFieldErrorString(v.input, key, raw)
		if stringErr != nil {
			return stringErr
		}
		v.failures[key] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", value)}
		return nil
	}
	v.values[key+"_id"] = id.String()
	return nil
}

func (r *LifecycleRepository) validateProjectCover(ctx context.Context, workspace, project string, v *projectFieldValidation, create bool) error {
	if err := r.validateProjectReference(ctx, v, "cover_image_asset", `SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1 AND deleted_at IS NULL)`); err != nil {
		return err
	}
	value, present := v.values["cover_image_asset_id"].(string)
	if !present {
		return nil
	}
	allowed := false
	if !create {
		var err error
		allowed, err = r.coverAssets(ctx, uuid.MustParse(workspace), uuid.MustParse(project), uuid.MustParse(value))
		if err != nil {
			return err
		}
	}
	if !allowed {
		delete(v.values, "cover_image_asset_id")
		v.failures["cover_image_asset"] = []string{"The provided asset is not a cover image for this project."}
	}
	return nil
}

func (r *LifecycleRepository) validateProjectNames(ctx context.Context, workspace, project string, v *projectFieldValidation) error {
	for _, key := range []string{"name", "identifier"} {
		value, ok := v.values[key].(string)
		if !ok {
			continue
		}
		upper := strings.ToUpper(key)
		if forbiddenProjectCharacters.MatchString(value) {
			v.failures[key] = []string{"PROJECT_" + upper + "_CANNOT_CONTAIN_SPECIAL_CHARACTERS"}
			continue
		}
		var duplicate bool
		if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE workspace_id=$1 AND `+key+`=$2 AND deleted_at IS NULL AND ($3='' OR id::text<>$3))`, workspace, value, project).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate {
			v.failures[key] = []string{"PROJECT_" + upper + "_ALREADY_EXIST"}
		}
	}
	return nil
}

func (v *projectFieldValidation) apiNames() error {
	for _, key := range []string{"name", "identifier"} {
		if value, ok := v.values[key].(string); ok && forbiddenProjectCharacters.MatchString(value) {
			label := "Project name"
			if key == "identifier" {
				label = "Project identifier"
			}
			return projectFailure(400, map[string]any{"non_field_errors": []string{label + " cannot contain special characters."}})
		}
	}
	return nil
}

func (r *LifecycleRepository) validateProjectRelationScopes(ctx context.Context, workspace, project string, values map[string]any, app bool) error {
	failures := map[string]any{}
	for _, key := range projectRelationFields {
		value, present := values[key+"_id"]
		if !present || value == nil {
			continue
		}
		query := `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active=true AND deleted_at IS NULL)`
		scope := workspace
		switch key {
		case "default_state":
			query = `SELECT EXISTS(SELECT 1 FROM states WHERE project_id::text=$1 AND id=$2 AND deleted_at IS NULL AND "group"<>'triage')`
			scope = project
		case "estimate":
			query = `SELECT EXISTS(SELECT 1 FROM estimates WHERE project_id::text=$1 AND id=$2 AND deleted_at IS NULL)`
			scope = project
		}
		var allowed bool
		if err := r.writer.QueryRow(ctx, query, scope, value).Scan(&allowed); err != nil {
			return err
		}
		if allowed {
			continue
		}
		failures[key] = []string{projectRelationScopeMessage(key, app)}
		if !app {
			return projectFailure(400, failures)
		}
	}
	if len(failures) > 0 {
		return projectFailure(400, failures)
	}
	return nil
}

func projectRelationScopeMessage(key string, app bool) string {
	switch key {
	case "default_state":
		return "The provided state does not belong to this project."
	case "estimate":
		return "The provided estimate does not belong to this project."
	default:
		if app {
			return "The provided user is not an active member of this workspace."
		}
		return "The provided user is not a member of this workspace."
	}
}
