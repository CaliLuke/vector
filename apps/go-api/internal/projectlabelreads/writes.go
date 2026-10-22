package projectlabelreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type HTTPError struct {
	Status int
	Body   map[string]any
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("label request failed (%d): %v", e.Status, e.Body)
}
func Failure(status int, field, message string) error {
	return &HTTPError{Status: status, Body: map[string]any{field: message}}
}

type Label struct {
	ID, Name, Color, WorkspaceID string
	ProjectID, Parent            *string
	SortOrder                    float64
}

type DeletionPublisher interface {
	SoftDelete(context.Context, string, string, string) error
}

// WriteRepository extends the existing label domain. Publication deliberately
// follows the saved soft delete, as it does in Django's SoftDeleteModel.
type WriteRepository struct {
	*Repository
	publisher DeletionPublisher
}

func NewWriteRepository(db database, publisher DeletionPublisher) (*WriteRepository, error) {
	if publisher == nil {
		return nil, errors.New("label deletion publisher is required")
	}
	repository, err := NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &WriteRepository{Repository: repository, publisher: publisher}, nil
}

func (r *WriteRepository) Authorize(ctx context.Context, user, slug, project, method string) error {
	var allowed bool
	var err error
	if method == http.MethodPost || method == http.MethodOptions || method == http.MethodGet || method == http.MethodHead {
		err = r.database.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM workspace_members member JOIN workspaces workspace ON workspace.id=member.workspace_id
 WHERE member.member_id=$1 AND workspace.slug=$2 AND member.is_active
 AND member.deleted_at IS NULL AND ($3 <> 'POST' OR member.role IN (15,20)))`, user, slug, method).Scan(&allowed)
	} else {
		err = r.database.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM project_members member JOIN workspaces workspace ON workspace.id=member.workspace_id
 WHERE member.member_id=$1 AND workspace.slug=$2 AND member.project_id=$3
 AND member.is_active AND member.deleted_at IS NULL
 AND (member.role=20 OR EXISTS (SELECT 1 FROM workspace_members wm
 WHERE wm.member_id=$1 AND wm.workspace_id=workspace.id AND wm.role=20 AND wm.is_active AND wm.deleted_at IS NULL)))`, user, slug, project).Scan(&allowed)
	}
	if err != nil {
		return err
	}
	if !allowed {
		return Failure(403, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

const labelColumns = `id::text,name,color,parent_id::text,project_id::text,workspace_id::text,sort_order`

func scanLabel(row pgx.Row) (*Label, error) {
	var result Label
	err := row.Scan(&result.ID, &result.Name, &result.Color, &result.Parent, &result.ProjectID, &result.WorkspaceID, &result.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, Failure(404, "detail", "No Label matches the given query.")
	}
	return &result, err
}

func (r *WriteRepository) Load(ctx context.Context, user, slug, project, id string) (*Label, error) {
	return scanLabel(r.database.QueryRow(ctx, `SELECT `+labelColumns+` FROM labels
 WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL
 AND workspace_id IN (SELECT id FROM workspaces WHERE slug=$3)
 AND EXISTS (SELECT 1 FROM project_members WHERE project_id=labels.project_id AND member_id=$4)`, id, project, slug, user))
}

func (r *WriteRepository) duplicate(ctx context.Context, project *string, name, exclude string, fold bool) (bool, error) {
	comparison := "name=$2"
	if fold {
		comparison = "UPPER(name)=UPPER($2)"
	}
	var exists bool
	err := r.database.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM labels WHERE deleted_at IS NULL
 AND project_id IS NOT DISTINCT FROM $1::uuid AND `+comparison+` AND ($3='' OR id::text<>$3))`, project, name, exclude).Scan(&exists)
	return exists, err
}

func (r *WriteRepository) Write(ctx context.Context, user, slug, project, id, method string, body io.Reader, contentType string) (*Label, error) {
	var label *Label
	var err error
	// Inherited PUT and DELETE fetch the target before ever asking DRF to parse a body.
	if method == http.MethodPut || method == http.MethodDelete {
		label, err = r.Load(ctx, user, slug, project, id)
		if err != nil {
			return nil, err
		}
	}
	if method == http.MethodDelete {
		_, err = scanLabel(r.database.QueryRow(ctx, `UPDATE labels SET deleted_at=clock_timestamp(),
 updated_at=clock_timestamp(),updated_by_id=$2,
 workspace_id=(SELECT workspace_id FROM projects WHERE id=labels.project_id)
 WHERE id=$1 RETURNING `+labelColumns, id, user))
		if err != nil {
			return nil, err
		}
		return nil, r.publisher.SoftDelete(ctx, "db", "label", id)
	}
	input, err := parseLabelInput(body, contentType)
	if err != nil {
		return nil, err
	}
	if method == http.MethodPatch {
		if input.null {
			return nil, Failure(500, "error", "Something went wrong please try again later")
		}
		if value, present := input.fields["name"]; present && value != nil {
			duplicate, err := r.duplicate(ctx, &project, pythonScalarString(value), id, false)
			if err != nil {
				return nil, err
			}
			if duplicate {
				return nil, Failure(400, "error", "Label with the same name already exists in the project")
			}
		}
		label, err = r.Load(ctx, user, slug, project, id)
		if err != nil {
			return nil, err
		}
	}
	if label == nil {
		label = &Label{ID: uuid.NewString(), ProjectID: &project, SortOrder: 65535}
	}
	if err = r.validateInput(ctx, input, label, project, method); err != nil {
		return nil, err
	}
	if method == http.MethodPost {
		label, err = scanLabel(r.database.QueryRow(ctx, `INSERT INTO labels
 (id,name,color,parent_id,project_id,workspace_id,sort_order,description,created_at,updated_at,created_by_id,updated_by_id)
 VALUES ($1,$2,$3,$4,$5,(SELECT workspace_id FROM projects WHERE id=$5),
 COALESCE((SELECT MAX(sort_order)+10000 FROM labels WHERE project_id=$5 AND deleted_at IS NULL),$6),
 '',clock_timestamp(),clock_timestamp(),$7,NULL) RETURNING `+labelColumns,
			label.ID, label.Name, label.Color, label.Parent, project, label.SortOrder, user))
	} else {
		label, err = scanLabel(r.database.QueryRow(ctx, `UPDATE labels SET name=$2,color=$3,parent_id=$4,sort_order=$5,
 updated_at=clock_timestamp(),updated_by_id=$6,
 workspace_id=(SELECT workspace_id FROM projects WHERE id=labels.project_id)
 WHERE id=$1 RETURNING `+labelColumns, id, label.Name, label.Color, label.Parent, label.SortOrder, user))
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) && strings.HasPrefix(postgres.Code, "23") {
		message := "The payload is not valid"
		if method == http.MethodPost {
			message = "Label with the same name already exists in the project"
		}
		return nil, Failure(400, "error", message)
	}
	return label, err
}
