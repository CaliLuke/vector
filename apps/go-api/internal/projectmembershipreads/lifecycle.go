package projectmembershipreads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	storeuuid "uuid"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type LifecycleCommand struct {
	ExpectedWorkspaceID                                  *storeuuid.UUID
	Operation, Method, UserID, Slug, ProjectID, MemberID string
	Params                                               map[string]string
	Present                                              map[string]bool
	Body                                                 io.Reader
	ContentType, BaseHost, Timezone                      string
}
type TaskPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}
type LifecycleRepository struct {
	*Repository
	writer   membershipDatabase
	baseHost string
	reader   database
}

func NewLifecycleRepository(db membershipDatabase, baseHost string) (*LifecycleRepository, error) {
	return NewLifecycleRepositoryWithReaderWriter(db, db, baseHost)
}
func NewLifecycleRepositoryWithReaderWriter(reader database, writer membershipDatabase, baseHost string) (*LifecycleRepository, error) {
	if writer == nil || reader == nil {
		return nil, errors.New("membership reader and writer are required")
	}
	r, err := NewRepository(writer)
	if err != nil {
		return nil, err
	}
	return &LifecycleRepository{Repository: r, writer: writer, baseHost: baseHost, reader: reader}, nil
}
func (r *LifecycleRepository) forCommand(c LifecycleCommand) *LifecycleRepository {
	if c.Operation == "v1-list-project-members" && c.Method == http.MethodGet {
		return r
	}
	if strings.HasPrefix(c.Operation, "v1-") && (c.Method == http.MethodGet || c.Method == http.MethodHead || c.Method == http.MethodOptions) {
		selected := *r
		selected.Repository = &Repository{database: r.reader}
		return &selected
	}
	return r
}

