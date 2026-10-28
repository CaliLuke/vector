package property

import (
	"fmt"
	"html"
	"strings"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
)

// PreservationHTML renders the former effective value, not the destination
// configuration. Reference labels must already be scoped by the caller.
func PreservationHTML(def Definition, config Configuration, value any, labels map[string]string) (string, error) {
	rendered, err := propertyValueHTML(def, config, value, labels)
	if err != nil || rendered == "" {
		return rendered, err
	}
	return "<p><strong>" + html.EscapeString(def.Title) + "</strong></p>" + rendered, nil
}

// ValueText formats historical values using their captured effective definition.
// Reference labels must already be scoped by the caller. It does not revalidate
// old selections against today's availability or required-field rules.
func ValueText(def Definition, config Configuration, value any, labels map[string]string) (string, error) {
	rendered, err := propertyValueHTML(def, config, value, labels)
	if err != nil {
		return "", err
	}
	return readableValueText(rendered), nil
}

func propertyValueHTML(def Definition, config Configuration, value any, labels map[string]string) (string, error) {
	if value == nil {
		return "", nil
	}
	var rendered string
	switch def.Kind {
	case RichText:
		text, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("invalid saved rich text")
		}
		clean, _, err := content.ValidateHTML(text)
		if err != nil {
			return "", err
		}
		rendered = clean
	case Dropdown, Member, Issue, Release, Cascading:
		ids, err := preservedSelections(value)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			name := "Unavailable selection"
			if def.Kind == Dropdown || def.Kind == Cascading {
				for _, option := range config.Options {
					if option.ID == id {
						name = option.Name
						break
					}
				}
			} else if label := labels[id]; label != "" {
				name = label
			}
			names = append(names, html.EscapeString(name))
		}
		separator := ", "
		if def.Kind == Cascading {
			separator = " → "
		}
		rendered = "<p>" + strings.Join(names, separator) + "</p>"
	default:
		text := html.EscapeString(fmt.Sprint(value))
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "<br>")
		rendered = "<p>" + text + "</p>"
	}
	return rendered, nil
}

func preservedSelections(value any) ([]string, error) {
	switch values := value.(type) {
	case string:
		return []string{values}, nil
	case []string:
		return values, nil
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			id, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("invalid saved selection")
			}
			out = append(out, id)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid saved selection")
	}
}
