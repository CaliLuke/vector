package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
)

var forbiddenProjectCharacters = regexp.MustCompile(`^.*[&+,:;$^}{*=?@#|'<>.()%!-].*$`)
var projectBooleanFields = []string{"module_view", "cycle_view", "issue_views_view", "page_view", "intake_view", "guest_view_all_features", "is_issue_type_enabled", "is_time_tracking_enabled"}

func (r *LifecycleRepository) validateProjectFields(ctx context.Context, workspace, project string, input djangoinput.Input, app, replace, create bool) (map[string]any, error) {
	fields, surrogates := input.Fields, input.Surrogates
	partial := !create && !replace
	values, failures := map[string]any{}, map[string]any{}
	if app {
		for _, key := range []string{"created_by", "updated_by", "archived_at"} {
			if _, supplied := fields[key]; supplied {
				failures[key] = []string{"This field is read-only."}
			}
		}
	}
	text := func(key string, required, blank, nullable bool, max int) {
		raw, present := input.FieldValue(key, required, blank, nullable)
		if !present {
			if required {
				failures[key] = []string{"This field is required."}
			}
			return
		}
		if raw == nil {
			if nullable {
				values[key] = nil
			} else {
				failures[key] = []string{"This field may not be null."}
			}
			return
		}
		value, messages := djangoinput.TextField(raw, blank, max, surrogates[key])
		if len(messages) > 0 {
			failures[key] = messages
			return
		}
		values[key] = value
	}
	text("name", create || replace, false, false, 255)
	text("identifier", create || replace, false, false, 12)
	text("description", false, true, false, 0)
	text("cover_image", false, true, true, 0)
	text("emoji", false, true, true, 255)
	text("external_source", false, true, true, 255)
	text("external_id", false, true, true, 255)
	if raw, present := input.FieldValue("timezone", false, false, false); present {
		zone, err := projectFieldErrorString(input, "timezone", raw)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			failures["timezone"] = []string{"This field may not be null."}
		} else if !validProjectTimezone(zone) {
			failures["timezone"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", zone)}
		} else {
			values["timezone"] = zone
		}
	}
	for _, key := range projectBooleanFields {
		raw, present := input.BooleanFieldValue(key, partial)
		if !present {
			continue
		}
		if raw == nil {
			failures[key] = []string{"This field may not be null."}
			continue
		}
		if value, ok := projectBoolean(raw); ok {
			values[key] = value
		} else {
			failures[key] = []string{"Must be a valid boolean."}
		}
	}
	for _, key := range []string{"archive_in", "close_in"} {
		raw, present := input.FieldValue(key, false, false, false)
		if !present {
			continue
		}
		if raw == nil {
			failures[key] = []string{"This field may not be null."}
			continue
		}
		number, err := djangoinput.IntegerField(raw)
		if err != nil || !number.IsInt64() {
			failures[key] = []string{"A valid integer is required."}
			continue
		}
		value := number.Int64()
		if value < 0 {
			failures[key] = []string{"Ensure this value is greater than or equal to 0."}
		} else if value > 12 {
			failures[key] = []string{"Ensure this value is less than or equal to 12."}
		} else {
			values[key] = value
		}
	}
	for _, key := range []string{"icon_prop", "description_html", "description_text", "logo_props"} {
		if !app && key != "icon_prop" {
			continue
		}
		if _, present := fields[key]; present && !projectJSONFieldUnicodeValid(input, key) {
			failures[key] = []string{"Value must be valid JSON."}
			continue
		}
		raw, present, err := input.JSONFieldValue(key)
		if err != nil {
			failures[key] = []string{"Value must be valid JSON."}
			continue
		}
		if present {
			if raw == nil && key == "logo_props" {
				failures[key] = []string{"This field may not be null."}
			} else {
				normalized, err := normalizeProjectJSONField(raw)
				if err != nil {
					failures[key] = []string{"Value must be valid JSON."}
				} else {
					values[key] = normalized
				}
			}
		}
	}
	if app && !replace {
		if raw, present := input.FieldValue("network", false, false, false); present {
			choice, err := projectFieldErrorString(input, "network", raw)
			if err != nil {
				return nil, err
			}
			if raw == nil {
				failures["network"] = []string{"This field may not be null."}
			} else if choice != "0" && choice != "2" {
				failures["network"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", choice)}
			} else {
				if choice == "0" {
					values["network"] = int64(0)
				} else {
					values["network"] = int64(2)
				}
			}
		}
	}
	for _, key := range []string{"project_lead", "default_assignee", "default_state", "estimate"} {
		if create && (key == "default_state" || key == "estimate") && !app {
			continue
		}
		raw, present := input.FieldValue(key, false, false, true)
		if !present {
			continue
		}
		column := key + "_id"
		if raw == nil || raw == "" {
			values[column] = nil
			continue
		}
		if _, boolean := raw.(bool); boolean {
			failures[key] = []string{"Incorrect type. Expected pk value, received bool."}
			continue
		}
		id, err := djangoinput.UUID(raw)
		if err != nil {
			value, stringErr := projectFieldErrorString(input, key, raw)
			if stringErr != nil {
				return nil, stringErr
			}
			failures[key] = []string{"“" + value + "” is not a valid UUID."}
			continue
		}
		table := "users"
		condition := ""
		if key == "default_state" {
			table = "states"
			condition = ` AND deleted_at IS NULL AND "group"<>'triage'`
		}
		if key == "estimate" {
			table = "estimates"
			condition = " AND deleted_at IS NULL"
		}
		var exists bool
		if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id=$1`+condition+`)`, id.String()).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			value, stringErr := projectFieldErrorString(input, key, raw)
			if stringErr != nil {
				return nil, stringErr
			}
			failures[key] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", value)}
			continue
		}
		values[column] = id.String()
	}
	if app {
		if raw, supplied := input.FieldValue("cover_image_asset", false, false, true); supplied {
			if raw == nil || raw == "" {
				values["cover_image_asset_id"] = nil
			} else if _, boolean := raw.(bool); boolean {
				failures["cover_image_asset"] = []string{"Incorrect type. Expected pk value, received bool."}
			} else {
				assetID, err := djangoinput.UUID(raw)
				if err != nil {
					value, stringErr := projectFieldErrorString(input, "cover_image_asset", raw)
					if stringErr != nil {
						return nil, stringErr
					}
					failures["cover_image_asset"] = []string{"“" + value + "” is not a valid UUID."}
				} else {
					var exists bool
					if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1 AND deleted_at IS NULL)`, assetID.String()).Scan(&exists); err != nil {
						return nil, err
					}
					if !exists {
						value, stringErr := projectFieldErrorString(input, "cover_image_asset", raw)
						if stringErr != nil {
							return nil, stringErr
						}
						failures["cover_image_asset"] = []string{fmt.Sprintf("Invalid pk %q - object does not exist.", value)}
					} else if create {
						failures["cover_image_asset"] = []string{"The provided asset is not a cover image for this project."}
					} else {
						allowed, err := r.coverAssets(ctx, uuid.MustParse(workspace), uuid.MustParse(project), assetID)
						if err != nil {
							return nil, err
						}
						if !allowed {
							failures["cover_image_asset"] = []string{"The provided asset is not a cover image for this project."}
						} else {
							values["cover_image_asset_id"] = assetID.String()
						}
					}
				}
			}
		}
		for _, key := range []string{"name", "identifier"} {
			value, ok := values[key].(string)
			if !ok {
				continue
			}
			upper := strings.ToUpper(key)
			if forbiddenProjectCharacters.MatchString(value) {
				failures[key] = []string{"PROJECT_" + upper + "_CANNOT_CONTAIN_SPECIAL_CHARACTERS"}
				continue
			}
			var duplicate bool
			if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE workspace_id=$1 AND `+key+`=$2 AND deleted_at IS NULL AND ($3='' OR id::text<>$3))`, workspace, value, project).Scan(&duplicate); err != nil {
				return nil, err
			}
			if duplicate {
				failures[key] = []string{"PROJECT_" + upper + "_ALREADY_EXIST"}
			}
		}
	}
	if len(failures) > 0 {
		return nil, projectFailure(400, failures)
	}
	if !app {
		for _, key := range []string{"name", "identifier"} {
			if value, ok := values[key].(string); ok && forbiddenProjectCharacters.MatchString(value) {
				label := "Project name"
				if key == "identifier" {
					label = "Project identifier"
				}
				return nil, projectFailure(400, map[string]any{"non_field_errors": []string{label + " cannot contain special characters."}})
			}
		}
	}
	relationErrors := map[string]any{}
	for _, key := range []string{"project_lead", "default_assignee", "default_state", "estimate"} {
		value, present := values[key+"_id"]
		if !present || value == nil {
			continue
		}
		var allowed bool
		query := `SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 AND is_active=true AND deleted_at IS NULL)`
		scope := workspace
		if key == "default_state" {
			query = `SELECT EXISTS(SELECT 1 FROM states WHERE project_id::text=$1 AND id=$2 AND deleted_at IS NULL AND "group"<>'triage')`
			scope = project
		}
		if key == "estimate" {
			query = `SELECT EXISTS(SELECT 1 FROM estimates WHERE project_id::text=$1 AND id=$2 AND deleted_at IS NULL)`
			scope = project
		}
		if err := r.writer.QueryRow(ctx, query, scope, value).Scan(&allowed); err != nil {
			return nil, err
		}
		if !allowed {
			if app {
				message := "The provided user is not an active member of this workspace."
				if key == "default_state" {
					message = "The provided state does not belong to this project."
				}
				if key == "estimate" {
					message = "The provided estimate does not belong to this project."
				}
				relationErrors[key] = []string{message}
			} else {
				switch key {
				case "project_lead":
					return nil, projectFailure(400, map[string]any{"project_lead": []string{"The provided user is not a member of this workspace."}})
				case "default_assignee":
					return nil, projectFailure(400, map[string]any{"default_assignee": []string{"The provided user is not a member of this workspace."}})
				case "default_state":
					return nil, projectFailure(400, map[string]any{"default_state": []string{"The provided state does not belong to this project."}})
				case "estimate":
					return nil, projectFailure(400, map[string]any{"estimate": []string{"The provided estimate does not belong to this project."}})
				}
			}
		}
	}
	if len(relationErrors) > 0 {
		return nil, projectFailure(400, relationErrors)
	}

	if app {
		if raw, present := values["description_html"]; present && pythoncompat.Truthy(raw) {
			text, err := pythoncompat.String(raw, input.JSONFieldSource("description_html"))
			if err != nil {
				return nil, err
			}
			clean, hasClean, err := content.ValidateHTML(text)
			if err != nil {
				return nil, projectFailure(400, map[string]any{"error": []string{"html content is not valid"}})
			}
			if hasClean {
				values["description_html"] = clean
			}
		}
	}
	return values, nil
}

func projectFieldErrorString(input djangoinput.Input, field string, value any) (string, error) {
	switch value.(type) {
	case []any, map[string]any:
		return pythoncompat.String(value, input.JSONFieldSource(field))
	default:
		return djangoinput.ScalarString(value), nil
	}
}

func projectBoolean(raw any) (bool, bool) {
	switch v := raw.(type) {
	case bool:
		return v, true
	case json.Number:
		label := djangoinput.ScalarString(v)
		if label == "1" || label == "1.0" {
			return true, true
		}
		if label == "0" || label == "0.0" {
			return false, true
		}
		if strings.ContainsAny(string(v), ".eE") {
			number, err := v.Float64()
			if err == nil {
				if number == 1 {
					return true, true
				}
				if number == 0 {
					return false, true
				}
			}
		}
	case string:
		switch strings.ToLower(v) {
		case "t", "y", "yes", "true", "on", "1":
			return true, true
		case "f", "n", "no", "false", "off", "0":
			return false, true
		}
	}
	return false, false
}

func normalizeProjectJSONField(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	normalized, err := pythoncompat.NormalizeJSONNumbers(raw)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(normalized)))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// projectJSONFieldUnicodeValid checks the original JSONField text before the
// request decoder can replace lone-surrogate escapes with U+FFFD. Duplicate
// names remain valid; an invalid escape in an overwritten duplicate is still
// rejected as part of validating the submitted field source.
func projectJSONFieldUnicodeValid(input djangoinput.Input, field string) bool {
	source := input.JSONFieldSource(field)
	if len(source) == 0 {
		value, present := input.Fields[field]
		if !present {
			return true
		}
		var err error
		source, err = json.Marshal(value)
		if err != nil {
			return false
		}
	}
	return jsontext.Value(source).IsValid(jsontext.AllowDuplicateNames(true))
}
