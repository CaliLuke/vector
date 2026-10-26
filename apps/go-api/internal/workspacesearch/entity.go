package workspacesearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
)

func (r *Repository) Entity(ctx context.Context, user uuid.UUID, slug, query, queryTypes, projectID string, count int) (json.RawMessage, error) {
	if count < 0 {
		return nil, fmt.Errorf("negative count")
	}
	if queryTypes == "" {
		queryTypes = "user_mention"
	}
	result := map[string]json.RawMessage{}
	like := "%" + workitemsearch.EscapeLike(query) + "%"
	sequences := strings.Join(integerMatches(query), ",")
	for _, kind := range stringsSplit(queryTypes) {
		sql, args := entitySearchQuery(kind, projectID != "", slug, user, projectID, query, like, sequences, count)
		if sql == "" {
			continue
		}
		body, err := scanJSON(ctx, r.db, sql, args...)
		if err != nil {
			return nil, err
		}
		result[kind] = body
	}
	encoded, err := json.Marshal(result)
	return encoded, err
}

func entitySearchQuery(kind string, projectSearch bool, slug string, user uuid.UUID, project, query, like, sequences string, count int) (string, []any) {
	if projectSearch {
		switch kind {
		case "user_mention":
			return `SELECT COALESCE(json_agg(item ORDER BY created_at DESC),'[]'::json) FROM (SELECT json_build_object('member__avatar_url',COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar),'member__display_name',u.display_name,'member__id',u.id) item,pm.created_at FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id JOIN users u ON u.id=pm.member_id WHERE w.slug=$1 AND pm.project_id=$2::uuid AND pm.is_active=true AND u.is_bot=false AND ($3='' OR u.first_name ILIKE $4 OR u.last_name ILIKE $4 OR u.display_name ILIKE $4) ORDER BY pm.created_at DESC LIMIT $5) x`, []any{slug, project, query, like, count}
		case "project":
			return `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT DISTINCT json_build_object('name',p.name,'id',p.id,'identifier',p.identifier,'logo_props',p.logo_props,'workspace__slug',w.slug) item,p.created_at FROM projects p JOIN workspaces w ON w.id=p.workspace_id LEFT JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 WHERE w.slug=$1 AND (pm.member_id IS NOT NULL OR p.network=2) AND ($3='' OR p.name ILIKE $4 OR p.identifier ILIKE $4) ORDER BY p.created_at DESC LIMIT $5)x`, []any{slug, user, query, like, count}
		case "issue":
			return `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT json_build_object('name',i.name,'id',i.id,'sequence_id',i.sequence_id,'project__identifier',p.identifier,'project_id',i.project_id,'priority',i.priority,'state_id',i.state_id,'type_id',i.type_id,'custom_property_match',property_match.snippet) item FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL LEFT JOIN states st ON st.id=i.state_id ` + workitemsearch.JoinSQL("i", "$4::text") + ` WHERE w.slug=$1 AND ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "w.id") + ` AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.project_id=$3::uuid AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND st."group" IS DISTINCT FROM 'triage' AND ($4='' OR property_match.snippet IS NOT NULL OR i.name ILIKE $5 OR p.identifier ILIKE $5 OR i.sequence_id::text=ANY(string_to_array($6,','))) ORDER BY i.created_at DESC LIMIT $7)x`, []any{slug, user, project, query, like, sequences, count}
		case "cycle":
			return `SELECT COALESCE(json_agg(json_build_object('name',c.name,'id',c.id,'project_id',c.project_id,'project__identifier',p.identifier,'status',CASE WHEN c.start_date<=now() AND c.end_date>=now() THEN 'CURRENT' WHEN c.start_date>now() THEN 'UPCOMING' WHEN c.end_date<now() THEN 'COMPLETED' ELSE 'DRAFT' END,'workspace__slug',w.slug) ORDER BY c.created_at DESC),'[]'::json) FROM cycles c JOIN projects p ON p.id=c.project_id JOIN workspaces w ON w.id=c.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND c.project_id=$3::uuid AND c.deleted_at IS NULL AND ($4='' OR c.name ILIKE $5) LIMIT $6`, []any{slug, user, project, query, like, count}
		case "module":
			return `SELECT COALESCE(json_agg(json_build_object('name',m.name,'id',m.id,'project_id',m.project_id,'project__identifier',p.identifier,'status',m.status,'workspace__slug',w.slug) ORDER BY m.created_at DESC),'[]'::json) FROM modules m JOIN projects p ON p.id=m.project_id JOIN workspaces w ON w.id=m.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND m.project_id=$3::uuid AND m.deleted_at IS NULL AND ($4='' OR m.name ILIKE $5) LIMIT $6`, []any{slug, user, project, query, like, count}
		case "page":
			return `SELECT COALESCE(json_agg(json_build_object('name',page.name,'id',page.id,'logo_props',page.logo_props,'projects__id',p.id,'workspace__slug',w.slug) ORDER BY page.created_at DESC),'[]'::json) FROM pages page JOIN workspaces w ON w.id=page.workspace_id JOIN project_pages pp ON pp.page_id=page.id AND pp.deleted_at IS NULL JOIN projects p ON p.id=pp.project_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND pp.project_id=$3::uuid AND page.access=0 AND ($4='' OR page.name ILIKE $5) LIMIT $6`, []any{slug, user, project, query, like, count}
		default:
			return "", nil
		}
	}
	switch kind {
	case "user_mention":
		return `SELECT COALESCE(json_agg(item ORDER BY created_at DESC),'[]'::json) FROM (SELECT json_build_object('member__avatar_url',COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar),'member__display_name',u.display_name,'member__id',u.id) item,wm.created_at FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id JOIN users u ON u.id=wm.member_id WHERE w.slug=$1 AND wm.is_active=true AND u.is_bot=false AND ($2='' OR u.first_name ILIKE $3 OR u.last_name ILIKE $3 OR u.display_name ILIKE $3) ORDER BY wm.created_at DESC LIMIT $4) x`, []any{slug, query, like, count}
	case "project":
		return `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT DISTINCT json_build_object('name',p.name,'id',p.id,'identifier',p.identifier,'logo_props',p.logo_props,'workspace__slug',w.slug) item,p.created_at FROM projects p JOIN workspaces w ON w.id=p.workspace_id LEFT JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 WHERE w.slug=$1 AND (pm.member_id IS NOT NULL OR p.network=2) AND ($3='' OR p.name ILIKE $4 OR p.identifier ILIKE $4) ORDER BY p.created_at DESC LIMIT $5)x`, []any{slug, user, query, like, count}
	case "issue":
		return `SELECT COALESCE(json_agg(item),'[]'::json) FROM (SELECT json_build_object('name',i.name,'id',i.id,'sequence_id',i.sequence_id,'project__identifier',p.identifier,'project_id',i.project_id,'priority',i.priority,'state_id',i.state_id,'type_id',i.type_id,'custom_property_match',property_match.snippet) item FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL LEFT JOIN states st ON st.id=i.state_id ` + workitemsearch.JoinSQL("i", "$3::text") + ` WHERE w.slug=$1 AND ` + djangoapi.IssueVisibleSQL("i", "$2::uuid", "w.id") + ` AND p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND st."group" IS DISTINCT FROM 'triage' AND ($3='' OR property_match.snippet IS NOT NULL OR i.name ILIKE $4 OR p.identifier ILIKE $4 OR i.sequence_id::text=ANY(string_to_array($5,','))) ORDER BY i.created_at DESC LIMIT $6)x`, []any{slug, user, query, like, sequences, count}
	case "cycle":
		return `SELECT COALESCE(json_agg(json_build_object('name',c.name,'id',c.id,'project_id',c.project_id,'project__identifier',p.identifier,'status',CASE WHEN c.start_date<=now() AND c.end_date>=now() THEN 'CURRENT' WHEN c.start_date>now() THEN 'UPCOMING' WHEN c.end_date<now() THEN 'COMPLETED' ELSE 'DRAFT' END,'workspace__slug',w.slug) ORDER BY c.created_at DESC),'[]'::json) FROM cycles c JOIN projects p ON p.id=c.project_id JOIN workspaces w ON w.id=c.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND c.deleted_at IS NULL AND ($3='' OR c.name ILIKE $4) LIMIT $5`, []any{slug, user, query, like, count}
	case "module":
		return `SELECT COALESCE(json_agg(json_build_object('name',m.name,'id',m.id,'project_id',m.project_id,'project__identifier',p.identifier,'status',m.status,'workspace__slug',w.slug) ORDER BY m.created_at DESC),'[]'::json) FROM modules m JOIN projects p ON p.id=m.project_id JOIN workspaces w ON w.id=m.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND m.deleted_at IS NULL AND ($3='' OR m.name ILIKE $4) LIMIT $5`, []any{slug, user, query, like, count}
	case "page":
		return `SELECT COALESCE(json_agg(json_build_object('name',page.name,'id',page.id,'logo_props',page.logo_props,'projects__id',p.id,'workspace__slug',w.slug) ORDER BY page.created_at DESC),'[]'::json) FROM pages page JOIN workspaces w ON w.id=page.workspace_id JOIN project_pages pp ON pp.page_id=page.id AND pp.deleted_at IS NULL JOIN projects p ON p.id=pp.project_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true WHERE w.slug=$1 AND page.access=0 AND ($3='' OR page.name ILIKE $4) LIMIT $5`, []any{slug, user, query, like, count}
	default:
		return "", nil
	}
}
