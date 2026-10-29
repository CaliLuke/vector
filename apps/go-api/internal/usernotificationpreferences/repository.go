package usernotificationpreferences

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("user notification preferences not found")
var ErrDuplicate = errors.New("multiple user notification preferences found")

type FieldError struct {
	Field   string
	Message string
}

func (err *FieldError) Error() string { return err.Field + ": " + err.Message }

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (repository *Repository) Patch(ctx context.Context, userID uuid.UUID, values map[string]any) (uuid.UUID, error) {
	preferenceID, err := repository.activeID(ctx, userID)
	if err != nil {
		return uuid.Nil(), err
	}

	allowed := []string{"created_by", "updated_by", "deleted_at", "user", "workspace", "project", "property_change", "state_change", "comment", "mention", "issue_completed"}
	set := make([]string, 0, len(allowed))
	args := []any{preferenceID, userID}
	for _, field := range allowed {
		value, ok := values[field]
		if !ok {
			continue
		}
		var converted any
		switch field {
		case "deleted_at":
			if value == nil {
				converted = nil
				break
			}
			parsed, parseErr := time.Parse(time.RFC3339Nano, fmt.Sprint(value))
			if parseErr != nil {
				return uuid.Nil(), &FieldError{Field: field, Message: "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
			}
			converted = parsed
		case "created_by", "updated_by", "user", "workspace", "project":
			if value == nil {
				if field == "user" {
					return uuid.Nil(), &FieldError{Field: field, Message: "This field may not be null."}
				}
				converted = nil
			} else {
				parsed, parseErr := uuid.Parse(fmt.Sprint(value))
				if parseErr != nil {
					return uuid.Nil(), &FieldError{Field: field, Message: fmt.Sprintf("\u201c%v\u201d is not a valid UUID.", value)}
				}
				if validateRelationErr := repository.validateRelation(ctx, field, parsed); validateRelationErr != nil {
					return uuid.Nil(), validateRelationErr
				}
				converted = parsed
			}
		default:
			boolean, valid := value.(bool)
			if !valid {
				return uuid.Nil(), fmt.Errorf("%s: Must be a valid boolean.", field)
			}
			converted = boolean
		}
		if field == "updated_by" {
			continue
		}
		set = append(set, fmt.Sprintf("%s = $%d", field, len(args)+1))
		args = append(args, converted)
	}
	set = append(set, "updated_by = $2", "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE user_notification_preferences SET %s WHERE id = $1 AND deleted_at IS NULL", strings.Join(set, ", "))
	result, err := repository.database.Exec(ctx, query, args...)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("patch user notification preferences: %w", err)
	}
	if result.RowsAffected() == 0 {
		return uuid.Nil(), ErrNotFound
	}
	return preferenceID, nil
}

func (repository *Repository) activeID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var rawID string
	var count int64
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(MIN(id::text), ''), COUNT(*)
FROM user_notification_preferences
WHERE user_id = $1 AND deleted_at IS NULL
`, userID).Scan(&rawID, &count)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("find user notification preferences: %w", err)
	}
	if count == 0 {
		return uuid.Nil(), ErrNotFound
	}
	if count > 1 {
		return uuid.Nil(), ErrDuplicate
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("parse user notification preference ID: %w", err)
	}
	return id, nil
}

func (repository *Repository) validateRelation(ctx context.Context, field string, id uuid.UUID) error {
	table := map[string]string{
		"created_by": "users",
		"updated_by": "users",
		"user":       "users",
		"workspace":  "workspaces",
		"project":    "projects",
	}[field]
	if table == "" {
		return nil
	}
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1 AND deleted_at IS NULL)", table)
	if err := repository.database.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return fmt.Errorf("validate %s relation: %w", field, err)
	}
	if !exists {
		return &FieldError{Field: field, Message: fmt.Sprintf("Invalid pk \"%s\" - object does not exist.", id)}
	}
	return nil
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("user notification preferences database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) Get(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	return repository.get(ctx, "p.user_id = $1", userID)
}

func (repository *Repository) GetByID(ctx context.Context, id uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, preferenceJSONQuery+` WHERE p.id = $1`, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user notification preferences by ID: %w", err)
	}
	return json.RawMessage(data), nil
}

const preferenceJSONQuery = `
SELECT json_build_object(
  'id', p.id, 'created_by', p.created_by_id, 'updated_by', p.updated_by_id,
  'created_at', p.created_at, 'updated_at', p.updated_at, 'deleted_at', p.deleted_at,
  'user', p.user_id, 'workspace', p.workspace_id, 'project', p.project_id,
  'property_change', p.property_change, 'state_change', p.state_change,
  'comment', p.comment, 'mention', p.mention, 'issue_completed', p.issue_completed
)
FROM user_notification_preferences AS p`

func (repository *Repository) get(ctx context.Context, predicate string, id uuid.UUID) (json.RawMessage, error) {
	var data []byte
	preferenceID, err := repository.activeID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = repository.database.QueryRow(ctx, preferenceJSONQuery+` WHERE `+predicate+` AND p.id = $2 AND p.deleted_at IS NULL`, id, preferenceID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user notification preferences: %w", err)
	}
	return json.RawMessage(data), nil
}
