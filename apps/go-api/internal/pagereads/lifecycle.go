package pagereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type lifecycleDatabase interface {
	database
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type pageLifecycleState struct {
	workspaceID uuid.UUID
	ownerID     uuid.UUID
	createdByID *uuid.UUID
	updatedByID *uuid.UUID
	role        int
	access      int
	locked      bool
	archivedAt  *time.Time
}

type lifecycleLabelReference struct {
	index  int
	id     uuid.UUID
	raw    any
	source json.RawMessage
}

type lifecycleLabelList struct {
	present    bool
	references []lifecycleLabelReference
	errors     any
}

type LifecycleError struct {
	Status int
	Body   any
}

func (failure *LifecycleError) Error() string {
	return fmt.Sprintf("page lifecycle HTTP %d", failure.Status)
}

// Lifecycle applies page writes through the existing page-read domain owner.
// Tenant and role checks are repeated at the write boundary so stale project
// memberships cannot authorize a mutation.
func (repository *Repository) Lifecycle(
	ctx context.Context,
	operation string,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
	timezone string,
	input djangoinput.Input,
) (int, json.RawMessage, error) {
	db, ok := repository.database.(lifecycleDatabase)
	if !ok {
		return 0, nil, fmt.Errorf("page lifecycle requires a transactional database")
	}
	if operation == "create" {
		return repository.createPage(ctx, db, userID, slug, projectID, timezone, input)
	}
	state, err := repository.pageLifecycleState(ctx, userID, slug, projectID, pageID)
	if err != nil {
		return 0, nil, err
	}
	owner := state.ownerID == userID
	if !owner && state.access == 1 {
		return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
	}
	if !owner {
		switch operation {
		case "update", "description_update", "duplicate", "access", "lock":
			if state.role < 15 {
				return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
			}
		case "delete", "unlock":
			if state.role < 20 {
				return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
			}
		}
	}
	switch operation {
	case "access":
		accessValue, _ := input.FieldValue("access", false, false, false)
		if accessValue == nil {
			if _, present := input.Fields["access"]; !present {
				accessValue = json.Number("0")
			}
		}
		value, valid := lifecycleChoice(accessValue)
		if !valid {
			if accessValue == nil {
				return 0, nil, &LifecycleError{Status: http.StatusBadRequest, Body: []string{"This field may not be null."}}
			}
			return 0, nil, &LifecycleError{Status: http.StatusBadRequest, Body: []string{lifecycleChoiceMessage(input, accessValue)}}
		}
		rawAccess, present := input.Fields["access"]
		if present && lifecycleRawAccessChanged(rawAccess, state, userID) {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
		if !owner && !present && lifecycleRawAccessChanged(accessValue, state, userID) {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
		if err := repository.writePage(ctx, db, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE pages SET access=$1, updated_by_id=$2, updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, value, userID, pageID, state.workspaceID)
			return err
		}); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, nil, nil
	case "lock", "unlock":
		if !owner && ((operation == "lock" && state.role < 15) || (operation == "unlock" && state.role < 20)) {
			return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
		locked := operation == "lock"
		if _, err := db.Exec(ctx, `UPDATE pages SET is_locked=$1, updated_by_id=$2, updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, locked, userID, pageID, state.workspaceID); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, nil, nil
	case "archive", "unarchive":
		if !owner && operation == "archive" && state.role == 15 {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Only the owner or admin can archive the page")
		}
		if !owner && state.role < 20 {
			return 0, nil, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
		if operation == "archive" {
			if err := repository.archivePage(ctx, db, projectID, state.workspaceID, pageID); err != nil {
				return 0, nil, err
			}
			data, _ := json.Marshal(map[string]any{"archived_at": time.Now().Format("2006-01-02 15:04:05.000000")})
			return http.StatusOK, data, nil
		}
		if err := repository.unarchivePage(ctx, db, projectID, state.workspaceID, pageID); err != nil {
			return 0, nil, err
		}
		return http.StatusNoContent, nil, nil
	case "delete":
		if state.archivedAt == nil {
			return 0, nil, lifecycleError(http.StatusBadRequest, "The page should be archived before deleting")
		}
		if !owner && state.role < 20 {
			return 0, nil, lifecycleError(http.StatusForbidden, "Only admin or owner can delete the page")
		}
		if err := repository.deletePage(ctx, db, userID, projectID, state.workspaceID, pageID); err != nil {
			return 0, nil, err
		}
		if repository.publisher != nil {
			if err := repository.publisher.SoftDelete(ctx, "db", "page", pageID.String()); err != nil {
				return 0, nil, err
			}
		}
		return http.StatusNoContent, nil, nil
	case "update", "description_update", "duplicate":
		if operation == "description_update" {
			return repository.updatePageDescription(ctx, db, userID, pageID, state, input)
		}
		return repository.updateOrDuplicatePage(ctx, db, operation, userID, slug, projectID, pageID, state, timezone, input)
	default:
		return 0, nil, fmt.Errorf("unknown page lifecycle operation %q", operation)
	}
}

func (repository *Repository) AuthorizeLifecycle(ctx context.Context, operation string, user uuid.UUID, slug string, project, page uuid.UUID) error {
	if operation == "create" || operation == "delete_collection" || operation == "unsupported_collection" || (operation == "options" && page == uuid.Nil()) {
		var role int
		err := repository.database.QueryRow(ctx, `SELECT pm.role FROM workspaces w JOIN projects pr ON pr.workspace_id=w.id AND pr.id=$2 AND pr.deleted_at IS NULL JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$1 AND wm.is_active AND wm.deleted_at IS NULL JOIN project_members pm ON pm.project_id=pr.id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL WHERE w.slug=$3 AND w.deleted_at IS NULL`, user, project, slug).Scan(&role)
		minimumRole := 15
		if operation == "options" {
			minimumRole = 5
		} else if operation == "delete_collection" {
			minimumRole = 20
		}
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (role < minimumRole || operation == "unsupported_collection")) {
			return lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
		return err
	}
	state, err := repository.pageLifecycleState(ctx, user, slug, project, page)
	if err != nil {
		return err
	}
	owner := state.ownerID == user
	if operation == "options" {
		if !owner && state.access == 1 {
			return lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
		return nil
	}
	if !owner && state.access == 1 {
		return lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
	}
	if !owner {
		minimumRole := 15
		switch operation {
		case "delete", "unarchive", "unlock":
			minimumRole = 20
		case "update", "description_update", "duplicate", "access", "archive", "lock":
		case "unsupported":
			return lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		default:
			return fmt.Errorf("unknown page lifecycle operation %q", operation)
		}
		if state.role < minimumRole {
			return lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
		}
	}
	return nil
}

// ValidateLifecycleHandler applies view-level checks after the source permission
// class and timezone validation, but before parsing lazy request.data.
func (repository *Repository) ValidateLifecycleHandler(ctx context.Context, operation string, user uuid.UUID, slug string, project, page uuid.UUID) error {
	if operation == "create" {
		return nil
	}
	state, err := repository.pageLifecycleState(ctx, user, slug, project, page)
	if err != nil {
		return err
	}
	owner := state.ownerID == user
	switch operation {
	case "delete":
		if state.archivedAt == nil {
			return lifecycleError(http.StatusBadRequest, "The page should be archived before deleting")
		}
	case "archive":
		if !owner && state.role == 15 {
			return lifecycleError(http.StatusBadRequest, "Only the owner or admin can archive the page")
		}
	case "update", "description_update":
		if state.locked {
			if operation == "description_update" {
				return &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error_code": 4701, "error_message": "PAGE_LOCKED"}}
			}
			return lifecycleError(http.StatusBadRequest, "Page is locked")
		}
		if state.archivedAt != nil && operation == "description_update" {
			return &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error_code": 4702, "error_message": "PAGE_ARCHIVED"}}
		}
	}
	return nil
}

func (repository *Repository) pageLifecycleState(ctx context.Context, user uuid.UUID, slug string, project, page uuid.UUID) (pageLifecycleState, error) {
	var state pageLifecycleState
	err := repository.database.QueryRow(ctx, `
SELECT w.id, p.owned_by_id, p.created_by_id, p.updated_by_id, pm.role, p.access, p.is_locked, p.archived_at
FROM pages p
JOIN workspaces w ON w.id=p.workspace_id AND w.slug=$2 AND w.deleted_at IS NULL
JOIN project_pages pp ON pp.page_id=p.id AND pp.project_id=$3 AND pp.workspace_id=w.id AND pp.deleted_at IS NULL
JOIN projects pr ON pr.id=pp.project_id AND pr.workspace_id=w.id AND pr.deleted_at IS NULL
JOIN project_members pm ON pm.project_id=pr.id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL
JOIN workspace_members wm ON wm.workspace_id=w.id AND wm.member_id=$1 AND wm.is_active AND wm.deleted_at IS NULL
WHERE p.id=$4 AND p.workspace_id=w.id AND p.deleted_at IS NULL`, user, slug, project, page).Scan(
		&state.workspaceID, &state.ownerID, &state.createdByID, &state.updatedByID, &state.role, &state.access, &state.locked, &state.archivedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, lifecycleError(http.StatusForbidden, "You do not have permission to perform this action.")
	}
	if err != nil {
		return state, err
	}
	return state, nil
}

func (repository *Repository) writePage(ctx context.Context, db lifecycleDatabase, operation func(pgx.Tx) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := operation(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *Repository) archivePage(ctx context.Context, db lifecycleDatabase, project, workspace, page uuid.UUID) error {
	return repository.writePage(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=now() WHERE entity_type='page' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, page, project, workspace); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
WITH RECURSIVE tree(id) AS (
  SELECT p.id FROM pages p JOIN project_pages pp ON pp.page_id=p.id
  WHERE p.id=$1 AND p.workspace_id=$2 AND pp.project_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL AND p.deleted_at IS NULL
  UNION
  SELECT child.id FROM pages child JOIN tree parent ON child.parent_id=parent.id
  JOIN project_pages pp ON pp.page_id=child.id AND pp.project_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL
  WHERE child.workspace_id=$2 AND child.deleted_at IS NULL
)
UPDATE pages SET archived_at=CURRENT_DATE WHERE id IN (SELECT id FROM tree) AND workspace_id=$2`, page, workspace, project)
		return err
	})
}

func (repository *Repository) unarchivePage(ctx context.Context, db lifecycleDatabase, project, workspace, page uuid.UUID) error {
	return repository.writePage(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE pages p SET parent_id=NULL WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND EXISTS (SELECT 1 FROM pages parent WHERE parent.id=p.parent_id AND parent.workspace_id=$2 AND parent.archived_at IS NOT NULL AND parent.deleted_at IS NULL)`, page, workspace); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
WITH RECURSIVE tree(id) AS (
  SELECT p.id FROM pages p JOIN project_pages pp ON pp.page_id=p.id
  WHERE p.id=$1 AND p.workspace_id=$2 AND pp.project_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL AND p.deleted_at IS NULL
  UNION
  SELECT child.id FROM pages child JOIN tree parent ON child.parent_id=parent.id
  JOIN project_pages pp ON pp.page_id=child.id AND pp.project_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL
  WHERE child.workspace_id=$2 AND child.deleted_at IS NULL
)
UPDATE pages SET archived_at=NULL WHERE id IN (SELECT id FROM tree) AND workspace_id=$2`, page, workspace, project)
		return err
	})
}

