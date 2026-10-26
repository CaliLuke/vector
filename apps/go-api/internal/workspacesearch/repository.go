package workspacesearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"

	"github.com/jackc/pgx/v5"
)

var ErrForbidden = errors.New("workspace membership required")

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace search database is required")
	}
	return &Repository{db: db}, nil
}
func (r *Repository) WorkspaceAllowed(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, slug, user).Scan(&ok)
	return ok, err
}
func (r *Repository) Global(ctx context.Context, user uuid.UUID, slug, query, entities, projectID, workspaceSearch string) (json.RawMessage, error) {
	selected := map[string]bool{}
	if entities == "" {
		for _, key := range []string{"workspace", "project", "issue", "cycle", "module", "issue_view", "page", "intake"} {
			selected[key] = true
		}
	} else {
		for _, key := range stringsSplit(entities) {
			switch key {
			case "workspace", "project", "issue", "cycle", "module", "issue_view", "page", "intake":
				selected[key] = true
			}
		}
	}
	results := map[string]json.RawMessage{}
	for _, entity := range []string{"workspace", "project", "issue", "cycle", "module", "issue_view", "page", "intake"} {
		if !selected[entity] {
			continue
		}
		body, err := r.globalEntity(ctx, user, slug, query, entity, projectID, workspaceSearch)
		if err != nil {
			return nil, err
		}
		results[entity] = body
	}
	encoded, err := json.Marshal(map[string]any{"results": results})
	return encoded, err
}

