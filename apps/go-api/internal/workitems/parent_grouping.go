package workitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

const parentGroupingPrefix = "parent_type_"

type parentGrouping struct {
	axis    string
	parents []map[string]any
	ids     []string
}

func parentGroupingAxis(q Request) string {
	for _, parameter := range []string{"group_by", "sub_group_by", "group_filter_by", "sub_group_filter_by"} {
		if axis := q.Params[parameter]; strings.HasPrefix(axis, parentGroupingPrefix) {
			return axis
		}
	}
	return ""
}

func (r *Repository) prepareParentGrouping(ctx context.Context, q Request) error {
	axis := parentGroupingAxis(q)
	if axis == "" {
		return nil
	}
	if strings.HasPrefix(q.Params["sub_group_by"], parentGroupingPrefix) || strings.HasPrefix(q.Params["sub_group_filter_by"], parentGroupingPrefix) {
		return failure(400, "error", "Parent type is available only as the primary grouping.")
	}
	typeID := strings.TrimPrefix(axis, parentGroupingPrefix)
	if r.typeDiscovery == nil || !r.typeDiscovery.TypeIDs()[typeID] {
		return failure(400, "error", "This parent type is unavailable. Choose another group.")
	}
	if selected := q.Params["group_filter_by"]; strings.HasPrefix(selected, parentGroupingPrefix) && selected != axis {
		return failure(400, "error", "Conflicting parent type selections.")
	}
	// A visible cross-project parent is included only through a visible child in
	// this project. Project-local parents also supply names for empty groups.
	parents, err := r.rows(ctx, `SELECT jsonb_build_object('id',parent.id,'name',parent.name,
 'project_id',parent.project_id,'identifier',p.identifier||'-'||parent.sequence_id::text)
 FROM issues parent JOIN projects p ON p.id=parent.project_id JOIN workspaces w ON w.id=parent.workspace_id
 WHERE w.slug=$1 AND parent.type_id=$2::uuid AND `+djangoapi.IssueVisibleSQL("parent", "$3::uuid", "w.id")+`
 AND ($4='' OR parent.project_id::text=$4 OR EXISTS(SELECT 1 FROM issues child
 WHERE child.parent_id=parent.id AND child.project_id::text=$4 AND `+djangoapi.IssueVisibleSQL("child", "$3::uuid", "w.id")+`))
 ORDER BY parent.created_at,parent.id`, q.Slug, typeID, q.ActorID, q.ProjectID)
	if err != nil {
		return err
	}
	group := &parentGrouping{axis: axis, parents: parents, ids: []string{}}
	for _, parent := range parents {
		group.ids = append(group.ids, stringValue(parent["id"]))
	}
	r.parentGrouping = group
	return nil
}

func (r *Repository) appendParentGroupSelector(q Request, where []string, args []any) ([]string, []any, error) {
	if r.parentGrouping == nil || q.Params["group_filter_by"] != r.parentGrouping.axis {
		return where, args, nil
	}
	value := q.Params["group_filter_value"]
	if value == "None" {
		args = append(args, r.parentGrouping.ids)
		return append(where, fmt.Sprintf("(issue.parent_id IS NULL OR NOT (issue.parent_id=ANY($%d::uuid[])))", len(args))), args, nil
	}
	for _, id := range r.parentGrouping.ids {
		if value == id {
			args = append(args, id)
			return append(where, fmt.Sprintf("issue.parent_id=$%d::uuid", len(args))), args, nil
		}
	}
	return nil, args, failure(400, "error", "This parent group is unavailable. Reload the view.")
}

func (r *Repository) projectParentGroup(row, result map[string]any) {
	if r.parentGrouping == nil {
		return
	}
	value := "None"
	for _, id := range r.parentGrouping.ids {
		if id == stringValue(row["parent_id"]) {
			value = id
			break
		}
	}
	result[r.parentGrouping.axis] = value
}
