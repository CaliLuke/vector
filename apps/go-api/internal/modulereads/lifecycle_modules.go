package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var moduleStatuses = map[string]struct{}{
	"backlog": {}, "planned": {}, "in-progress": {}, "paused": {}, "completed": {}, "cancelled": {},
}

// CheckModuleProject resolves the scoped project before serializer parsing,
// matching the app collection create view's project lookup order.
func (repository *LifecycleRepository) CheckModuleProject(ctx context.Context, slug, projectID string, apiV1 bool) error {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
		WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
	)`, projectID, slug).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check module project: %w", err)
	}
	if exists {
		return nil
	}
	if apiV1 {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
	}
	return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
}

// CheckModule resolves a write target before request-body validation, as DRF's
// update actions call get_object before constructing the serializer.
func (repository *LifecycleRepository) CheckModule(ctx context.Context, slug, projectID, moduleID string, apiV1, replace bool) error {
	current, err := repository.rawModule(ctx, slug, projectID, moduleID)
	if errors.Is(err, ErrNotFound) {
		switch {
		case apiV1:
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
		case replace:
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "No Module matches the given query."}}
		default:
			return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Module not found"}}
		}
	}
	if err != nil {
		return err
	}
	if !replace && current["archived_at"] != nil {
		message := "Archived module cannot be updated"
		if apiV1 {
			message = "Archived module cannot be edited"
		}
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": message}}
	}
	return nil
}

// ModuleExists reports whether the active-project detail queryset contains the
// requested module. OPTIONS uses this to omit serializer actions on missing
// detail resources without changing the source's 200 metadata response.
func (repository *LifecycleRepository) ModuleExists(ctx context.Context, slug, projectID, moduleID string) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM modules m
		JOIN projects p ON p.id=m.project_id AND p.workspace_id=m.workspace_id
		JOIN workspaces w ON w.id=m.workspace_id
		WHERE m.id=$1 AND p.id=$2 AND w.slug=$3 AND m.deleted_at IS NULL
	)`, moduleID, projectID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check module protocol resource: %w", err)
	}
	return exists, nil
}