func (r *Repository) globalEntity(ctx context.Context, user uuid.UUID, slug, term, entity, projectID, workspaceSearch string) (json.RawMessage, error) {
	like := "%" + workitemsearch.EscapeLike(term) + "%"
	scope := any(nil)
	if workspaceSearch == "false" {
		scope = nullableUUID(projectID)
	}
	var q string
	var args []any
	switch entity {
	case "workspace":
		q = `SELECT COALESCE(json_agg(json_build_object('name',w.name,'id',w.id,'slug',w.slug) ORDER BY w.created_at DESC),'[]'::json) FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id WHERE wm.member_id=$1 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL AND ($2='' OR w.name ILIKE $3)`
		args = []any{user, term, like}
	case "project":
		q = `SELECT COALESCE(json_agg(json_build_object('name',p.name,'id',p.id,'identifier',p.identifier,'workspace__slug',w.slug) ORDER BY p.created_at DESC),'[]'::json) FROM projects p JOIN workspaces w ON w.id=p.workspace_id JOIN project_members pm ON pm.project_id=p.id WHERE w.slug=$1 AND p.deleted_at IS NULL AND p.archived_at IS NULL AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL AND ($3='' OR p.name ILIKE $4 OR p.identifier ILIKE $4)`
		args = []any{slug, user, term, like}
	case "issue":
		q = `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT json_build_object('name',i.name,'id',i.id,'sequence_id',i.sequence_id,'project__identifier',p.identifier,'project_id',i.project_id,'workspace__slug',w.slug,'custom_property_match',property_match.snippet) item FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL LEFT JOIN states st ON st.id=i.state_id ` + workitemsearch.JoinSQL("i", "$4::text") + ` WHERE w.slug=$1 AND ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "w.id") + ` AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND st."group" IS DISTINCT FROM 'triage' AND ($3::uuid IS NULL OR i.project_id=$3::uuid) AND ($4='' OR property_match.snippet IS NOT NULL OR i.name ILIKE $5 OR p.identifier ILIKE $5 OR i.sequence_id::text=ANY(string_to_array($6,','))) ORDER BY i.created_at DESC LIMIT 100) x`
		args = []any{slug, user, scope, term, like, strings.Join(integerMatches(term), ",")}
	case "cycle":
		q = `SELECT COALESCE(json_agg(json_build_object('name',c.name,'id',c.id,'project_id',c.project_id,'project__identifier',p.identifier,'workspace__slug',w.slug) ORDER BY c.created_at DESC),'[]'::json) FROM cycles c JOIN projects p ON p.id=c.project_id JOIN workspaces w ON w.id=c.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND p.archived_at IS NULL AND c.deleted_at IS NULL AND ($3::uuid IS NULL OR c.project_id=$3::uuid) AND ($4='' OR c.name ILIKE $5)`
		args = []any{slug, user, scope, term, like}
	case "module":
		q = `SELECT COALESCE(json_agg(json_build_object('name',m.name,'id',m.id,'project_id',m.project_id,'project__identifier',p.identifier,'workspace__slug',w.slug) ORDER BY m.created_at DESC),'[]'::json) FROM modules m JOIN projects p ON p.id=m.project_id JOIN workspaces w ON w.id=m.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND p.archived_at IS NULL AND m.deleted_at IS NULL AND ($3::uuid IS NULL OR m.project_id=$3::uuid) AND ($4='' OR m.name ILIKE $5)`
		args = []any{slug, user, scope, term, like}
	case "issue_view":
		q = `SELECT COALESCE(json_agg(json_build_object('name',v.name,'id',v.id,'project_id',v.project_id,'project__identifier',p.identifier,'workspace__slug',w.slug) ORDER BY v.created_at DESC),'[]'::json) FROM issue_views v JOIN projects p ON p.id=v.project_id JOIN workspaces w ON w.id=v.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND p.archived_at IS NULL AND v.deleted_at IS NULL AND ($3::uuid IS NULL OR v.project_id=$3::uuid) AND ($4='' OR v.name ILIKE $5)`
		args = []any{slug, user, scope, term, like}
	case "page":
		q = `SELECT COALESCE(json_agg(item ORDER BY created_at DESC),'[]'::json) FROM (SELECT json_build_object('name',page.name,'id',page.id,'project_ids',COALESCE(array_agg(DISTINCT p.id) FILTER(WHERE p.id IS NOT NULL),'{}'::uuid[]),'project_identifiers',COALESCE(array_agg(DISTINCT p.identifier) FILTER(WHERE p.id IS NOT NULL),'{}'::text[]),'workspace__slug',w.slug) item,max(page.created_at) created_at FROM pages page JOIN workspaces w ON w.id=page.workspace_id JOIN project_pages pp ON pp.page_id=page.id AND pp.deleted_at IS NULL JOIN projects p ON p.id=pp.project_id AND p.deleted_at IS NULL AND p.archived_at IS NULL JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL WHERE w.slug=$1 AND ($3::uuid IS NULL OR ($4='false' AND pp.project_id=$3::uuid)) AND ($5='' OR page.name ILIKE $6) GROUP BY page.id,w.slug) x`
		args = []any{slug, user, scope, workspaceSearch, term, like}
	case "intake":
		q = `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT json_build_object('name',i.name,'id',i.id,'sequence_id',i.sequence_id,'project__identifier',p.identifier,'project_id',i.project_id,'workspace__slug',w.slug,'custom_property_match',property_match.snippet) item FROM issues i JOIN intake_issues intake ON intake.issue_id=i.id AND intake.workspace_id=i.workspace_id AND intake.project_id=i.project_id AND intake.deleted_at IS NULL JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true ` + workitemsearch.JoinSQL("i", "$4::text") + ` WHERE w.slug=$1 AND ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "w.id") + ` AND p.archived_at IS NULL AND i.deleted_at IS NULL AND intake.status IN (0,-2) AND ($3::uuid IS NULL OR i.project_id=$3::uuid) AND ($4='' OR property_match.snippet IS NOT NULL OR i.name ILIKE $5 OR p.identifier ILIKE $5) ORDER BY i.created_at DESC LIMIT 100) x`
		args = []any{slug, user, scope, term, like}
	}
	var body []byte
	if err := r.db.QueryRow(ctx, q, args...).Scan(&body); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func stringsSplit(v string) []string {
	var values []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
func nullableUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func scanJSON(ctx context.Context, db Database, query string, args ...any) (json.RawMessage, error) {
	var body []byte
	if err := db.QueryRow(ctx, query, args...).Scan(&body); err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("workspace search returned invalid JSON")
	}
	return json.RawMessage(body), nil
}

var integerPattern = regexp.MustCompile(`\b[0-9]+\b`)

func integerMatches(v string) []string { return integerPattern.FindAllString(v, -1) }
