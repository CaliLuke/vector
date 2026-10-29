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
	"time"
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

func (r *LifecycleRepository) lifecycleBulkCreate(ctx context.Context, c LifecycleCommand) (any, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return nil, err
	}
	if input.Null || input.Kind != "" {
		return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
	}
	var project bool
	if scanErr := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, c.ProjectID, c.Slug).Scan(&project); scanErr != nil {
		return nil, scanErr
	}
	if !project {
		return nil, lifecycleMissing(true)
	}
	if raw, present := input.Fields["members"]; present {
		if _, ok := raw.([]any); !ok {
			switch value := raw.(type) {
			case string:
				if value == "" {
					return nil, lifecycleFailure(400, "error", "At least one member is required")
				}
			case map[string]any:
				if len(value) == 0 {
					return nil, lifecycleFailure(400, "error", "At least one member is required")
				}
			}
			return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
	}
	members, _ := input.Fields["members"].([]any)
	if len(members) == 0 {
		return nil, lifecycleFailure(400, "error", "At least one member is required")
	}
	type memberRole struct {
		identity any
		role     any
		id       string
	}
	roles := map[string]*memberRole{}
	ordered := []*memberRole{}
	for _, item := range members {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		key, hashable := lifecycleBulkIdentityKey(m["member_id"])
		if !hashable {
			return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		entry, exists := roles[key]
		if !exists {
			entry = &memberRole{identity: m["member_id"]}
			roles[key] = entry
			ordered = append(ordered, entry)
		}
		entry.role = m["role"]
	}
	// Python constructs the complete dictionary before any ORM lookup. Equal
	// numeric keys replace the role but retain the first key and its position.
	for _, entry := range ordered {
		id, membershipErr := lifecycleBulkMemberUUID(entry.identity)
		if membershipErr != nil {
			return nil, membershipErr
		}
		entry.id = id
		var role int
		membershipErr = r.database.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL`, c.Slug, entry.id).Scan(&role)
		if errors.Is(membershipErr, pgx.ErrNoRows) {
			return nil, lifecycleMissing(true)
		}
		if membershipErr != nil {
			return nil, membershipErr
		}
		submitted, numeric := lifecycleNumericRole(entry.role)
		if role == 20 && numeric && (submitted == 5 || submitted == 15) {
			return nil, lifecycleFailure(400, "error", "You cannot add a user with role lower than the workspace role")
		}
		if role == 5 && numeric && (submitted == 15 || submitted == 20) {
			return nil, lifecycleFailure(400, "error", "You cannot add a user with role higher than the workspace role")
		}
	}
	// Match Django's independent bulk_update, bulk_create and property
	// bulk_create atomic calls. Earlier completed calls remain committed.
	// The ORM's next __in lookup prepares every original value, even values
	// whose equal dictionary keys were collapsed during the role checks.
	queryIDs := make([]string, 0, len(members))
	for _, item := range members {
		id, lifecycleBulkMemberUUIDErr := lifecycleBulkMemberUUID(item.(map[string]any)["member_id"])
		if lifecycleBulkMemberUUIDErr != nil {
			return nil, lifecycleBulkMemberUUIDErr
		}
		queryIDs = append(queryIDs, id)
	}
	existing, err := djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(member_id::text),'[]'::jsonb) FROM project_members WHERE project_id=$1 AND member_id::text=ANY($2) AND deleted_at IS NULL`, c.ProjectID, queryIDs)
	if err != nil {
		return nil, err
	}
	updates := make([]map[string]any, 0, len(existing.([]any)))
	for _, item := range existing.([]any) {
		id := item.(string)
		key, _ := lifecycleBulkIdentityKey(id)
		entry, exists := roles[key]
		if !exists {
			return nil, lifecycleFailure(400, "error", "The required key does not exist.")
		}
		role, lifecycleDatabaseRoleErr := lifecycleDatabaseRole(entry.role)
		if lifecycleDatabaseRoleErr != nil {
			return nil, lifecycleDatabaseRoleErr
		}
		updates = append(updates, map[string]any{"member_id": id, "role": role})
	}
	updateJSON, _ := json.Marshal(updates)
	_, err = r.database.Exec(ctx, `UPDATE project_members pm SET role=incoming.role,is_active=true FROM jsonb_to_recordset($2::jsonb) AS incoming(member_id uuid,role smallint) WHERE pm.project_id=$1 AND pm.member_id=incoming.member_id AND pm.deleted_at IS NULL`, c.ProjectID, string(updateJSON))
	if err != nil {
		return nil, lifecycleDBError(err)
	}
	inserts := make([]map[string]any, 0, len(members))
	properties := make([]map[string]any, 0, len(members))
	for _, item := range members {
		m := item.(map[string]any)
		key, _ := lifecycleBulkIdentityKey(m["member_id"])
		id := roles[key].id
		role, exists := m["role"]
		if !exists {
			role = 5
		}
		role, err = lifecycleDatabaseRole(role)
		if err != nil {
			return nil, err
		}
		inserts = append(inserts, map[string]any{"id": uuid.NewString(), "member_id": id, "role": role})
		properties = append(properties, map[string]any{"id": uuid.NewString(), "user_id": id, "lookup_key": djangoinput.ScalarString(m["member_id"])})
	}
	insertJSON, _ := json.Marshal(inserts)
	_, err = r.database.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,comment,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id,updated_by_id)
 SELECT incoming.id,$1,(SELECT workspace_id FROM projects WHERE id=$1),incoming.member_id,incoming.role,NULL,$3,$3,$4,65535,true,clock_timestamp(),clock_timestamp(),NULL,NULL FROM jsonb_to_recordset($2::jsonb) AS incoming(id uuid,member_id uuid,role smallint) ON CONFLICT DO NOTHING`, c.ProjectID, string(insertJSON), memberDefaultProps, memberPreferences)
	if err != nil {
		return nil, lifecycleDBError(err)
	}
	propertyJSON, _ := json.Marshal(properties)
	_, err = r.database.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id,updated_by_id)
 SELECT incoming.id,$1,(SELECT workspace_id FROM projects WHERE id=$1),incoming.user_id,$3,$4,$5,'{}',$6,COALESCE((SELECT MIN(sort_order)-10000 FROM project_user_properties WHERE user_id::text=incoming.lookup_key AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$1) AND deleted_at IS NULL),65535),clock_timestamp(),clock_timestamp(),NULL,NULL
 FROM jsonb_to_recordset($2::jsonb) AS incoming(id uuid,user_id uuid,lookup_key text) ON CONFLICT DO NOTHING`, c.ProjectID, string(propertyJSON), memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences)
	if err != nil {
		return nil, lifecycleDBError(err)
	}

	value, err := djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',id,'member',member_id,'role',role,'project',project_id,'original_role',role,'created_at',created_at) ORDER BY created_at DESC),'[]'::jsonb) FROM project_members WHERE project_id=$1 AND member_id::text=ANY($2) AND deleted_at IS NULL`, c.ProjectID, queryIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range value.([]any) {
		row := item.(map[string]any)
		if err = r.publisher.Task(ctx, "plane.bgtasks.project_add_user_email_task.project_add_user_email", []any{r.baseHost, row["id"], c.UserID}, nil); err != nil {
			return nil, err
		}
	}
	return value, nil
}
func (r *LifecycleRepository) lifecycleLegacyPatch(ctx context.Context, c LifecycleCommand) (any, error) {
	var targetUser *string
	var targetRole, targetWorkspace, requesterRole, requesterWorkspace int
	err := r.database.QueryRow(ctx, `SELECT pm.member_id::text,pm.role FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.is_active AND pm.deleted_at IS NULL`, c.MemberID, c.ProjectID, c.Slug).Scan(&targetUser, &targetRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, lifecycleMissing(true)
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range []struct {
		user   *string
		target *int
	}{{targetUser, &targetWorkspace}, {&c.UserID, &requesterWorkspace}} {
		err = r.database.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL`, c.Slug, entry.user).Scan(entry.target)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, lifecycleMissing(true)
		}
		if err != nil {
			return nil, err
		}
	}
	admin := requesterWorkspace == 20
	if targetUser != nil && *targetUser == c.UserID && !admin {
		return nil, lifecycleFailure(400, "error", "You cannot update your own role")
	}
	err = r.database.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, c.ProjectID, c.UserID).Scan(&requesterRole)
	if err != nil {
		return nil, err
	}
	input, err := lifecycleLegacyPatchInput(c)
	if err != nil {
		return nil, err
	}
	if value, ok := input.Fields["role"]; ok {
		if requesterRole < 20 && !admin {
			return nil, lifecycleFailure(403, "error", "You do not have permission to update roles")
		}
		if targetRole >= requesterRole && !admin {
			return nil, lifecycleFailure(403, "error", "You cannot update the role of a member with a role equal to or higher than your own")
		}
		if input.Kind != "" {
			return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		number, e := lifecyclePythonInt(value)
		if e != nil {
			return nil, lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		if number.Cmp(big.NewInt(int64(requesterRole))) >= 0 && !admin {
			return nil, lifecycleFailure(403, "error", "You cannot assign a role equal to or higher than your own")
		}
		if targetWorkspace == 5 && (number.Cmp(big.NewInt(15)) == 0 || number.Cmp(big.NewInt(20)) == 0) {
			return nil, lifecycleFailure(400, "error", "You cannot add a user with role higher than the workspace role")
		}
	}
	if _, ok := input.Fields["is_active"]; ok {
		if requesterRole < 20 && !admin {
			return nil, lifecycleFailure(403, "error", "You do not have permission to update member status")
		}
		if targetRole >= requesterRole && !admin {
			return nil, lifecycleFailure(403, "error", "You cannot update the status of a member with a role equal to or higher than your own")
		}
	}
	if input.Kind != "" {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	failures := map[string]any{}
	assignments := []string{"updated_at=clock_timestamp()", "updated_by_id=$2", "workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id)"}
	args := []any{c.MemberID, c.UserID}
	for _, field := range []string{"role", "is_active", "comment", "sort_order", "deleted_at", "view_props", "default_props", "preferences", "created_by", "updated_by"} {
		value, ok := input.Fields[field]
		if !ok {
			continue
		}
		// DRF's HTML get_value normalization belongs after the view's raw
		// role/status guards, which intentionally see the submitted form text.
		if input.Form && value == "" {
			switch field {
			case "role", "sort_order", "is_active":
				continue
			case "deleted_at", "created_by", "updated_by":
				value = nil
			}
		}
		column := field
		var failure string
		switch field {
		case "role":
			value, failure = lifecycleRole(value)
		case "is_active":
			value, failure = lifecycleBoolean(value)
		case "comment":
			if value != nil {
				var es []string
				value, es = djangoinput.TextField(value, true, 0, input.Surrogates[field])
				if len(es) > 0 {
					failure = es[0]
				}
			}
		case "sort_order":
			if value == nil {
				failure = "This field may not be null."
			} else {
				value, failure = djangoinput.FloatField(value)
			}
		case "deleted_at":
			if value != nil {
				location := time.UTC
				if c.Timezone != "" {
					if zone, e := time.LoadLocation(c.Timezone); e == nil {
						location = zone
					}
				}
				parsed, e := pythoncompat.DjangoDateTime(djangoinput.ScalarString(value), location)
				if e != nil {
					failure = "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."
				} else {
					value = parsed
				}
			}
		case "view_props", "default_props", "preferences":
			if value == nil {
				failure = "This field may not be null."
			} else {
				if input.Form {
					decoder := json.NewDecoder(strings.NewReader(djangoinput.ScalarString(value)))
					decoder.UseNumber()
					if decodeErr := decoder.Decode(&value); decodeErr != nil {
						failure = "Value must be valid JSON."
					} else if trailingDataErr := decoder.Decode(new(any)); trailingDataErr != io.EOF {
						failure = "Value must be valid JSON."
					}
					if failure != "" {
						break
					}
				}
				raw, e := json.Marshal(value)
				if e != nil {
					return nil, e
				}
				value = string(raw)
			}
		case "created_by", "updated_by":
			column += "_id"
			if _, boolean := value.(bool); boolean {
				failure = "Incorrect type. Expected pk value, received bool."
			} else if value != nil {
				parsed, e := djangoinput.UUID(value)
				if e != nil {
					failure = "“" + djangoinput.ScalarString(value) + "” is not a valid UUID."
				} else {
					value = parsed.String()
					var exists bool
					if scanErr := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, value).Scan(&exists); scanErr != nil {
						return nil, scanErr
					}
					if !exists {
						failure = fmt.Sprintf("Invalid pk %q - object does not exist.", value)
					}
				}
			}
		}
		if failure != "" {
			failures[field] = []string{failure}
			continue
		}
		if field == "updated_by" {
			continue
		}
		args = append(args, value)
		assignments = append(assignments, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if len(failures) > 0 {
		return nil, &djangoapi.HTTPError{Status: 400, Body: failures}
	}
	// BaseModel.save always replaces updated_by with the current user.
	for i := range assignments {
		if strings.HasPrefix(assignments[i], "updated_by_id=") {
			assignments[i] = "updated_by_id=$2"
		}
	}
	_, err = r.database.Exec(ctx, `UPDATE project_members SET `+strings.Join(assignments, ",")+` WHERE id=$1`, args...)
	if err != nil {
		return nil, lifecycleDBError(err)
	}
	return r.lifecycleLegacyRepresentation(ctx, c)
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

func lifecycleBulkMemberUUID(value any) (string, error) {
	if value == nil {
		return "", lifecycleMissing(true)
	}
	if boolean, ok := value.(bool); ok {
		value = json.Number("0")
		if boolean {
			value = json.Number("1")
		}
	}
	id, err := djangoinput.UUID(value)
	if err != nil {
		return "", lifecycleFailure(400, "error", "Please provide valid detail")
	}
	return id.String(), nil
}

// JSON object/list values are unhashable in Python. Its numeric dictionary
// keys compare by value, including bool/int/float equality, unlike strings.
func lifecycleBulkIdentityKey(value any) (string, bool) {
	switch value := value.(type) {
	case nil:
		return "none", true
	case string:
		return "str:" + value, true
	case bool:
		if value {
			return "number:1", true
		}
		return "number:0", true
	case json.Number:
		if !strings.ContainsAny(string(value), ".eE") {
			integer, ok := new(big.Int).SetString(string(value), 10)
			if !ok {
				return "", false
			}
			return "number:" + integer.String(), true
		}
		number, err := value.Float64()
		if err != nil && !math.IsInf(number, 0) {
			return "", false
		}
		if math.IsInf(number, 1) {
			return "number:inf", true
		}
		if math.IsInf(number, -1) {
			return "number:-inf", true
		}
		return "number:" + new(big.Rat).SetFloat64(number).RatString(), true
	}
	return "", false
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

func lifecycleDatabaseRole(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	integer, err := lifecyclePythonInt(value)
	if err != nil {
		return nil, err
	}
	return json.Number(integer.String()), nil
}

// The legacy view performs Python containment checks before passing data to
// its serializer. Retain those guard branches for non-dictionary JSON too.
func lifecycleLegacyPatchInput(c LifecycleCommand) (djangoinput.Input, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return input, err
	}
	if input.Kind == "" {
		return input, nil
	}
	if input.Null || (input.Kind != "list" && input.Kind != "str") {
		return input, lifecycleFailure(500, "error", "Something went wrong please try again later")
	}
	var raw any
	if err := json.Unmarshal(input.Raw, &raw); err != nil {
		return input, err
	}
	for _, field := range []string{"role", "is_active"} {
		present := false
		switch value := raw.(type) {
		case string:
			present = strings.Contains(value, field)
		case []any:
			for _, item := range value {
				if text, ok := item.(string); ok && text == field {
					present = true
					break
				}
			}
		}
		if present {
			input.Fields[field] = nil
		}
	}
	return input, nil
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
