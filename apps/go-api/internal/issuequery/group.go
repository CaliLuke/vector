// Package issuequery provides pure issue filtering, ordering, and grouping helpers.
package issuequery

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func ValidateGrouping(groupBy, subGroupBy string) error {
	allowed := map[string]bool{
		"state_id": true, "state__group": true, "priority": true,
		"assignees__id": true, "labels__id": true, "issue_module__module_id": true,
		"created_by": true, "project_id": true, "cycle_id": true,
		"target_date": true, "start_date": true,
	}
	if groupBy == "" {
		return nil
	}
	if groupBy != "" && !allowed[groupBy] {
		return fmt.Errorf("Invalid group_by field: %s", groupBy)
	}
	if subGroupBy != "" && !allowed[subGroupBy] {
		return fmt.Errorf("Invalid sub_group_by field: %s", subGroupBy)
	}
	if groupBy != "" && groupBy == subGroupBy {
		return errors.New("Group by and sub group by cannot have same parameters")
	}
	return nil
}

func Group(
	issues []map[string]any,
	groupBy, subGroupBy string,
	groupValues, _ []any,
	groupCounts, subgroupCounts map[string]int,
	limit, page int,
) (any, int, int, bool) {
	singleM2MGroup := subGroupBy == "" && GroupArrayField(groupBy) != ""
	groups := normalizeModuleIssueGroupValues(groupValues)
	if singleM2MGroup {
		groups = mergeModuleIssueGroupValues(nil, issues, groupBy)
	}
	grouped := make(map[string]any, len(groups))
	offset := page * limit
	maximumGroup := 0
	hasNext := false
	for _, groupCount := range groupCounts {
		if groupCount > maximumGroup {
			maximumGroup = groupCount
		}
	}
	for _, group := range groups {
		groupIssues := moduleIssuesForGroup(issues, groupBy, group)
		paginationCount, hasGroupCount := groupCounts[group]
		if !hasGroupCount {
			paginationCount = moduleIssueTopPaginationCount(groupIssues)
		}
		if paginationCount > maximumGroup {
			maximumGroup = paginationCount
		}
		groupTotal := paginationCount
		if hasGroupCount && groupTotal == 0 {
			groupTotal = 1
		}
		if subGroupBy == "" {
			selected := moduleIssuePageRows(groupIssues, offset, limit)
			if singleM2MGroup && len(selected) == 0 {
				continue
			}
			if moduleIssueHasNext(groupIssues, offset, len(selected)) {
				hasNext = true
			}
			grouped[group] = map[string]any{
				"results":       moduleIssueGroupedRows(selected, groupBy, group, "", ""),
				"total_results": groupTotal,
			}
			continue
		}
		subgroups := mergeModuleIssueGroupValues(moduleIssueSubgroupSeedValues(subgroupCounts, group), groupIssues, subGroupBy)
		nested := make(map[string]any, len(subgroups))
		for _, subgroup := range subgroups {
			subgroupIssues := moduleIssuesForGroup(groupIssues, subGroupBy, subgroup)
			selected := moduleIssuePageRows(subgroupIssues, offset, limit)
			if moduleIssueHasNext(subgroupIssues, offset, len(selected)) {
				hasNext = true
			}
			nested[subgroup] = map[string]any{
				"results":       moduleIssueGroupedRows(selected, groupBy, group, subGroupBy, subgroup),
				"total_results": moduleIssueSubgroupCount(subgroupCounts, group, subgroup, subgroupIssues),
			}
		}
		grouped[group] = map[string]any{"results": nested, "total_results": groupTotal}
	}
	if len(issues) == 0 {
		return map[string]any{}, 0, 0, hasNext
	}
	reconstructModuleIssueGroupArrays(grouped, groupBy, subGroupBy)
	return grouped, len(issues), maximumGroup, hasNext
}
func reconstructModuleIssueGroupArrays(grouped map[string]any, groupBy, subGroupBy string) {
	rows := make([]map[string]any, 0)
	for _, groupValue := range grouped {
		group := groupValue.(map[string]any)
		if subGroupBy == "" {
			rows = append(rows, group["results"].([]map[string]any)...)
			continue
		}
		for _, subgroupValue := range group["results"].(map[string]any) {
			subgroup := subgroupValue.(map[string]any)
			rows = append(rows, subgroup["results"].([]map[string]any)...)
		}
	}
	ordered := make(map[string]map[string][]any)
	first := make(map[string]map[string]any)
	for _, row := range rows {
		identifier := fmt.Sprint(row["id"])
		if ordered[identifier] == nil {
			ordered[identifier] = make(map[string][]any)
			first[identifier] = make(map[string]any)
		}
		for _, field := range []string{groupBy, subGroupBy} {
			arrayField := GroupArrayField(field)
			value := row[field]
			if arrayField == "" || value == nil {
				continue
			}
			if _, exists := first[identifier][arrayField]; !exists {
				first[identifier][arrayField] = value
			}
			if !moduleGroupSliceContains(ordered[identifier][arrayField], fmt.Sprint(value)) {
				ordered[identifier][arrayField] = append(ordered[identifier][arrayField], value)
			}
		}
	}
	for _, row := range rows {
		identifier := fmt.Sprint(row["id"])
		for _, field := range []string{groupBy, subGroupBy} {
			if arrayField := GroupArrayField(field); arrayField != "" {
				values := ordered[identifier][arrayField]
				if values == nil {
					values = []any{}
				}
				row[arrayField] = values
				if subGroupBy == "" && field == groupBy {
					row[field] = first[identifier][arrayField]
				}
			}
		}
	}
}