// CreateModule implements the app ModuleWriteSerializer create path. The
// caller must authorize before parsing the body, matching allow_permission.
func (repository *LifecycleRepository) CreateModule(ctx context.Context, actor, slug, projectID string, fields map[string]any, surrogates map[string][]string, origin string) (json.RawMessage, error) {
	if origin == "" {
		origin = repository.baseHost
	}
	var workspaceID string
	if err := repository.database.QueryRow(ctx, `SELECT p.workspace_id FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, projectID, slug).Scan(&workspaceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		return nil, fmt.Errorf("load module project: %w", err)
	}
	values, fieldErrors := validateModuleWriteFields(fields, surrogates, false)
	relationErrors, err := repository.validateModuleRelations(ctx, fields, []string{"lead", "lead_id"}, "member_ids")
	if err != nil {
		return nil, err
	}
	mergeModuleValidationErrors(fieldErrors, relationErrors)
	if len(fieldErrors) != 0 {
		return nil, &ModuleValidationError{Body: fieldErrors}
	}
	applyModuleCreateDefaults(values)
	if dateError := moduleDateOrderError(values); dateError != nil {
		return nil, &ModuleValidationError{Body: dateError}
	}
	name, _ := values["name"].(string)
	var duplicate bool
	if scanErr := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM modules WHERE project_id=$1 AND name=$2 AND deleted_at IS NULL)`, projectID, name).Scan(&duplicate); scanErr != nil {
		return nil, fmt.Errorf("check module name: %w", scanErr)
	}
	if duplicate {
		return nil, &ModuleValidationError{Body: map[string]any{"error": "Module with this name already exists"}}
	}
	var minimum pgtype.Float8
	if scanErr := repository.database.QueryRow(ctx, `SELECT MIN(sort_order)-10000 FROM modules WHERE project_id=$1 AND deleted_at IS NULL`, projectID).Scan(&minimum); scanErr != nil {
		return nil, fmt.Errorf("calculate module sort order: %w", scanErr)
	}
	sortOrder, _ := values["sort_order"].(float64)
	if minimum.Valid {
		sortOrder = minimum.Float64
	}
	moduleID, err := repository.insertModule(ctx, actor, workspaceID, projectID, values, sortOrder)
	if err != nil {
		return nil, err
	}
	if members, present := values["member_ids"]; present {
		if replaceModuleMembersErr := repository.replaceModuleMembers(ctx, workspaceID, projectID, moduleID, members.([]string)); replaceModuleMembersErr != nil {
			return nil, replaceModuleMembersErr
		}
	}
	result, err := repository.moduleMutationProjection(ctx, actor, slug, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{
		"model_name": "module", "model_id": moduleID, "requested_data": fields,
		"current_instance": nil, "actor_id": actor, "slug": slug, "origin": origin,
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateModuleV1 keeps the API serializer contract separate from the app
// ModuleWriteSerializer: API v1 accepts lead/members, gates on module_view,
// checks external identifiers, and returns the API ModuleSerializer shape.
func (repository *LifecycleRepository) CreateModuleV1(ctx context.Context, actor, slug, projectID string, fields map[string]any, surrogates map[string][]string, origin string) (json.RawMessage, error) {
	if origin == "" {
		origin = repository.baseHost
	}
	var workspaceID string
	var enabled bool
	err := repository.database.QueryRow(ctx, `SELECT p.workspace_id,p.module_view FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL`, projectID, slug).Scan(&workspaceID, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The requested resource does not exist."}}
		}
		return nil, fmt.Errorf("load API module project: %w", err)
	}
	values, validation := validateModuleAPIWriteFields(fields, surrogates, false)
	relationErrors, err := repository.validateModuleRelations(ctx, fields, []string{"lead"}, "members")
	if err != nil {
		return nil, err
	}
	mergeModuleValidationErrors(validation, remapModuleAPIValidationBody(relationErrors))
	if len(validation) > 0 {
		return nil, &ModuleValidationError{Body: validation}
	}
	applyModuleCreateDefaults(values)
	if !enabled {
		return nil, &ModuleValidationError{Body: map[string]any{"non_field_errors": []string{"Modules are not enabled for this project"}}}
	}
	if dateError := moduleDateOrderError(values); dateError != nil {
		return nil, &ModuleValidationError{Body: dateError}
	}
	if members, ok := values["member_ids"].([]string); ok {
		members, err = repository.projectModuleMembers(ctx, projectID, members)
		if err != nil {
			return nil, err
		}
		values["member_ids"] = members
	}
	externalID, hasExternalID := values["external_id"].(string)
	externalSource, hasExternalSource := values["external_source"].(string)
	if hasExternalID && hasExternalSource && externalID != "" && externalSource != "" {
		var duplicateID string
		scanErr := repository.database.QueryRow(ctx, `SELECT id::text FROM modules WHERE project_id=$1 AND workspace_id=$2 AND external_source=$3 AND external_id=$4 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`, projectID, workspaceID, externalSource, externalID).Scan(&duplicateID)
		if scanErr == nil {
			return nil, &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": "Module with the same external id and external source already exists", "id": duplicateID}}
		}
		if !errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check API module external identifier: %w", scanErr)
		}
	}
	name := values["name"].(string)
	var duplicateID string
	err = repository.database.QueryRow(ctx, `SELECT id::text FROM modules WHERE project_id=$1 AND name=$2 AND deleted_at IS NULL LIMIT 1`, projectID, name).Scan(&duplicateID)
	if err == nil {
		return nil, &ModuleValidationError{Body: map[string]any{"id": duplicateID, "code": "MODULE_NAME_ALREADY_EXISTS", "error": "Module with this name already exists", "message": "Module with this name already exists"}}
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check API module name: %w", err)
	}
	var minimum pgtype.Float8
	if scanErr := repository.database.QueryRow(ctx, `SELECT MIN(sort_order)-10000 FROM modules WHERE project_id=$1 AND deleted_at IS NULL`, projectID).Scan(&minimum); scanErr != nil {
		return nil, fmt.Errorf("calculate API module sort order: %w", scanErr)
	}
	sortOrder, _ := values["sort_order"].(float64)
	if minimum.Valid {
		sortOrder = minimum.Float64
	}
	id, err := repository.insertModule(ctx, actor, workspaceID, projectID, values, sortOrder)
	if err != nil {
		return nil, err
	}
	if members, ok := values["member_ids"].([]string); ok {
		if err = repository.replaceModuleMembers(ctx, workspaceID, projectID, id, members); err != nil {
			return nil, err
		}
	}
	result, err := repository.moduleWriteProjection(ctx, actor, slug, projectID, id)
	if err != nil {
		return nil, err
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "module", "model_id": id, "requested_data": fields, "current_instance": nil, "actor_id": actor, "slug": slug, "origin": origin}); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateModuleV1 follows ModuleDetailAPIEndpoint.patch and returns the narrow
// ModuleUpdateSerializer representation while recording the full current
// ModuleSerializer snapshot in the model-activity task.
func (repository *LifecycleRepository) UpdateModuleV1(ctx context.Context, actor, slug, projectID, moduleID string, fields map[string]any, surrogates map[string][]string, origin string) (json.RawMessage, error) {
	if origin == "" {
		origin = repository.baseHost
	}
	current, err := repository.rawModule(ctx, slug, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	if current["archived_at"] != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Archived module cannot be edited"}}
	}
	currentRaw, err := repository.moduleWriteProjection(ctx, actor, slug, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	currentValue, err := pythoncompat.DecodeObject(currentRaw)
	if err != nil {
		return nil, err
	}
	delete(currentValue, "member_ids")
	delete(currentValue, "lead_id")
	if moduleActivityErr := moduleActivityFloatFields(currentValue, "sort_order"); moduleActivityErr != nil {
		return nil, moduleActivityErr
	}
	currentJSONText, err := pythoncompat.Dumps(currentValue, currentRaw)
	if err != nil {
		return nil, err
	}
	currentJSON := []byte(currentJSONText)
	values, validation := validateModuleAPIWriteFields(fields, surrogates, true)
	relationErrors, err := repository.validateModuleRelations(ctx, fields, []string{"lead"}, "members")
	if err != nil {
		return nil, err
	}
	mergeModuleValidationErrors(validation, remapModuleAPIValidationBody(relationErrors))
	if len(validation) > 0 {
		return nil, &ModuleValidationError{Body: validation}
	}
	var enabled bool
	if scanErr := repository.database.QueryRow(ctx, `SELECT module_view FROM projects WHERE id=$1`, projectID).Scan(&enabled); scanErr != nil {
		return nil, fmt.Errorf("check API module feature: %w", scanErr)
	}
	if !enabled {
		return nil, &ModuleValidationError{Body: map[string]any{"non_field_errors": []string{"Modules are not enabled for this project"}}}
	}
	if dateError := moduleDateOrderError(values); dateError != nil {
		return nil, &ModuleValidationError{Body: dateError}
	}
	if members, present := values["member_ids"].([]string); present {
		members, err = repository.projectModuleMembers(ctx, projectID, members)
		if err != nil {
			return nil, err
		}
		values["member_ids"] = members
	}
	currentExternalID, _ := current["external_id"].(string)
	if externalID, present := values["external_id"].(string); present && externalID != "" && externalID != currentExternalID {
		externalSource, sourcePresent := values["external_source"]
		if !sourcePresent {
			externalSource = current["external_source"]
		}
		var duplicateID string
		scanErr := repository.database.QueryRow(ctx, `SELECT id::text FROM modules WHERE project_id=$1 AND workspace_id=$2 AND id<>$3 AND external_source IS NOT DISTINCT FROM $4 AND external_id=$5 AND deleted_at IS NULL LIMIT 1`, projectID, current["workspace_id"], moduleID, externalSource, externalID).Scan(&duplicateID)
		if scanErr == nil {
			return nil, &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": "Module with the same external id and external source already exists", "id": moduleID}}
		}
		if !errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check API module external identifier: %w", scanErr)
		}
	}
	if _, present := values["name"]; present {
		var duplicate bool
		if scanErr := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM modules WHERE project_id=$1 AND name=$2 AND id<>$3 AND deleted_at IS NULL)`, projectID, values["name"], moduleID).Scan(&duplicate); scanErr != nil {
			return nil, scanErr
		}
		if duplicate {
			return nil, &ModuleValidationError{Body: map[string]any{"error": "Module with this name already exists"}}
		}
	}
	if members, present := values["member_ids"]; present {
		if replaceModuleMembersErr := repository.replaceModuleMembers(ctx, current["workspace_id"].(string), projectID, moduleID, members.([]string)); replaceModuleMembersErr != nil {
			return nil, replaceModuleMembersErr
		}
	}
	if updateModuleFieldsErr := repository.updateModuleFields(ctx, actor, slug, projectID, moduleID, values); updateModuleFieldsErr != nil {
		return nil, updateModuleFieldsErr
	}
	result, err := repository.moduleWriteProjection(ctx, actor, slug, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	var representation map[string]any
	if err := json.Unmarshal(result, &representation); err != nil {
		return nil, err
	}
	selected := map[string]any{}
	for _, key := range []string{"name", "description", "start_date", "target_date", "status", "lead", "external_source", "external_id"} {
		if value, ok := representation[key]; ok {
			selected[key] = value
		}
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "module", "model_id": moduleID, "requested_data": fields, "current_instance": string(currentJSON), "actor_id": actor, "slug": slug, "origin": origin}); err != nil {
		return nil, err
	}
	return json.Marshal(selected)
}

func validateModuleAPIWriteFields(fields map[string]any, surrogates map[string][]string, partial bool) (map[string]any, map[string]any) {
	converted := make(map[string]any, len(fields))
	for key, value := range fields {
		switch key {
		case "lead":
			converted["lead_id"] = value
		case "members":
			converted["member_ids"] = value
		case "name", "description", "start_date", "target_date", "status", "external_source", "external_id":
			converted[key] = value
		}
	}
	values, failures := validateModuleWriteFields(converted, surrogates, partial)
	remapped := map[string]any{}
	for key, value := range failures {
		switch key {
		case "lead_id":
			remapped["lead"] = value
		case "member_ids":
			remapped["members"] = value
		default:
			remapped[key] = value
		}
	}
	return values, remapped
}
func remapModuleAPIValidationBody(body map[string]any) map[string]any {
	remapped := make(map[string]any, len(body))
	for key, value := range body {
		switch key {
		case "lead_id":
			key = "lead"
		case "member_ids":
			key = "members"
		}
		remapped[key] = value
	}
	return remapped
}

func mergeModuleValidationErrors(target, additional map[string]any) {
	for key, value := range additional {
		current, exists := target[key]
		if !exists {
			target[key] = value
			continue
		}
		currentFields, currentIsMap := current.(map[string]any)
		additionalFields, additionalIsMap := value.(map[string]any)
		if currentIsMap && additionalIsMap {
			for nestedKey, nestedValue := range additionalFields {
				if _, alreadyPresent := currentFields[nestedKey]; !alreadyPresent {
					currentFields[nestedKey] = nestedValue
				}
			}
		}
	}
}
func (repository *LifecycleRepository) projectModuleMembers(ctx context.Context, projectID string, ids []string) ([]string, error) {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		var exists bool
		if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND member_id=$2 AND deleted_at IS NULL)`, projectID, id).Scan(&exists); err != nil {
			return nil, fmt.Errorf("filter API module members: %w", err)
		}
		if exists {
			result = append(result, id)
		}
	}
	return result, nil
}