func (repository *Repository) deletePage(ctx context.Context, db lifecycleDatabase, user, project, workspace, page uuid.UUID) error {
	return repository.writePage(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE pages SET parent_id=NULL WHERE parent_id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND id IN (SELECT page_id FROM project_pages WHERE project_id=$3 AND workspace_id=$2 AND deleted_at IS NULL)`, page, workspace, project); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE pages SET deleted_at=now(), updated_by_id=$1, updated_at=now() WHERE id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, user, page, workspace); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE user_favorites SET deleted_at=now() WHERE entity_type='page' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, page, project, workspace); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM user_recent_visits WHERE entity_name='page' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=$3`, page, project, workspace)
		return err
	})
}

func lifecycleError(status int, message string) error {
	key := "error"
	if status == http.StatusUnauthorized {
		key = "detail"
	}
	if status == http.StatusForbidden {
		key = "detail"
	}
	return &djangoapi.HTTPError{Status: status, Body: map[string]any{key: message}}
}

func pageLifecycleDatabaseError(err error) error {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && strings.HasPrefix(postgres.Code, "23") {
		return &djangoapi.HTTPError{Status: http.StatusBadRequest, Body: map[string]any{"error": "The payload is not valid"}}
	}
	return err
}

func lifecycleChoice(value any) (int, bool) {
	// DRF ChoiceField compares the Python string spelling with its choices;
	// an integral-looking JSON float is still "0.0", not choice "0".
	switch djangoinput.ScalarString(value) {
	case "0":
		return 0, true
	case "1":
		return 1, true
	default:
		return 0, false
	}
}

