package workitems

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func (r *Repository) readGroupedListPage(ctx context.Context, q Request, shape issueListShape, plan issueListFilterPlan, from, order string, args []any) (Result, error) {
	limit, page, offset, err := apicontract.PageWindow(apicontract.Query{Params: q.Params, Present: q.Present})
	if err != nil {
		return Result{}, err
	}
	group, subgroup := plan.options.GroupBy, plan.options.SubGroupBy
	deriveGroups := r.deriveGroupedListUniverse(q, group, subgroup)
	var groups []any
	var visibleGroups []string
	if !deriveGroups {
		groups, err = r.groupUniverse(ctx, q, group, nil)
		if err != nil {
			return Result{}, err
		}
		visibleGroups = make([]string, 0, len(groups))
		for _, value := range groups {
			visibleGroups = append(visibleGroups, fmt.Sprint(value))
		}
	}
	// Custom axes use the authorized discovery snapshot and the same property
	// evaluation as their filters, including formulas and per-project schemas.
	var properties []map[string]any
	if r.propertyGroups[group] != nil || r.propertyGroups[subgroup] != nil {
		properties, err = r.propertyCandidateRows(ctx, plan.where, plan.args)
		if err != nil {
			return Result{}, err
		}
	}
	groupSQL := r.groupedListAxisSQL(group, shape, properties, &args)
	subgroupSQL := r.groupedListAxisSQL(subgroup, shape, properties, &args)
	args = append(args, visibleGroups, offset, limit)
	sql := groupedListPageSQL(from, order, groupSQL, subgroupSQL, issueListProjectionSQL(shape, true), len(args))
	records, err := r.rows(ctx, sql, args...)
	if err != nil {
		return Result{}, err
	}
	return r.projectGroupedListPage(ctx, q, shape, group, subgroup, limit, page, offset, groups, deriveGroups, records)
}

func (r *Repository) deriveGroupedListUniverse(q Request, group, subgroup string) bool {
	if subgroup == "" && issuequery.GroupArrayField(group) != "" {
		return true
	}
	if r.parentGrouping != nil && group == r.parentGrouping.axis || group == "type_id" && r.typeDiscovery != nil {
		return false
	}
	return r.propertyGroups[group] != nil || q.ProjectID == "" && usesDiscoveryGrouping(q) ||
		group == "created_by" || group == "target_date" || group == "start_date"
}

func (r *Repository) groupedListAxisSQL(axis string, shape issueListShape, properties []map[string]any, args *[]any) string {
	if field := r.propertyGroups[axis]; field != nil {
		values := make(map[string]string, len(properties))
		for _, item := range properties {
			projected := item["property_values"].(map[string]workitemtypes.ProjectedValue)
			values[stringValue(item["id"])] = workitemtypes.PropertyGroupValue(*field, projected)
		}
		*args = append(*args, values)
		return fmt.Sprintf("ARRAY[$%d::jsonb ->> issue.id::text]", len(*args))
	}
	if r.parentGrouping != nil && axis == r.parentGrouping.axis {
		*args = append(*args, r.parentGrouping.ids)
		return fmt.Sprintf("ARRAY[CASE WHEN issue.parent_id=ANY($%d::uuid[]) THEN (%s)::text END]",
			len(*args), apicontract.VisibleParentSQL("issue", "$1"))
	}
	expression := ""
	switch axis {
	case "state_id", "priority", "project_id", "type_id":
		expression = "issue." + axis
	case "created_by":
		expression = "issue.created_by_id"
	case "target_date", "start_date":
		expression = "to_jsonb(issue." + axis + ") #>> '{}'"
	case "cycle_id":
		expression = issueCycleSQL("issue")
	case "state__group":
		if shape.projectsStateGroup() {
			expression = `(SELECT "group" FROM states s WHERE s.id=issue.state_id)`
		}
	case "assignees__id":
		return `ARRAY(SELECT DISTINCT a.assignee_id::text FROM issue_assignees a WHERE a.issue_id=issue.id AND a.deleted_at IS NULL)`
	case "labels__id":
		return `ARRAY(SELECT DISTINCT l.label_id::text FROM issue_labels l WHERE ` + issuequery.LiveIssueLabelPredicate("issue", "l") + `)`
	case "issue_module__module_id":
		if shape.hasDetailProjection() {
			return `ARRAY(SELECT DISTINCT m.module_id::text FROM module_issues m WHERE m.issue_id=issue.id AND m.deleted_at IS NULL)`
		}
		return `ARRAY(SELECT DISTINCT m.module_id::text ` + issueModuleJoinSQL("issue") + liveIssueModuleConditions + ")"
	case "":
		return `ARRAY['']::text[]`
	}
	if expression == "" {
		return `ARRAY['None']::text[]`
	}
	return "ARRAY[(" + expression + ")::text]"
}