// UpdateModule performs the object lookup before serializer validation, as the
// viewset's update/partial_update actions do. replace selects PUT semantics.
func (repository *LifecycleRepository) UpdateModule(ctx context.Context, actor, slug, projectID, moduleID string, fields map[string]any, surrogates map[string][]string, replace bool, origin string) (json.RawMessage, error) {
	if origin == "" {
		origin = repository.baseHost
	}
	current, err := repository.rawModule(ctx, slug, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	if !replace && current["archived_at"] != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Archived module cannot be updated"}}
	}
	var currentProjection json.RawMessage
	if !replace {
		currentProjection, err = repository.moduleWriteSerializerProjection(ctx, actor, slug, projectID, moduleID)
		if err != nil {
			return nil, err
		}
	}
	values, fieldErrors := validateModuleWriteFields(fields, surrogates, !replace)
	relationErrors, err := repository.validateModuleRelations(ctx, fields, []string{"lead", "lead_id"}, "member_ids")
	if err != nil {
		return nil, err
	}
	mergeModuleValidationErrors(fieldErrors, relationErrors)
	if len(fieldErrors) != 0 {
		return nil, &ModuleValidationError{Body: fieldErrors}
	}
	if dateError := moduleDateOrderError(values); dateError != nil {
		return nil, &ModuleValidationError{Body: dateError}
	}
	if name, present := values["name"]; present {
		var duplicate bool
		if scanErr := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM modules WHERE project_id=$1 AND name=$2 AND id<>$3 AND deleted_at IS NULL)`, projectID, name, moduleID).Scan(&duplicate); scanErr != nil {
			return nil, fmt.Errorf("check module name: %w", scanErr)
		}
		if duplicate {
			return nil, &ModuleValidationError{Body: map[string]any{"error": "Module with this name already exists"}}
		}
	}
	if members, present := values["member_ids"]; present {
		if replaceModuleMembersErr := repository.replaceModuleMembers(ctx, current["workspace_id"].(string), projectID, moduleID, members.([]string)); replaceModuleMembersErr != nil {
			return nil, replaceModuleMembersErr
		}
	}
	if updateModuleFieldsErr := repository.updateModuleFields(ctx, actor, slug, projectID, moduleID, values); updateModuleFieldsErr != nil {
		return nil, updateModuleFieldsErr
	}
	var result json.RawMessage
	if replace {
		result, err = repository.moduleWriteProjection(ctx, actor, slug, projectID, moduleID)
	} else {
		result, err = repository.moduleMutationProjection(ctx, actor, slug, projectID, moduleID)
	}
	if err != nil {
		return nil, err
	}
	currentInstance := ""
	if !replace {
		currentValue, err := pythoncompat.DecodeObject(currentProjection)
		if err != nil {
			return nil, err
		}
		if moduleActivityErr := moduleActivityFloatFields(currentValue, "sort_order", "total_estimate_points", "completed_estimate_points"); moduleActivityErr != nil {
			return nil, moduleActivityErr
		}
		currentInstance, err = moduleActivitySnapshotJSON(currentValue, currentProjection)
		if err != nil {
			return nil, err
		}
	}
	if !replace {
		if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{
			"model_name": "module", "model_id": moduleID, "requested_data": fields,
			"current_instance": currentInstance, "actor_id": actor, "slug": slug, "origin": origin,
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// SetModuleArchived is shared by the app and API archive aliases. The app
// alias returns the archived timestamp; API v1 returns no body.
func (repository *LifecycleRepository) SetModuleArchived(ctx context.Context, actor, slug, projectID, moduleID string, archive bool) (time.Time, error) {
	current, err := repository.rawModule(ctx, slug, projectID, moduleID)
	if err != nil {
		return time.Time{}, err
	}
	if archive {
		status, _ := current["status"].(string)
		if status != "completed" && status != "cancelled" {
			return time.Time{}, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Only completed or cancelled modules can be archived"}}
		}
	}
	now := repository.now()
	var archivedAt *time.Time
	if archive {
		archivedAt = &now
	}
	var updated time.Time
	err = repository.database.QueryRow(ctx, `UPDATE modules SET archived_at=$4,updated_at=$5,updated_by_id=$6
		WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL
		RETURNING updated_at`, moduleID, projectID, slug, archivedAt, now, actor).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("set module archive state: %w", err)
	}
	if archive {
		if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$4
			WHERE entity_type='module' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL`, moduleID, projectID, slug, now); err != nil {
			return time.Time{}, fmt.Errorf("remove archived module favorites: %w", err)
		}
	}
	if !archive {
		return time.Time{}, nil
	}
	return now, nil
}

