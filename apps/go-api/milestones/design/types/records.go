// Package types defines milestone fields shared by native HTTP and MCP designs.
package types

import . "github.com/CaliLuke/loom/dsl"

func RecordFields() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("title", String)
	Attribute("target_date", String, func() { Nullable(); Format(FormatDate) })
	Attribute("external_source", String, func() { Nullable() })
	Attribute("external_id", String, func() { Nullable() })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Required("id", "title", "target_date", "external_source", "external_id", "created_at", "updated_at")
}

var Record = Type("MilestoneRecord", func() {
	RecordFields()
	Attribute("revision", String, "Present only when include_revision is requested.")
	Meta("openapi:additionalProperties", "false")
})

var Association = Type("MilestoneLink", func() {
	for _, field := range []string{"id", "issue", "milestone"} {
		Attribute(field, String, func() { Format(FormatUUID) })
	}
	Required("id", "issue", "milestone")
	Meta("openapi:additionalProperties", "false")
})

var MutationReceipt = Type("MilestoneMutationReceipt", func() {
	Attribute("milestone_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", Record, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String, func() { Format(FormatDateTime) })
	Attribute("replayed", Boolean)
	Required("milestone_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})

func MutationExtensions() {
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("expected_revision", String, func() { MinLength(1); MaxLength(128) })
}
