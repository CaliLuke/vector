package projectintakewrites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	intakereads "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type HTTPError struct {
	Status int
	Body   map[string]any
}

func (e *HTTPError) Error() string { return fmt.Sprintf("project intake write HTTP %d", e.Status) }

type Repository struct {
	db      Database
	readers *intakereads.Repository
}

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("project intake writes database is required")
	}
	readers, err := intakereads.NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db, readers: readers}, nil
}

func (r *Repository) Authorize(ctx context.Context, user, slug, project string) error {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1
		FROM project_members pm
		JOIN projects p ON p.id=pm.project_id
		JOIN workspaces w ON w.id=pm.workspace_id
		LEFT JOIN workspace_members wm ON wm.workspace_id=pm.workspace_id AND wm.member_id=pm.member_id AND wm.is_active
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
		  AND pm.is_active AND pm.deleted_at IS NULL
		  AND (pm.role IN(15,20) OR wm.role=20)
		  AND p.deleted_at IS NULL AND w.deleted_at IS NULL
	)`, user, project, slug).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return &HTTPError{403, map[string]any{"error": "You don't have the required permissions."}}
	}
	return nil
}

// ValidateCreate mirrors the IntakeSerializer checks that DRF runs before
// IntakeViewSet.perform_create applies its role decorator. It returns the
// normalized serializer values so the insert uses the same values it checked.
func (r *Repository) ValidateCreate(input djangoinput.Input) (map[string]any, error) {
	if input.Null {
		return nil, &HTTPError{400, map[string]any{"non_field_errors": []string{"No data provided"}}}
	}
	if input.Kind != "" {
		return nil, &HTTPError{400, map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	fields := make(map[string]any, len(input.Fields)+1)
	for key, value := range input.Fields {
		fields[key] = value
	}
	nameValue, present := input.FieldValue("name", true, false, false)
	if !present {
		return nil, &HTTPError{400, map[string]any{"name": []string{"This field is required."}}}
	}
	name, failures := djangoinput.TextField(nameValue, false, 255, input.Surrogates["name"])
	if len(failures) > 0 {
		return nil, &HTTPError{400, map[string]any{"name": failures}}
	}
	fields["name"] = name

	if description, present := input.FieldValue("description", false, true, false); present {
		text, messages := djangoinput.TextField(description, true, 0, input.Surrogates["description"])
		if len(messages) > 0 {
			return nil, &HTTPError{400, map[string]any{"description": messages}}
		}
		fields["description"] = text
	} else {
		fields["description"] = ""
	}

	if value, present := input.BooleanFieldValue("is_default", false); present {
		parsed, message := djangoinput.BooleanField(value)
		if message != "" {
			return nil, &HTTPError{400, map[string]any{"is_default": []string{message}}}
		}
		fields["is_default"] = parsed
	}
	for _, key := range []string{"view_props", "logo_props"} {
		value, present, err := input.JSONFieldValue(key)
		if err != nil {
			return nil, &HTTPError{400, map[string]any{key: []string{"Value must be valid JSON."}}}
		}
		if present && value == nil {
			return nil, &HTTPError{400, map[string]any{key: []string{"This field may not be null."}}}
		}
		if present {
			fields[key] = value
		}
	}
	return fields, nil
}

func (r *Repository) Create(ctx context.Context, user, slug, project string, fields map[string]any) (json.RawMessage, error) {
	validated, err := r.ValidateCreate(djangoinput.Input{Fields: fields})
	if err != nil {
		return nil, err
	}
	fields = validated
	name := fields["name"].(string)
	description, _ := fields["description"].(string)
	isDefault := false
	if value, exists := fields["is_default"]; exists {
		isDefault, _ = djangoinput.BooleanField(value)
	}
	viewProps, err := jsonField(fields, "view_props")
	if err != nil {
		return nil, err
	}
	logoProps, err := jsonField(fields, "logo_props")
	if err != nil {
		return nil, err
	}
	var id string
	err = r.db.QueryRow(ctx, `INSERT INTO intakes(id,workspace_id,project_id,name,description,is_default,view_props,logo_props,created_at,updated_at,created_by_id,updated_by_id,deleted_at) SELECT $1,p.workspace_id,p.id,$4,$5,$6,$7::jsonb,$8::jsonb,clock_timestamp(),clock_timestamp(),$9,NULL,NULL FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$2 AND w.slug=$3 AND p.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING id::text`, uuid.New().String(), project, slug, name, description, isDefault, viewProps, logoProps, user).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, &HTTPError{404, map[string]any{"detail": "Not found."}}
	}
	if err != nil {
		if uniqueViolation(err) {
			return nil, &HTTPError{400, map[string]any{"name": []string{"intake with this name already exists."}}}
		}
		return nil, err
	}
	data, err := r.detail(ctx, slug, project, id)
	if err != nil {
		return nil, err
	}
	return createdContainerResponse(data)
}

func (r *Repository) Patch(ctx context.Context, user, slug, project, id string, fields map[string]any) (json.RawMessage, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM intakes i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND p.id=$2 AND w.slug=$3 AND i.deleted_at IS NULL AND p.deleted_at IS NULL AND w.deleted_at IS NULL)`, id, project, slug).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &HTTPError{404, map[string]any{"detail": "No Intake matches the given query."}}
	}
	updates := []string{}
	args := []any{id}
	for _, field := range []string{"name", "description", "is_default", "view_props", "logo_props"} {
		value, present := fields[field]
		if !present {
			continue
		}
		if field == "name" {
			name, ok := value.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return nil, &HTTPError{400, map[string]any{"name": []string{"This field may not be blank."}}}
			}
			if len([]rune(name)) > 255 {
				return nil, &HTTPError{400, map[string]any{"name": []string{"Ensure this field has no more than 255 characters."}}}
			}
		}
		if field == "is_default" {
			if _, ok := value.(bool); !ok {
				return nil, &HTTPError{400, map[string]any{"is_default": []string{"Must be a valid boolean."}}}
			}
		}
		if field == "view_props" || field == "logo_props" {
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			value = encoded
			args = append(args, value)
			updates = append(updates, fmt.Sprintf("%s=$%d::jsonb", field, len(args)))
			continue
		}
		args = append(args, value)
		updates = append(updates, fmt.Sprintf("%s=$%d", field, len(args)))
	}
	if len(updates) > 0 {
		args = append(args, user)
		updates = append(updates, "updated_by_id=$"+fmt.Sprint(len(args)))
		if _, err := r.db.Exec(ctx, "UPDATE intakes SET "+strings.Join(updates, ",")+",updated_at=clock_timestamp() WHERE id=$1", args...); err != nil {
			if uniqueViolation(err) {
				return nil, &HTTPError{400, map[string]any{"name": []string{"intake with this name already exists."}}}
			}
			return nil, err
		}
	}
	return r.detail(ctx, slug, project, id)
}

