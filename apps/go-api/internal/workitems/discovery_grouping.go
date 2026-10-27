package workitems

import (
	"context"
	"fmt"
	"slices"

	"github.com/CaliLuke/go-plane/internal/issuequery"
)

// Discovery axes are accepted only by the authenticated reader that owns their
// catalog snapshot. Do not add them to the public/API grouping allowlist.
func usesTypeGrouping(q Request) bool {
	for _, key := range []string{"group_by", "sub_group_by", "group_filter_by", "sub_group_filter_by"} {
		if q.Params[key] == "type_id" {
			return true
		}
	}
	return parentGroupingAxis(q) != ""
}

func (r *Repository) validateDiscoveryGrouping(group, subgroup string) error {
	if group != "" && group == subgroup {
		return failure(400, "error", "Group by and sub group by cannot have same parameters")
	}
	for _, axis := range []string{group, subgroup} {
		if axis == "type_id" && r.typeDiscovery != nil || r.propertyGroups[axis] != nil || r.parentGrouping != nil && axis == r.parentGrouping.axis {
			continue
		}
		if err := issuequery.ValidateGrouping(axis, ""); err != nil {
			return failure(400, "detail", err.Error())
		}
	}
	return nil
}

// Bucket selectors are separate conjunctions, so loading a group's next page
// cannot replace an existing OR/NOT expression or accidentally broaden it.
func (r *Repository) appendDiscoveryGroupSelectors(ctx context.Context, q Request, where []string, args []any) ([]string, []any, error) {
	for _, prefix := range []string{"group_filter", "sub_group_filter"} {
		axis, value := q.Params[prefix+"_by"], q.Params[prefix+"_value"]
		if axis == "" && value == "" {
			continue
		}
		if q.API || value == "" || r.typeDiscovery == nil {
			return nil, args, failure(400, "error", "Invalid work item group selection.")
		}
		if r.propertyGroups[axis] != nil || r.parentGrouping != nil && axis == r.parentGrouping.axis {
			continue
		}
		if axis != "type_id" {
			return nil, args, failure(400, "error", "Invalid work item group selection.")
		}
		if value == "None" {
			where = append(where, "issue.type_id IS NULL")
		} else {
			if !r.typeDiscovery.TypeIDs()[value] {
				return nil, args, failure(400, "error", "This work item type is unavailable. Update this grouping.")
			}
			args = append(args, value)
			where = append(where, fmt.Sprintf("issue.type_id=$%d::uuid", len(args)))
		}
	}
	where, args, err := r.appendParentGroupSelector(q, where, args)
	if err != nil {
		return nil, args, err
	}
	return r.appendPropertyGroupSelectors(ctx, q, where, args)
}

func (r *Repository) discoveryTypeGroups() []any {
	values := make([]any, 0, len(r.typeDiscovery.Types)+1)
	for _, item := range r.typeDiscovery.Types {
		values = append(values, item.ID)
	}
	return append(values, "None")
}

// Workspace discovery grouping must derive other axes from visible candidates;
// the legacy workspace entity universes are not an authorization source.
func visibleGroupValues(rows []map[string]any, axis string) []any {
	values := []any{}
	for _, row := range rows {
		for _, value := range issuequery.GroupValues(row, axis) {
			if !slices.Contains(values, any(value)) {
				values = append(values, value)
			}
		}
	}
	return values
}
