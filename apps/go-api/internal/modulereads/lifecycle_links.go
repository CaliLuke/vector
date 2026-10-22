package modulereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ModuleLifecycleEffects contains the durable effects owned by module writes.
type ModuleLifecycleEffects interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

type lifecycleDatabase interface {
	database
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

// LifecycleRepository keeps module writes beside the existing module read
// repository so workspace and project scoping share one domain boundary.
type LifecycleRepository struct {
	database lifecycleDatabase
	reads    *Repository
	effects  ModuleLifecycleEffects
	now      func() time.Time
	baseHost string
}

func NewLifecycleRepository(database lifecycleDatabase, effects ModuleLifecycleEffects, baseHost ...string) (*LifecycleRepository, error) {
	if database == nil || effects == nil {
		return nil, errors.New("module lifecycle database and durable effects are required")
	}
	reads, err := NewRepository(database)
	if err != nil {
		return nil, err
	}
	host := ""
	if len(baseHost) > 0 {
		host = baseHost[0]
	}
	return &LifecycleRepository{database: database, reads: reads, effects: effects, now: time.Now, baseHost: host}, nil
}

// AuthorizeEntityProject matches ProjectEntityPermission for links and archive
// operations. Unlike app module writes, it has no workspace-admin fallback.
func (repository *LifecycleRepository) AuthorizeEntityProject(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM project_members pm
		JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		AND pm.is_active=true AND pm.deleted_at IS NULL AND pm.role IN (15,20)
	)`, userID, projectID, slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize module entity project: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// AuthorizeAppProject mirrors allow_permission([ADMIN, MEMBER]) for app
// module writes. Workspace-admin fallback still requires active project
// membership and is deliberately not shared with links/archive operations.
func (repository *LifecycleRepository) AuthorizeAppProject(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		AND pm.is_active=true AND pm.deleted_at IS NULL
		AND (pm.role IN (15,20) OR EXISTS (
			SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1
			AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
		))
	)`, userID, projectID, slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize app module project: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// CreateLink validates serializer fields before resolving the scoped module,
// then checks duplicate URLs before saving, matching the Django call order.
func (repository *LifecycleRepository) CreateLink(ctx context.Context, userID, slug, projectID, moduleID string, fields map[string]any, surrogates map[string][]string) (json.RawMessage, error) {
	values, fieldErrors := validateModuleLinkFields(fields, surrogates, "POST", true)
	if len(fieldErrors) != 0 {
		return nil, &ModuleValidationError{Body: fieldErrors}
	}
	moduleIDValue, err := repository.scopedModule(ctx, slug, projectID, moduleID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		}
		return nil, err
	}
	if err := repository.duplicateLink(ctx, moduleIDValue, values["url"].(string), ""); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	var result []byte
	metadata, err := moduleJSONValue(values["metadata"])
	if err != nil {
		return nil, fmt.Errorf("encode module link metadata: %w", err)
	}
	err = repository.database.QueryRow(ctx, `INSERT INTO module_links
		(id, workspace_id, project_id, module_id, created_by_id, title, url, metadata,created_at,updated_at)
		SELECT $2, m.workspace_id, m.project_id, m.id, $6, $7, $8, $9::jsonb,$10,$10
		FROM modules m JOIN projects p ON p.id=m.project_id
		JOIN workspaces w ON w.id=m.workspace_id
		WHERE m.id=$3 AND p.id=$4 AND w.slug=$5 AND m.deleted_at IS NULL
		RETURNING `+moduleLinkJSON("module_links"),
		userID, id, moduleIDValue, projectID, slug, userID, values["title"], values["url"], metadata, repository.now()).Scan(&result)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create module link: %w", err)
	}
	return result, nil
}

