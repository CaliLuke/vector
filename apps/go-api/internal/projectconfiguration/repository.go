// Package projectconfiguration owns the app estimate and project-label writes
// selected for the project configuration retirement.
package projectconfiguration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type HTTPError struct {
	Status int
	Body   any
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("project configuration request failed (%d)", e.Status)
}
func failure(status int, field, message string) error {
	return &HTTPError{Status: status, Body: map[string]any{field: message}}
}

func requireObjectInput(input djangoinput.Input) error {
	if input.Kind != "" || input.Null {
		return failure(http.StatusBadRequest, "error", "Please provide valid detail")
	}
	return nil
}

type Repository struct{ db database }

func NewRepository(db database) (*Repository, error) {
	if db == nil {
		return nil, errors.New("project configuration database is required")
	}
	return &Repository{db: db}, nil
}

func (r *Repository) DatabaseQueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return r.db.QueryRow(ctx, query, args...)
}

func (r *Repository) Authorize(ctx context.Context, user, slug, project string, write bool) error {
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, r.db, user, slug, project, write)
	if err != nil {
		return fmt.Errorf("authorize project configuration request: %w", err)
	}
	if !allowed {
		return failure(http.StatusForbidden, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

// AuthorizeAPIV1 applies ProjectMemberPermission: any active project member may
// access API v1 labels, without the app ProjectEntityPermission role threshold.
func (r *Repository) AuthorizeAPIV1(ctx context.Context, user, slug, project string) error {
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, r.db, user, slug, project, false)
	if err != nil {
		return fmt.Errorf("authorize API v1 project label request: %w", err)
	}
	if !allowed {
		return failure(http.StatusForbidden, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

// CreateEstimate validates every point before beginning the write transaction.
// The old Django handler persisted the estimate before point validation; the
// parity correction preserves the response while preventing that partial write.
func (r *Repository) CreateEstimate(ctx context.Context, user, slug, project string, body io.Reader, contentType string) (json.RawMessage, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if requireObjectInputErr := requireObjectInput(input); requireObjectInputErr != nil {
		return nil, requireObjectInputErr
	}
	estimate, ok := input.Fields["estimate"].(map[string]any)
	if !ok {
		return nil, failure(400, "estimate", "This field is required.")
	}
	name, exists := estimate["name"]
	if !exists {
		name = randomEstimateName()
	}
	nameText := djangoinput.ScalarString(name)
	typeValue := "categories"
	if value, present := estimate["type"]; present {
		typeValue = djangoinput.ScalarString(value)
	}
	var lastUsed any = false
	if value, present := estimate["last_used"]; present {
		if parsed, problem := djangoinput.BooleanField(value); problem == "" {
			lastUsed = parsed
		} else {
			lastUsed = value
		}
	}
	var points []any
	if value, present := input.Fields["estimate_points"]; present {
		var valid bool
		points, valid = value.([]any)
		if !valid {
			return nil, failure(400, "estimate_points", "Expected a list of items but got type \""+djangoinput.PythonType(value)+"\".")
		}
	}
	validated := make([]estimatePointInput, len(points))
	for index, raw := range points {
		point, valid := raw.(map[string]any)
		if !valid {
			return nil, failure(400, "estimate_points", "Expected a dictionary of items but got type \""+djangoinput.PythonType(raw)+"\".")
		}
		key, keyExists := point["key"]
		keyNumber, keyValid := int64(0), true
		if keyExists {
			keyNumber, keyValid = integerValue(key)
		}
		if !keyValid || keyNumber < 0 {
			return nil, failure(400, "estimate_points", "A valid integer is required.")
		}
		rawValue, valueExists := point["value"]
		if !valueExists {
			return nil, failure(400, "estimate_points", "This field is required.")
		}
		value, fieldErrors := djangoinput.CharField(rawValue, false, nil)
		if len(fieldErrors) > 0 {
			return nil, failure(400, "estimate_points", fieldErrors[0])
		}
		if utf8.RuneCountInString(value) > 20 {
			return nil, &HTTPError{Status: 400, Body: []any{map[string]any{"non_field_errors": []string{"Value can't be more than 20 characters"}}}}
		}
		description := ""
		if rawDescription, present := point["description"]; present {
			var ok bool
			description, ok = rawDescription.(string)
			if !ok {
				return nil, failure(400, "estimate_points", "Not a valid string.")
			}
		}
		validated[index] = estimatePointInput{key: keyNumber, value: value, description: description}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var estimateID string
	err = tx.QueryRow(ctx, `INSERT INTO estimates (id,project_id,workspace_id,name,type,last_used,description,created_at,updated_at,created_by_id,updated_by_id)
SELECT $1,p.id,p.workspace_id,$4,$5,$6,'',clock_timestamp(),clock_timestamp(),$7,NULL
FROM projects p JOIN workspaces w ON w.id=p.workspace_id
WHERE p.id=$2 AND w.slug=$3 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
RETURNING id::text`, uuid.NewString(), project, slug, nameText, typeValue, lastUsed, user).Scan(&estimateID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, failure(404, "detail", "Not found.")
	}
	if err != nil {
		return nil, mapWriteError(err)
	}
	for _, point := range validated {
		_, err = tx.Exec(ctx, `INSERT INTO estimate_points (id,estimate_id,project_id,workspace_id,key,value,description,created_at,updated_at,created_by_id,updated_by_id)
SELECT $1,$2,p.id,p.workspace_id,$4,$5,$6,clock_timestamp(),clock_timestamp(),$7,$7 FROM projects p WHERE p.id=$3`, uuid.NewString(), estimateID, project, point.key, point.value, point.description, user)
		if err != nil {
			return nil, mapWriteError(err)
		}
	}
	var data []byte
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object('id',e.id,'created_at',e.created_at,'updated_at',e.updated_at,'created_by',e.created_by_id,'updated_by',e.updated_by_id,'deleted_at',e.deleted_at,'workspace',e.workspace_id,'project',e.project_id,'name',e.name,'description',e.description,'type',e.type,'last_used',e.last_used,'points',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',p.id,'created_at',p.created_at,'updated_at',p.updated_at,'created_by',p.created_by_id,'updated_by',p.updated_by_id,'deleted_at',p.deleted_at,'estimate',p.estimate_id,'workspace',p.workspace_id,'project',p.project_id,'key',p.key,'value',p.value,'description',p.description) ORDER BY p.value) FROM estimate_points p WHERE p.estimate_id=e.id AND p.deleted_at IS NULL),'[]'::jsonb)) FROM estimates e WHERE e.id=$1`, estimateID).Scan(&data)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

type estimatePointInput struct {
	key                int64
	value, description string
}

func integerValue(value any) (int64, bool) {
	switch v := value.(type) {
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case int:
		return int64(v), true
	case int64:
		return v, true
	case string:
		n, err := djangoinput.IntegerField(v)
		if err != nil || !n.IsInt64() {
			return 0, false
		}
		return n.Int64(), true
	default:
		return 0, false
	}
}
func randomEstimateName() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, 10)
	for i := range bytes {
		bytes[i] = alphabet[uuid.New()[0]%byte(len(alphabet))]
	}
	return string(bytes)
}
func mapWriteError(err error) error {
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && strings.HasPrefix(postgres.Code, "23") {
		return failure(400, "estimate", "The payload is not valid.")
	}
	return err
}
