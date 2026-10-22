package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

// LifecycleEffects contains the durable side effects used by cycle writes.
// Keeping writes beside the cycle read repository gives the family one tenant
// and persistence boundary without introducing a second cycle domain owner.
type LifecycleEffects interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

type LifecycleRepository struct {
	database database
	effects  LifecycleEffects
	origin   string
	now      func() time.Time
}

type lifecycleOriginKey struct{}

func WithLifecycleRequestOrigin(ctx context.Context, scheme, host string) context.Context {
	return context.WithValue(ctx, lifecycleOriginKey{}, scheme+"://"+host)
}

func NewLifecycleRepository(database database, effects LifecycleEffects, origin string) (*LifecycleRepository, error) {
	if database == nil || effects == nil {
		return nil, errors.New("cycle lifecycle database and durable effects are required")
	}
	return &LifecycleRepository{database: database, effects: effects, origin: origin, now: time.Now}, nil
}

// AuthorizeProject is the write-side ProjectEntityPermission predicate. Unlike
// the API-v1 read permission, writes require project admin/member role.
func (repository *LifecycleRepository) AuthorizeProject(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND pm.project_id=$2 AND pm.is_active=true AND w.slug=$3 AND pm.role IN (15,20))`, userID, projectID, slug).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// AuthorizeAppProject matches the app's allow_permission([ADMIN, MEMBER])
// fallback for a workspace admin who is an active member of the project.
func (repository *LifecycleRepository) AuthorizeAppProject(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
		WHERE pm.member_id=$1 AND pm.project_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL AND w.slug=$3
		AND (pm.role IN (15,20) OR EXISTS(
			SELECT 1 FROM workspace_members wm
			WHERE wm.member_id=$1 AND wm.workspace_id=w.id AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
		))
	)`, userID, projectID, slug).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

// AuthorizeReadProject matches the inherited CycleViewSet queryset used by
// read-only update: active project membership is required, but no write role.
func (repository *LifecycleRepository) AuthorizeReadProject(ctx context.Context, userID, slug, projectID string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND pm.project_id=$2 AND pm.is_active=true AND p.deleted_at IS NULL AND p.archived_at IS NULL AND w.slug=$3)`, userID, projectID, slug).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrNotFound
	}
	return nil
}

func (repository *LifecycleRepository) DateCheck(ctx context.Context, slug, projectID string, start, end, cycleID *string) (bool, error) {
	var zone string
	if err := repository.database.QueryRow(ctx, `SELECT timezone FROM projects WHERE id=$1`, projectID).Scan(&zone); err != nil {
		return false, err
	}
	startAt, err := parseCycleDate(start, zone, true)
	if err != nil {
		return false, err
	}
	endAt, err := parseCycleDate(end, zone, false)
	if err != nil {
		return false, err
	}
	var overlap bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE w.slug=$1 AND c.project_id=$2 AND c.deleted_at IS NULL AND ($5::uuid IS NULL OR c.id<>$5::uuid) AND ((c.start_date<=$3 AND c.end_date>=$3) OR (c.start_date<=$4 AND c.end_date>=$4) OR (c.start_date>=$3 AND c.end_date<=$4)))`, slug, projectID, startAt, endAt, cycleID).Scan(&overlap); err != nil {
		return false, err
	}
	return !overlap, nil
}

type CycleCreate struct {
	Name, Description, Timezone string
	StartDate, EndDate          *string
	OwnedBy                     *string
	ExternalSource, ExternalID  *string
	SortOrder                   *float64
	ViewProps, ProgressSnapshot any
	LogoProps                   any
	Version                     *int32
}

type CycleLifecycleResource struct {
	ID, Name            string
	CreatedBy, OwnedBy  *string
	EndDate, ArchivedAt *time.Time
}