func (repository *LifecycleRepository) UpdateLink(ctx context.Context, userID, slug, projectID, moduleID, linkID string, fields map[string]any, surrogates map[string][]string, replace bool) (json.RawMessage, error) {
	method := "PATCH"
	if replace {
		method = "PUT"
	}
	if err := repository.authorizeLink(ctx, userID, slug, projectID, moduleID, linkID); err != nil {
		return nil, err
	}
	values, fieldErrors := validateModuleLinkFields(fields, surrogates, method, replace)
	if len(fieldErrors) != 0 {
		return nil, &ModuleValidationError{Body: fieldErrors}
	}
	current, err := repository.linkURL(ctx, moduleID, linkID)
	if err != nil {
		return nil, err
	}
	urlValue := current
	if next, exists := values["url"]; exists {
		urlValue = next.(string)
	}
	if err := repository.duplicateLink(ctx, moduleID, urlValue, linkID); err != nil {
		return nil, err
	}
	metadata, err := moduleJSONValue(values["metadata"])
	if err != nil {
		return nil, fmt.Errorf("encode module link metadata: %w", err)
	}
	var result []byte
	err = repository.database.QueryRow(ctx, `UPDATE module_links SET
		title=CASE WHEN $6 THEN $7 ELSE title END,
		url=$8,
		metadata=CASE WHEN $9 THEN $10::jsonb ELSE metadata END,
		updated_by_id=$11, updated_at=now()
		WHERE id=$2 AND module_id=$3 AND project_id=$4 AND deleted_at IS NULL
		AND EXISTS (SELECT 1 FROM workspaces w WHERE w.id=module_links.workspace_id AND w.slug=$5)
		RETURNING `+moduleLinkJSON("module_links"),
		userID, linkID, moduleID, projectID, slug, hasModuleField(fields, "title"), values["title"], urlValue,
		hasModuleField(fields, "metadata"), metadata, userID).Scan(&result)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update module link: %w", err)
	}
	return result, nil
}

// CheckLink resolves the detail object before request-body validation, matching
// DRF's update action order. UpdateLink repeats the scoped check in its write
// path so callers cannot bypass authorization by skipping this preflight.
func (repository *LifecycleRepository) CheckLink(ctx context.Context, userID, slug, projectID, moduleID, linkID string) error {
	err := repository.authorizeLink(ctx, userID, slug, projectID, moduleID, linkID)
	if errors.Is(err, ErrNotFound) {
		return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "No ModuleLink matches the given query."}}
	}
	return err
}

// LinkExists reports the public detail queryset independently of the actor's
// write permission so authenticated OPTIONS can return metadata without actions
// for guests and can omit actions for missing links.
func (repository *LifecycleRepository) LinkExists(ctx context.Context, slug, projectID, moduleID, linkID string) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM module_links ml
		JOIN modules m ON m.id=ml.module_id AND m.project_id=ml.project_id AND m.workspace_id=ml.workspace_id
		JOIN projects p ON p.id=ml.project_id AND p.workspace_id=ml.workspace_id
		JOIN workspaces w ON w.id=ml.workspace_id
		WHERE ml.id=$1 AND ml.module_id=$2 AND ml.project_id=$3 AND w.slug=$4
		AND ml.deleted_at IS NULL AND p.deleted_at IS NULL AND p.archived_at IS NULL AND w.deleted_at IS NULL
	)`, linkID, moduleID, projectID, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check module link protocol resource: %w", err)
	}
	return exists, nil
}

func (repository *LifecycleRepository) DeleteLink(ctx context.Context, userID, slug, projectID, moduleID, linkID string) error {
	if err := repository.authorizeLink(ctx, userID, slug, projectID, moduleID, linkID); err != nil {
		return err
	}
	command, err := repository.database.Exec(ctx, `UPDATE module_links SET deleted_at=now(),updated_at=now(),updated_by_id=$5
		WHERE id=$1 AND project_id=$2 AND module_id=$3 AND deleted_at IS NULL
		AND EXISTS (SELECT 1 FROM workspaces w WHERE w.slug=$4 AND w.id=module_links.workspace_id)
		AND EXISTS (SELECT 1 FROM projects p WHERE p.id=module_links.project_id AND p.archived_at IS NULL)`,
		linkID, projectID, moduleID, slug, userID)
	if err != nil {
		return fmt.Errorf("soft-delete module link: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return repository.effects.SoftDelete(ctx, "db", "modulelink", linkID)
}

func (repository *LifecycleRepository) scopedModule(ctx context.Context, slug, projectID, moduleID string) (string, error) {
	var id string
	err := repository.database.QueryRow(ctx, `SELECT m.id FROM modules m
		JOIN projects p ON p.id=m.project_id AND p.workspace_id=m.workspace_id
		JOIN workspaces w ON w.id=m.workspace_id
		WHERE m.id=$1 AND p.id=$2 AND w.slug=$3 AND m.deleted_at IS NULL`, moduleID, projectID, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("scope module link parent: %w", err)
	}
	return id, nil
}

func (repository *LifecycleRepository) authorizeLink(ctx context.Context, userID, slug, projectID, moduleID, linkID string) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM module_links ml
		JOIN modules m ON m.id=ml.module_id AND m.project_id=ml.project_id AND m.workspace_id=ml.workspace_id
		JOIN projects p ON p.id=ml.project_id AND p.workspace_id=ml.workspace_id
		JOIN workspaces w ON w.id=ml.workspace_id
		WHERE ml.id=$1 AND ml.module_id=$2 AND ml.project_id=$3 AND w.slug=$4
		AND ml.deleted_at IS NULL AND p.archived_at IS NULL
		AND EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id
			AND pm.member_id=$5 AND pm.is_active=true AND pm.deleted_at IS NULL AND pm.role IN (15,20))
	)`, linkID, moduleID, projectID, slug, userID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize module link: %w", err)
	}
	if !allowed {
		return ErrNotFound
	}
	return nil
}

