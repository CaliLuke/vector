package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	membership "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type LifecycleEffects interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}
type lifecycleDatabase interface {
	database
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type LifecycleRepository struct {
	*Repository
	writer               lifecycleDatabase
	effects              LifecycleEffects
	baseHost             string
	coverAssets          ProjectCoverAssetValidator
	transactionalEffects bool
}

// ProjectCoverAssetValidator reuses the asset owner's scoped lookup without
// importing its transport/service package into the project domain.
type ProjectCoverAssetValidator func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error)

func NewLifecycleRepository(db lifecycleDatabase, effects LifecycleEffects, baseHost string, coverAssets ProjectCoverAssetValidator) (*LifecycleRepository, error) {
	if db == nil || effects == nil || coverAssets == nil {
		return nil, errors.New("project lifecycle database, effects and cover asset validator required")
	}
	reads, err := NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &LifecycleRepository{Repository: reads, writer: db, effects: effects, baseHost: baseHost, coverAssets: coverAssets}, nil
}
func projectFailure(status int, body map[string]any) error {
	return &djangoapi.HTTPError{Status: status, Body: body}
}

func normalizeProjectIdentifier(value string) string {
	return cases.Upper(language.Und).String(strings.TrimSpace(value))
}

func lifecycleRequestedData(input djangoinput.Input) (any, error) {
	if input.Form || len(input.Raw) == 0 {
		return input.Fields, nil
	}
	normalized, err := pythoncompat.NormalizeDecodedJSONNumbers(input.Raw)
	if err != nil {
		return nil, err
	}
	// Keep the normalized object encoded until Celery serialization: decoding it
	// here would merge lone-surrogate keys with U+FFFD and replace their values.
	return json.RawMessage(normalized), nil
}

// AuthorizeProjectWrite keeps the source route policies distinct after the
// approved shared active-workspace-membership precondition.
func (r *LifecycleRepository) AuthorizeProjectWrite(ctx context.Context, user, slug, project, operation string, app bool) error {
	if !app && operation != "create" && project == "" {
		// Unsupported unsafe collection methods have no project scope. Django
		// finds no matching ProjectMember; do not bind an empty UUID to SQL.
		return ErrForbidden
	}
	var allowed bool
	rolePolicy := "wm.role IN (15,20)"
	if operation != "create" {
		pm := `EXISTS(SELECT 1 FROM project_members pm WHERE pm.member_id=$1 AND pm.project_id=$3 AND pm.workspace_id=w.id AND pm.is_active AND pm.deleted_at IS NULL`
		if app && (operation == "patch" || operation == "put" || operation == "delete") {
			rolePolicy = "(wm.role=20 OR " + pm + ` AND pm.role=20))`
		} else if app {
			rolePolicy = pm + ` AND (pm.role IN (15,20) OR wm.role=20))`
		} else if operation == "archive" {
			rolePolicy = "wm.role IN (15,20)" // v1 POST enters ProjectBasePermission's create branch.
		} else {
			rolePolicy = pm + ` AND (pm.role=20 OR wm.role=20))`
		}
	}
	args := []any{user, slug}
	if strings.Contains(rolePolicy, "$3") {
		args = append(args, project)
	}
	err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE w.slug=$2 AND wm.member_id=$1 AND wm.is_active AND wm.deleted_at IS NULL AND `+rolePolicy+`)`, args...).Scan(&allowed)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	if app {
		return projectFailure(403, map[string]any{"error": "You don't have the required permissions."})
	}
	return ErrForbidden
}
func (r *LifecycleRepository) DeleteProject(ctx context.Context, user, slug, id string, app bool) error {
	var exists bool
	if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, id, slug).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if !app {
		if _, err := r.writer.Exec(ctx, `UPDATE user_favorites SET deleted_at=clock_timestamp() WHERE project_id=$1 AND entity_identifier=$1 AND entity_type='project' AND deleted_at IS NULL`, id); err != nil {
			return err
		}
	}
	if _, err := r.writer.Exec(ctx, `UPDATE projects SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, id, user); err != nil {
		return err
	}
	if err := r.effects.SoftDelete(ctx, "db", "project", id); err != nil {
		return err
	}
	if err := r.effects.Task(ctx, "plane.bgtasks.webhook_task.webhook_activity", []any{}, map[string]any{"event": "project", "verb": "deleted", "field": nil, "old_value": nil, "new_value": nil, "actor_id": user, "slug": slug, "current_site": r.baseHost, "event_id": id, "old_identifier": nil, "new_identifier": nil}); err != nil {
		return err
	}
	if app {
		if _, err := r.writer.Exec(ctx, `UPDATE deploy_boards SET deleted_at=clock_timestamp() WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND deleted_at IS NULL`, id, slug); err != nil {
			return err
		}
		_, err := r.writer.Exec(ctx, `UPDATE user_favorites SET deleted_at=clock_timestamp() WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND deleted_at IS NULL`, id, slug)
		return err
	}
	return nil
}

type projectExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertProjectRecordExec(ctx context.Context, db projectExecer, values map[string]any) error {
	columns := make([]string, 0, len(values))
	for column := range values {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}
	list := strings.Join(columns, ",")
	_, err = db.Exec(ctx, `INSERT INTO projects (`+list+`) SELECT `+list+` FROM jsonb_populate_record(NULL::projects,$1::jsonb)`, string(raw))
	return err
}
func (r *LifecycleRepository) CreateProject(ctx context.Context, user, slug string, input djangoinput.Input, app bool) (json.RawMessage, error) {
	var workspace, timezone string
	if err := r.writer.QueryRow(ctx, `SELECT id,timezone FROM workspaces WHERE slug=$1`, slug).Scan(&workspace, &timezone); err != nil {
		return nil, err
	}
	values, err := r.validateProjectFields(ctx, workspace, "", input, app, false, true)
	if err != nil {
		return nil, err
	}
	requestedData, err := lifecycleRequestedData(input)
	if err != nil {
		return nil, err
	}
	defaults := map[string]any{"id": uuid.NewString(), "workspace_id": workspace, "created_at": time.Now().UTC(), "updated_at": time.Now().UTC(), "created_by_id": user, "updated_by_id": nil, "deleted_at": nil, "name": "", "identifier": "", "description": "", "description_html": nil, "description_text": nil, "network": 2, "default_assignee_id": nil, "project_lead_id": nil, "emoji": nil, "icon_prop": nil, "module_view": false, "cycle_view": false, "issue_views_view": false, "page_view": true, "intake_view": false, "is_time_tracking_enabled": false, "is_issue_type_enabled": false, "guest_view_all_features": false, "cover_image": nil, "cover_image_asset_id": nil, "estimate_id": nil, "archive_in": 0, "close_in": 0, "logo_props": map[string]any{}, "default_state_id": nil, "archived_at": nil, "timezone": timezone, "external_source": nil, "external_id": nil}
	if !app {
		defaults["logo_props"] = map[string]any{"in_use": "icon", "icon": map[string]any{"name": projectDefaultIcons[rand.IntN(len(projectDefaultIcons))], "color": projectDefaultColors[rand.IntN(len(projectDefaultColors))]}}
	}
	for key, value := range values {
		defaults[key] = value
	}
	defaults["identifier"] = normalizeProjectIdentifier(defaults["identifier"].(string))
	var reserved bool
	if scanErr := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_identifiers WHERE workspace_id=$1 AND name=$2 AND deleted_at IS NULL)`, workspace, defaults["identifier"]).Scan(&reserved); scanErr != nil {
		return nil, scanErr
	}
	if reserved && !app {
		return nil, projectFailure(409, map[string]any{"identifier": "The project identifier is already taken"})
	}
	tx, err := r.writer.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = insertProjectRecordExec(ctx, tx, defaults); err != nil {
		return nil, r.projectIntegrityError(err, app)
	}
	id := defaults["id"].(string)
	if _, err = tx.Exec(ctx, `INSERT INTO project_identifiers(project_id,workspace_id,name,created_at,updated_at) VALUES($1,$2,$3,clock_timestamp(),clock_timestamp())`, id, workspace, defaults["identifier"]); err != nil {
		return nil, err
	}
	if err = membership.InitializeProjectAdministrator(ctx, tx, user, id, user); err != nil {
		return nil, err
	}
	if lead, ok := defaults["project_lead_id"].(string); ok && lead != user {
		if err = membership.InitializeProjectAdministrator(ctx, tx, user, id, lead); err != nil {
			return nil, err
		}
	}
	for i, state := range []struct{ name, color, group string }{{"Backlog", "#60646C", "backlog"}, {"Todo", "#60646C", "unstarted"}, {"In Progress", "#F59E0B", "started"}, {"Done", "#46A758", "completed"}, {"Cancelled", "#9AA4BC", "cancelled"}, {"Triage", "#4E5355", "triage"}} {
		if _, err = tx.Exec(ctx, `INSERT INTO states(id,workspace_id,project_id,name,color,"group",sequence,"default",description,slug,is_triage,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'','',false,clock_timestamp(),clock_timestamp(),$9)`, uuid.NewString(), workspace, id, state.name, state.color, state.group, 15000+i*10000, i == 0, user); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	result, err := r.projectWriteProjection(ctx, user, slug, id, app)
	if err != nil {
		return nil, err
	}
	effectErr := r.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "project", "model_id": id, "requested_data": requestedData, "current_instance": nil, "actor_id": user, "slug": slug, "origin": r.baseHost})
	if !app && !r.transactionalEffects && effectErr != nil {
		log.Print(ctx, log.KV{K: "msg", V: "Project create activity dispatch failed after commit"}, log.KV{K: "error_type", V: fmt.Sprintf("%T", effectErr)})
	}
	if (app || r.transactionalEffects) && effectErr != nil {
		return nil, effectErr
	} // API create's on_commit callback is robust.
	return result, nil
}
func (r *LifecycleRepository) projectIntegrityError(err error, app bool) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && !app {
		return projectFailure(409, map[string]any{"name": "The project name is already taken"})
	}
	return err
}
func (r *LifecycleRepository) projectWriteProjection(ctx context.Context, user, slug, id string, app bool) (json.RawMessage, error) {
	if !app {
		return r.apiProjectWriteResponse(ctx, user, slug, id)
	}
	var raw []byte
	err := r.writer.QueryRow(ctx, `WITH requester AS MATERIALIZED(SELECT set_config('TimeZone',COALESCE(NULLIF(user_timezone,''),'UTC'),true) AS user_timezone FROM users WHERE id=$1) SELECT (`+projectJSON+`)::jsonb || jsonb_build_object('anchor',(SELECT board.anchor FROM deploy_boards board WHERE board.entity_name='project' AND board.entity_identifier=project.id AND board.workspace_id=project.workspace_id AND board.deleted_at IS NULL LIMIT 1)) FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester WHERE workspace.slug=$2 AND project.id=$3 AND project.deleted_at IS NULL`, user, slug, id).Scan(&raw)
	if err != nil {
		return nil, err
	}
	return projectResponseFloatFields(raw)
}

func projectResponseFloatFields(raw json.RawMessage) (json.RawMessage, error) {
	values, err := pythoncompat.DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if number, ok := values["sort_order"].(json.Number); ok {
		value, err := number.Float64()
		if err != nil {
			return nil, err
		}
		values["sort_order"] = json.Number(pythoncompat.FloatString(value))
	}
	return json.Marshal(values)
}

func projectCurrentInstance(raw json.RawMessage) (string, error) {
	values, err := pythoncompat.DecodeObject(raw)
	if err != nil {
		return "", err
	}
	return pythoncompat.Dumps(values, raw)
}

func (r *LifecycleRepository) UpdateProject(ctx context.Context, user, slug, id string, input djangoinput.Input, app, replace bool) (json.RawMessage, error) {
	fields := input.Fields
	current, err := r.projectModelProjection(ctx, user, slug, id, app)
	if errors.Is(err, pgx.ErrNoRows) {
		if replace || !app {
			return nil, projectFailure(404, map[string]any{"error": "Project does not exist"})
		}
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var existing map[string]any
	if unmarshalErr := json.Unmarshal(current, &existing); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if existing["archived_at"] != nil {
		message := "Archived project cannot be updated"
		if app {
			message = "Archived projects cannot be updated"
		}
		return nil, projectFailure(400, map[string]any{"error": message})
	}
	if replace {
		boundaryErrors := map[string]any{}
		if workspace, present := fields["workspace"]; present && workspace != nil && fmt.Sprint(workspace) != fmt.Sprint(existing["workspace"]) {
			return nil, projectFailure(400, map[string]any{"workspace": "Project workspace cannot be changed."})
		}
		for _, key := range []string{"deleted_at", "archived_at", "created_by", "updated_by", "id"} {
			if _, present := fields[key]; present {
				boundaryErrors[key] = "This field cannot be changed."
			}
		}
		if len(boundaryErrors) > 0 {
			return nil, projectFailure(400, boundaryErrors)
		}
		allowed := map[string]bool{}
		for _, key := range []string{"name", "identifier", "description", "description_html", "project_lead", "default_assignee", "icon_prop", "emoji", "cover_image", "cover_image_asset", "module_view", "cycle_view", "issue_views_view", "page_view", "intake_view", "inbox_view", "guest_view_all_features", "archive_in", "close_in", "timezone", "external_source", "external_id", "is_issue_type_enabled", "is_time_tracking_enabled", "default_state", "estimate", "workspace"} {
			allowed[key] = true
		}
		for key := range fields {
			if !allowed[key] {
				boundaryErrors[key] = "This field is not writable."
			}
		}
		if len(boundaryErrors) > 0 {
			return nil, projectFailure(400, boundaryErrors)
		}
	}
	// The session view's inbox_view alias takes precedence and falls back to the
	// stored intake setting. Direct intake_view is replaced before validation.
	validationFields := make(map[string]any, len(fields)+1)
	for k, v := range fields {
		validationFields[k] = v
	}
	intake := existing["intake_view"]
	intakeKey := "intake_view"
	if app {
		intakeKey = "inbox_view"
	}
	if supplied, ok := fields[intakeKey]; ok {
		intake = supplied
	}
	validationFields["intake_view"] = intake
	validationInput := input
	validationInput.Fields = validationFields
	values, err := r.validateProjectFields(ctx, fmt.Sprint(existing["workspace"]), id, validationInput, app, replace, false)
	if err != nil {
		return nil, err
	}
	requestedData, err := lifecycleRequestedData(input)
	if err != nil {
		return nil, err
	}
	if identifier, ok := values["identifier"].(string); ok {
		values["identifier"] = normalizeProjectIdentifier(identifier)
	}
	values["updated_at"] = time.Now().UTC()
	values["updated_by_id"] = user
	columns := make([]string, 0, len(values))
	for key := range values {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	assignments := make([]string, 0, len(columns))
	for _, key := range columns {
		assignments = append(assignments, key+"=incoming."+key)
	}
	_, err = r.writer.Exec(ctx, `UPDATE projects project SET `+strings.Join(assignments, ",")+` FROM jsonb_populate_record(NULL::projects,$1::jsonb) incoming WHERE project.id=$2`, string(raw), id)
	if err != nil {
		return nil, r.projectIntegrityError(err, app)
	}
	enabled, _ := existing["intake_view"].(bool)
	if value, ok := values["intake_view"].(bool); ok {
		enabled = value
	}
	if enabled {
		if err = EnsureProjectIntake(ctx, r.writer, id, user); err != nil {
			return nil, err
		}
	}
	result, err := r.projectWriteProjection(ctx, user, slug, id, app)
	if err != nil {
		return nil, err
	}
	currentJSON, err := projectCurrentInstance(current)
	if err != nil {
		return nil, err
	}
	if err = r.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "project", "model_id": id, "requested_data": requestedData, "current_instance": currentJSON, "actor_id": user, "slug": slug, "origin": r.baseHost}); err != nil {
		return nil, err
	}
	return result, nil
}
func (r *LifecycleRepository) projectModelProjection(ctx context.Context, user, slug, id string, app bool) (json.RawMessage, error) {
	projection := projectSerializerProjectionSQL(app)
	var raw []byte
	err := r.writer.QueryRow(ctx, `WITH requester AS MATERIALIZED(SELECT set_config('TimeZone',COALESCE(NULLIF(user_timezone,''),'UTC'),true) AS user_timezone FROM users WHERE id=$1) SELECT `+projection+` FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester WHERE workspace.slug=$2 AND project.id=$3 AND project.deleted_at IS NULL`, user, slug, id).Scan(&raw)
	return raw, err
}
func projectWorkspaceLogoURL() string {
	full := workspaceJSON(false)
	start := strings.Index(full, "'logo_url', ") + len("'logo_url', ")
	end := strings.Index(full[start:], ",\n  'name'")
	return full[start : start+end]
}

// ProjectSerializer field order also forms the serialized current_instance task
// argument; preserve it with json_build_object rather than jsonb sorting.
func projectSerializerProjectionSQL(app bool) string {
	fields := []string{"'id',project.id"}
	if app {
		fields = append(fields, "'workspace_detail',json_build_object('name',workspace.name,'slug',workspace.slug,'id',workspace.id,'logo_url',"+projectWorkspaceLogoURL()+")", "'inbox_view',project.intake_view")
	} else {
		fields = append(fields, "'cover_image_url',"+apiProjectCoverImageURL())
	}
	for _, key := range []string{"created_at", "updated_at", "deleted_at", "name", "description", "description_text", "description_html", "network", "identifier", "emoji", "icon_prop", "module_view", "cycle_view", "issue_views_view", "page_view", "intake_view", "is_time_tracking_enabled", "is_issue_type_enabled", "guest_view_all_features", "cover_image", "archive_in", "close_in", "logo_props", "archived_at", "timezone", "external_source", "external_id", "created_by", "updated_by", "workspace", "default_assignee", "project_lead", "cover_image_asset", "estimate", "default_state"} {
		expression := "project." + key
		switch key {
		case "created_at", "updated_at", "deleted_at", "archived_at":
			expression = djangoTimestamp(expression)
		case "created_by", "updated_by", "workspace", "default_assignee", "project_lead", "cover_image_asset", "estimate", "default_state":
			expression += "_id"
		}
		fields = append(fields, "'"+key+"',"+expression)
	}
	return "json_build_object(" + strings.Join(fields, ",") + ")"
}

var projectDefaultColors = []string{"#95999f", "#6d7b8a", "#5e6ad2", "#02b5ed", "#02b55c", "#f2be02", "#e57a00", "#f38e82"}
var projectDefaultIcons = []string{"home", "apps", "settings", "star", "favorite", "done", "check_circle", "add_task", "create_new_folder", "dataset", "terminal", "key", "rocket", "public", "quiz", "mood", "gavel", "eco", "diamond", "forest", "bolt", "sync", "cached", "library_add", "view_timeline", "view_kanban", "empty_dashboard", "cycle"}

// CheckProjectMutation preserves handler lookup/archive errors before lazy body
// parsing. Create's workspace lookup is also before the serializer is built.
func (r *LifecycleRepository) CheckProjectMutation(ctx context.Context, slug, id, operation string, app bool) error {
	if operation == "create" {
		var exists bool
		if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug=$1)`, slug).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return nil
	}
	var archived bool
	err := r.writer.QueryRow(ctx, `SELECT p.archived_at IS NOT NULL FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL`, id, slug).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		if operation == "put" || (!app && operation == "patch") {
			return projectFailure(404, map[string]any{"error": "Project does not exist"})
		}
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if archived && (operation == "put" || operation == "patch") {
		message := "Archived project cannot be updated"
		if app {
			message = "Archived projects cannot be updated"
		}
		return projectFailure(400, map[string]any{"error": message})
	}
	return nil
}

