package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func templateTarget(action string) {
	pageTarget(action, true)
	Attribute("kind", String, func() { Enum("workitem", "page", "project") })
	Required("kind")
	if action == "update" || action == "delete" {
		Attribute("template_id", String, func() { Format(FormatUUID) })
		Required("template_id")
	}
}
func templateFields() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("template_data", String, "JSON object. Updates merge top-level keys into the saved object.")
}

var templateList = Type("TemplateList", func() { templateTarget("list"); Attribute("include_revision", Boolean) })
var templateCreate = Type("TemplateCreate", func() {
	templateTarget("create")
	templateFields()
	Required("name", "template_data")
	mutationExtensions(false)
})
var templateUpdate = Type("TemplateUpdate", func() { templateTarget("update"); templateFields(); mutationExtensions(true) })
var templateDelete = Type("TemplateDelete", func() { templateTarget("delete"); mutationExtensions(true) })
var templateRecord = Type("TemplateRecord", func() {
	nullableStrings("id", "name", "created_at", "updated_at", "workspace", "project")
	Attribute("short_description", String, func() { Nullable() })
	Attribute("template_type", String, func() { Nullable() })
	Attribute("description_html", String, func() { Nullable() })
	Attribute("description", Any, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Attribute("template_data", Any, func() { Nullable() })
	Required("template_data")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var templateRecords = Type("TemplateRecords", ArrayOf(templateRecord))
var templateRequest = Type("TemplateRequest", OneOf(templateList, templateCreate, templateUpdate, templateDelete), func() { Untagged() })
var templateResult = Type("TemplateResult", OneOf(templateRecord, templateRecords, pageMutationReceipt), func() { Untagged() })

func templates() {
	Method("template", func() {
		Description("Reusable work-item, page and project templates. Workspace lists include accessible project templates; project_id restricts scope. Project-kind templates exist only at workspace scope. Create requires a JSON object; update merges top-level keys and rejects an empty update. Description is template metadata, separate from template_data. Workspace administrators manage workspace templates; project administrators manage project templates unless workspace governance owns them. Optional receipts and revisions support retries.")
		Payload(templateRequest)
		Result(templateResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("workspace_owned", func() {
			Remedy(func() {
				SafeMessage("This workspace owns its templates. Omit project_id to write at workspace scope; project templates remain readable.")
			})
		})
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("template", "List, create, edit and delete reusable templates.", mcp.ToolTitle("Templates"))
	})
}
