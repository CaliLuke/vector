package property

import (
	"fmt"
	"slices"
	"strings"
)

func specificity(c Context) int {
	rank := 0
	if len(c.Projects) > 0 {
		rank++
	}
	if len(c.Types) > 0 {
		rank++
	}
	return rank
}
func overlaps(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	ids := make(map[string]bool, len(right))
	for _, id := range right {
		ids[strings.ToLower(id)] = true
	}
	for _, id := range left {
		if ids[strings.ToLower(id)] {
			return true
		}
	}
	return false
}
func uniqueIDs(ids []string) bool {
	if len(ids) > 1000 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) || seen[strings.ToLower(id)] {
			return false
		}
		seen[strings.ToLower(id)] = true
	}
	return true
}
func validateContexts(def Definition) error {
	if len(def.Contexts) > 100 {
		return fmt.Errorf("a property supports at most 100 contexts")
	}
	ids := map[string]bool{}
	for i, c := range def.Contexts {
		if !validID(c.ID) || ids[strings.ToLower(c.ID)] || strings.TrimSpace(c.Name) == "" || len(c.Name) > 255 || specificity(c) == 0 || !uniqueIDs(c.Projects) || !uniqueIDs(c.Types) {
			return fmt.Errorf("contexts need unique identities and a specific project or type scope")
		}
		ids[strings.ToLower(c.ID)] = true
		for _, other := range def.Contexts[:i] {
			if specificity(c) == specificity(other) && overlaps(c.Projects, other.Projects) && overlaps(c.Types, other.Types) {
				return fmt.Errorf("equally specific contexts cannot overlap")
			}
		}
		effective, err := applyOverrides(def.Configuration, c, def.Kind)
		if err != nil {
			return err
		}
		if err = validateConfiguration(def.Kind, effective); err != nil {
			return fmt.Errorf("context %s: %w", c.Name, err)
		}
	}
	return nil
}

func applyOverrides(base Configuration, c Context, kind Kind) (Configuration, error) {
	seen := map[string]bool{}
	for _, key := range c.Overrides {
		if seen[key] {
			return base, fmt.Errorf("duplicate context override")
		}
		seen[key] = true
		switch key {
		case "active":
			base.Active = c.Configuration.Active
		case "required":
			base.Required = c.Configuration.Required
		case "default_value":
			base.Default = c.Configuration.Default
		case "format":
			if kind != Date {
				return base, fmt.Errorf("only date presentation can be overridden")
			}
			base.Format = c.Configuration.Format
		case "options":
			if kind != Cascading {
				return base, fmt.Errorf("only cascading catalogs can be overridden")
			}
			base.Options = c.Configuration.Options
		default:
			return base, fmt.Errorf("unsupported context override %q", key)
		}
	}
	// A required override clears an inherited default as a single effective change.
	if base.Required {
		base.Default = nil
	}
	return base, nil
}

// Effective resolves one winner and inherits only from base, never from a lower
// priority context. The context identity accompanies option selections/projections.
func Effective(def Definition, project, typeID string) (Configuration, string, error) {
	winner := -1
	for i, c := range def.Contexts {
		if len(c.Projects) > 0 && !slices.ContainsFunc(c.Projects, func(id string) bool { return strings.EqualFold(id, project) }) || len(c.Types) > 0 && !slices.ContainsFunc(c.Types, func(id string) bool { return strings.EqualFold(id, typeID) }) {
			continue
		}
		if winner >= 0 && specificity(c) == specificity(def.Contexts[winner]) {
			return Configuration{}, "", fmt.Errorf("ambiguous property context")
		}
		if winner < 0 || specificity(c) > specificity(def.Contexts[winner]) {
			winner = i
		}
	}
	if winner < 0 {
		return def.Configuration, "", nil
	}
	configuration, err := applyOverrides(def.Configuration, def.Contexts[winner], def.Kind)
	return configuration, def.Contexts[winner].ID, err
}
