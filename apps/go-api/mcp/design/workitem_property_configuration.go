package design

import . "github.com/CaliLuke/loom/dsl"

func propertyConfigurationLevelFields() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String, func() { MinLength(1); MaxLength(100) })
	Attribute("required", Boolean)
	Required("id", "name", "required")
	Meta("openapi:additionalProperties", "false")
}

func propertyConfigurationOptionFields() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("icon", String)
	Attribute("parent", String, "Parent option UUID; omit for a root option.")
	Attribute("level", Int, "Zero-based cascading level.", func() { Minimum(0); Maximum(3) })
	Attribute("next_label", String, func() { MaxLength(100) })
	Required("id", "name", "level")
	Meta("openapi:additionalProperties", "false")
}

var propertyCreateConfigurationLevel = Type("WorkitemPropertyCreateConfigurationLevel", propertyConfigurationLevelFields)
var propertyUpdateConfigurationLevel = Type("WorkitemPropertyUpdateConfigurationLevel", propertyConfigurationLevelFields)
var propertyCreateConfigurationOption = Type("WorkitemPropertyCreateConfigurationOption", propertyConfigurationOptionFields)
var propertyUpdateConfigurationOption = Type("WorkitemPropertyUpdateConfigurationOption", propertyConfigurationOptionFields)

func propertyConfigurationFields(create bool) {
	level, option := propertyUpdateConfigurationLevel, propertyUpdateConfigurationOption
	if create {
		level, option = propertyCreateConfigurationLevel, propertyCreateConfigurationOption
	}
	Description("Optional Vector configuration for formulas, cascading selections and read-only text. Omitted fields retain existing values on update; supplied arrays replace the whole catalogue. Native validation and preservation rules apply.")
	Attribute("expression", String, func() { MinLength(1); MaxLength(4096) })
	Attribute("result", String, func() { Enum("Number", "Text", "Boolean", "Date") })
	Attribute("content", String, func() { MaxLength(65536) })
	// The native validator enforces the two-level minimum only for cascading
	// properties. Restore MinLength after loom#1042 preserves optional absence:
	// https://github.com/CaliLuke/loom/issues/1042
	Attribute("levels", ArrayOf(level), func() { MaxLength(4) })
	Attribute("options", ArrayOf(option), func() { MaxLength(1000) })
	Meta("openapi:additionalProperties", "false")
}

// Merge these named types after a released generator deduplicates shared
// nested validators: https://github.com/CaliLuke/loom-mcp/issues/385.
var propertyCreateConfiguration = Type("WorkitemPropertyCreateConfiguration", func() { propertyConfigurationFields(true) })
var propertyUpdateConfiguration = Type("WorkitemPropertyUpdateConfiguration", func() { propertyConfigurationFields(false) })