func (repository *LifecycleRepository) Patch(ctx context.Context, slug, projectID, cycleID, actor string, api bool, requested, activityRequested map[string]any) (map[string]any, error) {
	before, err := repository.rawCycle(ctx, slug, projectID, cycleID, !api)
	if err != nil {
		return nil, err
	}
	if before["archived_at"] != nil {
		message := "Archived cycle cannot be updated"
		if api {
			message = "Archived cycle cannot be edited"
		}
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": message}}
	}
	currentInstance := before
	if !api {
		currentInstance, err = repository.appCycleSerializerSnapshot(ctx, actor, slug, projectID, cycleID)
		if err != nil {
			return nil, err
		}
	} else {
		currentInstance, err = repository.userCycleSerializerSnapshot(ctx, actor, slug, projectID, cycleID)
		if err != nil {
			return nil, err
		}
	}
	var projectTimezone string
	var actorTimezone string
	if err := repository.database.QueryRow(ctx, `SELECT p.timezone,COALESCE(NULLIF(u.user_timezone,''),'UTC') FROM projects p JOIN users u ON u.id=$2 WHERE p.id=$1`, projectID, actor).Scan(&projectTimezone, &actorTimezone); err != nil {
		return nil, err
	}
	end, _ := timestampValue(before["end_date"])
	completed := end != nil && end.Before(repository.now())
	effective := map[string]any{}
	activityData := maps.Clone(activityRequested)
	if completed {
		if _, ok := requested["sort_order"]; !ok {
			message := "The Cycle has already been completed so it cannot be edited"
			return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": message}}
		}
		effective["sort_order"] = requested["sort_order"]
		activityData = map[string]any{"sort_order": activityRequested["sort_order"]}
	} else {
		for key, value := range requested {
			effective[key] = value
		}
	}
	if api {
		if value, ok := effective["owned_by"]; ok {
			if value == nil || value == "" {
				effective["owned_by"] = actor
			}
		}
	}
	startDate, _ := timestampValue(before["start_date"])
	endDate, _ := timestampValue(before["end_date"])
	pairedDateValidation := false
	if startValue, startPresent := effective["start_date"]; startPresent && startValue != nil {
		if endValue, endPresent := effective["end_date"]; endPresent && endValue != nil {
			pairedDateValidation = true
		}
	}
	for key, destination := range map[string]**time.Time{"start_date": &startDate, "end_date": &endDate} {
		value, present := effective[key]
		if !present {
			continue
		}
		if value == nil {
			*destination = nil
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{key: []string{"Datetime has wrong format."}}}
		}
		var parsed *time.Time
		var err error
		parsed, err = parseCycleDateTime(text, actorTimezone)
		if err != nil {
			return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{key: []string{"Datetime has wrong format."}}}
		}
		*destination = parsed
		effective[key] = parsed
	}
	if pairedDateValidation && startDate != nil && endDate != nil {
		if startDate.After(*endDate) {
			return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Start date cannot exceed end date"}}}
		}
		actorLocation, err := time.LoadLocation(actorTimezone)
		if err != nil {
			return nil, err
		}
		startDay := startDate.In(actorLocation).Format("2006-01-02")
		endDay := endDate.In(actorLocation).Format("2006-01-02")
		startDate, err = parseCycleDate(&startDay, projectTimezone, true)
		if err != nil {
			return nil, err
		}
		endDate, err = parseCycleDate(&endDay, projectTimezone, false)
		if err != nil {
			return nil, err
		}
		effective["start_date"] = startDate
		effective["end_date"] = endDate
	}
	updates := make([]string, 0, 10)
	args := []any{actor, cycleID, projectID, slug}
	add := func(column string, value any, cast string) {
		args = append(args, value)
		updates = append(updates, column+"=$"+strconv.Itoa(len(args))+cast)
	}
	allowed := map[string]string{"name": "name", "description": "description", "start_date": "start_date", "end_date": "end_date", "view_props": "view_props", "sort_order": "sort_order", "external_source": "external_source", "external_id": "external_id", "progress_snapshot": "progress_snapshot", "logo_props": "logo_props", "timezone": "timezone", "version": "version"}
	if api {
		allowed = map[string]string{"name": "name", "description": "description", "start_date": "start_date", "end_date": "end_date", "sort_order": "sort_order", "external_source": "external_source", "external_id": "external_id", "timezone": "timezone", "owned_by": "owned_by_id"}
	}
	for key, value := range effective {
		column, ok := allowed[key]
		if !ok {
			continue
		}
		if key == "start_date" || key == "end_date" {
			// Values are normalized and paired-date validated above.
		}
		if key == "sort_order" {
			if _, ok := value.(float64); !ok {
				return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"sort_order": []string{"A valid number is required."}}}
			}
		}
		cast := ""
		if key == "view_props" || key == "progress_snapshot" || key == "logo_props" {
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			value = string(encoded)
			cast = "::jsonb"
		}
		if key == "owned_by" {
			if text, ok := value.(string); ok {
				var exists bool
				if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, text).Scan(&exists); err != nil {
					return nil, err
				}
				if !exists {
					return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"owned_by": []string{"Invalid pk \"" + text + "\" - object does not exist."}}}
				}
			}
		}
		add(column, value, cast)
	}
	if api {
		if err := repository.checkAPIExternalIDConflict(ctx, projectID, slug, cycleID, before, activityData); err != nil {
			return nil, err
		}
	}
	args = append(args, repository.now(), actor)
	updates = append(updates, "updated_at=$"+strconv.Itoa(len(args)-1), "updated_by_id=$"+strconv.Itoa(len(args)))
	query := "UPDATE cycles SET " + strings.Join(updates, ",") + " WHERE id=$2 AND project_id=$3 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$4) AND deleted_at IS NULL RETURNING to_jsonb(cycles)||jsonb_build_object('project',project_id,'workspace',workspace_id,'owned_by',owned_by_id,'created_by',created_by_id,'updated_by',updated_by_id," + cycleUserTimestampJSONFields("cycles") + ")"
	var raw []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var updated map[string]any
	updated, err = pythoncompat.DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	current, err := pythonCycleCurrentInstance(currentInstance, api)
	if err != nil {
		return nil, err
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{
		"model_name": "cycle", "model_id": cycleID, "requested_data": activityData, "current_instance": current,
		"actor_id": actor, "slug": slug, "origin": repository.taskOrigin(ctx),
	}); err != nil {
		return nil, err
	}
	if api {
		return updated, nil
	}
	return repository.appCycleMutationProjection(ctx, actor, slug, projectID, cycleID, false)
}