func lifecycleChoiceMessage(input djangoinput.Input, value any) string {
	text, err := pythoncompat.String(value, input.JSONFieldSource("access"))
	if err != nil {
		text = djangoinput.ScalarString(value)
	}
	return `"` + text + `" is not a valid choice.`
}

// lifecycleParentLookupUUID matches the raw Page.objects.get/filter lookup in
// the page views. Django accepts booleans as integer primary keys, while the
// later DRF relation field performs its own typed validation.
func lifecycleParentLookupUUID(value any) (uuid.UUID, error) {
	lookupValue := value
	if boolean, ok := value.(bool); ok {
		if boolean {
			lookupValue = json.Number("1")
		} else {
			lookupValue = json.Number("0")
		}
	}
	parsed, err := djangoinput.UUID(lookupValue)
	if err != nil {
		return uuid.Nil(), err
	}
	return uuid.Parse(parsed.String())
}

func lifecycleSerializerParentUUID(value any) (uuid.UUID, error) {
	if _, ok := value.(bool); ok {
		return uuid.Nil(), fmt.Errorf("boolean is not a primary key value")
	}
	parsed, err := djangoinput.UUID(value)
	if err != nil {
		return uuid.Nil(), err
	}
	return uuid.Parse(parsed.String())
}

func (repository *Repository) lifecycleSerializerParent(ctx context.Context, workspace, project uuid.UUID, value any, source []byte, prechecked *uuid.UUID) (any, string, error) {
	if value == nil || value == "" {
		return nil, "", nil
	}
	if _, ok := value.(bool); ok {
		return nil, "Incorrect type. Expected pk value, received bool.", nil
	}
	parent, err := lifecycleSerializerParentUUID(value)
	if err != nil {
		text, reprErr := pythoncompat.String(value, source)
		if reprErr != nil {
			text = djangoinput.ScalarString(value)
		}
		return nil, "“" + text + "” is not a valid UUID.", nil
	}
	if prechecked != nil {
		return parent, "", nil
	}
	exists, err := repository.lifecycleParentExists(ctx, parent, workspace, project)
	if err != nil {
		return nil, "", err
	}
	if !exists {
		text, reprErr := pythoncompat.String(value, source)
		if reprErr != nil {
			text = djangoinput.ScalarString(value)
		}
		return nil, `Invalid pk "` + text + `" - object does not exist.`, nil
	}
	return parent, "", nil
}

