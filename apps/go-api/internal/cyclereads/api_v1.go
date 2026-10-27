package cyclereads

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

type APIV1Request struct {
	// ExpectedWorkspaceID binds an admitted workspace and requires current primary-data authorization.
	ExpectedWorkspaceID *uuid.UUID
	Operation           string
	Params              map[string]string
	Present             map[string]bool
}

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

func (r *Repository) apiJSON(ctx context.Context, query string, args ...any) (any, error) {
	return djangoapi.JSON(ctx, r.database, query, args...)
}

// ReadAPIV1 retains the API serializers and queryset semantics. The app API's
// cycle projections, role requirements and distinct issue counts are different.
func (r *Repository) ReadAPIV1(ctx context.Context, user string, request APIV1Request) (any, error) {
	p := request.Params
	if request.ExpectedWorkspaceID != nil && request.Operation != "lite" && request.Operation != "archived" && request.Operation != "detail" {
		return nil, ErrForbidden
	}
	if request.ExpectedWorkspaceID == nil && (request.Operation == "list" || request.Operation == "detail") {
		var exists bool
		if err := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, p["project_id"], p["slug"]).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}
	if request.Operation == "issue-detail" {
		value, err := r.apiJSON(ctx, `SELECT to_jsonb(ci) FROM cycle_issues ci JOIN workspaces w ON w.id=ci.workspace_id WHERE ci.project_id=$1 AND w.slug=$2 AND ci.cycle_id=$3 AND ci.issue_id=$4 AND ci.deleted_at IS NULL`, p["project_id"], p["slug"], p["cycle_id"], p["issue_id"])
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		return r.apiSerialize(ctx, value.(map[string]any), "bridge", p["fields"], p["expand"], user)
	}
	detail := request.Operation == "detail"
	current := request.Operation == "list" && apiCycleStatus(request) == "current"
	limit, page, offset := 1000, 0, 0
	if !detail && !current {
		var err error
		limit, page, offset, err = djangoapi.PageWindow(djangoapi.Query{Params: p, Present: request.Present})
		if err != nil {
			return nil, err
		}
	}
	var value any
	var err error
	if request.Operation == "issues" {
		value, err = r.apiIssues(ctx, request, limit, offset)
	} else {
		value, err = r.apiCycles(ctx, user, request, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	document := value.(map[string]any)
	rows := document["results"].([]any)
	if detail && len(rows) == 0 {
		return nil, ErrNotFound
	}
	kind := "cycle"
	if request.Operation == "issues" {
		kind = "issue"
	}
	fields, expand := p["fields"], p["expand"]
	if request.Operation == "lite" {
		fields, expand = "", ""
	}
	for index, raw := range rows {
		rows[index], err = r.apiSerialize(ctx, raw.(map[string]any), kind, fields, expand, user)
		if err != nil {
			return nil, err
		}
	}
	if detail {
		return rows[0], nil
	}
	if current {
		return rows, nil
	}
	total, err := strconv.Atoi(document["total_count"].(json.Number).String())
	if err != nil {
		return nil, err
	}
	envelope := djangoapi.Envelope(rows, total, limit, page)
	// Django count() drops ordering-only joins. The ordered window may contain
	// repeated issues even while total_count remains the base queryset count.
	envelope["next_page_results"] = document["has_next"]
	return envelope, nil
}

func (r *Repository) apiCycles(ctx context.Context, user string, request APIV1Request, limit, offset int) (any, error) {
	p := request.Params
	predicate := "c.archived_at IS NULL"
	if request.Operation == "archived" {
		predicate = "c.archived_at IS NOT NULL"
	}
	if request.Operation == "detail" {
		predicate += " AND c.id=$4::uuid"
	}
	if request.Operation == "list" || request.Operation == "lite" {
		switch apiCycleStatus(request) {
		case "current":
			predicate += " AND c.start_date<=CURRENT_TIMESTAMP AND c.end_date>=CURRENT_TIMESTAMP"
		case "upcoming":
			predicate += " AND c.start_date>CURRENT_TIMESTAMP"
		case "completed":
			predicate += " AND c.end_date<CURRENT_TIMESTAMP"
		case "draft":
			predicate += " AND c.start_date IS NULL AND c.end_date IS NULL"
		case "incomplete":
			predicate += " AND (c.end_date>=CURRENT_TIMESTAMP OR c.end_date IS NULL)"
		}
	}
	order := "c.created_at DESC"
	if request.Operation == "lite" {
		order, _ = apiOrder(p["order_by"], "-created_at", false)
	}
	row := "to_jsonb(c)"
	joins := ""
	if request.Operation != "lite" {
		joins = apiCycleMetricJoins(request.ExpectedWorkspaceID != nil)
		row += "||stats.counts"
		if request.Operation == "archived" {
			row += "||jsonb_build_object('total_estimates',stats.total_estimates,'completed_estimates',stats.completed_estimates,'started_estimates',stats.started_estimates)"
		}
	}
	slice := "LIMIT $5 OFFSET $6"
	if request.Operation == "detail" || request.Operation == "list" && apiCycleStatus(request) == "current" {
		slice = ""
	}
	// The unused typed parameters keep one stable argument layout across shapes.
	scope, scopeJoin, scopeResult := "", "", ""
	args := []any{p["project_id"], p["slug"], user, nilIfEmpty(p["pk"]), limit, offset}
	if request.ExpectedWorkspaceID != nil {
		scope, scopeJoin, scopeResult = apiCycleScopeSQL, " JOIN scope ON scope.id=c.project_id AND scope.workspace_id=c.workspace_id ", " FROM scope"
		args = append(args, request.ExpectedWorkspaceID)
	}
	query := `WITH ` + scope + `base AS MATERIALIZED(SELECT ` + row + ` AS data,row_number() OVER(ORDER BY ` + order + `) AS ordinal FROM cycles c JOIN workspaces w ON w.id=c.workspace_id ` + scopeJoin + joins + ` WHERE c.project_id=$1::uuid AND w.slug=$2 AND c.deleted_at IS NULL AND ` + predicate + ` AND $3::uuid IS NOT NULL AND ($4::text IS NULL OR true)), page AS (SELECT * FROM base ORDER BY ordinal ` + slice + `) SELECT jsonb_build_object('total_count',(SELECT count(*) FROM base),'has_next',(SELECT count(*) FROM base)-$6::bigint>$5::bigint,'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))`
	value, err := r.apiJSON(ctx, query+scopeResult, args...)
	if request.ExpectedWorkspaceID != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	return value, err
}

func apiOrder(raw, fallback string, issue bool) (string, string) {
	if issue {
		return djangoapi.IssueOrderSQL(raw)
	}
	name := strings.TrimPrefix(raw, "-")
	allowed := map[string]string{"created_at": "created_at", "updated_at": "updated_at", "name": "name", "start_date": "start_date", "end_date": "end_date", "sort_order": "sort_order"}
	joins := ""
	alias := "c."

	if _, ok := allowed[name]; !ok {
		raw = fallback
		name = strings.TrimPrefix(raw, "-")
	}
	direction := " ASC"
	if strings.HasPrefix(raw, "-") {
		direction = " DESC"
	}

	return alias + allowed[name] + direction, joins
}

func (r *Repository) apiIssues(ctx context.Context, request APIV1Request, limit, offset int) (any, error) {
	p := request.Params
	order, joins := apiOrder(p["order_by"], "created_at", true)
	query := `WITH base AS MATERIALIZED(SELECT i.* FROM issues i JOIN cycle_issues ci ON ci.issue_id=i.id JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id LEFT JOIN states s ON s.id=i.state_id WHERE i.project_id=$1::uuid AND w.slug=$2 AND ci.cycle_id=$3::uuid AND ci.deleted_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND NOT i.is_draft AND p.archived_at IS NULL AND (s."group" IS NULL OR s."group"<>'triage')),
 ordered AS (SELECT ` + apiIssueRow("i") + ` AS data,row_number() OVER(ORDER BY ` + order + `) AS ordinal FROM base i ` + joins + `), page AS (SELECT * FROM ordered ORDER BY ordinal LIMIT $4 OFFSET $5)
 SELECT jsonb_build_object('total_count',(SELECT count(*) FROM base),'has_next',(SELECT count(*) FROM ordered)-$5::bigint>$4::bigint,'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))`
	return r.apiJSON(ctx, query, p["project_id"], p["slug"], p["cycle_id"], limit, offset)
}

func apiIssueRow(alias string) string { return djangoapi.IssueRowSQL(alias) }
