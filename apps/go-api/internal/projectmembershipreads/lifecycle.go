package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strings"
	storeuuid "uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
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
	publisher TaskPublisher
	baseHost  string
	reader    database
}

func NewLifecycleRepository(db database, publisher TaskPublisher, baseHost string) (*LifecycleRepository, error) {
	return NewLifecycleRepositoryWithReaderWriter(db, db, publisher, baseHost)
}
func NewLifecycleRepositoryWithReaderWriter(reader, writer database, publisher TaskPublisher, baseHost string) (*LifecycleRepository, error) {
	if publisher == nil || reader == nil {
		return nil, errors.New("membership reader and task publisher are required")
	}
	r, err := NewRepository(writer)
	if err != nil {
		return nil, err
	}
	return &LifecycleRepository{Repository: r, publisher: publisher, baseHost: baseHost, reader: reader}, nil
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
	return &djangoapi.HTTPError{Status: status, Body: map[string]any{field: message}}
}
func lifecycleMissing(legacy bool) error {
	if legacy {
		return lifecycleFailure(404, "error", "The required object does not exist.")
	}
	return lifecycleFailure(404, "error", "The requested resource does not exist.")
}
func (r *LifecycleRepository) AuthorizeLifecycle(ctx context.Context, c LifecycleCommand) error {
	r = r.forCommand(c)
	// Session views run only IsAuthenticated during initial(). Their action
	// decorators run after timezone conversion and request initialization.
	if strings.HasPrefix(c.Operation, "legacy-") {
		return nil
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
	r = r.forCommand(c)
	legacy := strings.HasPrefix(c.Operation, "legacy-")
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
	if legacy {
		if err := r.authorizeLegacyAction(ctx, c); err != nil {
			return nil, err
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
		return djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(`+lifecycleUserSQL+` ORDER BY u.created_at DESC),'[]'::jsonb) FROM users u WHERE u.id IN (SELECT pm.member_id FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE w.slug=$1 AND pm.project_id=$2 AND pm.deleted_at IS NULL)`, c.Slug, c.ProjectID)
	case strings.HasPrefix(c.Operation, "v1-detail-"):
		value, err := djangoapi.JSON(ctx, r.database, `SELECT `+lifecycleUserSQL+` FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id JOIN users u ON u.id=pm.member_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.deleted_at IS NULL`, c.MemberID, c.ProjectID, c.Slug)
		if errors.Is(err, pgx.ErrNoRows) {
			err = lifecycleMissing(false)
		}
		return value, err
	case strings.HasPrefix(c.Operation, "v1-create-"):
		return r.lifecycleV1Write(ctx, c, true)
	case strings.HasPrefix(c.Operation, "v1-patch-"):
		return r.lifecycleV1Write(ctx, c, false)
	case strings.HasPrefix(c.Operation, "v1-delete-"):
		return nil, r.deactivate(ctx, c, false)
	case c.Operation == "legacy-create-members":
		return r.lifecycleBulkCreate(ctx, c)
	case c.Operation == "legacy-patch-members":
		return r.lifecycleLegacyPatch(ctx, c)
	case c.Operation == "legacy-delete-members":
		return nil, r.deactivate(ctx, c, true)
	case c.Operation == "legacy-leave-members":
		return nil, r.lifecycleLeave(ctx, c)
	}
	return nil, fmt.Errorf("unknown membership lifecycle operation %q", c.Operation)
}

var lifecycleUserSQL = `(jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'avatar',u.avatar,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `)`

func (r *LifecycleRepository) lifecycleLite(ctx context.Context, c LifecycleCommand) (any, error) {
	var project bool
	if err := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, c.ProjectID, c.Slug).Scan(&project); err != nil {
		return nil, err
	}
	if !project {
		return nil, lifecycleFailure(404, "error", "Provided project does not exist")
	}
	limit, page, offset, err := djangoapi.PageWindow(djangoapi.Query{Params: c.Params, Present: c.Present})
	if err != nil {
		return nil, err
	}
	value, err := djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(row.value ORDER BY row.created_at DESC),'[]'::jsonb) FROM (SELECT (CASE WHEN u.id IS NULL THEN jsonb_build_object('avatar',NULL,'avatar_url',NULL) ELSE `+lifecycleUserSQL+`||jsonb_build_object('is_bot',u.is_bot) END)||jsonb_build_object('role',pm.role,'is_active',pm.is_active) AS value,pm.created_at FROM project_members pm LEFT JOIN users u ON u.id=pm.member_id JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.project_id=$1 AND w.slug=$2 AND pm.deleted_at IS NULL ORDER BY pm.created_at DESC) row`, c.ProjectID, c.Slug)
	if err != nil {
		return nil, err
	}
	all := value.([]any)
	start := min(offset, len(all))
	end := min(start+limit, len(all))
	return djangoapi.Envelope(all[start:end], len(all), limit, page), nil
}
func (r *LifecycleRepository) authorizeLegacyAction(ctx context.Context, c LifecycleCommand) error {
	var allowed bool
	roles := []int{5, 15, 20}
	if c.Operation == "legacy-create-members" || c.Operation == "legacy-delete-members" {
		roles = []int{20}
	}
	err := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3 AND pm.is_active AND pm.deleted_at IS NULL AND
 (pm.role=ANY($4) OR EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.member_id=$1 AND wm.workspace_id=w.id AND wm.is_active AND wm.role=20 AND wm.deleted_at IS NULL)))`, c.UserID, c.Slug, c.ProjectID, roles).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return lifecycleFailure(403, "error", "You don't have the required permissions.")
	}
	return nil
}
func lifecycleParse(c LifecycleCommand) (djangoinput.Input, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return input, err
	}
	if input.Null {
		return input, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"No data provided"}}}
	}
	if input.Kind != "" {
		return input, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	return input, nil
}
func lifecycleRole(v any) (int, string) {
	if v == nil {
		return 0, "This field may not be null."
	}
	value := djangoinput.ScalarString(v)
	switch value {
	case "5":
		return 5, ""
	case "15":
		return 15, ""
	case "20":
		return 20, ""
	}
	return 0, fmt.Sprintf("%q is not a valid choice.", value)
}