// Rank visible candidates once and expand their memberships before paging.
// Only issue IDs are carried through the grouped window. The expensive
// projection joins unique page IDs back to issues when the windows are known.
func groupedListPageSQL(from, order, group, subgroup, projection string, arguments int) string {
	visible := fmt.Sprintf("($%d::text[] IS NULL OR group_key=ANY($%d::text[]))", arguments-2, arguments-2)
	paged := fmt.Sprintf("bucket_rank > $%d::bigint AND bucket_rank <= $%d::bigint + $%d::bigint", arguments-1, arguments-1, arguments)
	const values = `COALESCE(%s, 'None')`
	return `WITH candidates AS (
 SELECT issue.id, row_number() OVER (ORDER BY ` + order + `,issue.created_at DESC,issue.id DESC) AS position,
 ` + group + ` AS group_values, ` + subgroup + ` AS subgroup_values,
 (issue.archived_at IS NULL AND NOT issue.is_draft AND (
  NOT EXISTS(SELECT 1 FROM intake_issues t WHERE t.issue_id=issue.id)
  OR EXISTS(SELECT 1 FROM intake_issues t WHERE t.issue_id=issue.id AND t.status IN (-1,1,2))
 )) AS counts
 ` + from + `
), group_members AS (
 SELECT id, position, counts, subgroup_values, ` + fmt.Sprintf(values, "member.value") + ` AS group_key
 FROM candidates
 CROSS JOIN LATERAL unnest(CASE WHEN cardinality(group_values)=0 THEN ARRAY['None'] ELSE group_values END) member(value)
), group_counts AS (
 SELECT group_key, count(*) FILTER(WHERE counts) AS group_count FROM group_members GROUP BY group_key
), memberships AS (
 SELECT id, position, counts, group_key, ` + fmt.Sprintf(values, "member.value") + ` AS subgroup_key
 FROM group_members
 CROSS JOIN LATERAL unnest(CASE WHEN cardinality(subgroup_values)=0 THEN ARRAY['None'] ELSE subgroup_values END) member(value)
), ranked AS (
 SELECT id, group_key, subgroup_key, counts, row_number() OVER (PARTITION BY group_key, subgroup_key ORDER BY position) AS bucket_rank
 FROM memberships
), buckets AS (
 SELECT group_key, subgroup_key, count(*) FILTER(WHERE counts) AS filtered_count, count(*) AS row_count,
 array_agg(id ORDER BY bucket_rank) FILTER(WHERE ` + paged + `) AS ids
 FROM ranked GROUP BY group_key, subgroup_key
), selected AS (
 SELECT DISTINCT unnest(ids) AS id FROM buckets WHERE ` + visible + `
)
SELECT jsonb_build_object('kind','total','total',COALESCE(sum(row_count),0)) FROM buckets
UNION ALL
SELECT jsonb_build_object('kind','group','group_key',group_key,'count',group_count) FROM group_counts
UNION ALL
SELECT jsonb_build_object('kind','bucket','group_key',group_key,'subgroup_key',subgroup_key,'count',filtered_count,'row_count',row_count,'ids',COALESCE(ids,ARRAY[]::uuid[])) FROM buckets WHERE ` + visible + `
UNION ALL
SELECT jsonb_build_object('kind','issue','issue',` + projection + `) FROM selected JOIN issues issue ON issue.id=selected.id`
}

func (r *Repository) projectGroupedListPage(
	ctx context.Context, q Request, shape issueListShape, group, subgroup string, limit, page, offset int,
	groups []any, deriveGroups bool, records []map[string]any,
) (Result, error) {
	groupCounts := make(map[string]int)
	rows := []map[string]any{}
	total := 0
	for _, record := range records {
		switch record["kind"] {
		case "total":
			total = groupRecordCount(record, "total")
		case "group":
			key := stringValue(record["group_key"])
			groupCounts[key] = groupRecordCount(record, "count")
			if deriveGroups {
				groups = append(groups, key)
			}
		case "issue":
			rows = append(rows, record["issue"].(map[string]any))
		}
	}
	if total == 0 {
		groups = nil
	}
	values, err := r.projectIssueListRows(ctx, q, shape, rows)
	if err != nil {
		return Result{}, err
	}
	projected := make(map[string]map[string]any, len(values))
	for _, value := range values {
		row := value.(map[string]any)
		projected[stringValue(row["id"])] = row
	}
	pages := []issuequery.GroupPage{}
	for _, record := range records {
		if record["kind"] != "bucket" {
			continue
		}
		bucket := issuequery.GroupPage{
			Group: stringValue(record["group_key"]), Subgroup: stringValue(record["subgroup_key"]),
			Count: groupRecordCount(record, "count"), RowCount: groupRecordCount(record, "row_count"),
		}
		for _, id := range record["ids"].([]any) {
			bucket.Rows = append(bucket.Rows, projected[stringValue(id)])
		}
		pages = append(pages, bucket)
	}
	grouped, count, maximum, next := issuequery.FormatGroupPages(group, subgroup, groups, groupCounts, pages, limit, offset)
	return r.groupedEnvelope(group, subgroup, limit, page, total, count, maximum, next, grouped), nil
}

func groupRecordCount(row map[string]any, column string) int {
	value, _ := strconv.Atoi(string(row[column].(json.Number)))
	return value
}
