package modulereads

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

var apiModuleFields = strings.Fields("id created_at updated_at deleted_at created_by updated_by workspace project name description description_text description_html start_date target_date status lead view_props sort_order external_source external_id archived_at logo_props members")
var apiModuleCounts = strings.Fields("total_issues cancelled_issues completed_issues started_issues unstarted_issues backlog_issues")

// ReadAPIV1 retains the API source contract; app modules have different permissions and querysets.
func (r *Repository) ReadAPIV1(ctx context.Context, user string, request APIV1Request) (any, error) {
	if request.ExpectedWorkspaceID != nil && (request.Operation != "lite" && request.Operation != "archived" && request.Operation != "detail") {
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
	var value any
	var err error
	if request.Operation == "issues" {
		value, err = r.apiIssues(ctx, request, limit, offset)
	} else {
		value, err = r.apiModules(ctx, user, request, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	doc := value.(map[string]any)
	rows := doc["results"].([]any)
	if detail && len(rows) == 0 {
		return nil, ErrNotFound
	}
	for i, raw := range rows {
		row := raw.(map[string]any)
		if request.Operation == "issues" {
			rows[i], err = djangoapi.SerializeVisibleIssue(ctx, r.database, row, p["fields"], p["expand"], user)
		} else {
			fields, expand := p["fields"], p["expand"]
			declared := append([]string{}, apiModuleFields...)
			if request.Operation == "lite" {
				fields, expand = "", ""
			} else {
				declared = append(declared, apiModuleCounts...)
			}
			var projected map[string]any
			projected, err = djangoapi.Serialize(ctx, r.database, row, declared, fields, expand, false)
			if err == nil {
				projected["members"] = row["members"]
				if request.ExpectedWorkspaceID != nil && detail && p["include_revision"] == "true" {
					revision, ok := row["revision"].(string)
					if !ok || revision == "" {
						return nil, errors.New("module revision unavailable")
					}
					projected["revision"] = revision
				}
				rows[i] = projected
			}
		}
		if err != nil {
			return nil, err
		}
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

func apiModuleOrder(raw string) string {
	name := strings.TrimPrefix(raw, "-")
	switch name {
	case "created_at", "updated_at", "name", "start_date", "target_date", "status", "sort_order":
	default:
		raw = "-created_at"
		name = "created_at"
	}
	direction := " ASC"
	if strings.HasPrefix(raw, "-") {
		direction = " DESC"
	}
	return "m." + name + direction
}

func (r *Repository) apiModules(ctx context.Context, user string, request APIV1Request, limit, offset int) (any, error) {
	p := request.Params
	predicate := "m.archived_at IS NULL"
	if request.Operation == "archived" {
		predicate = "m.archived_at IS NOT NULL"
	}
	if request.Operation == "detail" {
		predicate += " AND m.id=$3::uuid"
	}
	order := "m.created_at DESC"
	if request.Operation == "lite" {
		order = apiModuleOrder(p["order_by"])
	}
	scoped := request.ExpectedWorkspaceID != nil
	memberScope := ""
	if scoped {
		memberScope = " AND mm.project_id=m.project_id AND mm.workspace_id=m.workspace_id AND mm.deleted_at IS NULL"
	}
	row := `to_jsonb(m)||jsonb_build_object('members',COALESCE((SELECT jsonb_agg(u.id ORDER BY u.created_at DESC) FROM users u JOIN module_members mm ON mm.member_id=u.id WHERE mm.module_id=m.id` + memberScope + `),'[]'::jsonb))`
	if request.ExpectedWorkspaceID != nil && request.Operation == "detail" && p["include_revision"] == "true" {
		row += "||jsonb_build_object('revision','module:'||m.id::text||':'||(SELECT revision::text FROM module_edit_revisions WHERE module_id=m.id))"
	}
	joins := ""
	if request.Operation != "lite" {
		joins = apiModuleMetricJoins(scoped)
		row += "||stats.counts"
	}
	slice := "LIMIT $4 OFFSET $5"
	if request.Operation == "detail" {
		slice = ""
	}
	scope, scopeJoin, scopeResult := "", "", ""
	if scoped {
		scope = djangoapi.ProjectMemberReadScopeSQL(1, 2, 6, 7)
		scopeJoin = " JOIN scope ON scope.id=m.project_id AND scope.workspace_id=m.workspace_id "
		scopeResult = " FROM scope"
	}
	query := `WITH ` + scope + `base AS MATERIALIZED(SELECT ` + row + ` AS data,row_number() OVER(ORDER BY ` + order + `) AS ordinal FROM modules m JOIN workspaces w ON w.id=m.workspace_id ` + scopeJoin + joins + ` WHERE m.project_id=$1::uuid AND w.slug=$2 AND m.deleted_at IS NULL AND ` + predicate + ` AND ($3::text IS NULL OR true)), page AS(SELECT * FROM base ORDER BY ordinal ` + slice + `) SELECT jsonb_build_object('total_count',(SELECT count(*) FROM base),'has_next',(SELECT count(*) FROM base)-$5::bigint>$4::bigint,'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))`
	var target any
	if p["pk"] != "" {
		target = p["pk"]
	}
	args := []any{p["project_id"], p["slug"], target, limit, offset}
	if scoped {
		args = append(args, user, request.ExpectedWorkspaceID)
	}
	value, err := djangoapi.JSON(ctx, r.database, query+scopeResult, args...)
	if scoped && errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	return value, err
}

func (r *Repository) apiIssues(ctx context.Context, request APIV1Request, limit, offset int) (any, error) {
	p := request.Params
	order, joins := djangoapi.IssueOrderSQL(p["order_by"])
	// Django reuses the module bridge already constrained by the filter, unlike cycle issue ordering.
	if strings.TrimPrefix(p["order_by"], "-") == "issue_module__module__name" {
		joins = " LEFT JOIN modules ordering_module ON ordering_module.id=$3::uuid"
	}
	query := `WITH base AS MATERIALIZED(SELECT i.* FROM issues i JOIN module_issues mi ON mi.issue_id=i.id JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id LEFT JOIN states s ON s.id=i.state_id WHERE i.project_id=$1::uuid AND w.slug=$2 AND mi.module_id=$3::uuid AND mi.deleted_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND NOT i.is_draft AND p.archived_at IS NULL AND (s."group" IS NULL OR s."group"<>'triage')),
 ordered AS(SELECT ` + djangoapi.IssueRowSQL("i") + ` AS data,row_number() OVER(ORDER BY ` + order + `) AS ordinal FROM base i ` + joins + `), page AS(SELECT * FROM ordered ORDER BY ordinal LIMIT $4 OFFSET $5)
 SELECT jsonb_build_object('total_count',(SELECT count(*) FROM base),'has_next',(SELECT count(*) FROM ordered)-$5::bigint>$4::bigint,'results',COALESCE((SELECT jsonb_agg(data ORDER BY ordinal) FROM page),'[]'::jsonb))`
	return djangoapi.JSON(ctx, r.database, query, p["project_id"], p["slug"], p["module_id"], limit, offset)
}
