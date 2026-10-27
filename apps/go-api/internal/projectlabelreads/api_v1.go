package projectlabelreads

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

type APIV1Request = djangoapi.ProjectReadRequest

// Safe-method ProjectMemberPermission is identical to ProjectEntityPermission.
// Label writes use the projectconfiguration repository, including its distinct
// API v1 POST workspace policy.
func (r *Repository) AuthorizeAPIV1(ctx context.Context, user, slug, project string, write bool) error {
	allowed, err := djangoapi.ProjectEntityAllowed(ctx, r.database, user, slug, project, write)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

var apiLabelFields = strings.Fields("id created_at updated_at deleted_at created_by updated_by workspace project name description color sort_order external_source external_id parent")

// ReadAPIV1 preserves API labels separately from the compact app representation.
func (r *Repository) ReadAPIV1(ctx context.Context, user string, request APIV1Request) (any, error) {
	if request.ExpectedWorkspaceID != nil && (request.Operation != "list" && request.Operation != "detail") {
		return nil, ErrForbidden
	}
	p := request.Params
	detail := request.Operation == "detail"
	limit, page, offset := 1000, 0, 0
	if !detail {
		var err error
		limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: p, Present: request.Present})
		if err != nil {
			return nil, err
		}
	}
	predicate, slice := "", "LIMIT $5 OFFSET $6"
	if detail {
		predicate = " AND l.id=$4::uuid"
		slice = ""
	}
	scope, scopeJoin, scopeResult, parentScope, parentValue := "", "", "", "", ""
	if request.ExpectedWorkspaceID != nil {
		scope = djangoapi.ProjectMemberReadScopeSQL(1, 2, 3, 7)
		scopeJoin = " JOIN scope ON scope.id=l.project_id AND scope.workspace_id=l.workspace_id "
		scopeResult = " FROM scope"
		parentScope = " AND parent.project_id=l.project_id AND parent.workspace_id=l.workspace_id"
		parentValue = "||jsonb_build_object('parent_id',parent.id)"
	}
	query := `WITH ` + scope + `base AS MATERIALIZED (
 SELECT to_jsonb(l)` + parentValue + `||jsonb_build_object('expanded_parent',CASE WHEN parent.id IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('id',parent.id,'project_id',parent.project_id) END) AS data,
 row_number() OVER(ORDER BY l.created_at DESC) AS ordinal
 FROM labels l JOIN projects p ON p.id=l.project_id JOIN workspaces w ON w.id=l.workspace_id ` + scopeJoin + ` LEFT JOIN labels parent ON parent.id=l.parent_id` + parentScope + `
 WHERE l.deleted_at IS NULL AND l.project_id=$1::uuid AND w.slug=$2 AND p.archived_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$3::uuid AND pm.is_active=true)` + predicate + ` AND ($4::text IS NULL OR true)),
 page AS(SELECT * FROM base ORDER BY ordinal ` + slice + `)
 SELECT jsonb_build_object('total_count',(SELECT count(*) FROM base),'has_next',(SELECT count(*) FROM base)-$6::bigint>$5::bigint,'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))`
	var target any
	if detail {
		target = p["pk"]
	}
	args := []any{p["project_id"], p["slug"], user, target, limit, offset}
	if request.ExpectedWorkspaceID != nil {
		args = append(args, request.ExpectedWorkspaceID)
	}
	value, err := djangoapi.JSON(ctx, r.database, query+scopeResult, args...)
	if request.ExpectedWorkspaceID != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	doc := value.(map[string]any)
	rows := doc["results"].([]any)
	if detail && len(rows) == 0 {
		return nil, ErrNotFound
	}
	fields, expand := p["fields"], p["expand"]
	if detail {
		fields, expand = "", ""
	}
	// BaseSerializer passes a Label parent to IssueLiteSerializer. Resolve that
	// relation from labels, preserving omitted sequence_id and deleted parents.
	parentExpanded := false
	tokens := []string{}
	for _, token := range strings.Split(expand, ",") {
		if token == "parent" {
			parentExpanded = true
		} else {
			tokens = append(tokens, token)
		}
	}
	for i, raw := range rows {
		row := raw.(map[string]any)
		projected, projectionErr := djangoapi.Serialize(ctx, r.database, row, apiLabelFields, fields, strings.Join(tokens, ","), false)
		if projectionErr != nil {
			return nil, projectionErr
		}
		if _, selected := projected["parent"]; parentExpanded && selected {
			projected["parent"] = row["expanded_parent"]
		}
		rows[i] = projected
	}
	if detail {
		return rows[0], nil
	}
	total, err := strconv.Atoi(doc["total_count"].(json.Number).String())
	if err != nil {
		return nil, err
	}
	result := djangoapi.Envelope(rows, total, limit, page)
	result["next_page_results"] = doc["has_next"]
	return result, nil
}
