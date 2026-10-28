// Package workitemhistory reads the source issue history and version models.
package workitemhistory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden       = errors.New("active project membership required")
	ErrProjectEntity   = errors.New("active project membership required by entity permission")
	ErrNotFound        = errors.New("work item or version not found")
	ErrGuestRestricted = errors.New("guest cannot read this work item")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("work item history database is required")
	}
	return &Repository{database: database}, nil
}

type Request struct {
	Operation    string
	Slug         string
	ProjectID    uuid.UUID
	ResourceID   uuid.UUID
	VersionID    uuid.UUID
	PropertyID   uuid.UUID
	BeforeID     *uuid.UUID
	UserID       uuid.UUID
	Timezone     string
	Cursor       *string
	CreatedAtGT  *string
	ActivityType *string
}

func (repository *Repository) Read(ctx context.Context, request Request) (json.RawMessage, error) {
	if request.Operation == "history" {
		allowed, err := repository.ProjectEntityAllowed(ctx, request.UserID.String(), request.Slug, request.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("authorize issue history entity: %w", err)
		}
		if !allowed {
			return nil, ErrProjectEntity
		}
	}
	if err := repository.authorizeProject(ctx, request); err != nil {
		return nil, err
	}
	switch request.Operation {
	case "meta":
		return repository.meta(ctx, request)
	case "history":
		return repository.history(ctx, request)
	case "versions":
		return repository.versions(ctx, request)
	case "version_detail":
		return repository.versionDetail(ctx, request)
	case "description_versions":
		return repository.descriptionVersions(ctx, request)
	case "description_version_detail":
		return repository.descriptionVersionDetail(ctx, request)
	case "property_history":
		return repository.propertyHistory(ctx, request)
	case "property_versions":
		return repository.propertyVersions(ctx, request)
	case "property_version_detail":
		return repository.propertyVersionDetail(ctx, request)
	default:
		return nil, fmt.Errorf("unknown work item history operation %q", request.Operation)
	}
}

// ProjectEntityAllowed mirrors IssueActivityEndpoint's class permission. It
// intentionally checks active membership without applying the GET role decorator.
func (repository *Repository) ProjectEntityAllowed(ctx context.Context, userID, slug string, projectID uuid.UUID) (bool, error) {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
 WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
   AND pm.is_active=true AND pm.deleted_at IS NULL
)`, userID, projectID, slug).Scan(&allowed)
	return allowed, err
}

func (repository *Repository) authorizeProject(ctx context.Context, request Request) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
 WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
   AND pm.is_active=true AND pm.deleted_at IS NULL
   AND (pm.role IN (5,15,20) OR EXISTS (
     SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1
       AND wm.is_active=true AND wm.deleted_at IS NULL AND wm.role=20
   ))
)`, request.UserID, request.ProjectID, request.Slug).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize work item history: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) meta(ctx context.Context, request Request) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(`SELECT jsonb_build_object('sequence_id',i.sequence_id,'project_identifier',p.identifier)
 FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id
 LEFT JOIN states s ON s.id=i.state_id
 WHERE i.id=$1 AND i.project_id=$2 AND w.slug=$3 AND i.deleted_at IS NULL
   AND i.archived_at IS NULL AND p.archived_at IS NULL AND NOT i.is_draft
   AND COALESCE(s."group",'') <> 'triage' LIMIT 1`, "$1", "$3", "$4::uuid"), request.ResourceID, request.ProjectID, request.Slug, request.UserID).Scan(&data)
	return scannedJSON(data, err, "get issue metadata")
}

func (repository *Repository) versions(ctx context.Context, request Request) (json.RawMessage, error) {
	page, err := parseCursor(request.Cursor)
	if err != nil {
		return nil, err
	}
	if page.limit <= 0 {
		return nil, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	offset, err := page.offset()
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, visibleItemHistorySQL(issueVersionListSQL, "$3", "$1", "$7::uuid"), request.Slug, request.ProjectID, request.ResourceID, page.limit, offset, page.page, request.UserID).Scan(&data)
	return transformedJSON(data, err, "list issue versions", request.Timezone, false)
}

func (repository *Repository) versionDetail(ctx context.Context, request Request) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(issueVersionDetailSQL, "$3", "$1", "$5::uuid"), request.Slug, request.ProjectID, request.ResourceID, request.VersionID, request.UserID).Scan(&data)
	return transformedJSON(data, err, "get issue version", request.Timezone, true)
}

