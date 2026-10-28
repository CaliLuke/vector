package workitemtypes

import (
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func applyAPIPropertyConfiguration(def *property.Definition, changes PropertyAPIChanges) error {
	c := changes.Configuration
	if c == nil {
		return nil
	}
	if (c.Expression != nil || c.Result != nil) && def.Kind != property.Formula {
		return reject(422, "Only formula properties accept expression and result configuration.")
	}
	if c.Content != nil && (def.Kind != property.Text || def.Configuration.Format != "Read-only") {
		return reject(422, "Only read-only text properties accept fixed content.")
	}
	if (c.Levels != nil || c.Options != nil) && def.Kind != property.Cascading {
		return reject(422, "Levels and option trees require a cascading property.")
	}
	if c.Options != nil && changes.Options != nil {
		return reject(422, "Supply cascading options only in configuration.options.")
	}
	if c.Expression != nil {
		def.Configuration.Expression = *c.Expression
	}
	if c.Result != nil {
		def.Configuration.Result = *c.Result
	}
	if c.Content != nil {
		def.Configuration.Content = *c.Content
	}
	if c.Levels != nil {
		def.Configuration.Levels = slices.Clone(*c.Levels)
	}
	if c.Options != nil {
		def.Configuration.Options = slices.Clone(*c.Options)
	}
	return nil
}
