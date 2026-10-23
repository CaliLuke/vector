// Package pagereads contains the read-only project page boundary.
package pagereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"uuid"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden       = errors.New("active project page access required")
	ErrNotFound        = errors.New("page read model not found")
	ErrRestrictedGuest = errors.New("guest cannot view this page")
	ErrInvalidOrder    = errors.New("invalid page order field")
	ErrInvalidSearch   = errors.New("null characters are not allowed in page search")
	ErrDjangoPageNil   = errors.New("Django page detail dereferenced a missing searched page")
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Publisher interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

// NativePageTaskAdmitter is the optional typed durable path for page effects.
// Celery publishing remains available for unrelated page tasks during rollout.
type NativePageTaskAdmitter interface {
	AdmitPageTransaction(context.Context, string, any, any, string) error
	AdmitPageVersion(context.Context, string, string, any, string) error
	AdmitPageDuplicateAssets(context.Context, string, string, string, string, string, string) error
}

type NativeRecentVisitAdmitter interface {
	AdmitRecentVisit(context.Context, string, string, string, string, string, string) error
}

type Repository struct {
	database     database
	publisher    Publisher
	nativeTasks  NativePageTaskAdmitter
	recentVisits NativeRecentVisitAdmitter
}

func NewRepository(database database, publishers ...Publisher) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("page reads database is required")
	}
	var publisher Publisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}
	return &Repository{database: database, publisher: publisher}, nil
}

func NewRepositoryWithNativeTasks(database database, publisher Publisher, nativeTasks NativePageTaskAdmitter, recentVisits ...NativeRecentVisitAdmitter) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("page reads database is required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("page task publisher is required")
	}
	if len(recentVisits) > 1 {
		return nil, fmt.Errorf("at most one native recent visit adapter may be configured")
	}
	var recent NativeRecentVisitAdmitter
	if len(recentVisits) == 1 {
		recent = recentVisits[0]
	}
	return &Repository{database: database, publisher: publisher, nativeTasks: nativeTasks, recentVisits: recent}, nil
}

func (repository *Repository) admitRecentVisit(ctx context.Context, userID uuid.UUID, slug string, projectID, pageID uuid.UUID) error {
	if repository.recentVisits != nil {
		return repository.recentVisits.AdmitRecentVisit(ctx, uuid.New().String(), slug, "page", pageID.String(), userID.String(), projectID.String())
	}
	if repository.publisher == nil {
		return nil
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{
		"slug": slug, "entity_name": "page", "entity_identifier": pageID.String(), "user_id": userID.String(), "project_id": projectID.String(),
	})
}

func (repository *Repository) admitPageTransaction(ctx context.Context, newHTML, oldHTML any, pageID string) error {
	if repository.nativeTasks != nil {
		return repository.nativeTasks.AdmitPageTransaction(ctx, uuid.New().String(), newHTML, oldHTML, pageID)
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.page_transaction_task.page_transaction", []any{}, map[string]any{
		"new_description_html": newHTML, "old_description_html": oldHTML, "page_id": pageID,
	})
}

func (repository *Repository) admitPageVersion(ctx context.Context, pageID string, existing any, userID string) error {
	if repository.nativeTasks != nil {
		return repository.nativeTasks.AdmitPageVersion(ctx, uuid.New().String(), pageID, existing, userID)
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.page_version_task.track_page_version", []any{}, map[string]any{
		"page_id": pageID, "existing_instance": existing, "user_id": userID,
	})
}

