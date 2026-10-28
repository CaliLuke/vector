package customers

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	nativeproperty "github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func normalize(in *Command) error {
	allowed, contractErr := validateCommand(in)
	if contractErr != nil {
		return contractErr
	}
	fields, fieldsErr := normalizeFields(in.Fields, allowed)
	if fieldsErr != nil {
		return fieldsErr
	}
	in.Fields = fields
	if in.Action == "create" {
		name := "name"
		if in.Tool == "customer_property" {
			name = "display_name"
			if in.Fields["property_type"] == nil {
				return projectmutations.ErrInvalid
			}
		}
		if in.Fields[name] == nil {
			return projectmutations.ErrInvalid
		}
	}
	if in.Tool == "customer" && in.Action == "delete" && in.ID == uuid.Nil && (in.Fields["external_source"] == nil || in.Fields["external_id"] == nil) {
		return projectmutations.ErrInvalid
	}
	for _, ids := range []*[]uuid.UUID{&in.Add, &in.Remove} {
		if len(*ids) > 1000 {
			return projectmutations.ErrInvalid
		}
		*ids = slices.Clone(*ids)
		for _, id := range *ids {
			if id == uuid.Nil {
				return projectmutations.ErrInvalid
			}
		}
		slices.SortFunc(*ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		*ids = slices.Compact(*ids)
	}
	if len(in.Options) > 1000 || len(in.Values) > 1000 {
		return projectmutations.ErrInvalid
	}
	valuesByID := make(map[string][]string, len(in.Values))
	for id, values := range in.Values {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil {
			return projectmutations.ErrInvalid
		}
		canonicalID := parsed.String()
		if _, duplicate := valuesByID[canonicalID]; duplicate {
			return projectmutations.ErrInvalid
		}
		if len(values) > 1000 {
			return projectmutations.ErrInvalid
		}
		for _, value := range values {
			if len(value) > 65536 || strings.ContainsRune(value, 0) {
				return projectmutations.ErrInvalid
			}
		}
		valuesByID[canonicalID] = slices.Clone(values)
	}
	raw, err := json.Marshal(in.Options)
	if err != nil {
		return projectmutations.ErrInvalid
	}
	if in.Options != nil {
		if err = json.Unmarshal(raw, &in.Options); err != nil {
			return projectmutations.ErrInvalid
		}
	}
	if in.Values != nil {
		in.Values = valuesByID
	}
	return nil
}

// validateCommand checks action, identity, and collection shape before field conversion.
func validateCommand(in *Command) ([]string, error) {
	if in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return nil, projectmutations.ErrInvalid
	}
	var allowed []string
	switch in.Tool {
	case "customer":
		allowed = []string{"name", "description_html", "email", "website_url", "domain", "employees", "stage", "contract_status", "revenue", "external_source", "external_id"}
		if in.Action == "manage_workitems" {
			allowed = nil
		}
		if in.Action == "delete" {
			allowed = []string{"external_source", "external_id"}
		}
	case "customer_request":
		allowed = []string{"name", "description_html", "link"}
		if in.Action == "delete" {
			allowed = nil
		}
		if in.CustomerID == uuid.Nil {
			return nil, projectmutations.ErrInvalid
		}
	case "customer_property":
		allowed = []string{"display_name", "description", "relation_type", "is_required", "is_multi", "is_active", "default_value", "external_source", "external_id"}
		if in.Action == "create" {
			allowed = append(allowed, "property_type", "display_format")
		}
		if in.Action == "delete" {
			allowed = nil
		}
		if in.Action == "set_values" {
			allowed = nil
			if in.CustomerID == uuid.Nil || in.Values == nil {
				return nil, projectmutations.ErrInvalid
			}
		}
	default:
		return nil, projectmutations.ErrInvalid
	}
	switch in.Action {
	case "create", "update", "delete", "set_values", "manage_workitems":
	default:
		return nil, projectmutations.ErrInvalid
	}
	if in.Action == "create" && in.ID != uuid.Nil || in.Action == "update" && in.ID == uuid.Nil || in.Action == "delete" && in.Tool != "customer" && in.ID == uuid.Nil {
		return nil, projectmutations.ErrInvalid
	}
	if in.Action == "manage_workitems" && (in.Tool != "customer" || in.CustomerID == uuid.Nil || len(in.Add)+len(in.Remove) == 0) {
		return nil, projectmutations.ErrInvalid
	}
	if in.Action == "set_values" && in.Tool != "customer_property" {
		return nil, projectmutations.ErrInvalid
	}
	if len(in.Add)+len(in.Remove) > 0 && in.Action != "manage_workitems" && (in.Tool != "customer_request" || in.Action != "create" || len(in.Remove) != 0) {
		return nil, projectmutations.ErrInvalid
	}
	if in.Values != nil && in.Action != "set_values" || in.Options != nil && (in.Tool != "customer_property" || in.Action != "create" && in.Action != "update") {
		return nil, projectmutations.ErrInvalid
	}
	return allowed, nil
}

// normalizeFields owns a field-map copy and applies customer scalar and rich-text rules.
func normalizeFields(fields map[string]any, allowed []string) (map[string]any, error) {
	fields = maps.Clone(fields)
	if fields == nil {
		fields = map[string]any{}
	}
	for key, value := range fields {
		if !slices.Contains(allowed, key) {
			return nil, projectmutations.ErrInvalid
		}
		switch key {
		case "employees":
			n, ok := value.(int64)
			if !ok || n < 0 {
				return nil, projectmutations.ErrInvalid
			}
		case "is_required", "is_multi", "is_active":
			if _, ok := value.(bool); !ok {
				return nil, projectmutations.ErrInvalid
			}
		case "default_value":
			values, ok := value.([]string)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = slices.Clone(values)
		default:
			text, ok := value.(string)
			if !ok || strings.ContainsRune(text, 0) {
				return nil, projectmutations.ErrInvalid
			}
			if key != "description_html" && key != "description" && key != "link" && key != "website_url" && utf8.RuneCountInString(text) > 255 {
				return nil, projectmutations.ErrInvalid
			}
			if (key == "name" || key == "display_name") && strings.TrimSpace(text) == "" {
				return nil, projectmutations.ErrInvalid
			}
			if key == "description_html" {
				clean, _, err := content.ValidateHTML(text)
				if err != nil {
					return nil, projectmutations.ErrInvalid
				}
				fields[key] = clean
			}
			if key == "email" || key == "website_url" || key == "link" {
				kind := nativeproperty.URL
				if key == "email" {
					kind = nativeproperty.Email
				}
				if _, err := nativeproperty.ValidateValue(kind, nativeproperty.Configuration{}, text, false, nil); err != nil {
					return nil, projectmutations.ErrInvalid
				}
			}
		}
	}
	return fields, nil
}

func propertyName(display string) string {
	var out strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(display)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out.WriteRune(r)
			dash = false
		} else if (unicode.IsSpace(r) || r == '-') && out.Len() > 0 && !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