func (repository *LifecycleRepository) linkURL(ctx context.Context, moduleID, linkID string) (string, error) {
	var value string
	err := repository.database.QueryRow(ctx, `SELECT url FROM module_links WHERE id=$1 AND module_id=$2 AND deleted_at IS NULL`, linkID, moduleID).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (repository *LifecycleRepository) duplicateLink(ctx context.Context, moduleID, value, excludingID string) error {
	var exists bool
	var exclusion any
	if excludingID != "" {
		exclusion = excludingID
	}
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM module_links WHERE module_id=$1 AND url=$2 AND deleted_at IS NULL AND ($3::uuid IS NULL OR id<>$3::uuid))`, moduleID, value, exclusion).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check duplicate module link URL: %w", err)
	}
	if exists {
		message := "URL already exists."
		if excludingID != "" {
			message = "URL already exists for this Issue"
		}
		return &ModuleValidationError{Body: map[string]any{"error": message}}
	}
	return nil
}

func validateModuleLinkFields(fields map[string]any, surrogates map[string][]string, method string, requiredURL bool) (map[string]any, map[string]any) {
	values := map[string]any{}
	failures := map[string]any{}
	if value, present := fields["title"]; present {
		if value == nil {
			values["title"] = nil
		} else if text, messages := djangoinput.TextField(value, true, 255, surrogates["title"]); len(messages) > 0 {
			failures["title"] = messages
		} else {
			values["title"] = text
		}
	}
	if value, present := fields["metadata"]; present {
		if value == nil {
			failures["metadata"] = []string{"This field may not be null."}
		} else {
			values["metadata"] = value
		}
	}
	value, present := fields["url"]
	if !present && requiredURL {
		failures["url"] = []string{"This field is required."}
	} else if !present {
		// Missing-URL validation is a serializer update/save check, after field
		// conversion. Defer it until we know no supplied field failed.
	} else if value == nil {
		failures["url"] = []string{"This field may not be null."}
	} else if text, ok := value.(string); ok {
		// The serializer mutates the input by adding its scheme before URLField
		// performs max-length validation.
		if text != "" && !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
			text = "http://" + text
		}
		if text, messages := djangoinput.TextField(text, false, 200, surrogates["url"]); len(messages) > 0 {
			failures["url"] = messages
		} else if text == "" {
			failures["url"] = []string{"This field may not be blank."}
		} else if !djangoinput.ValidURL(text) {
			failures["url"] = []string{"Enter a valid URL."}
		} else {
			values["url"] = text
		}
	} else if _, messages := djangoinput.TextField(value, false, 200, surrogates["url"]); len(messages) > 0 {
		failures["url"] = messages
	} else {
		failures["url"] = []string{"Enter a valid URL."}
	}
	if !present && !requiredURL && method == "PATCH" && len(failures) == 0 {
		failures["error"] = "Invalid URL format."
	}
	return values, failures
}

func moduleJSONValue(value any) (string, error) {
	if value == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func hasModuleField(fields map[string]any, key string) bool { _, ok := fields[key]; return ok }

type ModuleValidationError struct{ Body map[string]any }

func (err *ModuleValidationError) Error() string { return "module lifecycle validation failed" }