// CheckDeleteModule preserves the different authorization and lookup order of
// the app and API-v1 delete views before DeleteModule emits any effects.
func (repository *LifecycleRepository) CheckDeleteModule(ctx context.Context, actor, slug, projectID, moduleID string, api bool) error {
	if api {
		if err := repository.AuthorizeEntityProject(ctx, actor, slug, projectID); err != nil {
			return err
		}
		module, err := repository.rawModule(ctx, slug, projectID, moduleID)
		if err != nil {
			return err
		}
		if module["created_by"] == actor {
			return nil
		}
		var isProjectAdmin bool
		err = repository.database.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM project_members pm
			JOIN workspaces w ON w.id=pm.workspace_id
			WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
			AND pm.role=20 AND pm.is_active=true AND pm.deleted_at IS NULL
		)`, actor, projectID, slug).Scan(&isProjectAdmin)
		if err != nil {
			return fmt.Errorf("check API module delete permission: %w", err)
		}
		if !isProjectAdmin {
			return &djangoapi.HTTPError{Status: 403, Body: map[string]any{"error": "Only admin or creator can delete the module"}}
		}
		return nil
	}

	var isWorkspaceMember bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id
		WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL
	)`, actor, slug).Scan(&isWorkspaceMember)
	if err != nil {
		return fmt.Errorf("check app module delete workspace membership: %w", err)
	}
	if !isWorkspaceMember {
		return ErrForbidden
	}
	var isCreator bool
	err = repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM modules WHERE id=$1 AND created_by_id=$2 AND deleted_at IS NULL
	)`, moduleID, actor).Scan(&isCreator)
	if err != nil {
		return fmt.Errorf("check app module creator: %w", err)
	}
	if isCreator {
		_, rawModuleErr := repository.rawModule(ctx, slug, projectID, moduleID)
		return rawModuleErr
	}
	var isAdmin bool
	err = repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		AND pm.role=20 AND pm.is_active=true AND pm.deleted_at IS NULL
	) OR EXISTS (
		SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
		JOIN workspace_members wm ON wm.workspace_id=w.id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		AND pm.is_active=true AND pm.deleted_at IS NULL
		AND wm.member_id=$1 AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
	)`, actor, projectID, slug).Scan(&isAdmin)
	if err != nil {
		return fmt.Errorf("check app module delete admin permission: %w", err)
	}
	if !isAdmin {
		return ErrForbidden
	}
	_, err = repository.rawModule(ctx, slug, projectID, moduleID)
	return err
}

