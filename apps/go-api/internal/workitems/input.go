package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
)

type issueValidation struct {
	API, Partial, AllowTriage, Draft bool
	ProjectID, WorkspaceID, Timezone string
}
type issueChanges struct {
	values                  map[string]any
	assignees, labels       []string
	assigneesSet, labelsSet bool
}

func (r *Repository) validateIssue(ctx context.Context, input djangoinput.Input, options issueValidation) (issueChanges, error) {
	changes := issueChanges{values: map[string]any{}}
	if input.Null {
		return changes, invalid(map[string]any{"non_field_errors": []string{"No data provided"}})
	}
	if input.Kind != "" {
		return changes, invalid(map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}})
	}
	failures := map[string]any{}
	fields := input.Fields
	for _, field := range []string{"name", "description_html", "external_source", "external_id"} {
		value, present := fields[field]
		if !present {
			if field == "name" && !options.Partial && !options.Draft {
				failures[field] = []string{"This field is required."}
			}
			continue
		}
		if value == nil && (field == "external_source" || field == "external_id" || field == "name" && options.Draft) {
			changes.values[field] = nil
			continue
		}
		maximum := 255
		if field == "description_html" {
			maximum = 0
		}
		parsed, errs := djangoinput.TextField(value, field != "name" || options.Draft, maximum, input.Surrogates[field])
		if len(errs) > 0 {
			failures[field] = errs
		} else {
			changes.values[field] = parsed
		}
	}
	if value, present := fields["priority"]; present {
		if value == nil {
			failures["priority"] = []string{"This field may not be null."}
		} else {
			v := djangoinput.ScalarString(value)
			if !slices.Contains([]string{"urgent", "high", "medium", "low", "none"}, v) {
				failures["priority"] = []string{fmt.Sprintf("%q is not a valid choice.", v)}
			} else {
				changes.values["priority"] = v
			}
		}
	}
	for _, field := range []string{"start_date", "target_date", "archived_at"} {
		if value, present := fields[field]; present {
			if value == nil || input.Form && value == "" {
				changes.values[field] = nil
				continue
			}
			text, ok := value.(string)
			parsed, err := parseDate(text)
			if !ok || err != nil {
				failures[field] = []string{"Date has wrong format. Use one of these formats instead: YYYY-MM-DD."}
			} else {
				changes.values[field] = parsed
			}
		}
	}
	datetimeFields := []string{"deleted_at"}
	if options.API {
		datetimeFields = append(datetimeFields, "created_at")
	}
	for _, field := range datetimeFields {
		if value, present := fields[field]; present {
			if value == nil || input.Form && value == "" {
				if field == "created_at" {
					failures[field] = []string{"This field may not be null."}
				} else {
					changes.values[field] = nil
				}
			} else {
				raw, ok := value.(string)
				parsed, err := parseQueryDateTime(raw, options.Timezone)
				if !ok || err != nil {
					failures[field] = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
				} else {
					changes.values[field] = parsed
				}
			}
		}
	}

	if value, present := fields["sort_order"]; present && !(input.Form && value == "") {
		if value == nil {
			failures["sort_order"] = []string{"This field may not be null."}
		} else if parsed, message := djangoinput.FloatField(value); message != "" {
			failures["sort_order"] = []string{message}
		} else {
			changes.values["sort_order"] = parsed
		}
	}
	for _, field := range []string{"point", "sequence_id"} {
		if value, present := fields[field]; present {
			if value == nil {
				if field == "point" {
					changes.values[field] = nil
				} else {
					failures[field] = []string{"This field may not be null."}
				}
				continue
			}
			integer, err := integerField(value)
			if err != nil {
				failures[field] = []string{"A valid integer is required."}
				continue
			}
			minimum, maximum := int64(-2147483648), int64(2147483647)
			if field == "point" {
				minimum, maximum = 0, 12
			}
			if integer.Cmp(big.NewInt(minimum)) < 0 {
				failures[field] = []string{fmt.Sprintf("Ensure this value is greater than or equal to %d.", minimum)}
			} else if integer.Cmp(big.NewInt(maximum)) > 0 {
				failures[field] = []string{fmt.Sprintf("Ensure this value is less than or equal to %d.", maximum)}
			} else {
				changes.values[field] = integer.Int64()
			}
		}
	}
	if value, present := fields["is_draft"]; present {
		if parsed, ok := booleanField(value); ok {
			changes.values["is_draft"] = parsed
		} else {
			failures["is_draft"] = []string{"Must be a valid boolean."}
		}
	} else if input.Form && !options.Partial {
		changes.values["is_draft"] = false
	}
	if !options.API {
		if value, present := fields["description_json"]; present {
			if value == nil {
				failures["description_json"] = []string{"This field may not be null."}
			} else if input.Form {
				var parsed any
				decoder := json.NewDecoder(strings.NewReader(djangoinput.ScalarString(value)))
				decoder.UseNumber()
				if err := decoder.Decode(&parsed); err != nil {
					failures["description_json"] = []string{"Value must be valid JSON."}
				} else {
					changes.values["description_json"] = parsed
				}
			} else {
				changes.values["description_json"] = value
			}
		}
	}
	// Declared aliases are evaluated before model relationship fields, as DRF's
	// ModelSerializer does when two input fields have the same source.
	fieldsToRelations := [][3]string{{"estimate_point", "estimate_point_id", "estimate_points"}}
	if options.API {
		fieldsToRelations = append(fieldsToRelations, [][3]string{{"type_id", "type_id", "issue_types"}, {"type", "type_id", "issue_types"}, {"state", "state_id", "states"}, {"parent", "parent_id", "issues"}, {"created_by", "created_by_id", "users"}}...)
	} else {
		fieldsToRelations = append([][3]string{{"state_id", "state_id", "states"}, {"parent_id", "parent_id", "issues"}}, append(fieldsToRelations, [][3]string{{"type", "type_id", "issue_types"}, {"state", "state_id", "states"}, {"parent", "parent_id", "issues"}}...)...)
	}
	for _, relation := range fieldsToRelations {
		field, column, table := relation[0], relation[1], relation[2]
		value, present := fields[field]
		if !present {
			continue
		}
		if value == nil || value == "" {
			changes.values[column] = nil
			continue
		}
		id, messages, err := r.primaryKey(ctx, value, table, field == "state_id" && !options.Draft)
		if err != nil {
			return changes, err
		}
		if len(messages) > 0 {
			failures[field] = messages
		} else {
			changes.values[column] = id
		}
	}
	assignees, labels := "assignee_ids", "label_ids"
	if options.API {
		assignees, labels = "assignees", "labels"
	}
	for _, item := range []struct {
		field, table string
		values       *[]string
		set          *bool
	}{{assignees, "users", &changes.assignees, &changes.assigneesSet}, {labels, "labels", &changes.labels, &changes.labelsSet}} {
		value, present := fields[item.field]
		if !present {
			continue
		}
		*item.set = true
		if value == nil {
			failures[item.field] = []string{"This field may not be null."}
			continue
		}
		values, ok := value.([]any)
		if !ok {
			failures[item.field] = []string{fmt.Sprintf("Expected a list of items but got type %q.", djangoinput.PythonType(value))}
			continue
		}
		indexed := map[string]any{}
		for index, v := range values {
			if v == nil {
				indexed[strconv.Itoa(index)] = []string{"This field may not be null."}
				continue
			}
			id, messages, err := r.primaryKey(ctx, v, item.table, false)
			if err != nil {
				return changes, err
			}
			if len(messages) > 0 {
				indexed[strconv.Itoa(index)] = messages
			} else {
				*item.values = append(*item.values, id)
			}
		}
		if len(indexed) > 0 {
			failures[item.field] = indexed
		}
	}
	if len(failures) > 0 {
		return changes, invalid(failures)
	}
	if start, ok := changes.values["start_date"].(string); ok {
		if target, ok := changes.values["target_date"].(string); ok && start > target {
			return changes, invalid(map[string]any{"non_field_errors": []string{"Start date cannot exceed target date"}})
		}
	}
	if html, present := changes.values["description_html"]; present {
		text := html.(string)
		if options.API {
			clean, err := normalizeAPIHTML(text)
			if errors.Is(err, errInvalidAPIHTML) {
				return changes, invalid(map[string]any{"non_field_errors": []string{"Invalid HTML passed"}})
			}
			if err != nil {
				return changes, invalid(map[string]any{"error": []string{"html content is not valid"}})
			}
			text = clean
		} else if text != "" {
			clean, _, err := content.ValidateHTML(text)
			if err != nil {
				return changes, invalid(map[string]any{"error": []string{"html content is not valid"}})
			}
			text = clean
		}
		changes.values["description_html"] = text
	}
	if len(changes.assignees) > 0 {
		if options.ProjectID == "" && !options.API && !options.Draft {
			return changes, failure(400, "error", "The required key does not exist.")
		}
		ids, err := r.filteredIDs(ctx, `SELECT member_id::text FROM project_members WHERE project_id=NULLIF($1,'')::uuid AND member_id=ANY($2::uuid[]) AND role>=15 AND is_active AND deleted_at IS NULL ORDER BY created_at DESC`, options.ProjectID, changes.assignees)
		if err != nil {
			return changes, err
		}
		changes.assignees = ids
	}
	if len(changes.labels) > 0 {
		ids, err := r.filteredIDs(ctx, `SELECT id::text FROM labels WHERE project_id IS NOT DISTINCT FROM NULLIF($1,'')::uuid AND id=ANY($2::uuid[]) AND deleted_at IS NULL ORDER BY created_at DESC`, options.ProjectID, changes.labels)
		if err != nil {
			return changes, err
		}
		changes.labels = ids
	}
	for _, relation := range [][3]string{{"state_id", "states", "State is not valid please pass a valid state_id"}, {"parent_id", "issues", "Parent is not valid issue_id please pass a valid issue_id"}, {"estimate_point_id", "estimate_points", "Estimate point is not valid please pass a valid estimate_point_id"}} {
		id, present := changes.values[relation[0]]
		if !present || id == nil {
			continue
		}
		predicate := `deleted_at IS NULL`
		if relation[1] == "states" {
			if options.AllowTriage {
				predicate += ` AND "group"='triage'`
			} else {
				predicate += ` AND "group"<>'triage'`
			}
		}
		if options.API && relation[1] != "states" {
			predicate += ` AND workspace_id=NULLIF($3,'')::uuid`
		} else {
			predicate += ` AND $3::text IS NOT NULL`
		}
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+relation[1]+` WHERE id=$1 AND project_id=NULLIF($2,'')::uuid AND `+predicate+`)`, id, options.ProjectID, options.WorkspaceID).Scan(&exists)
		if err != nil {
			return changes, err
		}
		if !exists {
			return changes, invalid(map[string]any{"non_field_errors": []string{relation[2]}})
		}
	}
	return changes, nil
}

