package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func collectionTarget(action string, ids ...string) {
	if action != "list" && action != "create" {
		ids = append([]string{"collection_id"}, ids...)
	}
	pageTarget(action, false, ids...)
}

var collectionList = Type("CollectionList", func() { collectionTarget("list"); Attribute("include_revision", Boolean) })
var collectionRetrieve = Type("CollectionRetrieve", func() { collectionTarget("retrieve"); Attribute("include_revision", Boolean) })
var collectionCreate = Type("CollectionCreate", func() {
	collectionTarget("create")
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Required("name")
	Attribute("access", String, func() { Enum("", "public", "private") })
	mutationExtensions(false)
})
var collectionUpdate = Type("CollectionUpdate", func() {
	collectionTarget("update")
	Attribute("name", String)
	Attribute("sort_order", Float64)
	mutationExtensions(true)
})
var collectionDelete = Type("CollectionDelete", func() { collectionTarget("delete"); Attribute("archive_pages", Boolean); mutationExtensions(true) })
var collectionListPages = Type("CollectionListPages", func() {
	collectionTarget("list_pages")
	Attribute("search", String)
	Attribute("parent_id", String, func() { Format(FormatUUID) })
	pagePagination()
	Attribute("include_revision", Boolean)
})
var collectionSearchPages = Type("CollectionSearchPages", func() {
	collectionTarget("search_pages")
	Attribute("search", String)
	Attribute("include_revision", Boolean)
})
var collectionAddPages = Type("CollectionAddPages", func() {
	collectionTarget("add_pages")
	Attribute("page_ids", String, "JSON array or comma-separated page UUIDs.", func() { MinLength(1) })
	Required("page_ids")
	mutationExtensions(true)
})
var collectionRemovePage = Type("CollectionRemovePage", func() { collectionTarget("remove_page", "page_collection_id"); mutationExtensions(true) })
var collectionListMembers = Type("CollectionListMembers", func() { collectionTarget("list_members"); Attribute("include_revision", Boolean) })

func collectionMemberAccess() {
	Attribute("member_access", String, func() { Enum("view", "comment", "edit") })
	Required("member_access")
	mutationExtensions(true)
}

var collectionAddMember = Type("CollectionAddMember", func() { collectionTarget("add_member", "user_id"); collectionMemberAccess() })
var collectionUpdateMember = Type("CollectionUpdateMember", func() { collectionTarget("update_member", "collection_member_id"); collectionMemberAccess() })
var collectionRemoveMember = Type("CollectionRemoveMember", func() { collectionTarget("remove_member", "collection_member_id"); mutationExtensions(true) })
var collectionRecord = Type("CollectionRecord", func() {
	nullableStrings("id", "name", "owned_by_id", "workspace", "created_at", "updated_at", "created_by", "updated_by")
	for _, key := range []string{"access", "current_user_access"} {
		Attribute(key, Int, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"has_pages", "is_default", "is_global"} {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Required("sort_order", "logo_props")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var collectionMemberRecord = Type("CollectionMemberRecord", func() {
	nullableStrings("id", "collection", "member", "workspace", "created_at", "updated_at", "created_by", "updated_by")
	Attribute("access", Int, func() { Nullable() })
	Required("access")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var collectionPageRecord = Type("CollectionPageRecord", func() {
	nullableStrings("id", "collection", "page", "workspace", "created_at", "updated_at", "created_by", "updated_by")
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var collectionPageRow = Type("CollectionPageRow", func() {
	nullableStrings("page_collection_id", "collection_id", "parent_id")
	Attribute("sort_order", Float64)
	Attribute("page", pageRecord)
	Required("sort_order", "page")
	Meta("openapi:additionalProperties", "false")
})
var collectionPages = Type("CollectionPages", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(collectionPageRow))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var collectionRequest = Type("CollectionRequest", OneOf(collectionList, collectionRetrieve, collectionCreate, collectionUpdate, collectionDelete, collectionListPages, collectionSearchPages, collectionAddPages, collectionRemovePage, collectionListMembers, collectionAddMember, collectionUpdateMember, collectionRemoveMember), func() { Untagged() })

// Empty arrays cannot select a unique typed union branch. Use one array carrier
// until https://github.com/CaliLuke/loom/issues/1036 is fixed; native projections
// still emit the action-specific Collection, Member, Page and Search shapes.
var collectionListResult = Type("CollectionListResult", ArrayOf(Any))
var collectionResult = Type("CollectionResult", OneOf(collectionRecord, collectionMemberRecord, collectionPageRecord, collectionPages, collectionListResult, pageMutationReceipt), func() { Untagged() })

func collections() {
	Method("collection", func() {
		Description("Workspace page collections. Access public/private is fixed at creation. Member access view/comment/edit uses a separate scale. Private collections require membership. Only owners manage members or delete a collection; editors may rename and file pages. add_pages and remove_page operate on roots and their descendants, preserving private page visibility. Delete keeps pages and archives them by default; archive_pages=false leaves them unfiled. Removal requires membership IDs from the list actions. Optional receipts and revisions support retries.")
		Payload(collectionRequest)
		Result(collectionResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("collection", "Manage page collections, their pages and members.", mcp.ToolTitle("Page collections"))
	})
}