// DeleteModule reproduces the two source delete protocols: the app route emits
// one issue activity per linked issue and removes the requester's recent visit;
// API v1 emits one aggregate issue activity and keeps recent visits.
func (repository *LifecycleRepository) DeleteModule(ctx context.Context, actor, slug, projectID, moduleID string, api bool, origin string) error {
	if origin == "" {
		origin = repository.baseHost
	}
	module, err := repository.rawModule(ctx, slug, projectID, moduleID)
	if err != nil {
		return err
	}
	name, _ := module["name"].(string)
	var rawIssueIDs []byte
	err = repository.database.QueryRow(ctx, `SELECT COALESCE(json_agg(issue_id ORDER BY created_at DESC),'[]'::json) FROM module_issues WHERE module_id=$1 AND deleted_at IS NULL`, moduleID).Scan(&rawIssueIDs)
	if err != nil {
		return fmt.Errorf("list module issues for deletion: %w", err)
	}
	var issueIDs []string
	if unmarshalErr := json.Unmarshal(rawIssueIDs, &issueIDs); unmarshalErr != nil {
		return fmt.Errorf("decode module issue IDs: %w", unmarshalErr)
	}
	now := repository.now()
	current, err := pythoncompat.Dumps(map[string]any{"module_name": name}, nil)
	if err != nil {
		return err
	}
	if api {
		requested, dumpsErr := pythoncompat.Dumps(map[string]any{"module_id": moduleID, "module_name": name, "issues": issueIDs}, nil)
		if dumpsErr != nil {
			return dumpsErr
		}
		if taskErr := repository.effects.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
			"type": "module.activity.deleted", "requested_data": requested, "actor_id": actor,
			"issue_id": nil, "project_id": projectID, "current_instance": current, "epoch": now.Unix(), "origin": origin,
		}); taskErr != nil {
			return taskErr
		}
	} else {
		requested, dumpsErr := pythoncompat.Dumps(map[string]any{"module_id": moduleID}, nil)
		if dumpsErr != nil {
			return dumpsErr
		}
		for _, issueID := range issueIDs {
			if taskErr := repository.effects.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{
				"type": "module.activity.deleted", "requested_data": requested,
				"actor_id": actor, "issue_id": issueID, "project_id": projectID,
				"current_instance": current, "epoch": now.Unix(), "notification": true, "origin": origin,
			}); taskErr != nil {
				return taskErr
			}
		}
	}
	command, err := repository.database.Exec(ctx, `UPDATE modules SET deleted_at=$4,updated_at=$4,updated_by_id=$5
		WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL`, moduleID, projectID, slug, now, actor)
	if err != nil {
		return fmt.Errorf("soft-delete module: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := repository.effects.SoftDelete(ctx, "db", "module", moduleID); err != nil {
		return err
	}
	if _, err := repository.database.Exec(ctx, `UPDATE module_issues SET deleted_at=$3 WHERE module_id=$1 AND project_id=$2 AND deleted_at IS NULL`, moduleID, projectID, now); err != nil {
		return fmt.Errorf("soft-delete module issues: %w", err)
	}
	if api {
		if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$3 WHERE entity_type='module' AND entity_identifier=$1 AND project_id=$2 AND deleted_at IS NULL`, moduleID, projectID, now); err != nil {
			return fmt.Errorf("soft-delete module favorites: %w", err)
		}
		return nil
	}
	if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$4 WHERE user_id=$1 AND entity_type='module' AND entity_identifier=$2 AND project_id=$3 AND deleted_at IS NULL`, actor, moduleID, projectID, now); err != nil {
		return fmt.Errorf("soft-delete requester module favorites: %w", err)
	}
	if _, err := repository.database.Exec(ctx, `DELETE FROM user_recent_visits WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND entity_identifier=$3 AND entity_name='module'`, projectID, slug, moduleID); err != nil {
		return fmt.Errorf("delete module recent visits: %w", err)
	}
	return nil
}