func (repository *LifecycleRepository) checkAPIExternalIDConflict(ctx context.Context, projectID, slug, cycleID string, before, requestData map[string]any) error {
	rawExternalID, hasExternalID := requestData["external_id"]
	if !hasExternalID || !cycleExternalIDTruthy(rawExternalID) || reflect.DeepEqual(rawExternalID, before["external_id"]) {
		return nil
	}
	externalID := djangoinput.ScalarString(rawExternalID)
	externalSource, hasExternalSource := requestData["external_source"]
	if !hasExternalSource {
		externalSource = before["external_source"]
	}
	if externalSource != nil {
		externalSource = djangoinput.ScalarString(externalSource)
	}
	var duplicate bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM cycles
		WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2)
		AND external_source IS NOT DISTINCT FROM $3 AND external_id=$4 AND deleted_at IS NULL
	)`, projectID, slug, externalSource, externalID).Scan(&duplicate); err != nil {
		return err
	}
	if duplicate {
		return &djangoapi.HTTPError{Status: 409, Body: map[string]any{
			"error": "Cycle with the same external id and external source already exists",
			"id":    cycleID,
		}}
	}
	return nil
}

func cycleExternalIDTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return typed != ""
	case json.Number:
		number, err := strconv.ParseFloat(string(typed), 64)
		return err != nil || number != 0
	default:
		return true
	}
}

func parseCycleDateTime(value, zone string) (*time.Time, error) {
	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, err
	}
	parsed, err := pythoncompat.DjangoDateTime(value, location)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (repository *LifecycleRepository) appCycleSerializerSnapshot(ctx context.Context, actor, slug, projectID, cycleID string) (map[string]any, error) {
	value, err := djangoapi.JSON(ctx, repository.database, userCycleMutationDetailSQL, actor, slug, projectID, cycleID)
	if err != nil {
		return nil, err
	}
	cycle, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cycle serializer snapshot returned %T", value)
	}
	return cycle, nil
}

func (repository *LifecycleRepository) userCycleSerializerSnapshot(ctx context.Context, actor, slug, projectID, cycleID string) (map[string]any, error) {
	query := `SELECT to_jsonb(c)||jsonb_build_object('workspace',c.workspace_id,'project',c.project_id,'owned_by',c.owned_by_id,'created_by',c.created_by_id,'updated_by',c.updated_by_id,` + cycleUserTimestampJSONFields("c") + `) FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE c.id=$2 AND c.project_id=$3 AND w.slug=$4 AND c.deleted_at IS NULL`
	value, err := djangoapi.JSON(ctx, repository.database, query, actor, cycleID, projectID, slug)
	if err != nil {
		return nil, err
	}
	cycle, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cycle serializer snapshot returned %T", value)
	}
	return cycle, nil
}

func (repository *LifecycleRepository) PutNoop(ctx context.Context, slug, projectID, cycleID, actor string) (map[string]any, error) {
	if _, err := repository.rawCycle(ctx, slug, projectID, cycleID, true); err != nil {
		return nil, err
	}
	var updatedID string
	if err := repository.database.QueryRow(ctx, `UPDATE cycles SET updated_at=$4,updated_by_id=$5 WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL RETURNING id::text`, cycleID, projectID, slug, repository.now(), actor).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if updatedID == "" {
		return nil, ErrNotFound
	}
	return repository.appCycleMutationProjection(ctx, actor, slug, projectID, cycleID, true)
}

func (repository *LifecycleRepository) rawCycle(ctx context.Context, slug, projectID, cycleID string, appScope bool) (map[string]any, error) {
	query := `SELECT to_jsonb(c)||jsonb_build_object('project',c.project_id,'workspace',c.workspace_id,'owned_by',c.owned_by_id,'created_by',c.created_by_id,'updated_by',c.updated_by_id,` + cycleTimestampJSONFields("c") + `) FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE c.id=$1 AND c.project_id=$2 AND w.slug=$3 AND c.deleted_at IS NULL`
	if appScope {
		query += ` AND EXISTS(SELECT 1 FROM projects p WHERE p.id=c.project_id AND p.deleted_at IS NULL AND p.archived_at IS NULL)`
	}
	value, err := djangoapi.JSON(ctx, repository.database, query, cycleID, projectID, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cycle query returned %T", value)
	}
	return result, nil
}

func cycleTimestampJSONFields(table string) string {
	fields := []string{"created_at", "updated_at", "deleted_at", "archived_at", "start_date", "end_date"}
	pairs := make([]string, 0, len(fields))
	for _, field := range fields {
		pairs = append(pairs, "'"+field+"',"+utcTimestampSQL(table+"."+field))
	}
	return strings.Join(pairs, ",")
}

func cycleUserTimestampJSONFields(table string) string {
	fields := []string{"created_at", "updated_at", "deleted_at", "archived_at", "start_date", "end_date"}
	pairs := make([]string, 0, len(fields))
	for _, field := range fields {
		pairs = append(pairs, "'"+field+"',"+userTimestampSQL(table+"."+field))
	}
	return strings.Join(pairs, ",")
}

func (repository *LifecycleRepository) taskOrigin(ctx context.Context) string {
	if repository.origin != "" {
		return repository.origin
	}
	origin, _ := ctx.Value(lifecycleOriginKey{}).(string)
	return origin
}

var projectCycleMutationDetailSQL = strings.Replace(
	strings.Replace(projectCycleDetailSQL,
		"'total_issues',counts.total,'completed_issues',counts.completed,",
		"'total_issues',counts.total,'completed_issues',counts.completed,'cancelled_issues',counts.cancelled,", 1),
	"SELECT raw_counts.total,raw_counts.completed)",
	"SELECT raw_counts.total,raw_counts.completed,raw_counts.cancelled)", 1,
)

var userCycleMutationDetailSQL = strings.ReplaceAll(
	strings.ReplaceAll(projectCycleMutationDetailSQL,
		projectTimestampSQL("cycle.start_date"), userTimestampSQL("cycle.start_date")),
	projectTimestampSQL("cycle.end_date"), userTimestampSQL("cycle.end_date"),
)

func (repository *LifecycleRepository) appCycleMutationProjection(ctx context.Context, actor, slug, projectID, cycleID string, put bool) (map[string]any, error) {
	query := projectCycleMutationDetailSQL
	if put {
		query = userCycleMutationDetailSQL
	}
	value, err := djangoapi.JSON(ctx, repository.database, query, actor, slug, projectID, cycleID)
	if err != nil {
		return nil, err
	}
	cycle, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cycle mutation projection returned %T", value)
	}
	fields := []string{"id", "workspace_id", "project_id", "name", "description", "start_date", "end_date", "owned_by_id", "view_props", "sort_order", "external_source", "external_id", "progress_snapshot", "logo_props", "is_favorite", "total_issues", "completed_issues", "status"}
	if put {
		fields = append(fields, "cancelled_issues")
	} else {
		fields = append(fields, "assignee_ids", "version", "created_by")
	}
	projection := make(map[string]any, len(fields))
	for _, field := range fields {
		projection[field] = cycle[field]
	}
	return projection, nil
}

func timestampValue(value any) (*time.Time, bool) {
	text, ok := value.(string)
	if !ok || text == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func pythonCycleCurrentInstance(cycle map[string]any, api bool) (string, error) {
	order := []string{"id", "created_at", "updated_at", "deleted_at", "name", "description", "start_date", "end_date", "sub_issues", "view_props", "sort_order", "external_source", "external_id", "progress_snapshot", "archived_at", "logo_props", "timezone", "version", "created_by", "updated_by", "project", "workspace", "owned_by"}
	if !api {
		order = []string{"id", "workspace_id", "project_id", "name", "description", "start_date", "end_date", "owned_by_id", "view_props", "sort_order", "external_source", "external_id", "progress_snapshot", "logo_props", "is_favorite", "total_issues", "cancelled_issues", "completed_issues", "status"}
	}
	projection := make(map[string]any, len(order))
	var skeleton strings.Builder
	skeleton.WriteByte('{')
	first := true
	for _, key := range order {
		child, ok := cycle[key]
		if !ok {
			continue
		}
		projection[key] = child
		if !first {
			skeleton.WriteByte(',')
		}
		first = false
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return "", err
		}
		skeleton.Write(encodedKey)
		skeleton.WriteString(":null")
	}
	skeleton.WriteByte('}')
	return pythoncompat.Dumps(projection, []byte(skeleton.String()))
}

func (repository *LifecycleRepository) Cycle(ctx context.Context, slug, projectID, cycleID string) (CycleLifecycleResource, error) {
	var result CycleLifecycleResource
	err := repository.database.QueryRow(ctx, `SELECT c.id::text,c.name,c.created_by_id::text,c.owned_by_id::text,c.end_date,c.archived_at FROM cycles c JOIN workspaces w ON w.id=c.workspace_id WHERE c.id=$1 AND c.project_id=$2 AND w.slug=$3 AND c.deleted_at IS NULL`, cycleID, projectID, slug).Scan(&result.ID, &result.Name, &result.CreatedBy, &result.OwnedBy, &result.EndDate, &result.ArchivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CycleLifecycleResource{}, ErrNotFound
	}
	return result, err
}

func (repository *LifecycleRepository) SetArchived(ctx context.Context, slug, projectID, cycleID, actor string, archive bool) (time.Time, error) {
	cycle, err := repository.Cycle(ctx, slug, projectID, cycleID)
	if err != nil {
		return time.Time{}, err
	}
	now := repository.now()
	if archive && (cycle.EndDate == nil || !cycle.EndDate.Before(now)) {
		return time.Time{}, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Only completed cycles can be archived"}}
	}
	var archivedAt *time.Time
	if archive {
		archivedAt = &now
	}
	var updated time.Time
	if err := repository.database.QueryRow(ctx, `UPDATE cycles SET archived_at=$4,updated_at=$5,updated_by_id=$6 WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL RETURNING updated_at`, cycleID, projectID, slug, archivedAt, now, actor).Scan(&updated); err != nil {
		return time.Time{}, err
	}
	if archive {
		if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$4 WHERE entity_type='cycle' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL`, cycleID, projectID, slug, now); err != nil {
			return time.Time{}, err
		}
	}
	if archivedAt == nil {
		return time.Time{}, nil
	}
	return *archivedAt, nil
}

