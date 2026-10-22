package activityreads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

type HTTPError = djangoapi.HTTPError

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("activity reads database is required")
	}
	return &Repository{database: database}, nil
}

type Principal struct {
	UserID        uuid.UUID
	Authenticated bool
	Timezone      string
	APIKey        string
}

type Request struct {
	Operation                string
	Params                   map[string]string
	Present                  map[string]bool
	RequireProjectMemberRole bool
}

type Result struct{ Body any }

func (repository *Repository) Authorize(ctx context.Context, principal Principal, request Request) error {
	if err := requiredPrincipal(principal); err != nil {
		return err
	}
	switch request.Operation {
	case "get_api_users_me_activities":
		return nil
	case "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities",
		"get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities_by_pk",
		"get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities",
		"get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities_by_pk":
		return repository.authorizeProject(ctx, principal.UserID, param(request, "slug"), param(request, "project_id"), request.RequireProjectMemberRole)
	default:
		return fmt.Errorf("unknown activity read operation %q", request.Operation)
	}
}

func (repository *Repository) Read(ctx context.Context, principal Principal, request Request) (Result, error) {
	if err := requiredPrincipal(principal); err != nil {
		return Result{}, err
	}
	switch request.Operation {
	case "get_api_users_me_activities":
		return repository.userActivities(ctx, principal, request)
	case "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities",
		"get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities":
		return repository.apiActivities(ctx, principal, request, false)
	case "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities_by_pk",
		"get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities_by_pk":
		return repository.apiActivities(ctx, principal, request, true)
	default:
		return Result{}, fmt.Errorf("unknown activity read operation %q", request.Operation)
	}
}

func (repository *Repository) json(ctx context.Context, query string, args ...any) (any, error) {
	var data []byte
	if err := repository.database.QueryRow(ctx, query, args...).Scan(&data); err != nil {
		return nil, err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode activity read JSON: %w", err)
	}
	return value, nil
}