func (r *LifecycleRepository) lifecycleV1Write(ctx context.Context, c LifecycleCommand, create bool) (any, error) {
	var currentMember *string
	role := 5
	if !create {
		err := r.database.QueryRow(ctx, `SELECT pm.member_id::text,pm.role FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.deleted_at IS NULL`, c.MemberID, c.ProjectID, c.Slug).Scan(&currentMember, &role)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, lifecycleMissing(false)
		}
		if err != nil {
			return nil, err
		}
	}
	input, err := lifecycleParse(c)
	if err != nil {
		return nil, err
	}
	failures := map[string]any{}
	if value, ok := input.Fields["member"]; ok {
		if _, boolean := value.(bool); boolean {
			failures["member"] = []string{"Incorrect type. Expected pk value, received bool."}
		} else if value == nil {
			failures["member"] = []string{"This field may not be null."}
		} else {
			parsed, e := djangoinput.UUID(value)
			if e != nil {
				failures["member"] = []string{"“" + djangoinput.ScalarString(value) + "” is not a valid UUID."}
			} else {
				memberID := parsed.String()
				currentMember = &memberID
				var exists, inWorkspace bool
				e = r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1),EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.deleted_at IS NULL)`, *currentMember, c.Slug).Scan(&exists, &inWorkspace)
				if e != nil {
					return nil, e
				}
				if !exists {
					failures["member"] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", *currentMember)}
				} else if !inWorkspace {
					failures["member"] = []string{"Member not found in workspace"}
				}
			}
		}
	} else if create {
		failures["member"] = []string{"This field is required."}
	}
	if value, ok := input.Fields["role"]; ok && (!input.Form || value != "") {
		parsed, failure := lifecycleRole(value)
		if failure != "" {
			failures["role"] = []string{failure}
		} else {
			role = parsed
		}
	}
	if len(failures) > 0 {
		return nil, &djangoapi.HTTPError{Status: 400, Body: failures}
	}
	if create {
		// Django creates the user property in ProjectMember.save before inserting
		// the membership. A duplicate membership can therefore retain this effect.
		if err = r.insertLifecycleProperty(ctx, c, *currentMember, false); err != nil {
			return nil, lifecycleDBError(err)
		}
		c.MemberID = uuid.NewString()
		_, err = r.database.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,comment,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,(SELECT workspace_id FROM projects WHERE id=$2),$3,$4,NULL,$5,$5,$6,65535,true,clock_timestamp(),clock_timestamp(),$7,NULL)`, c.MemberID, c.ProjectID, currentMember, role, memberDefaultProps, memberPreferences, c.UserID)
	} else {
		_, err = r.database.Exec(ctx, `UPDATE project_members SET member_id=$2,role=$3,updated_at=clock_timestamp(),updated_by_id=$4,workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id) WHERE id=$1`, c.MemberID, currentMember, role, c.UserID)
	}
	if err != nil {
		return nil, lifecycleDBError(err)
	}
	return map[string]any{"id": c.MemberID, "member": currentMember, "role": role}, nil
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

