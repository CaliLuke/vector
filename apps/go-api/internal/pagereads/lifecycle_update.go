package pagereads

import (
	"context"
	"encoding/json"
	"net/http"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/jackc/pgx/v5"
)

func (repository *Repository) updateOrDuplicatePage(ctx context.Context, db lifecycleDatabase, operation string, user uuid.UUID, slug string, project, page uuid.UUID, state pageLifecycleState, timezone string, input djangoinput.Input) (int, json.RawMessage, error) {
	if operation == "duplicate" {
		return repository.duplicatePage(ctx, db, user, slug, project, page, state, timezone)
	}
	if state.locked {
		return 0, nil, lifecycleError(http.StatusBadRequest, "This page is locked")
	}
	fields, labelList, err := repository.validatePageUpdateFields(ctx, user, project, page, state, input)
	if err != nil {
		return 0, nil, err
	}
	var oldDescription string
	if descriptionErr := repository.database.QueryRow(ctx, `SELECT description_html FROM pages WHERE id=$1 AND workspace_id=$2`, page, state.workspaceID).Scan(&oldDescription); descriptionErr != nil {
		return 0, nil, descriptionErr
	}
	newDescription, descriptionChanged := fields["description_html"].(string)
	if writePageErr := repository.writePage(ctx, db, func(tx pgx.Tx) error {
		if _, collaborative := input.Fields["collaboration_revision"]; collaborative || fields["name"] != nil || fields["description_html"] != nil {
			if checkPageErr := checkPageCollaborationRevision(ctx, tx, page, state.workspaceID, input.Fields); checkPageErr != nil {
				return checkPageErr
			}
		}
		for key, column := range map[string]string{"name": "name", "color": "color", "access": "access", "parent": "parent_id", "description_html": "description_html", "view_props": "view_props", "logo_props": "logo_props", "is_locked": "is_locked", "archived_at": "archived_at"} {
			value, ok := fields[key]
			if !ok {
				continue
			}
			if key == "parent" && (value == nil || value == "") {
				value = nil
			}
			if key == "view_props" || key == "logo_props" {
				jsonValue, _, jsonFieldErr := input.JSONFieldValue(key)
				if jsonFieldErr != nil {
					return &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{key: []string{"Value must be valid JSON."}}}
				}
				encoded, jsonFieldErr := json.Marshal(jsonValue)
				if jsonFieldErr != nil {
					return jsonFieldErr
				}
				value = encoded
			}
			_, execErr := tx.Exec(ctx, `UPDATE pages SET `+column+`=$1, updated_by_id=$2, updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, value, user, page, state.workspaceID)
			if execErr != nil {
				return execErr
			}
		}
		if labelList.present {
			if _, err = tx.Exec(ctx, `UPDATE page_labels SET deleted_at=now() WHERE page_id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, page, state.workspaceID); err != nil {
				return err
			}
			for _, label := range labelList.references {
				if _, err := tx.Exec(ctx, `INSERT INTO page_labels(id,workspace_id,page_id,label_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,now(),now())`, uuid.New(), state.workspaceID, page, label.id, state.createdByID, state.updatedByID); err != nil {
					return err
				}
			}
		}
		// DRF's ModelSerializer calls save() even when the input contains no
		// writable fields, updating BaseModel audit columns.
		var stripped any
		if descriptionChanged {
			if newDescription != "" {
				stripped = content.StripTags(newDescription)
			}
		} else if oldDescription != "" {
			stripped = content.StripTags(oldDescription)
		}
		if _, err := tx.Exec(ctx, `UPDATE pages SET description_stripped=$1, updated_by_id=$2, updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, stripped, user, page, state.workspaceID); err != nil {
			return err
		}
		return nil
	}); writePageErr != nil {
		return 0, nil, writePageErr
	}
	if rawDescription, present := input.Fields["description_html"]; (repository.publisher != nil || repository.nativeTasks != nil) && descriptionChanged && present && pythoncompat.Truthy(rawDescription) {
		if err := repository.admitPageTransaction(ctx, rawDescription, oldDescription, page.String()); err != nil {
			return 0, nil, err
		}
	}
	return repository.pageLifecycleDetail(ctx, user, slug, project, page, timezone, http.StatusOK, "update")
}

func (repository *Repository) validatePageUpdateFields(ctx context.Context, user, project, page uuid.UUID, state pageLifecycleState, input djangoinput.Input) (map[string]any, lifecycleLabelList, error) {
	fields := input.Fields
	// The view resolves the submitted parent before it compares access or
	// validates serializer fields. A stale or foreign parent therefore wins
	// over both later checks.
	var handlerParent uuid.UUID
	var serializerPrecheckedParent *uuid.UUID
	parentValue, parentPresent := fields["parent"]
	parentTruthy := parentPresent && pythoncompat.Truthy(parentValue)
	if parentTruthy {
		parent, err := lifecycleParentLookupUUID(parentValue)
		if err != nil {
			return nil, lifecycleLabelList{}, lifecycleError(http.StatusBadRequest, "Please provide valid detail")
		}
		valid, err := repository.lifecycleParentExists(ctx, parent, state.workspaceID, project)
		if err != nil {
			return nil, lifecycleLabelList{}, err
		}
		if !valid {
			return nil, lifecycleLabelList{}, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
		handlerParent = parent
		serializerPrecheckedParent = &parent
	}
	// The source view compares the raw request scalar for this ownership guard
	// before DRF applies ChoiceField coercion. Thus, for example, raw "0" is
	// distinct from the stored integer 0 for a non-owner.
	if raw, present := fields["access"]; present && lifecycleRawAccessChanged(raw, state, user) {
		return nil, lifecycleLabelList{}, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
	}
	if parentTruthy {
		var cycle bool
		if err := repository.database.QueryRow(ctx, `WITH RECURSIVE descendants(id) AS (SELECT $1::uuid UNION SELECT p.id FROM pages p JOIN descendants d ON p.parent_id=d.id WHERE p.workspace_id=$2) SELECT EXISTS (SELECT 1 FROM descendants WHERE id=$3)`, page, state.workspaceID, handlerParent).Scan(&cycle); err != nil {
			return nil, lifecycleLabelList{}, err
		}
		if cycle {
			return nil, lifecycleLabelList{}, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"parent": []string{"Page hierarchy cannot contain a cycle."}}}
		}
	}
	fields, fieldErrors, labelList := validatePageLifecycleFields(input)
	labelList, err := repository.validateLifecycleLabelReferences(ctx, state.workspaceID, labelList)
	if err != nil {
		return nil, lifecycleLabelList{}, err
	}
	if labelList.errors != nil {
		fieldErrors["labels"] = labelList.errors
	}
	if parentPresent {
		parentValueNormalized, message, parentErr := repository.lifecycleSerializerParent(ctx, state.workspaceID, project, parentValue, input.JSONFieldSource("parent"), serializerPrecheckedParent)
		if parentErr != nil {
			return nil, lifecycleLabelList{}, parentErr
		}
		if message != "" {
			fieldErrors["parent"] = []string{message}
		} else {
			fields["parent"] = parentValueNormalized
		}
	}
	if len(fieldErrors) > 0 {
		return nil, lifecycleLabelList{}, &djangoapi.HTTPError{Status: 400, Body: fieldErrors}
	}
	if value, ok := fields["access"]; ok {
		access, valid := lifecycleChoice(value)
		if !valid {
			return nil, lifecycleLabelList{}, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"access": []string{lifecycleChoiceMessage(input, value)}}}
		}
		if access != state.access && state.ownerID != user {
			return nil, lifecycleLabelList{}, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
	}
	return fields, labelList, nil
}