func (r *Repository) primaryKey(ctx context.Context, value any, table string, allStates bool) (string, []string, error) {
	if _, ok := value.(bool); ok {
		return "", []string{"Incorrect type. Expected pk value, received bool."}, nil
	}
	id, err := djangoinput.UUID(value)
	if err != nil {
		return "", []string{"“" + djangoinput.ScalarString(value) + "” is not a valid UUID."}, nil
	}
	predicate := ` AND deleted_at IS NULL`
	if table == "users" || allStates {
		predicate = ""
	}
	if table == "states" && !allStates {
		predicate += ` AND "group"<>'triage'`
	}
	var exists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1`+predicate+`)`, id.String()).Scan(&exists)
	if err != nil {
		return "", nil, err
	}
	if !exists {
		return "", []string{fmt.Sprintf("Invalid pk %q - object does not exist.", djangoinput.ScalarString(value))}, nil
	}
	return id.String(), nil, nil
}

func (r *Repository) filteredIDs(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func parseDate(value string) (string, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid date")
	}
	month, e1 := strconv.Atoi(parts[1])
	day, e2 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil {
		return "", fmt.Errorf("invalid date")
	}
	text := fmt.Sprintf("%s-%02d-%02d", parts[0], month, day)
	_, err := time.Parse("2006-01-02", text)
	return text, err
}
func integerField(value any) (*big.Int, error) {
	if _, ok := value.(bool); ok {
		return nil, fmt.Errorf("boolean integer")
	}
	text := strings.TrimSpace(djangoinput.ScalarString(value))
	if index := strings.IndexByte(text, '.'); index >= 0 {
		if strings.Trim(text[index+1:], "0") == "" {
			text = text[:index]
		}
	}
	return pythoncompat.DecimalInteger(text)
}
func booleanField(value any) (bool, bool) {
	if v, ok := value.(bool); ok {
		return v, true
	}
	text := strings.ToLower(djangoinput.ScalarString(value))
	switch text {
	case "t", "y", "yes", "true", "on", "1", "1.0":
		return true, true
	case "f", "n", "no", "false", "off", "0", "0.0":
		return false, true
	}
	return false, false
}
