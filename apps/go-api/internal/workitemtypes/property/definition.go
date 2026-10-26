// Package property owns reusable property configuration and value validation.
// Database writers and projections share this contract rather than trusting UI validation.
package property

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/google/uuid"
)

type Kind string

const (
	Text      Kind = "Text"
	Number    Kind = "Number"
	Dropdown  Kind = "Dropdown"
	Boolean   Kind = "Boolean"
	Date      Kind = "Date"
	Member    Kind = "Member picker"
	Release   Kind = "Release picker"
	RichText  Kind = "Rich text"
	URL       Kind = "URL"
	Formula   Kind = "Formula"
	Cascading Kind = "Cascading"
)

type Option struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Parent    string `json:"parent"`
	Level     int    `json:"level"`
	NextLabel string `json:"next_label"`
}
type Level struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
}
type Configuration struct {
	Active     bool         `json:"active"`
	Required   bool         `json:"required"`
	Format     string       `json:"format"`
	Multiple   bool         `json:"multiple"`
	Default    any          `json:"default_value"`
	Content    string       `json:"content"`
	Options    []Option     `json:"options"`
	Levels     []Level      `json:"levels"`
	Expression string       `json:"expression"`
	Result     formula.Kind `json:"result"`
}
type Context struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Projects      []string      `json:"project_ids"`
	Types         []string      `json:"type_ids"`
	Overrides     []string      `json:"override_keys"`
	Configuration Configuration `json:"configuration"`
}
type Definition struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Icon          string        `json:"icon"`
	Kind          Kind          `json:"kind"`
	Configuration Configuration `json:"configuration"`
	Contexts      []Context     `json:"contexts"`
}

func validID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

// ValidateDefinition validates structure, including configurations that might
// only apply to a future item. Scope/foreign-ID checks belong to the caller's transaction.
func ValidateDefinition(def Definition, previous *Definition, used bool) error {
	if !validID(def.ID) {
		return fmt.Errorf("invalid property identity")
	}
	if strings.TrimSpace(def.Title) == "" || utf8.RuneCountInString(def.Title) > 255 || len(def.Description) > 10000 {
		return fmt.Errorf("enter a title of 1–255 characters and a description of at most 10000 bytes")
	}
	if def.Name == "" || len(def.Name) > 255 || !utf8.ValidString(def.Name) || strings.ContainsAny(def.Name, "{}") || strings.ContainsFunc(def.Name, unicode.IsControl) || formula.ReferenceKey(def.Name) != def.Name {
		return fmt.Errorf("invalid internal property name")
	}
	if slices.Contains([]string{"state", "due_date", "cycle", "modules", "name", "priority", "estimate_point", "start_date"}, formula.ReferenceKey(def.Title)) || slices.Contains([]string{"state", "due_date", "cycle", "modules", "name", "priority", "estimate_point", "start_date"}, def.Name) {
		return fmt.Errorf("this property name is reserved")
	}
	if previous != nil {
		if def.ID != previous.ID || def.Name != previous.Name {
			return fmt.Errorf("property identity and internal name cannot change")
		}
		if previous.Kind == Text && (def.Kind != Text || def.Configuration.Format != previous.Configuration.Format) {
			return fmt.Errorf("text format cannot change after creation")
		}
		if used && (def.Kind != previous.Kind || def.Configuration.Multiple != previous.Configuration.Multiple || def.Kind == Dropdown && !reflect.DeepEqual(def.Configuration.Options, previous.Configuration.Options)) {
			return fmt.Errorf("used properties cannot change kind, cardinality, or dropdown structure")
		}
		if def.Kind == Cascading && previous.Kind == Cascading {
			for index := 0; index < min(len(def.Configuration.Levels), len(previous.Configuration.Levels)); index++ {
				if def.Configuration.Levels[index].ID != previous.Configuration.Levels[index].ID {
					return fmt.Errorf("cascading levels can be renamed, appended, or removed from the end; they cannot be reordered")
				}
			}
		}
	}
	if err := validateConfiguration(def.Kind, def.Configuration); err != nil {
		return err
	}
	return validateContexts(def)
}