func (r *LifecycleRepository) apiProjectWriteResponse(ctx context.Context, user, slug, id string) (json.RawMessage, error) {
	var raw []byte
	err := r.writer.QueryRow(ctx, `WITH requester AS MATERIALIZED(SELECT set_config('TimeZone',COALESCE(NULLIF(user_timezone,''),'UTC'),true) AS user_timezone FROM users WHERE id=$1) SELECT (`+projectJSON+`)::jsonb - ARRAY['inbox_view','is_favorite','members','next_work_item_sequence','sort_order','anchor']::text[] || jsonb_build_object(
  'is_member',EXISTS(SELECT 1 FROM project_members member WHERE member.project_id=project.id AND member.workspace_id=project.workspace_id AND member.member_id=$1 AND member.is_active AND member.deleted_at IS NULL),
  'total_members',(SELECT count(*) FROM project_members member JOIN users member_user ON member_user.id=member.member_id WHERE member.project_id=project.id AND member.is_active AND member.deleted_at IS NULL AND member_user.is_bot=false),
  'total_cycles',(SELECT count(*) FROM cycles cycle WHERE cycle.project_id=project.id AND cycle.deleted_at IS NULL),
  'total_modules',(SELECT count(*) FROM modules module WHERE module.project_id=project.id AND module.deleted_at IS NULL),
  'is_deployed',EXISTS(SELECT 1 FROM deploy_boards board WHERE board.project_id=project.id AND board.workspace_id=project.workspace_id AND board.deleted_at IS NULL)
 ) FROM projects project JOIN workspaces workspace ON workspace.id=project.workspace_id CROSS JOIN requester WHERE workspace.slug=$2 AND project.id=$3 AND project.deleted_at IS NULL`, user, slug, id).Scan(&raw)
	return raw, err
}
