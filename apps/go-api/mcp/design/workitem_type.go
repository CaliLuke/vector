package design

import (
	. "github.com/CaliLuke/loom/dsl"
)

func workitemTypeTarget(action string, detail, project bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, "Omit or use empty string for workspace scope.")
	projectWorkspaceSelector()
	Required("action")
	if project {
		Required("project_id")
	}
	if detail {
		Attribute("workitem_type_id", String, func() { Format(FormatUUID) })
		Required("workitem_type_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func workitemTypeFields() {
	for _, key := range []string{"name", "description", "project_ids", "external_source", "external_id"} {
		Attribute(key, String)
	}
	Attribute("is_active", Boolean, func() { Nullable() })
}

var workitemTypeList = Type("WorkitemTypeList", func() {
	workitemTypeTarget("list", false, false)
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
})
var workitemTypeRetrieve = Type("WorkitemTypeRetrieve", func() { workitemTypeTarget("retrieve", true, false); Attribute("include_revision", Boolean) })
var workitemTypeCreate = Type("WorkitemTypeCreate", func() {
	workitemTypeTarget("create", false, false)
	workitemTypeFields()
	Required("name")
	mutationExtensions(true)
})
var workitemTypeUpdate = Type("WorkitemTypeUpdate", func() { workitemTypeTarget("update", true, false); workitemTypeFields(); mutationExtensions(true) })
var workitemTypeDelete = Type("WorkitemTypeDelete", func() { workitemTypeTarget("delete", true, false); mutationExtensions(true) })
var workitemTypeResolve = Type("WorkitemTypeResolve", func() {
	workitemTypeTarget("resolve", false, true)
	Attribute("name", String)
	Required("name")
	mutationExtensions(true)
})
var workitemTypeImport = Type("WorkitemTypeImport", func() {
	workitemTypeTarget("import_to_project", false, true)
	Attribute("workitem_type_ids", String, "Comma-separated identifiers or a JSON array.")
	Required("workitem_type_ids")
	mutationExtensions(true)
})
var workitemTypeRecord = Type("WorkitemTypeRecord", func() {
	Attribute("name", String)
	Required("name")
	for _, key := range []string{"id", "deleted_at", "created_at", "updated_at", "description", "external_source", "external_id", "created_by", "updated_by", "workspace"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	Attribute("project_ids", ArrayOf(String), func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Attribute("level", Int, func() { Nullable() })
	Required("project_ids", "logo_props", "level")
	for _, key := range []string{"is_epic", "is_default", "is_active"} {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var workitemTypes = Type("WorkitemTypes", ArrayOf(workitemTypeRecord))
var workitemTypeReceipt = Type("WorkitemTypeMutationReceipt", func() {
	Attribute("resource_id", String, "Type ID, or project ID for imports.", func() { Format(FormatUUID) })
	Attribute("revision", String, "Workspace type catalogue revision.")
	Attribute("result", workitemTypeRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var workitemTypeRequest = Type("WorkitemTypeRequest", OneOf(workitemTypeList, workitemTypeRetrieve, workitemTypeCreate, workitemTypeUpdate, workitemTypeDelete, workitemTypeResolve, workitemTypeImport), func() { Untagged() })
var workitemTypeResult = Type("WorkitemTypeResult", OneOf(workitemTypes, workitemTypeRecord, workitemTypeReceipt), func() { Untagged() })

func workitemType() {
	Method("workitem_type", func() {
		Description("Workspace-owned types and project imports. Definitions are created inactive unless explicitly activated. Workspace administrators manage definitions; project administrators import or resume them. Resolve returns an available exact name or performs the authorized native creation, activation and import. Enable workspace types in settings first.")
		Payload(workitemTypeRequest)
		Result(workitemTypeResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("workspace_owned", func() {
			Remedy(func() {
				SafeMessage("Types are managed by the workspace. Omit project_id for definition changes; use import_to_project or resolve for a project.")
			})
		})
		Error("conflict", func() {
			Remedy(func() { SafeMessage("The operation conflicts with the current work item type configuration.") })
		})
		mutationTool("workitem_type", "Work item types", "Read and manage workspace work item types and their project imports.")
	})
}
