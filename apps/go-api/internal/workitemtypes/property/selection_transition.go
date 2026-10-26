package property

import "slices"

// Scope identifies where a property's configuration is resolved. Empty IDs
// represent a private draft without that destination, not a wildcard query.
type Scope struct {
	ProjectID string
	TypeID    string
}

// CascadingSelectionRequiresPreservation compares selected identities, not labels
// or complete option arrays. Independent context catalogs never share identity,
// even when administrators copied the same option IDs into both catalogs.
func CascadingSelectionRequiresPreservation(before, after Definition, previous, current Scope, value any) (bool, error) {
	if before.Kind != Cascading || value == nil {
		return false, nil
	}
	oldConfig, oldOwner, err := OptionCatalog(before, previous)
	if err != nil {
		return false, err
	}
	oldConfig = withoutRequiredLevels(oldConfig)
	selected, err := ValidateValue(Cascading, oldConfig, value, false, nil)
	if err != nil {
		return false, err
	}
	if selected == nil {
		return false, nil
	}
	config, owner, err := OptionCatalog(after, current)
	if err != nil {
		return false, err
	}
	if oldOwner != owner || after.Kind != Cascading {
		return true, nil
	}
	_, err = ValidateValue(Cascading, withoutRequiredLevels(config), value, false, nil)
	return err != nil, nil
}

// OptionCatalog distinguishes effective presentation context from option ownership.
// Inherited options retain base identity even when another context wins behavior.
func OptionCatalog(def Definition, scope Scope) (Configuration, string, error) {
	config, winner, err := Effective(def, scope.ProjectID, scope.TypeID)
	if err != nil {
		return Configuration{}, "", err
	}
	for _, context := range def.Contexts {
		if context.ID == winner && slices.Contains(context.Overrides, "options") {
			return config, winner, nil
		}
	}
	return config, "", nil
}

func withoutRequiredLevels(config Configuration) Configuration {
	config.Required = false
	config.Levels = slices.Clone(config.Levels)
	for i := range config.Levels {
		config.Levels[i].Required = false
	}
	return config
}
