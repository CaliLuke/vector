package design

import (
	. "github.com/CaliLuke/loom/dsl"
)

func stateTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, "Omit or use an empty string for workspace scope.")
	projectWorkspaceSelector()
	Required("action")
	if detail {
		Attribute("state_id", String, func() { Format(FormatUUID) })
		Required("state_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func stateFields() {
	for _, name := range []string{"name", "color", "description", "group", "external_source", "external_id"} {
		Attribute(name, String)
	}
	Attribute("sequence", Float64, func() { Nullable() })
	Attribute("default", Boolean, func() { Nullable() })
}
func mutationExtensions(revision bool) {
	Attribute("request_id", String, func() { Format(FormatUUID) })
	if revision {
		Attribute("expected_revision", String, func() { MinLength(1) })
	}
}

var stateList = Type("StateList", func() {
	stateTarget("list", false)
	Attribute("cursor", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
})
var stateRetrieve = Type("StateRetrieve", func() { stateTarget("retrieve", true); Attribute("include_revision", Boolean) })
var stateCreate = Type("StateCreate", func() {
	stateTarget("create", false)
	stateFields()
	mutationExtensions(false)
	Required("name", "color")
})
var stateUpdate = Type("StateUpdate", func() { stateTarget("update", true); stateFields(); mutationExtensions(true) })
var stateDelete = Type("StateDelete", func() { stateTarget("delete", true); mutationExtensions(true) })
var stateRecord = Type("StateRecord", func() {
	Attribute("name", String)
	Attribute("color", String)
	Required("name", "color")
	for _, name := range []string{"id", "created_at", "updated_at", "deleted_at", "description", "group", "external_source", "external_id", "created_by", "updated_by", "project", "workspace"} {
		Attribute(name, String, func() { Nullable() })
		Required(name)
	}
	Attribute("sequence", Float64, func() { Nullable() })
	Required("sequence")
	for _, name := range []string{"is_triage", "default"} {
		Attribute(name, Boolean, func() { Nullable() })
		Required(name)
	}
	Attribute("slug", String)
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var statePage = Type("StatePage", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(stateRecord))
	Meta("openapi:additionalProperties", "false")
})
var stateReceipt = Type("StateMutationReceipt", func() {
	Attribute("state_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", stateRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("state_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var stateRequest = Type("StateRequest", OneOf(stateList, stateRetrieve, stateCreate, stateUpdate, stateDelete), func() { Untagged() })
var stateResult = Type("StateResult", OneOf(statePage, stateRecord, stateReceipt), func() { Untagged() })

func state() {
	Method("state", func() {
		Description("Workflow states within a project, or accessible project states at workspace scope. Workspace writes require governance and are unavailable in Vector.")
		Payload(stateRequest)
		Result(stateResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("conflict", func() { Remedy(func() { SafeMessage("The operation conflicts with an existing resource.") }) })
		Error("workspace_not_managed", func() {
			Remedy(func() { SafeMessage("Workspace state mutations require workspace governance. Supply a project_id.") })
		})
		mutationTool("state", "Workflow states", "Workflow states. Omit project_id for workspace reads; mutations require a project. Use an advertised action.")
	})
}