func validateConfiguration(kind Kind, c Configuration) error {
	if !slices.Contains([]Kind{Text, Number, Dropdown, Boolean, Date, Member, Release, RichText, URL, Formula, Cascading}, kind) {
		return fmt.Errorf("unknown property kind")
	}
	if c.Required && (kind == Boolean || kind == Formula || kind == Cascading || kind == Text && c.Format == "Read-only") {
		return fmt.Errorf("this property kind cannot be required")
	}
	if c.Default != nil && (c.Required || !slices.Contains([]Kind{Number, Dropdown, Member}, kind)) {
		return fmt.Errorf("defaults are supported only for optional numbers, dropdowns, and members")
	}
	if c.Multiple && !slices.Contains([]Kind{Dropdown, Member, Release}, kind) {
		return fmt.Errorf("this property kind does not support multiple selections")
	}
	if kind == Text && !slices.Contains([]string{"Single line", "Paragraph", "Read-only"}, c.Format) {
		return fmt.Errorf("choose a supported text format")
	}
	if kind == Date && !slices.Contains([]string{"Jan 15, 2025", "15/01/2025", "01/15/2025", "2025/01/15"}, c.Format) {
		return fmt.Errorf("choose a supported date format")
	}
	if c.Format != "" && kind != Text && kind != Date {
		return fmt.Errorf("this property kind has no display format")
	}
	if c.Content != "" && (kind != Text || c.Format != "Read-only") {
		return fmt.Errorf("fixed content is only supported for read-only text")
	}
	if len(c.Content) > 65536 {
		return fmt.Errorf("fixed text exceeds 65536 bytes")
	}
	if kind == Formula {
		if c.Expression == "" || len(c.Expression) > 4096 || !slices.Contains([]formula.Kind{formula.Text, formula.Number, formula.Boolean, formula.Date}, c.Result) {
			return fmt.Errorf("choose a formula expression and result kind")
		}
	} else if c.Expression != "" || c.Result != "" {
		return fmt.Errorf("only formula properties accept an expression")
	}
	if kind != Cascading && len(c.Levels) > 0 {
		return fmt.Errorf("only cascading properties have levels")
	}
	if kind != Cascading && kind != Dropdown && len(c.Options) > 0 {
		return fmt.Errorf("this property kind has no option catalog")
	}
	if kind == Dropdown || kind == Cascading {
		if err := validateOptions(kind, c); err != nil {
			return err
		}
	}
	if c.Default != nil {
		_, err := ValidateValue(kind, c, c.Default, false, func(Kind, string) bool { return true })
		return err
	}
	return nil
}

func validateOptions(kind Kind, c Configuration) error {
	if len(c.Options) > 1000 {
		return fmt.Errorf("a property supports at most 1000 options")
	}
	if kind == Dropdown && len(c.Options) == 0 {
		return fmt.Errorf("add at least one dropdown option")
	}
	if kind == Cascading {
		if len(c.Levels) < 2 || len(c.Levels) > 4 {
			return fmt.Errorf("cascading properties require two to four levels")
		}
		ids := map[string]bool{}
		optional := false
		for _, level := range c.Levels {
			if !validID(level.ID) || ids[level.ID] || strings.TrimSpace(level.Name) == "" || utf8.RuneCountInString(level.Name) > 100 {
				return fmt.Errorf("levels need unique identities and names of 1–100 characters")
			}
			ids[level.ID] = true
			if level.Required && optional {
				return fmt.Errorf("a required level requires all ancestor levels")
			}
			optional = optional || !level.Required
		}
	}
	options := map[string]Option{}
	names := map[string]bool{}
	for _, option := range c.Options {
		key := option.Parent + ":" + strings.ToLower(strings.TrimSpace(option.Name))
		_, duplicate := options[option.ID]
		if !validID(option.ID) || duplicate || names[key] || strings.TrimSpace(option.Name) == "" || utf8.RuneCountInString(option.Name) > 255 || utf8.RuneCountInString(option.NextLabel) > 100 {
			return fmt.Errorf("options require unique identities and sibling names")
		}
		options[option.ID] = option
		names[key] = true
	}
	for _, option := range c.Options {
		if kind == Cascading && (option.Level < 0 || option.Level >= len(c.Levels) || (option.Parent == "") != (option.Level == 0)) {
			return fmt.Errorf("invalid cascading option level")
		}
		seen := map[string]bool{option.ID: true}
		parent := option.Parent
		for parent != "" {
			ancestor, exists := options[parent]
			if !exists || seen[parent] {
				return fmt.Errorf("option parents must exist without cycles")
			}
			seen[parent] = true
			parent = ancestor.Parent
		}
		if kind == Cascading && option.Parent != "" && options[option.Parent].Level != option.Level-1 {
			return fmt.Errorf("cascading options must follow adjacent levels")
		}
	}
	return nil
}
