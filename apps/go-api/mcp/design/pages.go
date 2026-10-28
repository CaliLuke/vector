package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func pageTarget(action string, project bool, ids ...string) {
	Attribute("action", String, func() { Enum(action) })
	Required("action")
	projectWorkspaceSelector()
	if project {
		Attribute("project_id", String, func() { Format(FormatUUID) })
	}
	for _, id := range ids {
		Attribute(id, String, func() { Format(FormatUUID) })
		Required(id)
	}
	Meta("openapi:additionalProperties", "false")
}
func pagePagination() {
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
}

var pageList = Type("PageList", func() { pageTarget("list", true); pagePagination(); Attribute("include_revision", Boolean) })
var pageRetrieve = Type("PageRetrieve", func() { pageTarget("retrieve", true, "page_id"); Attribute("include_revision", Boolean) })
var pageCreate = Type("PageCreate", func() {
	pageTarget("create", true)
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("description_html", String, func() { MinLength(1) })
	Required("name", "description_html")
	Attribute("parent_id", String, func() { Format(FormatUUID) })
	Attribute("collection_id", String, func() { Format(FormatUUID) })
	Attribute("access", Int, func() { Enum(0, 1) })
	Attribute("color", String)
	Attribute("is_locked", Boolean)
	Attribute("external_source", String)
	Attribute("external_id", String)
	mutationExtensions(false)
})
var pageUpdate = Type("PageUpdate", func() {
	pageTarget("update", true, "page_id")
	Attribute("name", String)
	Attribute("description_html", String)
	mutationExtensions(true)
})
var pageArchive = Type("PageArchive", func() {
	pageTarget("archive", true, "page_id")
	Attribute("archive", Boolean)
	mutationExtensions(true)
})
var pageDelete = Type("PageDelete", func() { pageTarget("delete", true, "page_id"); mutationExtensions(true) })
var pageSetCollection = Type("PageSetCollection", func() { pageTarget("set_collection", false, "page_id", "collection_id"); mutationExtensions(true) })
var pageListWorkitems = Type("PageListWorkitems", func() {
	pageTarget("list_workitem_pages", false, "project_id", "workitem_id")
	Attribute("include_revision", Boolean)
})
var pageAttachWorkitem = Type("PageAttachWorkitem", func() {
	pageTarget("attach_to_workitem", false, "project_id", "workitem_id", "page_id")
	mutationExtensions(true)
})
var pageDetachWorkitem = Type("PageDetachWorkitem", func() {
	pageTarget("detach_from_workitem", false, "project_id", "workitem_id", "workitem_page_id")
	mutationExtensions(true)
})
var pageRecord = Type("PageRecord", func() {
	nullableStrings("id", "name", "description_stripped", "description_html", "description_binary", "created_at", "updated_at", "owned_by", "anchor", "workspace", "archived_at", "parent_id", "collection_id", "page_collection_id")
	Attribute("description", Any, func() { Nullable() })
	Attribute("projects", ArrayOf(String), func() { Nullable() })
	Attribute("access", Int, func() { Nullable() })
	Attribute("is_locked", Boolean, func() { Nullable() })
	Required("description", "projects", "access", "is_locked")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var pagePage = Type("PagePage", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(pageRecord))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var workitemPageLite = Type("WorkitemPageLite", func() {
	nullableStrings("id", "name", "created_at", "updated_at", "created_by")
	Attribute("is_global", Boolean, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Required("is_global", "logo_props")
	Meta("openapi:additionalProperties", "false")
})
var workitemPageRecord = Type("WorkitemPageRecord", func() {
	nullableStrings("id", "issue", "project", "workspace", "created_at", "updated_at", "created_by")
	Attribute("page", workitemPageLite, func() { Nullable() })
	Required("page")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var pageArchiveResult = Type("PageArchiveResult", func() {
	Attribute("page_id", String)
	Attribute("archived", Boolean)
	Required("page_id", "archived")
	Meta("openapi:additionalProperties", "false")
})
var pageFiledResult = Type("PageFiledResult", func() {
	Attribute("page_id", String)
	Attribute("collection_id", String)
	Attribute("page_collection_id", String)
	Required("page_id", "collection_id", "page_collection_id")
	Meta("openapi:additionalProperties", "false")
})
var pageMutationReceipt = Type("PageMutationReceipt", func() {
	Attribute("resource_id", String)
	Attribute("revision", String)
	// Nested untagged result workaround: https://github.com/CaliLuke/loom/issues/1036.
	// Replace Any with the result union when the generator supports nested unions.
	Attribute("result", Any, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var pageRequest = Type("PageRequest", OneOf(pageList, pageRetrieve, pageCreate, pageUpdate, pageArchive, pageDelete, pageSetCollection, pageListWorkitems, pageAttachWorkitem, pageDetachWorkitem), func() { Untagged() })
var workitemPageListResult = Type("WorkitemPageListResult", ArrayOf(workitemPageRecord))
var pageResult = Type("PageResult", OneOf(pageRecord, pagePage, workitemPageRecord, workitemPageListResult, pageArchiveResult, pageFiledResult, pageMutationReceipt), func() { Untagged() })

func pages() {
	Method("page", func() {
		Description("Pages at workspace or project scope. Omit project_id for workspace pages. Create accepts parent_id or collection_id, never both; children inherit their parent's collection. Updates replace the supplied HTML body and refuse locked or archived pages. Archive defaults to true; false restores. Delete requires an archived page. File or move a workspace root and its descendants with set_collection. Work-item detach uses workitem_page_id, the link ID. Optional receipts and revisions support retries.")
		Payload(pageRequest)
		Result(pageResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("page", "Read, create, edit, archive, file and link pages.", mcp.ToolTitle("Pages"))
	})
}