func lifecycleFailure(status int, field, message string) error {
	return &apicontract.HTTPError{Status: status, Body: map[string]any{field: message}}
}
func lifecycleMissing(legacy bool) error {
	if legacy {
		return lifecycleFailure(404, "error", "The required object does not exist.")
	}
	return lifecycleFailure(404, "error", "The requested resource does not exist.")
}
func (r *LifecycleRepository) AuthorizeLifecycle(ctx context.Context, c LifecycleCommand) error {
	r = r.forCommand(c)
	if isProjectMemberWrite(c) {
		return authorizeProjectMemberWrite(ctx, r.database, c)
	}
	var allowed bool
	read := c.Method == http.MethodGet
	var err error
	if c.Operation == "v1-list-members-lite" && c.Method == http.MethodPost {
		err = r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN (15,20))`, c.UserID, c.Slug).Scan(&allowed)
	} else if c.Operation == "v1-list-members-lite" && c.Method != http.MethodGet && c.Method != http.MethodHead && c.Method != http.MethodOptions {
		err = r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (15,20))`, c.UserID, c.Slug, c.ProjectID).Scan(&allowed)
	} else {
		read = read || c.Operation == "v1-list-members-lite"
		err = r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
 WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND ($4 OR pm.role=20))`, c.UserID, c.Slug, c.ProjectID, read).Scan(&allowed)
	}
	if err != nil {
		return err
	}
	if !allowed {
		return lifecycleFailure(403, "detail", "You do not have permission to perform this action.")
	}
	return nil
}
func (r *LifecycleRepository) ExecuteLifecycle(ctx context.Context, c LifecycleCommand) (any, error) {
	if isProjectMemberWrite(c) {
		return r.mutateProjectMembers(ctx, c)
	}
	r = r.forCommand(c)
	if strings.HasPrefix(c.Operation, "v1-list-") || strings.HasPrefix(c.Operation, "v1-detail-") {
		var workspace bool
		if err := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug=$1 AND deleted_at IS NULL)`, c.Slug).Scan(&workspace); err != nil {
			return nil, err
		}
		if !workspace {
			status := 400
			if strings.Contains(c.Operation, "lite") {
				status = 404
			}
			return nil, lifecycleFailure(status, "error", "Provided workspace does not exist")
		}
	}
	if c.Method == http.MethodHead || c.Method == http.MethodOptions {
		return nil, lifecycleFailure(405, "detail", fmt.Sprintf("Method %q not allowed.", c.Method))
	}
	switch {
	case c.Operation == "v1-list-project-members":
		return r.projectMembers(ctx, c)
	case strings.Contains(c.Operation, "lite"):
		return r.lifecycleLite(ctx, c)
	case strings.HasPrefix(c.Operation, "v1-list-"):
		return apicontract.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(`+lifecycleUserSQL+` ORDER BY u.created_at DESC),'[]'::jsonb) FROM users u WHERE u.id IN (SELECT pm.member_id FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE w.slug=$1 AND pm.project_id=$2 AND pm.deleted_at IS NULL)`, c.Slug, c.ProjectID)
	case strings.HasPrefix(c.Operation, "v1-detail-"):
		value, err := apicontract.JSON(ctx, r.database, `SELECT `+lifecycleUserSQL+` FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id JOIN users u ON u.id=pm.member_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.deleted_at IS NULL`, c.MemberID, c.ProjectID, c.Slug)
		if errors.Is(err, pgx.ErrNoRows) {
			err = lifecycleMissing(false)
		}
		return value, err

	}
	return nil, fmt.Errorf("unknown membership lifecycle operation %q", c.Operation)
}

var lifecycleUserSQL = `(jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'avatar',u.avatar,'display_name',u.display_name)||` + apicontract.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `)`

func (r *LifecycleRepository) lifecycleLite(ctx context.Context, c LifecycleCommand) (any, error) {
	var project bool
	if err := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, c.ProjectID, c.Slug).Scan(&project); err != nil {
		return nil, err
	}
	if !project {
		return nil, lifecycleFailure(404, "error", "Provided project does not exist")
	}
	limit, page, offset, err := apicontract.PageWindow(apicontract.Query{Params: c.Params, Present: c.Present})
	if err != nil {
		return nil, err
	}
	value, err := apicontract.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(row.value ORDER BY row.created_at DESC),'[]'::jsonb) FROM (SELECT (CASE WHEN u.id IS NULL THEN jsonb_build_object('avatar',NULL,'avatar_url',NULL) ELSE `+lifecycleUserSQL+`||jsonb_build_object('is_bot',u.is_bot) END)||jsonb_build_object('role',pm.role,'is_active',pm.is_active) AS value,pm.created_at FROM project_members pm LEFT JOIN users u ON u.id=pm.member_id JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.project_id=$1 AND w.slug=$2 AND pm.deleted_at IS NULL ORDER BY pm.created_at DESC) row`, c.ProjectID, c.Slug)
	if err != nil {
		return nil, err
	}
	all := value.([]any)
	start := min(offset, len(all))
	end := min(start+limit, len(all))
	return apicontract.Envelope(all[start:end], len(all), limit, page), nil
}
func lifecycleDBError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && strings.HasPrefix(pg.Code, "23") {
		return lifecycleFailure(400, "error", "The payload is not valid")
	}
	return err
}

const memberFilters = `{"priority":null,"state":null,"state_group":null,"assignees":null,"created_by":null,"labels":null,"start_date":null,"target_date":null,"subscriber":null}`
const memberDisplayFilters = `{"group_by":null,"order_by":"-created_at","type":null,"sub_issue":true,"show_empty_groups":true,"layout":"list","calendar_date_range":""}`
const memberDefaultProps = `{"filters":` + memberFilters + `,"display_filters":` + memberDisplayFilters + `}`
const memberPreferences = `{"pages":{"block_display":true},"navigation":{"default_tab":"work_items","hide_in_more_menu":[]}}`
const memberDisplayProperties = `{"assignee":true,"attachment_count":true,"created_on":true,"due_date":true,"estimate":true,"key":true,"labels":true,"link":true,"priority":true,"start_date":true,"state":true,"sub_issue_count":true,"updated_on":true}`

const projectMembershipJSON = `jsonb_build_object('id',pm.id,'member',pm.member_id,'role',pm.role,'project',pm.project_id,'original_role',pm.role,'is_active',pm.is_active,'created_at',pm.created_at)`

func projectMemberRepresentation(ctx context.Context, db database, member string) (any, error) {
	return apicontract.JSON(ctx, db, `SELECT `+projectMembershipJSON+` FROM project_members pm WHERE pm.id=$1`, member)
}