func (repository *Repository) lifecycleParentExists(ctx context.Context, parent, workspace, project uuid.UUID) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pages p JOIN project_pages pp ON pp.page_id=p.id WHERE p.id=$1 AND p.workspace_id=$2 AND p.deleted_at IS NULL AND pp.project_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL)`, parent, workspace, project).Scan(&exists)
	return exists, err
}

func lifecycleJSONField(input djangoinput.Input, key string, fallback any) ([]byte, error) {
	value, present, err := input.JSONFieldValue(key)
	if err != nil {
		return nil, err
	}
	if present && value == nil {
		return nil, fmt.Errorf("null")
	}
	if !present {
		value = fallback
	}
	return json.Marshal(value)
}

func lifecycleJSONFieldMessage(err error) string {
	if err.Error() == "null" {
		return "This field may not be null."
	}
	return "Value must be valid JSON."
}

func lifecycleBoolean(value any) (bool, bool) {
	text := strings.ToLower(djangoinput.ScalarString(value))
	switch text {
	case "true", "t", "yes", "y", "on", "1":
		return true, true
	case "false", "f", "no", "n", "off", "0":
		return false, true
	default:
		if number, ok := value.(json.Number); ok {
			if parsed, err := number.Float64(); err == nil {
				if parsed == 1 {
					return true, true
				}
				if parsed == 0 {
					return false, true
				}
			}
		}
		return false, false
	}
}

func (repository *Repository) updateOrDuplicatePage(ctx context.Context, db lifecycleDatabase, operation string, user uuid.UUID, slug string, project, page uuid.UUID, state pageLifecycleState, timezone string, input djangoinput.Input) (int, json.RawMessage, error) {
	fields := input.Fields
	if operation == "duplicate" {
		return repository.duplicatePage(ctx, db, user, slug, project, page, state, timezone)
	}
	if state.locked {
		return 0, nil, lifecycleError(http.StatusBadRequest, "This page is locked")
	}
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
			return 0, nil, lifecycleError(http.StatusBadRequest, "Please provide valid detail")
		}
		valid, err := repository.lifecycleParentExists(ctx, parent, state.workspaceID, project)
		if err != nil {
			return 0, nil, err
		}
		if !valid {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
		handlerParent = parent
		serializerPrecheckedParent = &parent
	}
	// The source view compares the raw request scalar for this ownership guard
	// before DRF applies ChoiceField coercion. Thus, for example, raw "0" is
	// distinct from the stored integer 0 for a non-owner.
	if raw, present := fields["access"]; present && lifecycleRawAccessChanged(raw, state, user) {
		return 0, nil, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
	}
	if parentTruthy {
		var cycle bool
		if err := repository.database.QueryRow(ctx, `WITH RECURSIVE descendants(id) AS (SELECT $1::uuid UNION SELECT p.id FROM pages p JOIN descendants d ON p.parent_id=d.id WHERE p.workspace_id=$2) SELECT EXISTS (SELECT 1 FROM descendants WHERE id=$3)`, page, state.workspaceID, handlerParent).Scan(&cycle); err != nil {
			return 0, nil, err
		}
		if cycle {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"parent": []string{"Page hierarchy cannot contain a cycle."}}}
		}
	}
	fields, fieldErrors, labelList := validatePageLifecycleFields(input)
	labelList, err := repository.validateLifecycleLabelReferences(ctx, state.workspaceID, labelList)
	if err != nil {
		return 0, nil, err
	}
	if labelList.errors != nil {
		fieldErrors["labels"] = labelList.errors
	}
	if parentPresent {
		parentValueNormalized, message, parentErr := repository.lifecycleSerializerParent(ctx, state.workspaceID, project, parentValue, input.JSONFieldSource("parent"), serializerPrecheckedParent)
		if parentErr != nil {
			return 0, nil, parentErr
		}
		if message != "" {
			fieldErrors["parent"] = []string{message}
		} else {
			fields["parent"] = parentValueNormalized
		}
	}
	if len(fieldErrors) > 0 {
		return 0, nil, &djangoapi.HTTPError{Status: 400, Body: fieldErrors}
	}
	if value, ok := fields["access"]; ok {
		access, valid := lifecycleChoice(value)
		if !valid {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"access": []string{lifecycleChoiceMessage(input, value)}}}
		}
		if access != state.access && state.ownerID != user {
			return 0, nil, lifecycleError(http.StatusBadRequest, "Access cannot be updated since this page is owned by someone else")
		}
	}
	var oldDescription string
	_ = repository.database.QueryRow(ctx, `SELECT description_html FROM pages WHERE id=$1 AND workspace_id=$2`, page, state.workspaceID).Scan(&oldDescription)
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

func lifecycleRawAccessChanged(raw any, state pageLifecycleState, user uuid.UUID) bool {
	if state.ownerID == user {
		return false
	}
	// Python compares the raw request value with the model integer before
	// ChoiceField coercion. Numeric JSON values compare equal (including bool,
	// which is an int subclass in Python); strings never equal the integer.
	switch value := raw.(type) {
	case string, nil:
		return true
	case bool:
		return (value && state.access != 1) || (!value && state.access != 0)
	case json.Number:
		number, err := value.Float64()
		return err != nil || number != float64(state.access)
	case float64:
		return value != float64(state.access)
	case float32:
		return float64(value) != float64(state.access)
	case int:
		return value != state.access
	case int8:
		return int(value) != state.access
	case int16:
		return int(value) != state.access
	case int32:
		return int(value) != state.access
	case int64:
		return value != int64(state.access)
	default:
		return true
	}
}

func validatePageLifecycleFields(input djangoinput.Input) (map[string]any, map[string]any, lifecycleLabelList) {
	values := make(map[string]any, len(input.Fields))
	errorsByField := map[string]any{}
	for key, maximum := range map[string]int{"name": 0, "color": 255, "description_html": 0} {
		value, present := input.FieldValue(key, false, true, false)
		if !present {
			continue
		}
		var text string
		var failures []string
		if key == "color" {
			text, failures = djangoinput.CharField(value, true, input.Surrogates[key])
		} else {
			text, failures = djangoinput.TextField(value, true, maximum, input.Surrogates[key])
		}
		if len(failures) > 0 {
			errorsByField[key] = failures
		} else if key == "description_html" && text == "" {
			errorsByField[key] = []string{"This field may not be blank."}
		} else {
			values[key] = text
		}
	}
	value, present := input.ListValue("labels", true)
	labels := lifecycleParseLabelList(value, present, input.JSONFieldSource("labels"))
	if labels.errors != nil {
		errorsByField["labels"] = labels.errors
	}
	if value, present := input.FieldValue("access", false, false, false); present {
		choice, valid := lifecycleChoice(value)
		if !valid {
			message := lifecycleChoiceMessage(input, value)
			if value == nil {
				message = "This field may not be null."
			}
			errorsByField["access"] = []string{message}
		} else {
			values["access"] = choice
		}
	}
	for _, key := range []string{"view_props", "logo_props"} {
		_, present := input.Fields[key]
		if !present {
			continue
		}
		value, _, err := input.JSONFieldValue(key)
		if err != nil {
			errorsByField[key] = []string{"Value must be valid JSON."}
		} else if value == nil {
			errorsByField[key] = []string{"This field may not be null."}
		} else {
			encoded, err := json.Marshal(value)
			if err != nil {
				errorsByField[key] = []string{"Value must be valid JSON."}
			} else {
				values[key] = encoded
			}
		}
	}
	if value, present := input.BooleanFieldValue("is_locked", true); present {
		if locked, valid := lifecycleBoolean(value); !valid {
			errorsByField["is_locked"] = []string{"Must be a valid boolean."}
		} else {
			values["is_locked"] = locked
		}
	}
	if value, present := input.FieldValue("archived_at", false, false, true); present {
		if value == nil {
			values["archived_at"] = nil
		} else if date, ok := value.(string); !ok {
			errorsByField["archived_at"] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
		} else if _, err := time.Parse(time.DateOnly, date); err != nil {
			errorsByField["archived_at"] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
		} else {
			values["archived_at"] = date
		}
	}
	if value, present := input.Fields["parent"]; present {
		values["parent"] = value
	}
	return values, errorsByField, labels
}

func (repository *Repository) updatePageDescription(ctx context.Context, db lifecycleDatabase, user, page uuid.UUID, state pageLifecycleState, input djangoinput.Input) (int, json.RawMessage, error) {
	fields := input.Fields
	var oldHTML string
	if err := repository.database.QueryRow(ctx, `SELECT description_html FROM pages WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, page, state.workspaceID).Scan(&oldHTML); err != nil {
		return 0, nil, err
	}
	update := map[string]any{}
	if value, ok := fields["description_html"]; ok {
		if value == nil {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_html": []string{"This field may not be null."}}}
		}
		html, failures := djangoinput.TextField(value, true, 0, input.Surrogates["description_html"])
		if len(failures) > 0 {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_html": failures}}
		}
		clean, _, err := content.ValidateHTML(html)
		if err != nil {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_html": []string{err.Error()}}}
		}
		update["description_html"] = clean
	}
	if value, ok := fields["description_binary"]; ok {
		if value == nil {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_binary": []string{"This field may not be null."}}}
		}
		encoded, failures := djangoinput.TextField(value, true, 0, input.Surrogates["description_binary"])
		if len(failures) > 0 {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_binary": failures}}
		}
		if encoded == "" {
			update["description_binary"] = []byte{}
		} else {
			decoded, err := content.DecodeBase64Binary(encoded)
			if err != nil {
				if errors.Is(err, content.ErrInvalidBase64) {
					return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_binary": []string{"Failed to decode base64 data"}}}
				}
				return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_binary": []string{"Invalid binary data: " + err.Error()}}}
			}
			update["description_binary"] = decoded
		}
	}
	if _, ok := fields["description_json"]; ok {
		value, _, err := input.JSONFieldValue("description_json")
		if err != nil {
			return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_json": []string{"Value must be valid JSON."}}}
		}
		if value == nil {
			update["description_json"] = nil
		} else {
			encoded, err := json.Marshal(value)
			if err != nil {
				return 0, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"description_json": []string{"Value must be valid JSON."}}}
			}
			update["description_json"] = encoded
		}
	}
	if err := repository.writePage(ctx, db, func(tx pgx.Tx) error {
		if err := checkPageCollaborationRevision(ctx, tx, page, state.workspaceID, fields); err != nil {
			return err
		}
		for key, value := range update {
			if _, err := tx.Exec(ctx, `UPDATE pages SET `+key+`=$1, updated_by_id=$2, updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, value, user, page, state.workspaceID); err != nil {
				return err
			}
		}
		var stripped any
		currentHTML := oldHTML
		if html, ok := update["description_html"].(string); ok {
			currentHTML = html
		}
		if currentHTML != "" {
			stripped = content.StripTags(currentHTML)
		}
		_, err := tx.Exec(ctx, `UPDATE pages SET description_stripped=$1,updated_by_id=$2,updated_at=now() WHERE id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, stripped, user, page, state.workspaceID)
		return err
	}); err != nil {
		return 0, nil, pageLifecycleDatabaseError(err)
	}
	if repository.publisher != nil || repository.nativeTasks != nil {
		if value, ok := fields["description_html"]; ok && pythoncompat.Truthy(value) {
			if err := repository.admitPageTransaction(ctx, value, oldHTML, page.String()); err != nil {
				return 0, nil, err
			}
		}
		existing, _ := pythoncompat.Dumps(map[string]any{"description_html": oldHTML}, nil)
		if err := repository.admitPageVersion(ctx, page.String(), existing, user.String()); err != nil {
			return 0, nil, err
		}
	}
	result, _ := json.Marshal(map[string]any{"message": "Updated successfully"})
	return http.StatusOK, result, nil
}