func (repository *Repository) authorizeProject(ctx context.Context, user uuid.UUID, slug, projectID string, requireMemberRole bool) error {
	id, err := uuid.Parse(projectID)
	if err != nil {
		return ErrNotFound
	}
	var allowed bool
	allowed, err = djangoapi.ProjectEntityAllowed(ctx, repository.database, user, slug, id, requireMemberRole)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func requiredPrincipal(principal Principal) error {
	if !principal.Authenticated {
		return &HTTPError{Status: 401, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	return nil
}

func param(request Request, names ...string) string {
	for _, name := range names {
		if value, ok := request.Params[name]; ok {
			return value
		}
	}
	return ""
}

func pageSize(request Request) (int, error) {
	return djangoapi.PageSize(djangoapi.Query{Params: request.Params, Present: request.Present})
}
func cursorPage(request Request, limit int) (int, error) {
	return djangoapi.CursorPage(djangoapi.Query{Params: request.Params, Present: request.Present}, limit)
}
func pageWindow(request Request) (int, int, int, error) {
	return djangoapi.PageWindow(djangoapi.Query{Params: request.Params, Present: request.Present})
}
func envelope(results []any, total, limit, page int) map[string]any {
	return djangoapi.Envelope(results, total, limit, page)
}

func paginatedResult(value any, limit, page int) (Result, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return Result{}, fmt.Errorf("expected paginated JSON object, got %T", value)
	}
	results, ok := object["results"].([]any)
	if !ok {
		return Result{}, fmt.Errorf("expected paginated JSON results, got %T", object["results"])
	}
	totalNumber, ok := object["total_count"].(json.Number)
	if !ok {
		return Result{}, fmt.Errorf("expected paginated JSON count, got %T", object["total_count"])
	}
	total64, err := strconv.ParseInt(totalNumber.String(), 10, 64)
	if err != nil || int64(int(total64)) != total64 {
		return Result{}, fmt.Errorf("invalid paginated JSON count %q", totalNumber)
	}
	return Result{Body: envelope(results, int(total64), limit, page)}, nil
}

const activityAPIJSON = `jsonb_build_object(
'id',a.id,'created_at',a.created_at,'updated_at',a.updated_at,'deleted_at',a.deleted_at,
'workspace',a.workspace_id,'project',a.project_id,'issue',a.issue_id,'verb',a.verb,'field',a.field,
'old_value',a.old_value,'new_value',a.new_value,'comment',a.comment,'attachments',a.attachments,
'issue_comment',a.issue_comment_id,'actor',a.actor_id,'old_identifier',a.old_identifier,
'new_identifier',a.new_identifier,'epoch',a.epoch)`

func fileAssetURLSQL(assetID, legacy string) string {
	return djangoapi.FileAssetURLSQL(assetID, legacy)
}

var activityActorExpansion = `COALESCE((SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'display_name',u.display_name,'avatar',u.avatar,'avatar_url',` + fileAssetURLSQL("u.avatar_asset_id", "u.avatar") + `) FROM users u WHERE u.id=a.actor_id),'{}'::jsonb)`

const activityWorkspaceExpansion = `(SELECT jsonb_build_object('id',w.id,'name',w.name,'slug',w.slug) FROM workspaces w WHERE w.id=a.workspace_id)`

var activityProjectExpansion = `(SELECT jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'description',p.description,'cover_image',p.cover_image,'cover_image_url',` + fileAssetURLSQL("p.cover_image_asset_id", "p.cover_image") + `,'emoji',p.emoji,'icon_prop',p.icon_prop,'archived_at',p.archived_at) FROM projects p WHERE p.id=a.project_id)`

const activityIssueExpansion = `(SELECT jsonb_build_object('id',i.id,'assignees',COALESCE((SELECT jsonb_agg(x.assignee_id ORDER BY x.created_at DESC) FROM issue_assignees x WHERE x.issue_id=i.id AND x.deleted_at IS NULL),'[]'::jsonb),'labels',COALESCE((SELECT jsonb_agg(x.label_id ORDER BY x.created_at DESC) FROM issue_labels x WHERE x.issue_id=i.id AND x.deleted_at IS NULL),'[]'::jsonb),'type',i.type_id,'type_id',i.type_id,'created_at',i.created_at,'updated_at',i.updated_at,'created_by',i.created_by_id,'updated_by',i.updated_by_id,'deleted_at',i.deleted_at,'project',i.project_id,'workspace',i.workspace_id,'parent',i.parent_id,'state',i.state_id,'point',i.point,'estimate_point',i.estimate_point_id,'name',i.name,'description_html',i.description_html,'description_binary',CASE WHEN i.description_binary IS NULL THEN NULL ELSE replace(encode(i.description_binary,'base64'),E'\n','') END,'priority',i.priority,'start_date',i.start_date,'target_date',i.target_date,'sequence_id',i.sequence_id,'sort_order',i.sort_order,'completed_at',i.completed_at,'archived_at',i.archived_at,'is_draft',i.is_draft,'external_source',i.external_source,'external_id',i.external_id) FROM issues i WHERE i.id=a.issue_id)`

var activityAPISerializerExpansions = map[string]string{
	"actor":     activityActorExpansion,
	"workspace": activityWorkspaceExpansion,
	"project":   activityProjectExpansion,
	"issue":     activityIssueExpansion,
}

var activityAPISerializerFields = map[string]struct{}{
	"id": {}, "created_at": {}, "updated_at": {}, "deleted_at": {},
	"workspace": {}, "project": {}, "issue": {}, "verb": {}, "field": {},
	"old_value": {}, "new_value": {}, "comment": {}, "attachments": {},
	"issue_comment": {}, "actor": {}, "old_identifier": {}, "new_identifier": {}, "epoch": {},
}

func activityAPISerializer(request Request) string {
	serializer := activityAPIJSON
	seen := make(map[string]struct{}, len(activityAPISerializerExpansions))
	for _, field := range strings.Split(param(request, "expand"), ",") {
		expression, expanded := activityAPISerializerExpansions[field]
		if !expanded {
			if _, serialized := activityAPISerializerFields[field]; !serialized {
				continue
			}
			expression = "NULL"
			if field == "issue_comment" {
				expression = "a.issue_comment_id"
			}
		}
		if _, duplicate := seen[field]; duplicate {
			continue
		}
		seen[field] = struct{}{}
		serializer = "(" + serializer + " || jsonb_build_object('" + field + "'," + expression + "))"
	}
	return serializer
}

var activityJSON = `jsonb_build_object(
'id',a.id,'created_at',a.created_at,'updated_at',a.updated_at,'created_by',a.created_by_id,'updated_by',a.updated_by_id,'deleted_at',a.deleted_at,
'workspace',a.workspace_id,'project',a.project_id,'issue',a.issue_id,'verb',a.verb,'field',a.field,
'old_value',a.old_value,'new_value',a.new_value,'comment',a.comment,'attachments',a.attachments,
'issue_comment',a.issue_comment_id,'actor',a.actor_id,'old_identifier',a.old_identifier,
'new_identifier',a.new_identifier,'epoch',a.epoch,'source_data',NULL,
'actor_detail',COALESCE((SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'display_name',u.display_name,'avatar',u.avatar,'avatar_url',` + fileAssetURLSQL("u.avatar_asset_id", "u.avatar") + `,'is_bot',u.is_bot) FROM users u WHERE u.id=a.actor_id),'{}'::jsonb),
'workspace_detail',(SELECT jsonb_build_object('id',w.id,'name',w.name,'slug',w.slug,'logo_url',` + fileAssetURLSQL("w.logo_asset_id", "w.logo") + `) FROM workspaces w WHERE w.id=a.workspace_id),
'project_detail',(SELECT jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'cover_image_url',` + fileAssetURLSQL("p.cover_image_asset_id", "p.cover_image") + `,'logo_props',p.logo_props,'description',p.description) FROM projects p WHERE p.id=a.project_id),
'issue_detail',(SELECT jsonb_build_object('id',i.id,'name',i.name,'description_html',i.description_html,'description_json',i.description_json,'priority',i.priority,'sequence_id',i.sequence_id,'sort_order',i.sort_order,'start_date',i.start_date,'target_date',i.target_date,'is_draft',i.is_draft) FROM issues i WHERE i.id=a.issue_id))`

func activityOrder(request Request, fallback string) string {
	value := param(request, "order_by")
	switch value {
	case "created_at":
		return "a.created_at ASC, a.id DESC"
	case "-created_at":
		return "a.created_at DESC, a.id DESC"
	case "updated_at":
		return "a.updated_at ASC NULLS LAST, a.created_at DESC, a.id DESC"
	case "-updated_at":
		return "a.updated_at DESC NULLS LAST, a.created_at DESC, a.id DESC"
	default:
		return fallback + ", a.id DESC"
	}
}

func (repository *Repository) userActivities(ctx context.Context, principal Principal, request Request) (Result, error) {
	limit, page, offset, err := pageWindow(request)
	if err != nil {
		return Result{}, err
	}
	order := activityOrder(request, "a.created_at DESC")
	value, err := repository.json(ctx, `WITH matching AS (
SELECT a.* FROM issue_activities a WHERE a.actor_id=$1 AND a.deleted_at IS NULL
), page AS (
SELECT * FROM matching a ORDER BY `+order+` LIMIT $2 OFFSET $3
)
SELECT jsonb_build_object('total_count',(SELECT count(*) FROM matching),'results',COALESCE(jsonb_agg(`+activityJSON+` ORDER BY `+order+`),'[]'::jsonb)) FROM page a`, principal.UserID, limit, offset)
	if err != nil {
		return Result{}, err
	}
	return paginatedResult(value, limit, page)
}
func (repository *Repository) apiActivities(ctx context.Context, principal Principal, request Request, detail bool) (Result, error) {
	projectID, err := uuid.Parse(param(request, "project_id"))
	if err != nil {
		return Result{}, ErrNotFound
	}
	issueID, err := uuid.Parse(param(request, "issue_id"))
	if err != nil {
		return Result{}, ErrNotFound
	}
	serializer := activityAPISerializer(request)
	if detail {
		pk, err := uuid.Parse(param(request, "pk"))
		if err != nil {
			return Result{}, ErrNotFound
		}
		value, err := repository.json(ctx, `SELECT `+serializer+` FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects p ON p.id=a.project_id WHERE a.id=$1 AND a.issue_id=$2 AND a.project_id=$3 AND w.slug=$4 AND a.deleted_at IS NULL AND p.archived_at IS NULL AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft')) AND EXISTS (SELECT 1 FROM project_members membership WHERE membership.project_id=p.id AND membership.member_id=$5 AND membership.is_active=true) LIMIT 1`, pk, issueID, projectID, param(request, "slug"), principal.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Result{}, &HTTPError{Status: 404, Body: map[string]any{"message": "Activity not found.", "code": "NOT_FOUND"}}
		}
		return Result{Body: value}, err
	}
	limit, page, offset, err := pageWindow(request)
	if err != nil {
		return Result{}, err
	}
	order := activityOrder(request, "a.created_at ASC")
	value, err := repository.json(ctx, `WITH matching AS (
SELECT a.* FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects p ON p.id=a.project_id JOIN project_members membership ON membership.project_id=p.id AND membership.member_id=$4 AND membership.is_active=true WHERE a.issue_id=$1 AND a.project_id=$2 AND w.slug=$3 AND a.deleted_at IS NULL AND p.archived_at IS NULL AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft'))
), page AS (
SELECT * FROM matching a ORDER BY `+order+` LIMIT $5 OFFSET $6
)
SELECT jsonb_build_object('total_count',(SELECT count(*) FROM matching),'results',COALESCE(jsonb_agg(`+serializer+` ORDER BY `+order+`),'[]'::jsonb)) FROM page a`, issueID, projectID, param(request, "slug"), principal.UserID, limit, offset)
	if err != nil {
		return Result{}, err
	}
	return paginatedResult(value, limit, page)
}