func (repository *Repository) descriptionVersions(ctx context.Context, request Request) (json.RawMessage, error) {
	// The source handler resolves the project and issue before parsing its cursor.
	if err := repository.ensureProjectAndIssue(ctx, request); err != nil {
		return nil, err
	}
	if err := repository.authorizeDescriptionGuest(ctx, request); err != nil {
		return nil, err
	}
	page, err := parseCursor(request.Cursor)
	if err != nil {
		return nil, err
	}
	if page.limit <= 0 {
		return nil, &djangoapi.HTTPError{Status: 500, Body: map[string]any{"error": "Something went wrong please try again later"}}
	}
	offset, err := page.offset()
	if err != nil {
		return nil, err
	}
	var data []byte
	err = repository.database.QueryRow(ctx, visibleItemHistorySQL(descriptionVersionListSQL, "$3", "$1", "$7::uuid"), request.Slug, request.ProjectID, request.ResourceID, page.limit, offset, page.page, request.UserID).Scan(&data)
	return transformedJSON(data, err, "list work item description versions", request.Timezone, false)
}

func (repository *Repository) descriptionVersionDetail(ctx context.Context, request Request) (json.RawMessage, error) {
	if err := repository.ensureProjectAndIssue(ctx, request); err != nil {
		return nil, err
	}
	if err := repository.authorizeDescriptionGuest(ctx, request); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(descriptionVersionDetailSQL, "$3", "$1", "$5::uuid"), request.Slug, request.ProjectID, request.ResourceID, request.VersionID, request.UserID).Scan(&data)
	return transformedJSON(data, err, "get work item description version", request.Timezone, true)
}

