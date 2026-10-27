package workitems

import (
	"context"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func propertyGroupingKeys(q Request) []string {
	keys := []string{}
	for _, parameter := range []string{"group_by", "sub_group_by", "group_filter_by", "sub_group_filter_by"} {
		if axis := q.Params[parameter]; strings.HasPrefix(axis, "customproperty_") {
			keys = append(keys, axis)
		}
	}
	return keys
}

func usesDiscoveryGrouping(q Request) bool {
	return usesTypeGrouping(q) || len(propertyGroupingKeys(q)) > 0
}

func (r *Repository) preparePropertyGrouping(ctx context.Context, q Request) error {
	keys := propertyGroupingKeys(q)
	if len(keys) == 0 {
		return nil
	}
	var err error
	r.propertyGroups, err = r.propertyDiscovery.GroupingProperties(ctx, keys)
	return discoveryReadError(err)
}

type propertyGroupSelection struct{ axis, value string }

// Select groups before SQL counts/windows, under the same authorized snapshot
// as the structured filter. A selector never rewrites the user's OR/NOT tree.
func (r *Repository) appendPropertyGroupSelectors(ctx context.Context, q Request, where []string, args []any) ([]string, []any, error) {
	selected := []propertyGroupSelection{}
	for _, prefix := range []string{"group_filter", "sub_group_filter"} {
		axis := q.Params[prefix+"_by"]
		if r.propertyGroups[axis] != nil {
			selected = append(selected, propertyGroupSelection{axis, q.Params[prefix+"_value"]})
		}
	}
	if len(selected) == 0 {
		return where, args, nil
	}
	rows, err := r.propertyCandidateRows(ctx, where, args)
	if err != nil {
		return nil, args, err
	}
	ids := []string{}
	for _, row := range rows {
		if r.matchesPropertyGroups(row, selected) {
			ids = append(ids, stringValue(row["id"]))
		}
	}
	args = append(args, ids)
	return append(where, fmt.Sprintf("issue.id=ANY($%d::uuid[])", len(args))), args, nil
}

func (r *Repository) matchesPropertyGroups(row map[string]any, selected []propertyGroupSelection) bool {
	values := row["property_values"].(map[string]workitemtypes.ProjectedValue)
	for _, selection := range selected {
		if workitemtypes.PropertyGroupValue(*r.propertyGroups[selection.axis], values) != selection.value {
			return false
		}
	}
	return true
}

func (r *Repository) projectPropertyGroups(row, result map[string]any) {
	values, _ := row["property_values"].(map[string]workitemtypes.ProjectedValue)
	for axis, field := range r.propertyGroups {
		result[axis] = workitemtypes.PropertyGroupValue(*field, values)
	}
}