func validateModuleWriteFields(fields map[string]any, surrogates map[string][]string, partial bool) (map[string]any, map[string]any) {
	values, failures := map[string]any{}, map[string]any{}
	text := func(key string, required, allowBlank bool, maximum int) {
		value, present := fields[key]
		if !present {
			if required {
				failures[key] = []string{"This field is required."}
			}
			return
		}
		if value == nil && !required && (key == "external_source" || key == "external_id") {
			values[key] = nil
			return
		}
		if value == nil {
			failures[key] = []string{"This field may not be null."}
			return
		}
		if value, messages := djangoinput.TextField(value, allowBlank, maximum, surrogates[key]); len(messages) > 0 {
			failures[key] = messages
		} else {
			values[key] = value
		}
	}
	text("name", !partial, false, 255)
	text("description", false, true, 0)
	text("external_source", false, true, 255)
	text("external_id", false, true, 255)
	if raw, present := fields["status"]; present {
		if value, ok := raw.(string); !ok {
			failures["status"] = []string{"Not a valid string."}
		} else if _, valid := moduleStatuses[value]; !valid {
			failures["status"] = []string{"\"" + value + "\" is not a valid choice."}
		} else {
			values["status"] = value
		}
	}
	for _, key := range []string{"start_date", "target_date"} {
		value, present := fields[key]
		if !present {
			continue
		}
		if value == nil {
			values[key] = nil
			continue
		}
		textValue, ok := value.(string)
		if !ok {
			failures[key] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
			continue
		}
		if _, err := time.Parse("2006-01-02", textValue); err != nil {
			failures[key] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
			continue
		}
		values[key] = textValue
	}
	// Both names are serializer fields targeting the same relation. Validate
	// each independently so a valid alias cannot hide an invalid supplied
	// field; source iteration order determines which valid value is applied.
	for _, key := range []string{"lead_id", "lead"} {
		raw, present := fields[key]
		if !present {
			continue
		}
		if raw == nil || raw == "" {
			values["lead_id"] = nil
		} else if id, err := djangoinput.UUID(raw); err != nil {
			failures[key] = []string{"Invalid pk \"" + djangoinput.ScalarString(raw) + "\" - object does not exist."}
		} else {
			values["lead_id"] = id.String()
		}
	}
	for _, key := range []string{"description_text", "description_html", "view_props", "logo_props"} {
		if raw, present := fields[key]; present {
			if key == "view_props" || key == "logo_props" {
				if raw == nil {
					failures[key] = []string{"This field may not be null."}
					continue
				}
			}
			values[key] = raw
		}
	}
	if raw, present := fields["sort_order"]; present {
		value, message := djangoinput.FloatField(raw)
		if message != "" {
			failures["sort_order"] = []string{message}
		} else {
			values["sort_order"] = value
		}
	}
	if raw, present := fields["member_ids"]; present {
		members, ok := raw.([]any)
		if !ok {
			if stringsValue, stringSlice := raw.([]string); stringSlice {
				members = make([]any, len(stringsValue))
				for i := range stringsValue {
					members[i] = stringsValue[i]
				}
			} else if raw == nil {
				failures["member_ids"] = []string{"This field may not be null."}
			} else {
				failures["member_ids"] = []string{"Expected a list of items but got type \"" + djangoinput.PythonType(raw) + "\"."}
			}
		}
		if members != nil {
			ids := make([]string, 0, len(members))
			memberErrors := map[string]any{}
			for index, member := range members {
				if member == nil || member == "" {
					memberErrors[strconv.Itoa(index)] = []string{"This field may not be null."}
					continue
				}
				if _, isBool := member.(bool); isBool {
					memberErrors[strconv.Itoa(index)] = []string{"Incorrect type. Expected pk value, received bool."}
					continue
				}
				switch member.(type) {
				case []any:
					memberErrors[strconv.Itoa(index)] = []string{"Incorrect type. Expected pk value, received list."}
					continue
				case map[string]any:
					memberErrors[strconv.Itoa(index)] = []string{"Incorrect type. Expected pk value, received dict."}
					continue
				}
				id, err := djangoinput.UUID(member)
				if err != nil {
					memberErrors[strconv.Itoa(index)] = []string{fmt.Sprintf("“%s” is not a valid UUID.", djangoinput.ScalarString(member))}
					continue
				}
				ids = append(ids, id.String())
			}
			if len(memberErrors) > 0 {
				failures["member_ids"] = memberErrors
			} else {
				values["member_ids"] = ids
			}
		}
	}
	return values, failures
}

func moduleDateOrderError(values map[string]any) map[string]any {
	start, startOK := values["start_date"].(string)
	target, targetOK := values["target_date"].(string)
	if startOK && targetOK && start > target {
		return map[string]any{"non_field_errors": []string{"Start date cannot exceed target date"}}
	}
	return nil
}

// applyModuleCreateDefaults matches Django model defaults. They belong on
// object creation only; PUT still validates required fields but must preserve
// omitted optional values on an existing instance.
func applyModuleCreateDefaults(values map[string]any) {
	if _, exists := values["status"]; !exists {
		values["status"] = "planned"
	}
	if _, exists := values["view_props"]; !exists {
		values["view_props"] = map[string]any{}
	}
	if _, exists := values["logo_props"]; !exists {
		values["logo_props"] = map[string]any{}
	}
	if _, exists := values["sort_order"]; !exists {
		values["sort_order"] = float64(65535)
	}
}

func (repository *LifecycleRepository) validateModuleRelations(ctx context.Context, fields map[string]any, leadFields []string, memberField string) (map[string]any, error) {
	fieldErrors := map[string]any{}
	checkedLeadIDs := map[string]bool{}
	for _, key := range leadFields {
		raw, present := fields[key]
		if !present || raw == nil || raw == "" {
			continue
		}
		id, err := djangoinput.UUID(raw)
		if err != nil {
			continue // Shape errors have already been collected by field conversion.
		}
		idText := id.String()
		exists, checked := checkedLeadIDs[idText]
		if !checked {
			if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, idText).Scan(&exists); err != nil {
				return nil, err
			}
			checkedLeadIDs[idText] = exists
		}
		if !exists {
			fieldErrors[key] = []string{"Invalid pk \"" + djangoinput.ScalarString(raw) + "\" - object does not exist."}
		}
	}
	if _, supplied := fields[memberField]; supplied {
		if members, ok := moduleMemberList(fields[memberField]); ok {
			memberErrors := map[string]any{}
			for index, member := range members {
				if member == nil || member == "" {
					continue
				}
				if _, isBool := member.(bool); isBool {
					continue
				}
				switch member.(type) {
				case []any, map[string]any:
					continue
				}
				id, err := djangoinput.UUID(member)
				if err != nil {
					continue // The field converter already records this child's shape error.
				}
				idText := id.String()
				var exists bool
				if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, idText).Scan(&exists); err != nil {
					return nil, err
				}
				if !exists {
					memberErrors[strconv.Itoa(index)] = []string{"Invalid pk \"" + djangoinput.ScalarString(member) + "\" - object does not exist."}
				}
			}
			if len(memberErrors) > 0 {
				fieldErrors[memberField] = memberErrors
			}
		}
	}
	return fieldErrors, nil
}

func moduleMemberList(raw any) ([]any, bool) {
	switch members := raw.(type) {
	case []any:
		return members, true
	case []string:
		values := make([]any, len(members))
		for index := range members {
			values[index] = members[index]
		}
		return values, true
	default:
		return nil, false
	}
}