func (repository *Repository) duplicatePage(ctx context.Context, db lifecycleDatabase, user uuid.UUID, slug string, project, page uuid.UUID, state pageLifecycleState, timezone string) (int, json.RawMessage, error) {
	var duplicate uuid.UUID
	var descriptionHTML string
	err := repository.writePage(ctx, db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO pages (id,workspace_id,name,description_json,description_binary,description_html,description_stripped,owned_by_id,access,color,parent_id,view_props,logo_props,is_global,moved_to_page,moved_to_project,sort_order,external_id,external_source,created_by_id,updated_by_id,created_at,updated_at,is_locked,archived_at) SELECT $4,workspace_id,name||' (Copy)',description_json,NULL,description_html,description_stripped,$1,access,color,parent_id,view_props,logo_props,is_global,moved_to_page,moved_to_project,sort_order,external_id,external_source,$1,$1,now(),now(),is_locked,archived_at FROM pages WHERE id=$2 AND workspace_id=$3 AND deleted_at IS NULL RETURNING id,description_html`, user, page, state.workspaceID, uuid.New()).Scan(&duplicate, &descriptionHTML)
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT pp.project_id FROM project_pages pp JOIN projects pr ON pr.id=pp.project_id AND pr.workspace_id=$2 AND pr.deleted_at IS NULL JOIN project_members pm ON pm.project_id=pr.id AND pm.member_id=$1 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (15,20) JOIN workspace_members wm ON wm.workspace_id=$2 AND wm.member_id=$1 AND wm.is_active AND wm.deleted_at IS NULL WHERE pp.page_id=$3 AND pp.workspace_id=$2 AND pp.deleted_at IS NULL UNION SELECT $4::uuid`, user, state.workspaceID, page, project)
		if err != nil {
			return err
		}
		var links []uuid.UUID
		for rows.Next() {
			var linked uuid.UUID
			if err := rows.Scan(&linked); err != nil {
				rows.Close()
				return err
			}
			links = append(links, linked)
		}
		rowsErr := rows.Err()
		rows.Close()
		if rowsErr != nil {
			return rowsErr
		}
		for _, linked := range links {
			if _, err := tx.Exec(ctx, `INSERT INTO project_pages(id,workspace_id,project_id,page_id,created_by_id,updated_by_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,NULL,now(),now())`, uuid.New(), state.workspaceID, linked, duplicate, user); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	if repository.publisher != nil || repository.nativeTasks != nil {
		if err := repository.admitPageTransaction(ctx, descriptionHTML, nil, duplicate.String()); err != nil {
			return 0, nil, err
		}
		if err := repository.admitPageDuplicateAssets(ctx, "PAGE", duplicate.String(), project.String(), slug, user.String()); err != nil {
			return 0, nil, err
		}
	}
	return repository.pageLifecycleDetail(ctx, user, slug, project, duplicate, timezone, http.StatusCreated, "duplicate")
}

func (repository *Repository) pageLifecycleDetail(ctx context.Context, user uuid.UUID, slug string, project, page uuid.UUID, timezone string, status int, projection string) (int, json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, pageLifecycleDetailSQL, user, slug, project, page, timezone).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, lifecycleError(http.StatusNotFound, "The required object does not exist.")
	}
	if err != nil {
		return 0, nil, err
	}
	var body map[string]any
	if decodeErr := json.Unmarshal(data, &body); decodeErr != nil {
		return 0, nil, decodeErr
	}
	delete(body, "issue_ids")
	switch projection {
	case "update":
		delete(body, "is_favorite")
		delete(body, "label_ids")
		delete(body, "project_ids")
	case "duplicate":
		delete(body, "is_favorite")
		delete(body, "label_ids")
	}
	data, err = json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	return status, json.RawMessage(data), nil
}

