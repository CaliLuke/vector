package property

import (
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"golang.org/x/net/html"
)

// ReferenceAllowed must enforce workspace/project eligibility using the same
// transaction snapshot as the write. Selecting a member never grants access.
type ReferenceAllowed func(Kind, string) bool

// ValidateValue returns a canonical value. Defaults are deliberately not applied
// here: defaults belong to creation, never to clearing a saved field.
func ValidateValue(kind Kind, c Configuration, value any, draft bool, allowed ReferenceAllowed) (any, error) {
	if kind == Formula || kind == Text && c.Format == "Read-only" {
		return nil, fmt.Errorf("this property is read-only")
	}
	if kind == Cascading {
		if draft {
			return nil, fmt.Errorf("cascading properties are unavailable on drafts")
		}
		return cascadingValue(c, value)
	}
	if value == nil {
		if kind == Boolean {
			return false, nil
		}
		if c.Required && !draft {
			return nil, fmt.Errorf("this property is required")
		}
		return nil, nil
	}
	var result any
	var err error
	if kind == Dropdown || kind == Member || kind == Issue || kind == Release {
		result, err = selectionValue(kind, c, value, allowed)
	} else {
		result, err = scalarValue(kind, c, value)
	}
	if err != nil {
		return nil, err
	}
	if result == nil && c.Required && !draft {
		return nil, fmt.Errorf("this property is required")
	}
	return result, nil
}

func scalarValue(kind Kind, c Configuration, value any) (any, error) {
	switch kind {
	case Text, Email, File, URL, Date, RichText:
		return textValue(kind, c, value)
	case Number:
		number, ok := value.(float64)
		if raw, isNumber := value.(json.Number); isNumber {
			var err error
			number, err = raw.Float64()
			ok = err == nil
		}
		if !ok || math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, fmt.Errorf("enter a finite number")
		}
		return number, nil
	case Boolean:
		boolean, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("choose true or false")
		}
		return boolean, nil
	default:
		return nil, fmt.Errorf("unknown property kind")
	}
}

func textValue(kind Kind, c Configuration, value any) (any, error) {
	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("enter a text value")
	}
	if kind != RichText && len(text) > 65536 {
		return nil, fmt.Errorf("text exceeds 65536 bytes")
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	switch kind {
	case Text:
		if c.Format == "Single line" && strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("enter a single line")
		}
	case Email:
		address, err := mail.ParseAddress(text)
		if err != nil || address.Name != "" || address.Address != text {
			return nil, fmt.Errorf("enter an email address")
		}
	case URL:
		parsed, err := url.Parse(text)
		if err != nil || parsed.Host == "" || parsed.Scheme != "https" && parsed.Scheme != "http" || parsed.User != nil {
			return nil, fmt.Errorf("enter an HTTP or HTTPS URL without credentials")
		}
	case Date:
		if _, err := ParseDate(text); err != nil {
			return nil, err
		}
	case RichText:
		clean, _, err := content.ValidateHTML(text)
		if err != nil {
			return nil, err
		}
		if !meaningfulHTML(clean) {
			return nil, nil
		}
		return clean, nil
	}
	return text, nil
}

// ParseDate accepts calendar dates and date-time property values without losing
// their time component. An input without an explicit offset uses UTC.
func ParseDate(value string) (time.Time, error) {
	for _, layout := range []string{time.DateOnly, time.DateTime, time.RFC3339Nano} {
		parsed, err := time.Parse(layout, value)
		if err == nil && parsed.Year() >= 1 && parsed.Year() <= 9999 && parsed.UTC().Year() >= 1 && parsed.UTC().Year() <= 9999 && (layout == time.RFC3339Nano || parsed.Format(layout) == value) {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("enter a date as YYYY-MM-DD or a date-time as YYYY-MM-DD HH:MM:SS or RFC3339")
}

func meaningfulHTML(clean string) bool {
	if strings.TrimSpace(content.StripTags(clean)) != "" {
		return true
	}
	tokens := html.NewTokenizer(strings.NewReader(clean))
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return false
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokens.Token()
			if token.Data != "img" && token.Data != "image-component" {
				continue
			}
			for _, attribute := range token.Attr {
				if attribute.Key == "src" && strings.TrimSpace(attribute.Val) != "" {
					return true
				}
			}
		}
	}
}

func selectionIDs(value any, multiple bool) ([]string, error) {
	if !multiple {
		id, ok := value.(string)
		if !ok || !validID(id) {
			return nil, fmt.Errorf("choose one valid selection")
		}
		return []string{id}, nil
	}
	var ids []string
	switch list := value.(type) {
	case []string:
		ids = list
	case []any:
		for _, entry := range list {
			id, ok := entry.(string)
			if !ok {
				return nil, fmt.Errorf("selections must be identifiers")
			}
			ids = append(ids, id)
		}
	default:
		return nil, fmt.Errorf("choose a list of selections")
	}
	if len(ids) > 1000 || !uniqueIDs(ids) {
		return nil, fmt.Errorf("choose unique valid selections")
	}
	return ids, nil
}

func selectionValue(kind Kind, c Configuration, value any, allowed ReferenceAllowed) (any, error) {
	ids, err := selectionIDs(value, c.Multiple)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		valid := false
		if kind == Dropdown {
			for _, option := range c.Options {
				valid = valid || option.ID == id
			}
		} else {
			valid = allowed != nil && allowed(kind, id)
		}
		if !valid {
			return nil, fmt.Errorf("selection is unavailable in this property or project")
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if c.Multiple {
		return ids, nil
	}
	return ids[0], nil
}

func cascadingValue(c Configuration, value any) (any, error) {
	var ids []string
	if value != nil {
		var err error
		ids, err = selectionIDs(value, true)
		if err != nil {
			return nil, err
		}
	}
	if len(ids) > len(c.Levels) {
		return nil, fmt.Errorf("cascading path exceeds its levels")
	}
	parent := ""
	for index, level := range c.Levels {
		var choices []Option
		for _, option := range c.Options {
			if option.Level == index && option.Parent == parent {
				choices = append(choices, option)
			}
		}
		if index >= len(ids) {
			if level.Required && (index == 0 || len(choices) > 0) {
				return nil, fmt.Errorf("choose a value for %s", level.Name)
			}
			break
		}
		found := false
		for _, option := range choices {
			found = found || option.ID == ids[index]
		}
		if !found {
			return nil, fmt.Errorf("cascading selection does not belong to its parent")
		}
		parent = ids[index]
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return ids, nil
}