func (repository *LifecycleRepository) insertModule(ctx context.Context, actor, workspaceID, projectID string, values map[string]any, sortOrder float64) (string, error) {
	newID := uuid.NewString()
	name := values["name"]
	description := values["description"]
	if description == nil {
		description = ""
	}
	args := []any{
		newID, workspaceID, projectID, name, description,
		moduleSQLJSON(values["description_text"]), moduleSQLJSON(values["description_html"]), values["start_date"], values["target_date"],
		values["status"], values["lead_id"], moduleSQLJSON(values["view_props"]), sortOrder,
		values["external_source"], values["external_id"], moduleSQLJSON(values["logo_props"]), actor, repository.now(),
	}
	var insertedID string
	err := repository.database.QueryRow(ctx, `INSERT INTO modules
		(id,workspace_id,project_id,name,description,description_text,description_html,start_date,target_date,status,lead_id,view_props,sort_order,external_source,external_id,logo_props,created_by_id,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16::jsonb,$17,$18,$18) RETURNING id::text`, args...).Scan(&insertedID)
	if err != nil {
		return "", fmt.Errorf("create module: %w", err)
	}
	return insertedID, nil
}

func (repository *LifecycleRepository) replaceModuleMembers(ctx context.Context, workspaceID, projectID, moduleID string, memberIDs []string) error {
	now := repository.now()
	if _, err := repository.database.Exec(ctx, `UPDATE module_members SET deleted_at=$2 WHERE module_id=$1 AND deleted_at IS NULL`, moduleID, now); err != nil {
		return fmt.Errorf("soft-delete existing module members: %w", err)
	}
	var createdBy, updatedBy *string
	if err := repository.database.QueryRow(ctx, `SELECT created_by_id::text,updated_by_id::text FROM modules WHERE id=$1`, moduleID).Scan(&createdBy, &updatedBy); err != nil {
		return fmt.Errorf("load module membership audit fields: %w", err)
	}
	if len(memberIDs) == 0 {
		return nil
	}
	// Django's bulk_create wraps all insert batches in one atomic block, after
	// the queryset soft-delete above has already committed. Keep that exact
	// failure boundary: an insertion error rolls back every replacement row,
	// while the old active rows remain soft-deleted.
	return insertModuleMembersAtomically(ctx, func(ctx context.Context) (moduleMemberTransaction, error) {
		return repository.database.Begin(ctx)
	}, workspaceID, projectID, moduleID, memberIDs, createdBy, updatedBy, now)
}

type moduleMemberTransaction interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