func (r *Repository) Delete(ctx context.Context, user, slug, project, id string) error {
	var isDefault bool
	err := r.db.QueryRow(ctx, `SELECT i.is_default FROM intakes i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND p.id=$2 AND w.slug=$3 AND i.deleted_at IS NULL`, id, project, slug).Scan(&isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return &HTTPError{404, map[string]any{"detail": "No Intake matches the given query."}}
	}
	if err != nil {
		return err
	}
	if isDefault {
		return &HTTPError{400, map[string]any{"error": "You cannot delete the default intake"}}
	}
	_, err = r.db.Exec(ctx, `UPDATE intakes SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, user)
	return err
}

func (r *Repository) detail(ctx context.Context, slug, project, id string) (json.RawMessage, error) {
	data, err := r.readers.IntakeDetail(ctx, uuid.Nil(), slug, mustUUID(project), mustUUID(id))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func createdContainerResponse(data json.RawMessage) (json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	delete(fields, "pending_issue_count")
	return json.Marshal(fields)
}

func mustUUID(value string) uuid.UUID { id, _ := uuid.Parse(value); return id }
func jsonField(fields map[string]any, name string) ([]byte, error) {
	if value, ok := fields[name]; ok {
		return json.Marshal(value)
	}
	return []byte(`{}`), nil
}
func uniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
