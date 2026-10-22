package workitems

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
)

func (r *Repository) validateCollaboration(ctx context.Context, q Request, input djangoinput.Input, table string) (map[string]any, error) {
	values := map[string]any{}
	if input.Null {
		return nil, invalid(map[string]any{"non_field_errors": []string{"No data provided"}})
	}
	if input.Kind != "" {
		return nil, invalid(map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}})
	}
	failures := map[string]any{}
	texts := []string{}
	nullable := map[string]bool{}
	foreign := map[string]string{}
	jsonFields := []string{}
	dates := []string{}
	switch table {
	case "issue_comments":
		texts = []string{"comment_html", "external_source", "external_id"}
		nullable["external_source"], nullable["external_id"] = true, true
		jsonFields = []string{"comment_json"}
		if !q.API {
			texts = append(texts, "comment_stripped")
			foreign = map[string]string{"actor": "users", "parent": "issue_comments", "description": "descriptions"}
			dates = []string{"edited_at", "deleted_at"}
		}
	case "issue_links":
		texts = []string{"title", "url"}
		nullable["title"] = true
		if !q.API || q.Method != "POST" {
			jsonFields = []string{"metadata"}
			dates = []string{"deleted_at"}
		}
	case "issue_reactions":
		texts = []string{"reaction"}
		foreign = map[string]string{"created_by": "users", "updated_by": "users"}
	case "comment_reactions":
		texts = []string{"reaction"}
	case "issue_subscribers":
		foreign = map[string]string{"subscriber": "users", "created_by": "users", "updated_by": "users"}
		dates = []string{"deleted_at"}
	}
	for _, field := range texts {
		value, present := input.Fields[field]
		required := field == "reaction" || field == "url"
		if !present {
			if required && q.Method != "PATCH" {
				failures[field] = []string{"This field is required."}
			}
			continue
		}
		if value == nil && nullable[field] {
			values[field] = nil
			continue
		}
		maximum := 255
		if strings.HasPrefix(field, "comment_") || field == "url" || field == "reaction" {
			maximum = 0
		}
		parsed, messages := djangoinput.TextField(value, field != "reaction" && field != "url", maximum, input.Surrogates[field])
		if len(messages) > 0 {
			failures[field] = messages
			continue
		}
		if field == "url" {
			if !q.API && parsed != "" && !strings.HasPrefix(parsed, "http://") && !strings.HasPrefix(parsed, "https://") {
				parsed = "http://" + parsed
			}
			if !djangoinput.ValidURL(parsed) {
				if q.API {
					failures[field] = []string{"Enter a valid URL."}
					if q.Method == "POST" {
						failures[field] = []string{"Invalid URL format."}
					}
				} else {
					failures[field] = map[string]any{"error": "Invalid URL format."}
				}
				continue
			}
			if q.API && q.Method == "POST" && !strings.HasPrefix(parsed, "http://") && !strings.HasPrefix(parsed, "https://") {
				failures[field] = []string{"Invalid URL scheme."}
				continue
			}
		}
		values[field] = parsed
	}
	if value, present := input.Fields["access"]; present && table == "issue_comments" {
		if value == nil {
			failures["access"] = []string{"This field may not be null."}
		} else {
			access := djangoinput.ScalarString(value)
			if access != "INTERNAL" && access != "EXTERNAL" {
				failures["access"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", access)}
			} else {
				values["access"] = access
			}
		}
	}
	for _, field := range jsonFields {
		if value, present := input.Fields[field]; present {
			if value == nil {
				failures[field] = []string{"This field may not be null."}
			} else if input.Form {
				var parsed any
				decoder := json.NewDecoder(strings.NewReader(djangoinput.ScalarString(value)))
				decoder.UseNumber()
				if err := decoder.Decode(&parsed); err != nil {
					failures[field] = []string{"Value must be valid JSON."}
				} else {
					values[field] = parsed
				}
			} else {
				values[field] = value
			}
		}
	}
	for field, table := range foreign {
		value, present := input.Fields[field]
		if !present {
			if field == "subscriber" && q.Method == "POST" {
				failures[field] = []string{"This field is required."}
			}
			continue
		}
		if value == nil || value == "" {
			if field == "subscriber" {
				failures[field] = []string{"This field may not be null."}
			} else {
				values[field+"_id"] = nil
			}
			continue
		}
		id, messages, err := r.primaryKey(ctx, value, table, false)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			failures[field] = messages
		} else {
			values[field+"_id"] = id
		}
	}
	for _, field := range dates {
		if value, present := input.Fields[field]; present {
			if value == nil || input.Form && value == "" {
				values[field] = nil
				continue
			}
			raw, ok := value.(string)
			parsed, err := parseQueryDateTime(raw, q.Timezone)
			if !ok || err != nil {
				failures[field] = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
			} else {
				values[field] = parsed
			}
		}
	}
	if table == "issue_comments" && !q.API {
		if value, present := input.ListValue("attachments", q.Method == "PATCH"); present {
			list, ok := value.([]any)
			if !ok {
				if value == nil {
					failures["attachments"] = []string{"This field may not be null."}
				} else {
					failures["attachments"] = []string{fmt.Sprintf("Expected a list of items but got type \"%s\".", djangoinput.PythonType(value))}
				}
			} else if len(list) > 10 {
				failures["attachments"] = []string{"Ensure this field has no more than 10 elements."}
			} else {
				bad := map[string]any{}
				converted := []any{}
				for i, item := range list {
					raw, msg := djangoinput.TextField(item, false, 200, nil)
					if len(msg) == 0 && !djangoinput.ValidURL(raw) {
						msg = []string{"Enter a valid URL."}
					}
					if len(msg) > 0 {
						bad[fmt.Sprint(i)] = msg
					} else {
						converted = append(converted, raw)
					}
				}
				if len(bad) > 0 {
					failures["attachments"] = bad
				} else {
					values["attachments"] = converted
				}
			}
		}
	}
	if len(failures) > 0 {
		return nil, invalid(failures)
	}
	if html, ok := values["comment_html"].(string); ok && html != "" {
		sanitized, _, err := content.ValidateHTML(html)
		if err != nil {
			return nil, invalid(map[string]any{"comment_html": "HTML content is not valid"})
		}
		values["comment_html"] = sanitized
	}
	return values, nil
}