func (repository *Repository) Summary(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, pageSummarySQL, userID, slug, projectID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("summarize project pages: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) List(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	search string,
	orderBy string,
	userTimezone string,
) (json.RawMessage, error) {
	if err := repository.authorizeProject(ctx, userID, slug, projectID); err != nil {
		return nil, err
	}
	if !validPageOrder(orderBy) {
		return nil, ErrInvalidOrder
	}
	if strings.IndexByte(search, 0) >= 0 {
		return nil, ErrInvalidSearch
	}
	searchTerms := pageSearchPatterns(search)
	var data []byte
	err := repository.database.QueryRow(ctx, pageListSQL, userID, slug, projectID, searchTerms, userTimezone).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list project pages: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Detail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
	search string,
	orderBy string,
	userTimezone string,
) (json.RawMessage, error) {
	authorizationErr := repository.authorizePage(ctx, userID, slug, projectID, pageID, true)
	if authorizationErr != nil &&
		!errors.Is(authorizationErr, ErrRestrictedGuest) &&
		!errors.Is(authorizationErr, ErrDjangoPageNil) &&
		!errors.Is(authorizationErr, ErrNotFound) {
		return nil, authorizationErr
	}
	if !validPageOrder(orderBy) {
		return nil, ErrInvalidOrder
	}
	if strings.IndexByte(search, 0) >= 0 {
		return nil, ErrInvalidSearch
	}
	searchTerms := pageSearchPatterns(search)
	if errors.Is(authorizationErr, ErrDjangoPageNil) {
		return nil, ErrDjangoPageNil
	}
	if errors.Is(authorizationErr, ErrNotFound) {
		return nil, ErrNotFound
	}
	if errors.Is(authorizationErr, ErrRestrictedGuest) {
		if len(searchTerms) != 0 {
			var matches bool
			if err := repository.database.QueryRow(ctx, pageSearchMatchSQL, pageID, searchTerms).Scan(&matches); err != nil {
				return nil, fmt.Errorf("match restricted project page search: %w", err)
			}
			if !matches {
				return nil, ErrDjangoPageNil
			}
		}
		return nil, ErrRestrictedGuest
	}
	var data []byte
	err := repository.database.QueryRow(ctx, pageDetailSQL, userID, slug, projectID, pageID, searchTerms, userTimezone).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project page: %w", err)
	}
	if err := repository.admitRecentVisit(ctx, userID, slug, projectID, pageID); err != nil {
		return nil, fmt.Errorf("record recent page visit: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Description(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
) ([]byte, error) {
	if err := repository.authorizePage(ctx, userID, slug, projectID, pageID, false); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, pageDescriptionSQL, userID, slug, projectID, pageID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project page description: %w", err)
	}
	return data, nil
}

func (repository *Repository) VersionList(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
	userTimezone string,
) (json.RawMessage, error) {
	if err := repository.authorizePage(ctx, userID, slug, projectID, pageID, false); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, pageVersionListSQL, userID, slug, projectID, pageID, userTimezone).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list project page versions: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) VersionDetail(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
	versionID uuid.UUID,
	userTimezone string,
) (json.RawMessage, error) {
	if err := repository.authorizePage(ctx, userID, slug, projectID, pageID, false); err != nil {
		return nil, err
	}
	var data []byte
	err := repository.database.QueryRow(ctx, pageVersionDetailSQL, userID, slug, projectID, pageID, versionID, userTimezone).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project page version: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) authorizeProject(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
) error {
	var allowed bool
	err := repository.database.QueryRow(ctx, pageProjectAuthorizationSQL, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize project page read: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (repository *Repository) AuthorizeOptions(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID *uuid.UUID,
) error {
	if pageID == nil {
		return repository.authorizeProject(ctx, userID, slug, projectID)
	}
	return repository.authorizePage(ctx, userID, slug, projectID, *pageID, false)
}

func (repository *Repository) authorizePage(
	ctx context.Context,
	userID uuid.UUID,
	slug string,
	projectID uuid.UUID,
	pageID uuid.UUID,
	restrictGuest bool,
) error {
	var access string
	err := repository.database.QueryRow(
		ctx,
		pageAuthorizationSQL,
		userID,
		slug,
		projectID,
		pageID,
		restrictGuest,
	).Scan(&access)
	if err != nil {
		return fmt.Errorf("authorize page read: %w", err)
	}
	switch access {
	case "allowed":
		return nil
	case "restricted_guest":
		return ErrRestrictedGuest
	case "django_page_nil":
		return ErrDjangoPageNil
	case "not_found":
		return ErrNotFound
	default:
		return ErrForbidden
	}
}

type pageOrder struct {
	field      string
	descending bool
}

func validPageOrder(orderBy string) bool {
	_, valid := parsePageOrder(orderBy)
	return valid
}

func parsePageOrder(orderBy string) (pageOrder, bool) {
	if orderBy == "?" {
		return pageOrder{field: "?"}, true
	}
	descending := strings.HasPrefix(orderBy, "-")
	field := strings.TrimPrefix(orderBy, "-")
	if field == "" {
		return pageOrder{}, false
	}
	first, suffix, hasSuffix := strings.Cut(field, "__")
	if first == "pk" {
		first = "id"
	}
	_, scalar := pageOrderScalars["page"][first]
	_, relation := pageOrderRelations["page"][first]
	if !scalar && !relation {
		return pageOrder{}, false
	}
	if hasSuffix {
		first += "__" + suffix
	}
	return pageOrder{field: first, descending: descending}, true
}

var pageSearchSplitPattern = regexp.MustCompile(`(?:(?:[^\s'"]*(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')[^\s'"]*)+|\S+)`)

var pageLikeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func pageSearchTerms(value string) []string {
	terms := make([]string, 0, 1)
	for _, term := range pageSearchSplitPattern.FindAllString(value, -1) {
		term = strings.Trim(term, ",")
		if len(term) >= 2 && (term[0] == '"' || term[0] == '\'') && term[len(term)-1] == term[0] {
			quote := string(term[0])
			unescaped := strings.ReplaceAll(term[1:len(term)-1], `\`+quote, quote)
			terms = append(terms, strings.ReplaceAll(unescaped, `\\`, `\`))
			continue
		}
		for subTerm := range strings.SplitSeq(term, ",") {
			if subTerm != "" {
				terms = append(terms, strings.TrimSpace(subTerm))
			}
		}
	}
	return terms
}

func pageSearchPatterns(value string) []string {
	terms := pageSearchTerms(value)
	for index := range terms {
		terms[index] = pageLikeEscaper.Replace(terms[index])
	}
	return terms
}

var pageOrderScalars = map[string]map[string]struct{}{
	"page": set(
		"access", "archived_at", "color", "created_at", "created_by_id", "deleted_at",
		"description_binary", "description_html", "description_json", "description_stripped",
		"external_id", "external_source", "id", "is_favorite", "is_global", "is_locked",
		"logo_props", "moved_to_page", "moved_to_project", "name", "owned_by_id", "parent_id",
		"sort_order", "updated_at", "updated_by_id", "view_props", "workspace_id",
	),
	"user": set(
		"id", "username", "mobile_number", "email", "display_name", "first_name", "last_name",
		"avatar", "cover_image", "date_joined", "created_at", "updated_at", "last_location",
		"created_location", "is_superuser", "is_managed", "is_password_expired", "is_active",
		"is_staff", "is_email_verified", "is_password_autoset", "is_password_reset_required",
		"token", "last_active", "last_login_time", "last_logout_time", "last_login_ip",
		"last_logout_ip", "last_login_medium", "last_login_uagent", "token_updated_at",
		"is_bot", "bot_type", "user_timezone", "is_email_valid", "masked_at",
	),
	"workspace": set(
		"id", "name", "slug", "organization_size", "logo", "timezone", "created_at", "updated_at",
		"deleted_at", "created_by_id", "updated_by_id",
	),
	"label": set(
		"id", "name", "description", "color", "sort_order", "external_id", "external_source",
		"created_at", "updated_at", "deleted_at", "created_by_id", "updated_by_id",
		"workspace_id", "project_id", "parent_id",
	),
	"project": set(
		"id", "name", "identifier", "description", "description_text", "description_html",
		"network", "emoji", "icon_prop", "module_view", "cycle_view", "issue_views_view",
		"page_view", "inbox_view", "cover_image", "archive_in", "close_in", "logo_props",
		"archived_at", "created_at", "updated_at", "deleted_at", "created_by_id",
		"updated_by_id", "workspace_id",
	),
	"page_log": set(
		"id", "transaction", "entity_identifier", "entity_name", "entity_type",
		"created_at", "updated_at", "deleted_at", "created_by_id", "updated_by_id",
		"page_id", "workspace_id",
	),
	"page_label": set(
		"id", "created_at", "updated_at", "deleted_at", "created_by_id", "updated_by_id",
		"page_id", "workspace_id", "label_id",
	),
	"project_page": set(
		"id", "created_at", "updated_at", "deleted_at", "created_by_id", "updated_by_id",
		"page_id", "workspace_id", "project_id",
	),
	"page_version": set(
		"id", "last_saved_at", "description_binary", "description_html", "description_stripped",
		"description_json", "sub_pages_data", "created_at", "updated_at", "deleted_at",
		"created_by_id", "updated_by_id", "page_id", "workspace_id", "owned_by_id",
	),
	"asset": set(
		"id", "attributes", "asset", "entity_type", "entity_identifier", "is_deleted", "is_archived",
		"external_id", "external_source", "size", "is_uploaded", "storage_metadata",
		"created_at", "updated_at", "deleted_at",
	),
}

var pageOrderRelations = map[string]map[string]string{
	"page": {
		"workspace": "workspace", "owned_by": "user", "created_by": "user", "updated_by": "user",
		"labels": "label", "parent": "page", "projects": "project", "child_page": "page",
		"page_log": "page_log", "page_labels": "page_label", "project_pages": "project_page",
		"page_versions": "page_version", "assets": "asset",
	},
	"workspace": {
		"created_by": "user", "updated_by": "user",
	},
	"label": {
		"created_by": "user", "updated_by": "user", "workspace": "workspace", "project": "project",
		"parent": "label", "pages": "page",
	},
	"project": {
		"created_by": "user", "updated_by": "user", "workspace": "workspace", "pages": "page",
	},
	"page_log": {
		"created_by": "user", "updated_by": "user", "page": "page", "workspace": "workspace",
	},
	"page_label": {
		"created_by": "user", "updated_by": "user", "page": "page", "workspace": "workspace",
		"label": "label",
	},
	"project_page": {
		"created_by": "user", "updated_by": "user", "page": "page", "workspace": "workspace",
		"project": "project",
	},
	"page_version": {
		"created_by": "user", "updated_by": "user", "page": "page", "workspace": "workspace",
		"owned_by": "user",
	},
	"asset": {
		"created_by": "user", "updated_by": "user", "page": "page", "workspace": "workspace",
		"user": "user", "project": "project",
	},
}

func set(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func djangoDateTimeSQL(column, timezoneParam string) string {
	localTimestamp := "(" + column + " AT TIME ZONE " + timezoneParam + ")"
	offsetSeconds := "(EXTRACT(EPOCH FROM (" + localTimestamp + " - (" + column + " AT TIME ZONE 'UTC')))::bigint)"
	return "CASE WHEN " + column + " IS NULL THEN NULL ELSE " +
		"to_char(" + localTimestamp + ", 'YYYY-MM-DD\"T\"HH24:MI:SS') || " +
		"CASE WHEN (EXTRACT(MICROSECONDS FROM " + localTimestamp + ")::bigint % 1000000) = 0 " +
		"THEN '' ELSE to_char(" + localTimestamp + ", '.US') END || " +
		"CASE WHEN " + offsetSeconds + " = 0 THEN 'Z' ELSE " +
		"CASE WHEN " + offsetSeconds + " < 0 THEN '-' ELSE '+' END || " +
		"lpad((abs(" + offsetSeconds + ") / 3600)::text, 2, '0') || ':' || " +
		"lpad(((abs(" + offsetSeconds + ") % 3600) / 60)::text, 2, '0') || " +
		"CASE WHEN (abs(" + offsetSeconds + ") % 60) = 0 THEN '' ELSE ':' || " +
		"lpad((abs(" + offsetSeconds + ") % 60)::text, 2, '0') END END END"
}

const pageProjectAuthorizationSQL = `
SELECT EXISTS (
  SELECT 1
  FROM project_members project_member
  JOIN projects project ON project.id = project_member.project_id
  JOIN workspaces workspace ON workspace.id = project.workspace_id
  WHERE project_member.member_id = $1
    AND workspace.slug = $2
    AND project.id = $3
    AND project_member.role IN (5, 15, 20)
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
)
`

const pageAuthorizationSQL = `
WITH page_access AS (
  SELECT page.owned_by_id, page.access, project_member.role, project.guest_view_all_features,
    page.parent_id IS NULL AS is_top_level,
    EXISTS (
      SELECT 1
      FROM project_pages visible_link
      JOIN projects visible_project
        ON visible_project.id = visible_link.project_id
        AND visible_project.workspace_id = page.workspace_id
        AND visible_project.archived_at IS NULL
        AND visible_project.deleted_at IS NULL
      JOIN project_members visible_member
        ON visible_member.project_id = visible_project.id
        AND visible_member.member_id = $1
        AND visible_member.is_active = true
        AND visible_member.deleted_at IS NULL
      WHERE visible_link.page_id = page.id
        AND visible_link.deleted_at IS NULL
    ) AS has_visible_project
  FROM pages page
  JOIN workspaces workspace ON workspace.id = page.workspace_id
  JOIN project_pages project_page
    ON project_page.page_id = page.id
    AND project_page.workspace_id = workspace.id
    AND project_page.project_id = $3
    AND project_page.deleted_at IS NULL
  JOIN projects project
    ON project.id = project_page.project_id
    AND project.workspace_id = workspace.id
  JOIN project_members project_member
    ON project_member.project_id = project.id
    AND project_member.workspace_id = workspace.id
    AND project_member.member_id = $1
    AND project_member.role IN (5, 15, 20)
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
  WHERE workspace.slug = $2
    AND page.id = $4
    AND page.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND workspace.deleted_at IS NULL
  LIMIT 1
)
SELECT COALESCE((
  SELECT CASE
    WHEN access = 1 AND owned_by_id <> $1 THEN 'forbidden'
    WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) AND role = 5 AND NOT guest_view_all_features THEN 'django_page_nil'
    WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) THEN 'not_found'
    WHEN owned_by_id = $1 THEN 'allowed'
    WHEN $5 AND role = 5 AND NOT guest_view_all_features THEN 'restricted_guest'
    ELSE 'allowed'
  END
  FROM page_access
), 'forbidden')
`

const pageSearchMatchSQL = `
SELECT EXISTS (
  SELECT 1
  FROM pages page
  WHERE page.id = $1
    AND NOT EXISTS (
      SELECT 1
      FROM unnest($2::text[]) AS search_term
      WHERE UPPER(page.name) NOT LIKE UPPER('%' || search_term || '%') ESCAPE E'\\'
    )
)
`

const pageSummarySQL = `
SELECT json_build_object(
  'public_pages', count(*) FILTER (WHERE page.access = 0 AND page.archived_at IS NULL),
  'private_pages', count(*) FILTER (WHERE page.access = 1 AND page.archived_at IS NULL),
  'archived_pages', count(*) FILTER (WHERE page.archived_at IS NOT NULL)
)
FROM pages page
JOIN workspaces workspace ON workspace.id = page.workspace_id
JOIN project_pages project_page
  ON project_page.page_id = page.id
  AND project_page.project_id = $3
  AND project_page.deleted_at IS NULL
JOIN projects project
  ON project.id = project_page.project_id
  AND project.workspace_id = workspace.id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.workspace_id = workspace.id
  AND project_member.member_id = $1
  AND project_member.is_active = true
  AND project_member.deleted_at IS NULL
WHERE workspace.slug = $2
  AND page.parent_id IS NULL
  AND page.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM project_pages visible_link
    JOIN projects visible_project
      ON visible_project.id = visible_link.project_id
      AND visible_project.workspace_id = page.workspace_id
      AND visible_project.archived_at IS NULL
      AND visible_project.deleted_at IS NULL
    JOIN project_members visible_member
      ON visible_member.project_id = visible_project.id
      AND visible_member.member_id = $1
      AND visible_member.is_active = true
      AND visible_member.deleted_at IS NULL
    WHERE visible_link.page_id = page.id
      AND visible_link.deleted_at IS NULL
  )
  AND workspace.deleted_at IS NULL
  AND (page.owned_by_id = $1 OR page.access = 0)
  AND (project_member.role <> 5 OR project.guest_view_all_features OR page.owned_by_id = $1)
`

var pageListSQL = `
WITH page_rows AS (
  SELECT page.*,
    EXISTS (
      SELECT 1
      FROM user_favorites favorite
      WHERE favorite.user_id = $1
        AND favorite.workspace_id = page.workspace_id
        AND favorite.entity_type = 'page'
        AND favorite.entity_identifier = page.id
        AND favorite.deleted_at IS NULL
    ) AS is_favorite,
    COALESCE((
      SELECT json_agg(DISTINCT page_label.label_id)
      FROM page_labels page_label
      WHERE page_label.page_id = page.id
        AND page_label.deleted_at IS NULL
    ), '[]'::json) AS label_ids,
    COALESCE((
      SELECT json_agg(DISTINCT linked_page.project_id)
      FROM project_pages linked_page
      WHERE linked_page.page_id = page.id
        AND linked_page.deleted_at IS NULL
    ), '[]'::json) AS project_ids
  FROM pages page
  JOIN workspaces workspace ON workspace.id = page.workspace_id
  JOIN project_pages project_page
    ON project_page.page_id = page.id
    AND project_page.project_id = $3
    AND project_page.deleted_at IS NULL
  JOIN projects project
    ON project.id = project_page.project_id
    AND project.workspace_id = workspace.id
  JOIN project_members project_member
    ON project_member.project_id = project.id
    AND project_member.workspace_id = workspace.id
    AND project_member.member_id = $1
    AND project_member.is_active = true
    AND project_member.deleted_at IS NULL
  WHERE workspace.slug = $2
    AND page.parent_id IS NULL
    AND page.deleted_at IS NULL
    AND project.deleted_at IS NULL
    AND EXISTS (
      SELECT 1
      FROM project_pages visible_link
      JOIN projects visible_project
        ON visible_project.id = visible_link.project_id
        AND visible_project.workspace_id = page.workspace_id
        AND visible_project.archived_at IS NULL
        AND visible_project.deleted_at IS NULL
      JOIN project_members visible_member
        ON visible_member.project_id = visible_project.id
        AND visible_member.member_id = $1
        AND visible_member.is_active = true
        AND visible_member.deleted_at IS NULL
      WHERE visible_link.page_id = page.id
        AND visible_link.deleted_at IS NULL
    )
    AND workspace.deleted_at IS NULL
    AND (page.owned_by_id = $1 OR page.access = 0)
    AND (project_member.role <> 5 OR project.guest_view_all_features OR page.owned_by_id = $1)
    AND NOT EXISTS (
      SELECT 1
      FROM unnest($4::text[]) AS search_term
      WHERE UPPER(page.name) NOT LIKE UPPER('%' || search_term || '%') ESCAPE E'\\'
    )
)
SELECT COALESCE(json_agg(json_build_object(
  'id', id,
  'name', name,
  'owned_by', owned_by_id,
  'access', access,
  'color', color,
  'parent', parent_id,
  'is_favorite', is_favorite,
  'is_locked', is_locked,
  'archived_at', archived_at,
  'workspace', workspace_id,
  'created_at', ` + djangoDateTimeSQL("created_at", "$5") + `,
  'updated_at', ` + djangoDateTimeSQL("updated_at", "$5") + `,
  'created_by', created_by_id,
  'updated_by', updated_by_id,
  'view_props', view_props,
  'logo_props', logo_props,
  'label_ids', label_ids,
  'project_ids', project_ids
) ORDER BY is_favorite DESC, created_at DESC), '[]'::json)
FROM page_rows
`

var pageDetailSQL = `
SELECT json_build_object(
  'id', page.id,
  'name', page.name,
  'owned_by', page.owned_by_id,
  'access', page.access,
  'color', page.color,
  'parent', page.parent_id,
  'is_favorite', EXISTS (
    SELECT 1
    FROM user_favorites favorite
    WHERE favorite.user_id = $1
      AND favorite.workspace_id = page.workspace_id
      AND favorite.entity_type = 'page'
      AND favorite.entity_identifier = page.id
      AND favorite.deleted_at IS NULL
  ),
  'is_locked', page.is_locked,
  'archived_at', page.archived_at,
  'workspace', page.workspace_id,
  'created_at', ` + djangoDateTimeSQL("page.created_at", "$6") + `,
  'updated_at', ` + djangoDateTimeSQL("page.updated_at", "$6") + `,
  'created_by', page.created_by_id,
  'updated_by', page.updated_by_id,
  'view_props', page.view_props,
  'logo_props', page.logo_props,
  'label_ids', COALESCE((
    SELECT json_agg(DISTINCT page_label.label_id)
    FROM page_labels page_label
    WHERE page_label.page_id = page.id
      AND page_label.deleted_at IS NULL
  ), '[]'::json),
  'project_ids', COALESCE((
    SELECT json_agg(DISTINCT linked_page.project_id)
    FROM project_pages linked_page
    WHERE linked_page.page_id = page.id
      AND linked_page.deleted_at IS NULL
  ), '[]'::json),
  'description_html', page.description_html,
  'issue_ids', COALESCE((
    SELECT json_agg(page_log.entity_identifier ORDER BY page_log.created_at DESC)
    FROM page_logs page_log
    WHERE page_log.page_id = page.id
      AND page_log.entity_name = 'issue'
      AND page_log.deleted_at IS NULL
  ), '[]'::json)
)
FROM pages page
JOIN workspaces workspace ON workspace.id = page.workspace_id
JOIN project_pages project_page
  ON project_page.page_id = page.id
  AND project_page.project_id = $3
  AND project_page.deleted_at IS NULL
JOIN projects project
  ON project.id = project_page.project_id
  AND project.workspace_id = workspace.id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.workspace_id = workspace.id
  AND project_member.member_id = $1
  AND project_member.role IN (5, 15, 20)
  AND project_member.is_active = true
  AND project_member.deleted_at IS NULL
WHERE workspace.slug = $2
  AND page.id = $4
  AND page.deleted_at IS NULL
  AND page.parent_id IS NULL
  AND project.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM project_pages visible_link
    JOIN projects visible_project
      ON visible_project.id = visible_link.project_id
      AND visible_project.workspace_id = page.workspace_id
      AND visible_project.archived_at IS NULL
      AND visible_project.deleted_at IS NULL
    JOIN project_members visible_member
      ON visible_member.project_id = visible_project.id
      AND visible_member.member_id = $1
      AND visible_member.is_active = true
      AND visible_member.deleted_at IS NULL
    WHERE visible_link.page_id = page.id
      AND visible_link.deleted_at IS NULL
  )
  AND (page.owned_by_id = $1 OR page.access = 0)
  AND (project_member.role <> 5 OR project.guest_view_all_features OR page.owned_by_id = $1)
  AND NOT EXISTS (
    SELECT 1
    FROM unnest($5::text[]) AS search_term
    WHERE UPPER(page.name) NOT LIKE UPPER('%' || search_term || '%') ESCAPE E'\\'
  )
  AND workspace.deleted_at IS NULL
LIMIT 1
`

const pageDescriptionSQL = `
SELECT COALESCE(page.description_binary, ''::bytea)
FROM pages page
JOIN workspaces workspace ON workspace.id = page.workspace_id
JOIN project_pages project_page
  ON project_page.page_id = page.id
  AND project_page.project_id = $3
  AND project_page.deleted_at IS NULL
JOIN projects project ON project.id = project_page.project_id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.workspace_id = workspace.id
  AND project_member.member_id = $1
  AND project_member.role IN (5, 15, 20)
  AND project_member.is_active = true
  AND project_member.deleted_at IS NULL
WHERE workspace.slug = $2
  AND page.id = $4
  AND page.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND workspace.deleted_at IS NULL
  AND (page.owned_by_id = $1 OR page.access = 0)
LIMIT 1
`

var pageVersionListSQL = `
SELECT COALESCE(json_agg(json_build_object(
  'id', version.id,
  'workspace', version.workspace_id,
  'page', version.page_id,
  'last_saved_at', ` + djangoDateTimeSQL("version.last_saved_at", "$5") + `,
  'owned_by', version.owned_by_id,
  'created_at', ` + djangoDateTimeSQL("version.created_at", "$5") + `,
  'updated_at', ` + djangoDateTimeSQL("version.updated_at", "$5") + `,
  'created_by', version.created_by_id,
  'updated_by', version.updated_by_id
) ORDER BY version.created_at DESC), '[]'::json)
FROM page_versions version
JOIN pages page ON page.id = version.page_id
JOIN workspaces workspace
  ON workspace.id = version.workspace_id
  AND workspace.id = page.workspace_id
JOIN project_pages project_page
  ON project_page.page_id = page.id
  AND project_page.project_id = $3
  AND project_page.deleted_at IS NULL
JOIN projects project ON project.id = project_page.project_id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.workspace_id = workspace.id
  AND project_member.member_id = $1
  AND project_member.role IN (5, 15, 20)
  AND project_member.is_active = true
  AND project_member.deleted_at IS NULL
WHERE workspace.slug = $2
  AND page.id = $4
  AND version.deleted_at IS NULL
  AND page.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND workspace.deleted_at IS NULL
  AND (page.owned_by_id = $1 OR page.access = 0)
`

var pageVersionDetailSQL = `
SELECT json_build_object(
  'id', version.id,
  'workspace', version.workspace_id,
  'page', version.page_id,
  'last_saved_at', ` + djangoDateTimeSQL("version.last_saved_at", "$6") + `,
  'description_binary', CASE
    WHEN version.description_binary IS NULL THEN NULL
    ELSE replace(encode(version.description_binary, 'base64'), E'\n', '')
  END,
  'description_html', version.description_html,
  'description_json', version.description_json,
  'owned_by', version.owned_by_id,
  'created_at', ` + djangoDateTimeSQL("version.created_at", "$6") + `,
  'updated_at', ` + djangoDateTimeSQL("version.updated_at", "$6") + `,
  'created_by', version.created_by_id,
  'updated_by', version.updated_by_id
)
FROM page_versions version
JOIN pages page ON page.id = version.page_id
JOIN workspaces workspace
  ON workspace.id = version.workspace_id
  AND workspace.id = page.workspace_id
JOIN project_pages project_page
  ON project_page.page_id = page.id
  AND project_page.project_id = $3
  AND project_page.deleted_at IS NULL
JOIN projects project ON project.id = project_page.project_id
JOIN project_members project_member
  ON project_member.project_id = project.id
  AND project_member.workspace_id = workspace.id
  AND project_member.member_id = $1
  AND project_member.role IN (5, 15, 20)
  AND project_member.is_active = true
  AND project_member.deleted_at IS NULL
WHERE workspace.slug = $2
  AND page.id = $4
  AND version.id = $5
  AND version.deleted_at IS NULL
  AND page.deleted_at IS NULL
  AND project.deleted_at IS NULL
  AND workspace.deleted_at IS NULL
  AND (page.owned_by_id = $1 OR page.access = 0)
LIMIT 1
`

func (repository *Repository) admitPageDuplicateAssets(ctx context.Context, entityName, entityID, projectID, slug, userID string) error {
	if repository.nativeTasks != nil {
		return repository.nativeTasks.AdmitPageDuplicateAssets(ctx, entityID, entityName, entityID, projectID, slug, userID)
	}
	if repository.publisher == nil {
		return nil
	}
	return repository.publisher.Task(ctx, "plane.bgtasks.copy_s3_object.copy_s3_objects_of_description_and_assets", []any{}, map[string]any{
		"entity_name": entityName, "entity_identifier": entityID, "project_id": projectID, "slug": slug, "user_id": userID,
	})
}