var pageLifecycleDetailSQL = `
SELECT json_build_object(
  'collaboration_revision', page.collaboration_revision,
  'id', page.id, 'name', page.name, 'owned_by', page.owned_by_id,
  'access', page.access, 'color', page.color, 'parent', page.parent_id,
  'is_favorite', EXISTS (
    SELECT 1 FROM user_favorites favorite
    WHERE favorite.user_id=$1 AND favorite.workspace_id=page.workspace_id
      AND favorite.project_id=$3 AND favorite.entity_type='page'
      AND favorite.entity_identifier=page.id AND favorite.deleted_at IS NULL
  ),
  'is_locked', page.is_locked, 'archived_at', page.archived_at,
  'workspace', page.workspace_id,
	  'created_at', ` + djangoDateTimeSQL("page.created_at", "$5") + `,
	  'updated_at', ` + djangoDateTimeSQL("page.updated_at", "$5") + `,
  'created_by', page.created_by_id, 'updated_by', page.updated_by_id,
  'view_props', page.view_props, 'logo_props', page.logo_props,
  'label_ids', COALESCE((
    SELECT json_agg(DISTINCT page_label.label_id)
    FROM page_labels page_label JOIN labels label ON label.id=page_label.label_id
      AND label.workspace_id=page.workspace_id AND label.deleted_at IS NULL
    WHERE page_label.page_id=page.id AND page_label.deleted_at IS NULL
  ), '[]'::json),
  'project_ids', COALESCE((
    SELECT json_agg(DISTINCT link.project_id)
    FROM project_pages link JOIN projects linked_project ON linked_project.id=link.project_id
      AND linked_project.workspace_id=page.workspace_id AND linked_project.deleted_at IS NULL
    WHERE link.page_id=page.id AND link.workspace_id=page.workspace_id AND link.deleted_at IS NULL
  ), '[]'::json),
  'description_html', page.description_html
)
FROM pages page
JOIN workspaces workspace ON workspace.id=page.workspace_id AND workspace.slug=$2 AND workspace.deleted_at IS NULL
JOIN project_pages requested_link ON requested_link.page_id=page.id AND requested_link.project_id=$3
  AND requested_link.workspace_id=workspace.id AND requested_link.deleted_at IS NULL
JOIN projects project ON project.id=requested_link.project_id AND project.workspace_id=workspace.id AND project.deleted_at IS NULL
JOIN project_members member ON member.project_id=project.id AND member.workspace_id=workspace.id
  AND member.member_id=$1 AND member.is_active AND member.deleted_at IS NULL
WHERE page.id=$4 AND page.workspace_id=workspace.id AND page.deleted_at IS NULL
LIMIT 1`