func moduleGroupSliceContains(values []any, candidate string) bool {
	for _, value := range values {
		if fmt.Sprint(value) == candidate {
			return true
		}
	}
	return false
}

func normalizeModuleIssueGroupValues(values []any) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		key := "None"
		if value != nil {
			key = fmt.Sprint(value)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, key)
		}
	}
	return result
}

func mergeModuleIssueGroupValues(values []any, issues []map[string]any, field string) []string {
	if field == "" {
		return nil
	}
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	add := func(value string) {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	for _, value := range values {
		if value == nil {
			add("None")
		} else {
			add(fmt.Sprint(value))
		}
	}
	for _, issue := range issues {
		for _, value := range GroupValues(issue, field) {
			add(value)
		}
	}
	return result
}

func moduleIssuesForGroup(issues []map[string]any, field, group string) []map[string]any {
	result := make([]map[string]any, 0)
	for _, issue := range issues {
		for _, value := range GroupValues(issue, field) {
			if value == group {
				result = append(result, issue)
				break
			}
		}
	}
	return result
}
func moduleIssuePageRows(issues []map[string]any, offset, limit int) []map[string]any {
	if len(issues) > 0 {
		if _, pagedInRepository := issues[0]["_group_page_count"]; pagedInRepository {
			return issues
		}
	}
	if offset >= len(issues) {
		return []map[string]any{}
	}
	stop := min(offset+limit, len(issues))
	return issues[offset:stop]
}

func moduleIssuePaginationCount(issues []map[string]any) int {
	if len(issues) > 0 {
		if count, exists := issues[0]["_group_page_count"].(float64); exists {
			return int(count)
		}
	}
	count := 0
	for _, issue := range issues {
		if included, exists := issue["_count_for_pages"].(bool); !exists || included {
			count++
		}
	}
	return count
}

func moduleIssueTopPaginationCount(issues []map[string]any) int {
	if len(issues) > 0 {
		if count, exists := issues[0]["_top_group_page_count"].(float64); exists {
			return int(count)
		}
	}
	return moduleIssuePaginationCount(issues)
}
func moduleIssueSubgroupSeedValues(counts map[string]int, group string) []any {
	prefix := group + ":::"
	values := make([]string, 0)
	for key := range counts {
		if strings.HasPrefix(key, prefix) {
			values = append(values, strings.TrimPrefix(key, prefix))
		}
	}
	sort.Strings(values)
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = value
	}
	return result
}

func moduleIssueSubgroupCount(counts map[string]int, group, subgroup string, issues []map[string]any) int {
	if count, exists := counts[group+":::"+subgroup]; exists {
		return count
	}
	return moduleIssuePaginationCount(issues)
}

func moduleIssueHasNext(issues []map[string]any, offset, selected int) bool {
	if len(issues) > 0 {
		if hasNext, exists := issues[0]["_group_has_next"].(bool); exists {
			return hasNext
		}
		if count, exists := issues[0]["_group_row_count"].(float64); exists {
			return int(count) > offset+selected
		}
	}
	return len(issues) > offset+selected
}

func moduleIssueGroupedRows(
	issues []map[string]any,
	groupBy, group, subGroupBy, subgroup string,
) []map[string]any {
	groupArray := GroupArrayField(groupBy)
	subgroupArray := GroupArrayField(subGroupBy)
	results := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		item := make(map[string]any, len(issue))
		for key, value := range issue {
			item[key] = value
		}
		delete(item, "_count_for_pages")
		delete(item, "_module_group_ids")
		delete(item, "_selected_group")
		delete(item, "_selected_group_by")
		delete(item, "_selected_subgroup")
		delete(item, "_selected_subgroup_by")
		delete(item, "_group_page_count")
		delete(item, "_top_group_page_count")
		delete(item, "_group_row_count")
		delete(item, "_group_has_next")
		if groupArray != "" {
			item[groupBy] = nullableModuleGroupValue(group)
		}
		if subgroupArray != "" {
			item[subGroupBy] = nullableModuleGroupValue(subgroup)
		}
		results = append(results, item)
	}
	return results
}

func GroupArrayField(field string) string {
	switch field {
	case "assignees__id":
		return "assignee_ids"
	case "labels__id":
		return "label_ids"
	case "issue_module__module_id":
		return "module_ids"
	default:
		return ""
	}
}

func nullableModuleGroupValue(value string) any {
	if value == "None" {
		return nil
	}
	return value
}

func GroupValues(issue map[string]any, field string) []string {
	if selectedBy, exists := issue["_selected_group_by"].(string); exists && selectedBy == field {
		return []string{fmt.Sprint(issue["_selected_group"])}
	}
	if selectedBy, exists := issue["_selected_subgroup_by"].(string); exists && selectedBy == field {
		return []string{fmt.Sprint(issue["_selected_subgroup"])}
	}
	mapped := ""
	switch field {
	case "assignees__id":
		mapped = "assignee_ids"
	case "labels__id":
		mapped = "label_ids"
	case "issue_module__module_id":
		mapped = "module_ids"
		if _, exists := issue["_module_group_ids"]; exists {
			mapped = "_module_group_ids"
		}
	}
	if mapped == "" {
		if value := issue[field]; value != nil {
			return []string{fmt.Sprint(value)}
		}
		return []string{"None"}
	}
	values, _ := issue[mapped].([]any)
	if len(values) == 0 {
		return []string{"None"}
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := fmt.Sprint(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}