func (repository *LifecycleRepository) AuthorizeDelete(ctx context.Context, slug, projectID, cycleID, actor string, api bool) error {
	cycle, err := repository.Cycle(ctx, slug, projectID, cycleID)
	if err != nil {
		return err
	}
	owner := cycle.CreatedBy
	if api {
		owner = cycle.OwnedBy
	}
	var admin bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.project_id=$1 AND w.slug=$2 AND pm.member_id=$3 AND pm.role=20 AND pm.is_active=true)`, projectID, slug, actor).Scan(&admin); err != nil {
		return err
	}
	if (owner == nil || *owner != actor) && !admin {
		return ErrForbidden
	}
	return nil
}

// AuthorizeAppDelete mirrors the creator decorator and its independent
// workspace-membership, project-admin, and workspace-admin checks.
func (repository *LifecycleRepository) AuthorizeAppDelete(ctx context.Context, slug, projectID, cycleID, actor string) error {
	var allowed bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id
		WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL
		AND (
			EXISTS(SELECT 1 FROM cycles c WHERE c.id=$3 AND c.created_by_id=$1 AND c.deleted_at IS NULL)
			OR EXISTS(
				SELECT 1 FROM project_members pm
				WHERE pm.member_id=$1 AND pm.project_id=$4 AND pm.workspace_id=w.id AND pm.role=20 AND pm.is_active=true AND pm.deleted_at IS NULL
			)
			OR EXISTS(
				SELECT 1 FROM project_members pm JOIN workspace_members admin ON admin.workspace_id=pm.workspace_id
				WHERE pm.member_id=$1 AND pm.project_id=$4 AND pm.workspace_id=w.id AND pm.is_active=true AND pm.deleted_at IS NULL
				AND admin.member_id=$1 AND admin.role=20 AND admin.is_active=true AND admin.deleted_at IS NULL
			)
		)
	)`, actor, slug, cycleID, projectID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *LifecycleRepository) Delete(ctx context.Context, slug, projectID, cycleID, actor string, api bool) error {
	cycle, err := repository.Cycle(ctx, slug, projectID, cycleID)
	if err != nil {
		return err
	}
	var issueIDs []byte
	if err := repository.database.QueryRow(ctx, `SELECT COALESCE(json_agg(ci.issue_id ORDER BY ci.issue_id),'[]'::json) FROM cycle_issues ci WHERE ci.cycle_id=$1 AND ci.deleted_at IS NULL`, cycleID).Scan(&issueIDs); err != nil {
		return err
	}
	var ids []string
	if err := json.Unmarshal(issueIDs, &ids); err != nil {
		return err
	}
	requested, err := json.Marshal(map[string]any{"cycle_id": cycleID, "cycle_name": cycle.Name, "issues": ids})
	if err != nil {
		return err
	}
	kwargs := map[string]any{"type": "cycle.activity.deleted", "requested_data": string(requested), "actor_id": actor, "project_id": projectID, "current_instance": nil, "epoch": repository.now().Unix()}
	if api {
		kwargs["issue_id"] = nil
	} else {
		kwargs["issue_id"] = cycleID
		kwargs["notification"] = true
		origin := repository.origin
		if origin == "" {
			origin, _ = ctx.Value(lifecycleOriginKey{}).(string)
		}
		kwargs["origin"] = origin
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, kwargs); err != nil {
		return err
	}
	var deleted string
	if err := repository.database.QueryRow(ctx, `UPDATE cycles SET deleted_at=$4,updated_at=$4,updated_by_id=$5 WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL RETURNING id::text`, cycleID, projectID, slug, repository.now(), actor).Scan(&deleted); err != nil {
		return err
	}
	if deleted == "" {
		return ErrNotFound
	}
	if err := repository.effects.SoftDelete(ctx, "db", "cycle", cycleID); err != nil {
		return err
	}
	if api {
		if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$3 WHERE entity_type='cycle' AND entity_identifier=$1 AND project_id=$2 AND deleted_at IS NULL`, cycleID, projectID, repository.now()); err != nil {
			return err
		}
	} else {
		if _, err := repository.database.Exec(ctx, `UPDATE user_favorites SET deleted_at=$4 WHERE entity_type='cycle' AND entity_identifier=$1 AND project_id=$2 AND user_id=$3 AND deleted_at IS NULL`, cycleID, projectID, actor, repository.now()); err != nil {
			return err
		}
		if err := repository.database.QueryRow(ctx, `DELETE FROM user_recent_visits WHERE entity_name='cycle' AND entity_identifier=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) RETURNING id::text`, cycleID, projectID, slug).Scan(&deleted); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return nil
}

// Create persists a cycle and returns the database row enriched with the
// workspace/project fields needed by the two source serializers.
func (repository *LifecycleRepository) Create(ctx context.Context, actor, slug, projectID string, input CycleCreate, api bool, requested map[string]any) (map[string]any, error) {
	var workspaceID, projectTimezone, actorTimezone string
	var cycleEnabled bool
	err := repository.database.QueryRow(ctx, `SELECT p.workspace_id::text,p.timezone,p.cycle_view,COALESCE(NULLIF(u.user_timezone,''),'UTC') FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN users u ON u.id=$3 WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL`, projectID, slug, actor).Scan(&workspaceID, &projectTimezone, &cycleEnabled, &actorTimezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if api && !cycleEnabled {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Cycles are not enabled for this project"}}}
	}
	start, end, err := parseCycleInputDates(input.StartDate, input.EndDate, actorTimezone, projectTimezone)
	if err != nil {
		return nil, err
	}
	owner := actor
	if input.OwnedBy != nil && *input.OwnedBy != "" {
		owner = *input.OwnedBy
	}
	var ownerExists bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, owner).Scan(&ownerExists); err != nil {
		return nil, err
	}
	if !ownerExists {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"owned_by": []string{"Invalid pk \"" + owner + "\" - object does not exist."}}}
	}
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	var duplicate string
	if api && input.ExternalID != nil && *input.ExternalID != "" && input.ExternalSource != nil && *input.ExternalSource != "" {
		err = repository.database.QueryRow(ctx, `SELECT id::text FROM cycles WHERE project_id=$1 AND workspace_id=$2 AND external_source=$3 AND external_id=$4 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`, projectID, workspaceID, *input.ExternalSource, *input.ExternalID).Scan(&duplicate)
		if err == nil {
			return nil, &djangoapi.HTTPError{Status: 409, Body: map[string]any{"error": "Cycle with the same external id and external source already exists", "id": duplicate}}
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	var cycleID string
	var cycle map[string]any
	viewProps, err := lifecycleJSON(input.ViewProps)
	if err != nil {
		return nil, err
	}
	if input.ViewProps == nil {
		viewProps = `{}`
	}
	progressSnapshot, err := lifecycleJSON(input.ProgressSnapshot)
	if err != nil {
		return nil, err
	}
	if input.ProgressSnapshot == nil {
		progressSnapshot = `{}`
	}
	logoProps, err := lifecycleJSON(input.LogoProps)
	if err != nil {
		return nil, err
	}
	if input.LogoProps == nil {
		logoProps = `{}`
	}
	version := int32(1)
	if input.Version != nil {
		version = *input.Version
	}
	var sortOrder any
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	query := `WITH smallest AS (SELECT min(sort_order) AS value FROM cycles WHERE project_id=@project_id AND deleted_at IS NULL)
INSERT INTO cycles (id,created_at,updated_at,deleted_at,name,description,start_date,end_date,owned_by_id,view_props,sort_order,external_source,external_id,progress_snapshot,archived_at,logo_props,timezone,version,created_by_id,updated_by_id,project_id,workspace_id)
SELECT gen_random_uuid(),NOW(),NOW(),NULL,@name,@description,@start_date,@end_date,@owner,@view_props::jsonb,COALESCE(@sort_order,smallest.value-10000,65535),@external_source,@external_id,@progress_snapshot::jsonb,NULL,@logo_props::jsonb,@timezone,@version,@actor,NULL,@project_id,@workspace_id FROM smallest
RETURNING id::text`
	args := pgx.StrictNamedArgs{
		"project_id": projectID, "workspace_id": workspaceID, "name": input.Name, "description": input.Description,
		"start_date": start, "end_date": end, "owner": owner, "view_props": viewProps, "sort_order": sortOrder,
		"external_source": input.ExternalSource, "external_id": input.ExternalID, "progress_snapshot": progressSnapshot,
		"logo_props": logoProps, "timezone": input.Timezone, "version": version, "actor": actor,
	}
	if err := repository.database.QueryRow(ctx, query, args).Scan(&cycleID); err != nil {
		return nil, err
	}
	value, err := djangoapi.JSON(ctx, repository.database, `SELECT to_jsonb(c) || jsonb_build_object('workspace',c.workspace_id,'project',c.project_id,'owned_by',c.owned_by_id,'created_by',c.created_by_id,'updated_by',c.updated_by_id,`+cycleUserTimestampJSONFields("c")+`) FROM cycles c WHERE c.id=$2`, actor, cycleID)
	if err != nil {
		return nil, err
	}
	cycle, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cycle create returned %T", value)
	}
	cycle["created_by"] = cycle["created_by_id"]
	cycle["is_favorite"] = false
	cycle["assignee_ids"] = []any{}
	cycle["status"] = cycleStatus(start, end, repository.now())
	origin := repository.origin
	if origin == "" {
		origin, _ = ctx.Value(lifecycleOriginKey{}).(string)
	}
	if err := repository.effects.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{
		"model_name": "cycle", "model_id": cycleID, "requested_data": requested, "current_instance": nil,
		"actor_id": actor, "slug": slug, "origin": origin,
	}); err != nil {
		return nil, err
	}
	if api {
		return cycle, nil
	}
	return repository.appCycleMutationProjection(ctx, actor, slug, projectID, cycleID, false)
}

func lifecycleJSON(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	encoded, err := json.Marshal(value)
	return string(encoded), err
}

func cycleStatus(start, end *time.Time, now time.Time) string {
	if start == nil && end == nil {
		return "DRAFT"
	}
	if end != nil && end.Before(now) {
		return "COMPLETED"
	}
	if start != nil && start.After(now) {
		return "UPCOMING"
	}
	return "CURRENT"
}

func parseCycleDate(value *string, zone string, start bool) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	var parsed time.Time
	var err error
	if len(*value) == len("2006-01-02") {
		var location *time.Location
		location, err = time.LoadLocation(zone)
		if err == nil {
			date, parseErr := time.Parse("2006-01-02", *value)
			err = parseErr
			if err == nil {
				hour, minute, second := 23, 59, 0
				if start {
					hour, minute, second = 0, 0, 1
				}
				parsed = time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, location).UTC()
			}
		}
	} else {
		var location *time.Location
		location, err = time.LoadLocation(zone)
		if err == nil {
			parsed, err = pythoncompat.DjangoDateTime(*value, location)
		}
	}
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"start_date": []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}}}
	}
	return &parsed, nil
}

func parseCycleInputDates(startValue, endValue *string, actorTimezone, projectTimezone string) (*time.Time, *time.Time, error) {
	parse := func(value *string) (*time.Time, error) {
		if value == nil || *value == "" {
			return nil, nil
		}
		return parseCycleDateTime(*value, actorTimezone)
	}
	start, err := parse(startValue)
	if err != nil {
		return nil, nil, err
	}
	end, err := parse(endValue)
	if err != nil {
		return nil, nil, err
	}
	if start == nil || end == nil {
		return start, end, nil
	}
	if start.After(*end) {
		return nil, nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Start date cannot exceed end date"}}}
	}
	actorLocation, err := time.LoadLocation(actorTimezone)
	if err != nil {
		return nil, nil, err
	}
	startDay := start.In(actorLocation).Format("2006-01-02")
	endDay := end.In(actorLocation).Format("2006-01-02")
	start, err = parseCycleDate(&startDay, projectTimezone, true)
	if err != nil {
		return nil, nil, err
	}
	end, err = parseCycleDate(&endDay, projectTimezone, false)
	if err != nil {
		return nil, nil, err
	}
	return start, end, nil
}