func lifecycleParseLabelList(value any, present bool, source []byte) lifecycleLabelList {
	labels := lifecycleLabelList{present: present}
	if !present {
		return labels
	}
	if value == nil {
		labels.errors = []string{"This field may not be null."}
		return labels
	}
	items, ok := value.([]any)
	if !ok {
		labels.errors = []string{fmt.Sprintf("Expected a list of items but got type %q.", djangoinput.PythonType(value))}
		return labels
	}
	childErrors := map[string]any{}
	var sourceChildren []json.RawMessage
	_ = json.Unmarshal(source, &sourceChildren)
	for index, item := range items {
		indexKey := strconv.Itoa(index)
		var childSource []byte
		if index < len(sourceChildren) {
			childSource = sourceChildren[index]
		}
		if item == nil || item == "" {
			childErrors[indexKey] = []string{"This field may not be null."}
			continue
		}
		if _, isBool := item.(bool); isBool {
			childErrors[indexKey] = []string{"Incorrect type. Expected pk value, received bool."}
			continue
		}
		parsed, err := djangoinput.UUID(item)
		if err != nil {
			childErrors[indexKey] = []string{"“" + lifecycleLabelValueString(item, childSource) + "” is not a valid UUID."}
			continue
		}
		id, err := uuid.Parse(parsed.String())
		if err != nil {
			childErrors[indexKey] = []string{"“" + lifecycleLabelValueString(item, childSource) + "” is not a valid UUID."}
			continue
		}
		labels.references = append(labels.references, lifecycleLabelReference{index: index, id: id, raw: item, source: childSource})
	}
	if len(childErrors) > 0 {
		labels.errors = childErrors
	}
	return labels
}

func (repository *Repository) validateLifecycleLabelReferences(ctx context.Context, workspace uuid.UUID, labels lifecycleLabelList) (lifecycleLabelList, error) {
	if len(labels.references) == 0 {
		return labels, nil
	}
	childErrors, hasChildErrors := labels.errors.(map[string]any)
	if !hasChildErrors {
		childErrors = map[string]any{}
	}
	for _, label := range labels.references {
		var exists bool
		if err := repository.database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM labels WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL)`, label.id, workspace).Scan(&exists); err != nil {
			return labels, err
		}
		if !exists {
			childErrors[strconv.Itoa(label.index)] = []string{"Invalid pk \"" + lifecycleLabelValueString(label.raw, label.source) + "\" - object does not exist."}
		}
	}
	if len(childErrors) > 0 {
		labels.errors = childErrors
	}
	return labels, nil
}

func lifecycleLabelValueString(value any, source []byte) string {
	text, err := pythoncompat.String(value, source)
	if err != nil {
		return djangoinput.ScalarString(value)
	}
	return text
}