func insertModuleMembersAtomically(
	ctx context.Context,
	begin func(context.Context) (moduleMemberTransaction, error),
	workspaceID, projectID, moduleID string,
	memberIDs []string,
	createdBy, updatedBy *string,
	now time.Time,
) (resultErr error) {
	tx, err := begin(ctx)
	if err != nil {
		return fmt.Errorf("begin module member replacement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr)
		}
	}()
	for offset := 0; offset < len(memberIDs); offset += 10 {
		end := offset + 10
		if end > len(memberIDs) {
			end = len(memberIDs)
		}
		var query strings.Builder
		query.WriteString(`INSERT INTO module_members (id,workspace_id,project_id,module_id,member_id,created_by_id,updated_by_id,created_at,updated_at) VALUES `)
		args := make([]any, 0, (end-offset)*8)
		for i, memberID := range memberIDs[offset:end] {
			if i > 0 {
				query.WriteByte(',')
			}
			base := i * 8
			fmt.Fprintf(&query, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+8)
			args = append(args, uuid.NewString(), workspaceID, projectID, moduleID, memberID, createdBy, updatedBy, now)
		}
		query.WriteString(" ON CONFLICT DO NOTHING")
		if _, err := tx.Exec(ctx, query.String(), args...); err != nil {
			return fmt.Errorf("replace module member batch: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit module member replacement: %w", err)
	}
	committed = true
	return nil
}

func (repository *LifecycleRepository) updateModuleFields(ctx context.Context, actor, slug, projectID, moduleID string, values map[string]any) error {
	columns := map[string]string{
		"name": "name", "description": "description", "description_text": "description_text", "description_html": "description_html",
		"start_date": "start_date", "target_date": "target_date", "status": "status", "lead_id": "lead_id",
		"view_props": "view_props", "sort_order": "sort_order", "external_source": "external_source", "external_id": "external_id", "logo_props": "logo_props",
	}
	keys := []string{"name", "description", "description_text", "description_html", "start_date", "target_date", "status", "lead_id", "view_props", "sort_order", "external_source", "external_id", "logo_props"}
	sets, args := []string{}, []any{moduleID, projectID, slug}
	for _, key := range keys {
		value, exists := values[key]
		if !exists {
			continue
		}
		args = append(args, value)
		cast := ""
		if key == "view_props" || key == "logo_props" || key == "description_text" || key == "description_html" {
			args[len(args)-1] = moduleSQLJSON(value)
			cast = "::jsonb"
		}
		sets = append(sets, columns[key]+"=$"+strconv.Itoa(len(args))+cast)
	}
	args = append(args, repository.now(), actor)
	sets = append(sets, "updated_at=$"+strconv.Itoa(len(args)-1), "updated_by_id=$"+strconv.Itoa(len(args)))
	command, err := repository.database.Exec(ctx, `UPDATE modules SET `+strings.Join(sets, ",")+` WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL`, args...)
	if err != nil {
		return fmt.Errorf("update module fields: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (repository *LifecycleRepository) rawModule(ctx context.Context, slug, projectID, moduleID string) (map[string]any, error) {
	value, err := djangoapi.JSON(ctx, repository.database, `SELECT to_jsonb(m)||jsonb_build_object('workspace',m.workspace_id,'project',m.project_id,'lead',m.lead_id,'created_by',m.created_by_id,'updated_by',m.updated_by_id) FROM modules m JOIN workspaces w ON w.id=m.workspace_id WHERE m.id=$1 AND m.project_id=$2 AND w.slug=$3 AND m.deleted_at IS NULL`, moduleID, projectID, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}

func (repository *LifecycleRepository) moduleMutationProjection(ctx context.Context, actor, slug, projectID, moduleID string) (json.RawMessage, error) {
	return repository.moduleProjection(ctx, actor, slug, projectID, moduleID, true)
}

func (repository *LifecycleRepository) moduleWriteSerializerProjection(ctx context.Context, actor, slug, projectID, moduleID string) (json.RawMessage, error) {
	return repository.moduleProjection(ctx, actor, slug, projectID, moduleID, false)
}

func (repository *LifecycleRepository) moduleProjection(ctx context.Context, actor, slug, projectID, moduleID string, valuesOnly bool) (json.RawMessage, error) {
	projection := moduleJSON("m", "active_serializer")
	if valuesOnly {
		projection = "(" + projection + ") - 'archived_at'"
	}
	var raw []byte
	err := repository.database.QueryRow(ctx, `SELECT `+projection+` FROM modules m JOIN workspaces w ON w.id=m.workspace_id WHERE m.id=$2 AND m.project_id=$3 AND w.slug=$4 AND m.deleted_at IS NULL`, actor, moduleID, projectID, slug).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read module serializer projection: %w", err)
	}
	return json.RawMessage(raw), nil
}

func (repository *LifecycleRepository) moduleWriteProjection(ctx context.Context, actor, slug, projectID, moduleID string) (json.RawMessage, error) {
	var raw []byte
	err := repository.database.QueryRow(ctx, moduleWriteProjectionSQL(), actor, moduleID, projectID, slug).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	values, err := pythoncompat.DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if moduleActivityErr := moduleActivityFloatFields(values, "sort_order"); moduleActivityErr != nil {
		return nil, moduleActivityErr
	}
	serialized, err := pythoncompat.Dumps(values, raw)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(serialized), nil
}

func moduleWriteProjectionSQL() string {
	return `SELECT json_build_object(
		'id',m.id,
		'created_at',` + requesterTimestampSQL("m.created_at") + `,'updated_at',` + requesterTimestampSQL("m.updated_at") + `,'deleted_at',` + requesterTimestampSQL("m.deleted_at") + `,
		'name',m.name,'description',m.description,'description_text',m.description_text,
		'description_html',m.description_html,'start_date',m.start_date,'target_date',m.target_date,
		'status',m.status,'view_props',m.view_props,'sort_order',m.sort_order,
		'external_source',m.external_source,'external_id',m.external_id,'archived_at',` + requesterTimestampSQL("m.archived_at") + `,
		'logo_props',m.logo_props,'created_by',m.created_by_id,'updated_by',m.updated_by_id,
		'project',m.project_id,'workspace',m.workspace_id,'lead',m.lead_id,'lead_id',m.lead_id,
		'member_ids',COALESCE((SELECT json_agg(mm.member_id ORDER BY member_user.created_at DESC) FROM module_members mm JOIN users member_user ON member_user.id=mm.member_id WHERE mm.module_id=m.id AND mm.deleted_at IS NULL),'[]'::json),
		'members',COALESCE((SELECT json_agg(mm.member_id ORDER BY member_user.created_at DESC) FROM module_members mm JOIN users member_user ON member_user.id=mm.member_id WHERE mm.module_id=m.id AND mm.deleted_at IS NULL),'[]'::json))
		FROM modules m JOIN workspaces w ON w.id=m.workspace_id
		WHERE m.id=$2 AND m.project_id=$3 AND w.slug=$4 AND m.deleted_at IS NULL`
}

func moduleActivityFloatFields(values map[string]any, fields ...string) error {
	for _, field := range fields {
		raw, present := values[field]
		if !present || raw == nil {
			continue
		}
		switch number := raw.(type) {
		case json.Number:
			value, err := number.Float64()
			if err != nil {
				return fmt.Errorf("decode module activity number %s: %w", field, err)
			}
			values[field] = value
		case float64:
		default:
			return fmt.Errorf("module activity field %s has non-numeric type %T", field, raw)
		}
	}
	return nil
}

// moduleActivitySnapshotJSON serializes the app ModuleSerializer snapshot in
// Django's declared field order while preserving original nested JSON values.
func moduleActivitySnapshotJSON(values map[string]any, original []byte) (string, error) {
	var source map[string]json.RawMessage
	if err := json.Unmarshal(original, &source); err != nil {
		return "", fmt.Errorf("decode module activity snapshot order: %w", err)
	}
	order := []string{
		"id", "workspace_id", "project_id", "name", "description", "description_text", "description_html",
		"start_date", "target_date", "status", "lead_id", "member_ids", "view_props", "sort_order",
		"external_source", "external_id", "logo_props", "total_estimate_points", "completed_estimate_points",
		"is_favorite", "total_issues", "cancelled_issues", "completed_issues", "started_issues",
		"unstarted_issues", "backlog_issues", "created_at", "updated_at", "archived_at",
	}
	var ordered strings.Builder
	ordered.WriteByte('{')
	first := true
	for _, key := range order {
		if _, present := values[key]; !present {
			continue
		}
		raw := source[key]
		if len(raw) == 0 {
			var err error
			raw, err = json.Marshal(values[key])
			if err != nil {
				return "", fmt.Errorf("encode module activity snapshot field %s: %w", key, err)
			}
		}
		if !first {
			ordered.WriteByte(',')
		}
		first = false
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return "", fmt.Errorf("encode module activity snapshot key: %w", err)
		}
		ordered.Write(keyJSON)
		ordered.WriteByte(':')
		ordered.Write(raw)
	}
	ordered.WriteByte('}')
	serialized, err := pythoncompat.Dumps(values, []byte(ordered.String()))
	if err != nil {
		return "", fmt.Errorf("serialize module activity snapshot: %w", err)
	}
	return serialized, nil
}

func moduleSQLJSON(value any) string {
	if value == nil {
		return "null"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(encoded)
}
