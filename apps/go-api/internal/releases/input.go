package releases

import (
	"encoding/json"
	"html"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/google/uuid"
)

var colorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

func normalize(in *Command) error {
	allowed, contractErr := validateCommand(in)
	if contractErr != nil {
		return contractErr
	}
	fields, fieldsErr := normalizeFields(in.Fields, allowed)
	if fieldsErr != nil {
		return fieldsErr
	}
	if in.Action == "update_changelog" {
		body, _ := fields["description_html"].(string)
		plain, _ := fields["description_stripped"].(string)
		if body == "" && plain == "" {
			return projectmutations.ErrInvalid
		}
		if body == "" {
			body = "<p>" + strings.ReplaceAll(html.EscapeString(plain), "\n", "<br/>") + "</p>"
			paragraphs := []any{}
			for _, line := range strings.Split(plain, "\n") {
				paragraph := map[string]any{"type": "paragraph"}
				if line != "" {
					paragraph["content"] = []any{map[string]any{"type": "text", "text": line}}
				}
				paragraphs = append(paragraphs, paragraph)
			}
			fields["description_json"] = map[string]any{"type": "doc", "content": paragraphs}
		}
		fields["description_html"] = body
		delete(fields, "description_stripped")
	}
	if body, ok := fields["description_html"].(string); ok {
		clean, _, err := content.ValidateHTML(body)
		if err != nil {
			return projectmutations.ErrInvalid
		}
		fields["description_html"] = clean
		if fields["description_json"] == nil {
			fields["description_json"] = map[string]any{}
		}
		key := "description"
		if in.Action == "update_changelog" {
			key = "description_stripped"
		}
		fields[key] = content.StripTags(clean)
	}
	for _, ids := range []*[]uuid.UUID{&in.Add, &in.Remove} {
		if len(*ids) > 1000 {
			return projectmutations.ErrInvalid
		}
		for _, id := range *ids {
			if id == uuid.Nil {
				return projectmutations.ErrInvalid
			}
		}
		slices.SortFunc(*ids, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
		*ids = slices.Compact(*ids)
	}
	if in.Action != "manage_workitems" && in.Action != "attach" && in.Action != "detach" && len(in.Add)+len(in.Remove) > 0 {
		return projectmutations.ErrInvalid
	}
	// Verify JSON encodability before a receipt can be admitted.
	if _, err := json.Marshal(fields); err != nil {
		return projectmutations.ErrInvalid
	}
	in.Fields = fields
	return nil
}

// validateCommand checks the tool, action, identity, and required field shape.
func validateCommand(in *Command) ([]string, error) {
	if table(in.Tool) == "" || in.ExpectedRevision != nil && *in.ExpectedRevision == "" {
		return nil, projectmutations.ErrInvalid
	}
	allowed := []string{}
	switch in.Action {
	case "create", "update":
		switch in.Tool {
		case "release":
			allowed = []string{"name", "description_html", "status", "target_date", "release_date", "tag_id", "lead_id", "is_prerelease"}
			if in.Action == "create" {
				allowed = append(allowed, "external_source", "external_id")
			}
		case "release_tag":
			allowed = []string{"version", "description", "commit_hash", "git_tag"}
		case "release_label":
			allowed = []string{"name", "color", "sort_order"}
		}
	case "delete":
	case "update_changelog":
		if in.Tool != "release" {
			return nil, projectmutations.ErrInvalid
		}
		allowed = []string{"description_html", "description_stripped"}
	case "manage_workitems":
		if in.Tool != "release" || len(in.Add)+len(in.Remove) == 0 {
			return nil, projectmutations.ErrInvalid
		}
	case "attach", "detach":
		if in.Tool != "release_label" || in.ReleaseID == uuid.Nil || len(in.Add) == 0 {
			return nil, projectmutations.ErrInvalid
		}
	default:
		return nil, projectmutations.ErrInvalid
	}
	if in.Action == "create" {
		if in.ID != uuid.Nil {
			return nil, projectmutations.ErrInvalid
		}
		key := "name"
		if in.Tool == "release_tag" {
			key = "version"
		}
		if _, ok := in.Fields[key]; !ok {
			return nil, projectmutations.ErrInvalid
		}
	} else if in.Action != "attach" && in.Action != "detach" && in.ID == uuid.Nil {
		return nil, projectmutations.ErrInvalid
	}
	return allowed, nil
}

// normalizeFields owns a field-map copy and converts scalar catalogue values.
func normalizeFields(fields map[string]any, allowed []string) (map[string]any, error) {
	fields = maps.Clone(fields)
	if fields == nil {
		fields = map[string]any{}
	}
	for key, value := range fields {
		if !slices.Contains(allowed, key) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "is_prerelease" {
			if _, ok := value.(bool); !ok {
				return nil, projectmutations.ErrInvalid
			}
			continue
		}
		if key == "sort_order" {
			if _, ok := value.(int64); !ok {
				return nil, projectmutations.ErrInvalid
			}
			continue
		}
		text, ok := value.(string)
		if !ok || strings.ContainsRune(text, 0) {
			return nil, projectmutations.ErrInvalid
		}
		if key != "description_html" && key != "description_stripped" && key != "description" && utf8.RuneCountInString(text) > 255 {
			return nil, projectmutations.ErrInvalid
		}
		if key == "description" && utf8.RuneCountInString(text) > 10000 {
			return nil, projectmutations.ErrInvalid
		}
		if key == "name" || key == "version" {
			text = strings.TrimSpace(text)
			if text == "" {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = text
		}
		if key == "status" && !slices.Contains([]string{"unreleased", "released", "cancelled"}, text) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "color" && !colorPattern.MatchString(text) {
			return nil, projectmutations.ErrInvalid
		}
		if key == "release_date" || key == "target_date" {
			parsed, err := time.Parse(time.DateOnly, text)
			if err != nil || parsed.Year() < 1 || parsed.Format(time.DateOnly) != text {
				return nil, projectmutations.ErrInvalid
			}
		}
		if key == "tag_id" || key == "lead_id" {
			id, err := uuid.Parse(text)
			if err != nil || id == uuid.Nil {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = id.String()
		}
	}
	return fields, nil
}