func (repository *Repository) ensureProjectAndIssue(ctx context.Context, request Request) error {
	var exists bool
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1 AND deleted_at IS NULL)`, request.ProjectID).Scan(&exists); err != nil {
		return fmt.Errorf("look up work item history project: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	if err := repository.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE w.slug=$1 AND i.project_id=$2 AND i.id=$3 AND i.deleted_at IS NULL)`, request.Slug, request.ProjectID, request.ResourceID).Scan(&exists); err != nil {
		return fmt.Errorf("look up work item history issue: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (repository *Repository) authorizeDescriptionGuest(ctx context.Context, request Request) error {
	var denied bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM project_members pm JOIN projects p ON p.id=pm.project_id
 JOIN issues i ON i.id=$3 JOIN workspaces w ON w.id=pm.workspace_id
 WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$4 AND pm.role=5
   AND pm.is_active=true AND pm.deleted_at IS NULL AND NOT p.guest_view_all_features
   AND i.created_by_id IS DISTINCT FROM $1
)`, request.UserID, request.ProjectID, request.ResourceID, request.Slug).Scan(&denied)
	if err != nil {
		return fmt.Errorf("authorize guest work item description history: %w", err)
	}
	if denied {
		return ErrGuestRestricted
	}
	return nil
}

func (repository *Repository) history(ctx context.Context, request Request) (json.RawMessage, error) {
	var createdAfter any
	if request.CreatedAtGT != nil && *request.CreatedAtGT != "" {
		parsed, err := pythoncompat.ISODateTime(*request.CreatedAtGT, time.UTC)
		if err != nil {
			return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Please provide valid detail"}}
		}
		createdAfter = parsed
	}
	query := historyMixedSQL
	if request.ActivityType != nil && *request.ActivityType == "issue-property" {
		query = historyActivitiesSQL
	} else if request.ActivityType != nil && *request.ActivityType == "issue-comment" {
		query = historyCommentsSQL
	}
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(query, "$1", "$3", "$5::uuid"), request.ResourceID, request.ProjectID, request.Slug, createdAfter, request.UserID).Scan(&data)
	return transformedJSON(data, err, "get issue history", request.Timezone, true)
}

type cursorPage struct{ limit, page int }

func (page cursorPage) offset() (int, error) {
	if page.limit <= 0 {
		return 0, invalidCursor()
	}
	if page.page < 0 {
		return 0, nil // Django's queryset slicing path treats negative pages as its first page.
	}
	if page.page > math.MaxInt/page.limit {
		return 0, invalidCursor()
	}
	return page.page * page.limit, nil
}

func parseCursor(raw *string) (cursorPage, error) {
	if raw == nil {
		return cursorPage{limit: 1000}, nil
	}
	parts := strings.Split(*raw, ":")
	if len(parts) != 3 {
		return cursorPage{}, invalidCursor()
	}
	values := make([]*big.Int, len(parts))
	for index, part := range parts {
		value, err := pythoncompat.DecimalInteger(part)
		if err != nil {
			return cursorPage{}, invalidCursor()
		}
		values[index] = value
	}
	limitValue := values[0]
	if limitValue.Sign() <= 0 {
		return cursorPage{}, invalidCursor()
	}
	if limitValue.Cmp(big.NewInt(1000)) > 0 {
		limitValue = big.NewInt(1000)
	}
	if !limitValue.IsInt64() || !values[1].IsInt64() || int64(int(values[1].Int64())) != values[1].Int64() || int64(int(limitValue.Int64())) != limitValue.Int64() {
		return cursorPage{}, invalidCursor()
	}
	return cursorPage{limit: int(limitValue.Int64()), page: int(values[1].Int64())}, nil
}

func invalidCursor() error {
	return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"detail": "Invalid cursor parameter."}}
}

func scannedJSON(data []byte, err error, operation string) (json.RawMessage, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("%s returned invalid JSON", operation)
	}
	return json.RawMessage(data), nil
}

func transformedJSON(data []byte, err error, operation, timezone string, detail bool) (json.RawMessage, error) {
	result, err := scannedJSON(data, err, operation)
	if err != nil {
		return nil, err
	}
	location := time.UTC
	if timezone != "" {
		location, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("load response timezone: %w", err)
		}
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(result)))
	decoder.UseNumber()
	if decodeErr := decoder.Decode(&value); decodeErr != nil {
		return nil, decodeErr
	}
	convertResponseTimes(value, location, detail)
	encoded, err := json.Marshal(value)
	return encoded, err
}

func convertResponseTimes(value any, location *time.Location, detail bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if timestamp, ok := nested.(string); ok && (key == "created_at" || key == "updated_at" || key == "last_saved_at") {
				if parsed, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
					timestampLocation := location
					if key == "last_saved_at" && !detail {
						timestampLocation = time.UTC
					}
					typed[key] = pythoncompat.DRFDateTimeString(parsed, timestampLocation)
				}
			} else {
				convertResponseTimes(nested, location, detail)
			}
		}
	case []any:
		for _, nested := range typed {
			convertResponseTimes(nested, location, detail)
		}
	}
}

const issueVersionListSQL = `WITH matching AS (
 SELECT v.* FROM issue_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.project_id=$2 AND v.issue_id=$3 AND v.deleted_at IS NULL
), page AS (SELECT * FROM matching ORDER BY created_at DESC LIMIT $4 OFFSET $5), totals AS (SELECT count(*)::int AS total FROM matching)
SELECT jsonb_build_object(
 'prev_cursor',concat($4::int,':',$6::bigint-1,':0'), 'cursor',concat($4::int,':',$6::bigint,':0'),
 'next_cursor',CASE WHEN totals.total>$5::numeric+$4::int THEN concat($4::int,':',$6::bigint+1,':0') ELSE NULL END,
 'prev_page_results',$6::bigint>0, 'next_page_results',totals.total>$5::numeric+$4::int,
 'page_count',count(page.id)::int,'total_results',totals.total,'total_pages',ceil(totals.total::numeric/NULLIF($4::int,0))::int,
 'results',COALESCE(jsonb_agg(jsonb_build_object('id',page.id,'workspace',page.workspace_id,'project',page.project_id,'issue',page.issue_id,'last_saved_at',page.last_saved_at,'owned_by',page.owned_by_id,'created_at',page.created_at,'updated_at',page.updated_at,'created_by',page.created_by_id,'updated_by',page.updated_by_id) ORDER BY page.created_at DESC) FILTER(WHERE page.id IS NOT NULL),'[]'::jsonb)
) FROM totals LEFT JOIN page ON true GROUP BY totals.total`

var issueVersionDetailSQL = `SELECT jsonb_build_object(
 'id',v.id,'workspace',v.workspace_id,'project',v.project_id,'issue',v.issue_id,'parent',(SELECT related.id FROM issues related WHERE related.id=v.parent AND ` + djangoapi.IssueVisibleSQL("related", "$5::uuid", "v.workspace_id") + `),'state',v.state,
 'estimate_point',v.estimate_point,'name',v.name,'priority',v.priority,'start_date',v.start_date,'target_date',v.target_date,
 'assignees',v.assignees,'sequence_id',v.sequence_id,'labels',v.labels,'sort_order',v.sort_order,'completed_at',v.completed_at,
 'archived_at',v.archived_at,'is_draft',v.is_draft,'external_source',v.external_source,'external_id',v.external_id,'type',v.type,
 'cycle',v.cycle,'modules',v.modules,'meta',v.meta,'last_saved_at',v.last_saved_at,'owned_by',v.owned_by_id,
 'created_at',v.created_at,'updated_at',v.updated_at,'created_by',v.created_by_id,'updated_by',v.updated_by_id)
 FROM issue_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.project_id=$2 AND v.issue_id=$3 AND v.id=$4 AND v.deleted_at IS NULL LIMIT 1`

const descriptionVersionListSQL = `WITH matching AS (
 SELECT v.* FROM issue_description_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.project_id=$2 AND v.issue_id=$3 AND v.deleted_at IS NULL
), page AS (SELECT * FROM matching ORDER BY created_at DESC LIMIT $4 OFFSET $5), totals AS (SELECT count(*)::int AS total FROM matching)
SELECT jsonb_build_object(
 'prev_cursor',concat($4::int,':',$6::bigint-1,':0'), 'cursor',concat($4::int,':',$6::bigint,':0'),
 'next_cursor',CASE WHEN totals.total>$5::numeric+$4::int THEN concat($4::int,':',$6::bigint+1,':0') ELSE NULL END,
 'prev_page_results',$6::bigint>0, 'next_page_results',totals.total>$5::numeric+$4::int,
 'page_count',count(page.id)::int,'total_results',totals.total,'total_pages',ceil(totals.total::numeric/NULLIF($4::int,0))::int,
 'results',COALESCE(jsonb_agg(jsonb_build_object('id',page.id,'workspace',page.workspace_id,'project',page.project_id,'issue',page.issue_id,'last_saved_at',page.last_saved_at,'owned_by',page.owned_by_id,'created_at',page.created_at,'updated_at',page.updated_at,'created_by',page.created_by_id,'updated_by',page.updated_by_id) ORDER BY page.created_at DESC) FILTER(WHERE page.id IS NOT NULL),'[]'::jsonb)
) FROM totals LEFT JOIN page ON true GROUP BY totals.total`

const descriptionVersionDetailSQL = `SELECT jsonb_build_object(
 'id',v.id,'workspace',v.workspace_id,'project',v.project_id,'issue',v.issue_id,
 'description_binary',CASE WHEN v.description_binary IS NULL THEN NULL ELSE replace(encode(v.description_binary,'base64'),E'\\n','') END,
 'description_html',v.description_html,'description_stripped',v.description_stripped,'description_json',v.description_json,
 'last_saved_at',v.last_saved_at,'owned_by',v.owned_by_id,'created_at',v.created_at,'updated_at',v.updated_at,
 'created_by',v.created_by_id,'updated_by',v.updated_by_id)
 FROM issue_description_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.project_id=$2 AND v.issue_id=$3 AND v.id=$4 AND v.deleted_at IS NULL LIMIT 1`

var activityBase = `((to_jsonb(a)-'workspace_id'-'project_id'-'issue_id'-'issue_comment_id'-'actor_id'-'created_by_id'-'updated_by_id') || jsonb_build_object(
 'workspace',a.workspace_id,'project',a.project_id,'issue',a.issue_id,'issue_comment',a.issue_comment_id,'actor',a.actor_id,
 'created_by',a.created_by_id,'updated_by',a.updated_by_id,
 'actor_detail',COALESCE((SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + ` FROM users u WHERE u.id=a.actor_id),'{}'::jsonb),
 'issue_detail',(SELECT jsonb_build_object('id',i.id,'name',i.name,'description_json',i.description_json,'description_html',i.description_html,'priority',i.priority,'start_date',i.start_date,'target_date',i.target_date,'sequence_id',i.sequence_id,'sort_order',i.sort_order,'is_draft',i.is_draft) FROM issues i WHERE i.id=a.issue_id),
 'project_detail',(SELECT jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'logo_props',p.logo_props,'description',p.description)||` + djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url") + ` FROM projects p WHERE p.id=a.project_id),
 'workspace_detail',(SELECT jsonb_build_object('name',w.name,'slug',w.slug,'id',w.id)||` + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + ` FROM workspaces w WHERE w.id=a.workspace_id),
 'source_data',(SELECT jsonb_build_object('source',ii.source,'source_email',ii.source_email,'extra',ii.extra) FROM intake_issues ii WHERE ii.issue_id=a.issue_id AND ii.deleted_at IS NULL ORDER BY ii.created_at DESC LIMIT 1)
 ))` + parentActivityVisibilitySQL()

var activityBaseWithoutSourceData = strings.Replace(
	activityBase,
	"'source_data',(SELECT jsonb_build_object('source',ii.source,'source_email',ii.source_email,'extra',ii.extra) FROM intake_issues ii WHERE ii.issue_id=a.issue_id AND ii.deleted_at IS NULL ORDER BY ii.created_at DESC LIMIT 1)",
	"'source_data',NULL",
	1,
)

var commentBase = `((to_jsonb(c)-'workspace_id'-'project_id'-'issue_id'-'description_id'-'actor_id'-'parent_id'-'created_by_id'-'updated_by_id') || jsonb_build_object(
 'workspace',c.workspace_id,'project',c.project_id,'issue',c.issue_id,'description',c.description_id,'actor',c.actor_id,'parent',c.parent_id,
 'created_by',c.created_by_id,'updated_by',c.updated_by_id,
 'actor_detail',COALESCE((SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + ` FROM users u WHERE u.id=c.actor_id),'{}'::jsonb),
 'issue_detail',(SELECT jsonb_build_object('id',i.id,'name',i.name,'description_json',i.description_json,'description_html',i.description_html,'priority',i.priority,'start_date',i.start_date,'target_date',i.target_date,'sequence_id',i.sequence_id,'sort_order',i.sort_order,'is_draft',i.is_draft) FROM issues i WHERE i.id=c.issue_id),
 'project_detail',(SELECT jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'logo_props',p.logo_props,'description',p.description)||` + djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url") + ` FROM projects p WHERE p.id=c.project_id),
 'workspace_detail',(SELECT jsonb_build_object('name',w.name,'slug',w.slug,'id',w.id)||` + djangoapi.FileAssetURLFieldSQL("w.logo_asset_id", "w.logo", "logo_url") + ` FROM workspaces w WHERE w.id=c.workspace_id),
 'comment_reactions',COALESCE((SELECT jsonb_agg((to_jsonb(cr)-'workspace_id'-'project_id'-'comment_id'-'actor_id'-'created_by_id'-'updated_by_id')||jsonb_build_object('workspace',cr.workspace_id,'project',cr.project_id,'comment',cr.comment_id,'actor',cr.actor_id,'created_by',cr.created_by_id,'updated_by',cr.updated_by_id,'display_name',(SELECT u.display_name FROM users u WHERE u.id=cr.actor_id)) ORDER BY cr.created_at DESC) FROM comment_reactions cr WHERE cr.comment_id=c.id AND cr.deleted_at IS NULL),'[]'::jsonb)
 ))`

var activityFilter = `a.issue_id=$1 AND a.project_id=$2 AND w.slug=$3 AND a.deleted_at IS NULL AND p.archived_at IS NULL
 AND (a.field IS NULL OR a.field NOT IN ('comment','vote','reaction','draft')) AND ($4::timestamptz IS NULL OR a.created_at>$4) AND ` + issuequery.MilestoneActivityVisibleSQL("a.field", "a.issue_id", "a.project_id", "a.workspace_id", "$5::uuid")

const commentFilter = `$5::uuid IS NOT NULL AND c.issue_id=$1 AND c.project_id=$2 AND w.slug=$3 AND c.deleted_at IS NULL AND p.archived_at IS NULL AND ($4::timestamptz IS NULL OR c.created_at>$4)`

var historyActivitiesSQL = `SELECT COALESCE(jsonb_agg(row_json ORDER BY created_at ASC,id), '[]'::jsonb) FROM (
 SELECT a.id,a.created_at,` + activityBase + ` AS row_json FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects p ON p.id=a.project_id WHERE ` + activityFilter + `
) history`

var historyCommentsSQL = `SELECT COALESCE(jsonb_agg(row_json ORDER BY created_at ASC,id), '[]'::jsonb) FROM (
 SELECT c.id,c.created_at,` + commentBase + ` AS row_json FROM issue_comments c JOIN workspaces w ON w.id=c.workspace_id JOIN projects p ON p.id=c.project_id WHERE ` + commentFilter + `
) history`

var historyMixedSQL = `SELECT COALESCE(jsonb_agg(row_json ORDER BY created_at ASC,kind ASC,id), '[]'::jsonb) FROM (
 SELECT a.id,a.created_at,0 AS kind,` + activityBaseWithoutSourceData + ` AS row_json FROM issue_activities a JOIN workspaces w ON w.id=a.workspace_id JOIN projects p ON p.id=a.project_id WHERE ` + activityFilter + `
 UNION ALL
 SELECT c.id,c.created_at,1 AS kind,` + commentBase + ` AS row_json FROM issue_comments c JOIN workspaces w ON w.id=c.workspace_id JOIN projects p ON p.id=c.project_id WHERE ` + commentFilter + `
) history`