func (r *LifecycleRepository) insertLifecycleProperty(ctx context.Context, c LifecycleCommand, user string, bulk bool) error {
	conflict := ""
	var creator any = c.UserID
	if bulk {
		conflict = " ON CONFLICT DO NOTHING"
		creator = nil
	}
	_, err := r.database.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,(SELECT workspace_id FROM projects WHERE id=$2),$3,$4,$5,$6,'{}',$7,COALESCE((SELECT MIN(sort_order)-10000 FROM project_user_properties WHERE user_id=$3 AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$2) AND deleted_at IS NULL),65535),clock_timestamp(),clock_timestamp(),$8,NULL)`+conflict, uuid.NewString(), c.ProjectID, user, memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences, creator)
	return err
}
func (r *LifecycleRepository) deactivate(ctx context.Context, c LifecycleCommand, legacy bool) error {
	var targetUser *string
	var targetRole, requesterRole int
	err := r.database.QueryRow(ctx, `SELECT pm.member_id::text,pm.role FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id LEFT JOIN users u ON u.id=pm.member_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.deleted_at IS NULL AND (NOT $4 OR (pm.is_active AND NOT u.is_bot))`, c.MemberID, c.ProjectID, c.Slug, legacy).Scan(&targetUser, &targetRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return lifecycleMissing(legacy)
	}
	if err != nil {
		return err
	}
	if legacy {
		if targetUser != nil && *targetUser == c.UserID {
			return lifecycleFailure(400, "error", "You cannot remove yourself from the workspace. Please use leave workspace")
		}
		err = r.database.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, c.ProjectID, c.UserID).Scan(&requesterRole)
		if err != nil {
			return err
		}
		if requesterRole < targetRole {
			return lifecycleFailure(400, "error", "You cannot remove a user having role higher than you")
		}
	}
	_, err = r.database.Exec(ctx, `UPDATE project_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$2,workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id) WHERE id=$1`, c.MemberID, c.UserID)
	return err
}
func (r *LifecycleRepository) lifecycleLeave(ctx context.Context, c LifecycleCommand) error {
	var id string
	var role, admins int
	err := r.database.QueryRow(ctx, `SELECT pm.id::text,pm.role,(SELECT count(*) FROM project_members admins WHERE admins.project_id=pm.project_id AND admins.workspace_id=pm.workspace_id AND admins.role=20 AND admins.is_active AND admins.deleted_at IS NULL) FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.project_id=$1 AND pm.member_id=$2 AND w.slug=$3 AND pm.is_active AND pm.deleted_at IS NULL`, c.ProjectID, c.UserID, c.Slug).Scan(&id, &role, &admins)
	if errors.Is(err, pgx.ErrNoRows) {
		return lifecycleMissing(true)
	}
	if err != nil {
		return err
	}
	if role == 20 && admins <= 1 {
		return lifecycleFailure(400, "error", "You cannot leave the project as your the only admin of the project you will have to either delete the project or create an another admin")
	}
	_, err = r.database.Exec(ctx, `UPDATE project_members SET is_active=false,updated_at=clock_timestamp(),updated_by_id=$2,workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id) WHERE id=$1`, id, c.UserID)
	return err
}

var ErrLifecycleRendering = errors.New("Django cannot render a nonfinite membership sort_order")

func (r *LifecycleRepository) lifecycleLegacyRepresentation(ctx context.Context, c LifecycleCommand) (any, error) {
	// Reuse the complete legacy read serializer, including a just-deleted target.
	value, err := djangoapi.JSON(ctx, r.database, `SELECT (to_jsonb(pm)-'workspace_id'-'project_id'-'member_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('created_by',pm.created_by_id,'updated_by',pm.updated_by_id,'member',jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||`+djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url")+`,'project',jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'logo_props',p.logo_props,'description',p.description)||`+djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url")+`,'workspace',jsonb_build_object('id',w.id,'name',w.name,'slug',w.slug)||`+djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url")+`) FROM project_members pm JOIN users u ON u.id=pm.member_id JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.id=$1`, c.MemberID)
	if err != nil {
		return nil, err
	}
	if order, ok := value.(map[string]any)["sort_order"].(string); ok && (order == "NaN" || order == "Infinity" || order == "-Infinity") {
		return nil, ErrLifecycleRendering
	}
	return value, nil
}

// Python's `in [5,15]` compares numeric values but does not coerce strings.
func lifecycleNumericRole(value any) (float64, bool) {
	switch value := value.(type) {
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	case int:
		return float64(value), true
	case float64:
		return value, true
	case bool:
		if value {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}
func lifecyclePythonInt(value any) (*big.Int, error) {
	switch value := value.(type) {
	case string:
		return pythoncompat.DecimalInteger(value)
	case bool:
		if value {
			return big.NewInt(1), nil
		}
		return big.NewInt(0), nil
	case json.Number:
		if !strings.ContainsAny(string(value), ".eE") {
			return pythoncompat.DecimalInteger(string(value))
		}
		number, err := value.Float64()
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, errors.New("invalid Python integer")
		}
		result, _ := new(big.Float).SetFloat64(number).Int(nil)
		return result, nil
	case int:
		return big.NewInt(int64(value)), nil
	}
	return nil, errors.New("invalid Python integer")
}

func lifecycleBoolean(value any) (bool, string) {
	if value == nil {
		return false, "This field may not be null."
	}
	if number, numeric := lifecycleNumericRole(value); numeric {
		if number == 0 {
			return false, ""
		}
		if number == 1 {
			return true, ""
		}
		return false, "Must be a valid boolean."
	}
	switch strings.ToLower(djangoinput.ScalarString(value)) {
	case "true", "1", "yes", "on", "t", "y":
		return true, ""
	case "false", "0", "no", "off", "f", "n":
		return false, ""
	}
	return false, "Must be a valid boolean."
}
