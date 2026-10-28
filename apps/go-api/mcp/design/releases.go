package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func releaseTarget(action, key string, required bool) {
	Attribute("action", String, func() { Enum(action) })
	Required("action")
	projectWorkspaceSelector()
	if key != "" {
		Attribute(key, String, func() { Format(FormatUUID) })
		if required {
			Required(key)
		}
	}
	Meta("openapi:additionalProperties", "false")
}
func releasePageInput() {
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
	Attribute("include_revision", Boolean)
}
func releaseFields(create bool) {
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("description_html", String)
	Attribute("status", String, func() { Enum("", "unreleased", "released", "cancelled") })
	for _, key := range []string{"release_date", "target_date", "tag_id", "lead_id"} {
		Attribute(key, String)
	}
	Attribute("is_prerelease", Boolean)
	if create {
		Required("name")
		Attribute("external_source", String)
		Attribute("external_id", String)
	}
}

var releaseList = Type("ReleaseList", func() { releaseTarget("list", "", false); releasePageInput() })
var releaseRetrieve = Type("ReleaseRetrieve", func() { releaseTarget("retrieve", "release_id", true); Attribute("include_revision", Boolean) })
var releaseCreate = Type("ReleaseCreate", func() { releaseTarget("create", "", false); releaseFields(true); mutationExtensions(true) })
var releaseUpdate = Type("ReleaseUpdate", func() { releaseTarget("update", "release_id", true); releaseFields(false); mutationExtensions(true) })
var releaseDelete = Type("ReleaseDelete", func() { releaseTarget("delete", "release_id", true); mutationExtensions(true) })
var releaseChangelogGet = Type("ReleaseChangelogGet", func() { releaseTarget("get_changelog", "release_id", true); Attribute("include_revision", Boolean) })
var releaseChangelogUpdate = Type("ReleaseChangelogUpdate", func() {
	releaseTarget("update_changelog", "release_id", true)
	Attribute("description_html", String)
	Attribute("description_stripped", String)
	mutationExtensions(true)
})
var releaseWorkitems = Type("ReleaseWorkitems", func() { releaseTarget("list_workitems", "release_id", true); releasePageInput() })
var releaseManage = Type("ReleaseManage", func() {
	releaseTarget("manage_workitems", "release_id", true)
	Attribute("add_ids", String)
	Attribute("remove_ids", String)
	mutationExtensions(true)
})
var releaseBody = Type("ReleaseBody", func() {
	nullableStrings("id", "description_html", "description_stripped")
	Attribute("description_json", Any, func() { Nullable() })
	Required("description_json")
	Meta("openapi:additionalProperties", "false")
})
var releaseRecord = Type("ReleaseRecord", func() {
	nullableStrings("id", "name", "status", "target_date", "release_date", "lead", "tag", "external_source", "external_id", "created_at", "updated_at", "created_by", "updated_by", "workspace", "start_date")
	Attribute("description", releaseBody, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Attribute("is_latest", Boolean, func() { Nullable() })
	Attribute("is_prerelease", Boolean, func() { Nullable() })
	Attribute("revision", String)
	Required("description", "logo_props", "is_latest", "is_prerelease")
	Meta("openapi:additionalProperties", "false")
})
var releaseChangelogRecord = Type("ReleaseChangelogRecord", func() {
	nullableStrings("id", "created_at", "updated_at", "created_by", "updated_by", "workspace", "release")
	Attribute("changelog", releaseBody, func() { Nullable() })
	Required("changelog")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releasePage = Type("ReleasePage", func() {
	membertypes.MemberPageFields()
	// One page carrier avoids ambiguous empty arrays of two record variants.
	// Restore a typed item union after https://github.com/CaliLuke/loom/issues/1036.
	Attribute("results", ArrayOf(Any))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releaseReceipt = Type("ReleaseMutationReceipt", func() {
	Attribute("resource_id", String)
	Attribute("revision", String)
	// Nested result union carrier: https://github.com/CaliLuke/loom/issues/1036.
	Attribute("result", Any, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var releaseRequest = Type("ReleaseRequest", OneOf(releaseList, releaseRetrieve, releaseCreate, releaseUpdate, releaseDelete, releaseChangelogGet, releaseChangelogUpdate, releaseWorkitems, releaseManage), func() { Untagged() })
var releaseResult = Type("ReleaseResult", OneOf(releasePage, releaseRecord, releaseChangelogRecord, releaseReceipt), func() { Untagged() })

func releaseTagFields(create bool) {
	Attribute("version", String, func() { MinLength(1); MaxLength(255) })
	for _, key := range []string{"description", "commit_hash", "git_tag"} {
		Attribute(key, String)
	}
	if create {
		Required("version")
	}
}

var releaseTagList = Type("ReleaseTagList", func() { releaseTarget("list", "", false); releasePageInput() })
var releaseTagRetrieve = Type("ReleaseTagRetrieve", func() { releaseTarget("retrieve", "tag_id", true); Attribute("include_revision", Boolean) })
var releaseTagCreate = Type("ReleaseTagCreate", func() { releaseTarget("create", "", false); releaseTagFields(true); mutationExtensions(true) })
var releaseTagUpdate = Type("ReleaseTagUpdate", func() { releaseTarget("update", "tag_id", true); releaseTagFields(false); mutationExtensions(true) })
var releaseTagDelete = Type("ReleaseTagDelete", func() { releaseTarget("delete", "tag_id", true); mutationExtensions(true) })
var releaseTagRecord = Type("ReleaseTagRecord", func() {
	nullableStrings("id", "version", "description", "commit_hash", "git_tag", "created_at", "updated_at", "workspace", "name", "color")
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releaseTagPage = Type("ReleaseTagPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(releaseTagRecord))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releaseTagRequest = Type("ReleaseTagRequest", OneOf(releaseTagList, releaseTagRetrieve, releaseTagCreate, releaseTagUpdate, releaseTagDelete), func() { Untagged() })
var releaseTagResult = Type("ReleaseTagResult", OneOf(releaseTagRecord, releaseTagPage, releaseReceipt), func() { Untagged() })

func releaseLabelFields(create bool) {
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("color", String)
	Attribute("sort_order", Int64)
	if create {
		Required("name")
	}
}

var releaseLabelList = Type("ReleaseLabelList", func() { releaseTarget("list", "release_id", false); releasePageInput() })
var releaseLabelCreate = Type("ReleaseLabelCreate", func() { releaseTarget("create", "", false); releaseLabelFields(true); mutationExtensions(true) })
var releaseLabelUpdate = Type("ReleaseLabelUpdate", func() { releaseTarget("update", "label_id", true); releaseLabelFields(false); mutationExtensions(true) })
var releaseLabelDelete = Type("ReleaseLabelDelete", func() { releaseTarget("delete", "label_id", true); mutationExtensions(true) })
var releaseLabelAttach = Type("ReleaseLabelAttach", func() {
	releaseTarget("attach", "release_id", true)
	Attribute("label_ids", String)
	Required("label_ids")
	mutationExtensions(true)
})
var releaseLabelDetach = Type("ReleaseLabelDetach", func() {
	releaseTarget("detach", "release_id", true)
	Attribute("label_ids", String)
	Required("label_ids")
	mutationExtensions(true)
})
var releaseLabelRecord = Type("ReleaseLabelRecord", func() {
	nullableStrings("id", "name", "color", "workspace")
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releaseLabelPage = Type("ReleaseLabelPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(releaseLabelRecord))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var releaseLabelRequest = Type("ReleaseLabelRequest", OneOf(releaseLabelList, releaseLabelCreate, releaseLabelUpdate, releaseLabelDelete, releaseLabelAttach, releaseLabelDetach), func() { Untagged() })
var releaseLabelResult = Type("ReleaseLabelResult", OneOf(releaseLabelRecord, releaseLabelPage, releaseReceipt), func() { Untagged() })

func releaseTool(name, title, description string, request, result expr.UserType) {
	Method(name, func() {
		Description(description)
		Payload(request)
		Result(result, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool(name, description, mcp.ToolTitle(title))
	})
}
func workspaceReleases() {
	releaseTool("release", "Releases", "Workspace releases, changelogs and linked work items. Workspace administrators manage the catalogue. release_date is the UI target date; target_date is stored separately. Item changes validate current project permissions. Labels and version tags have separate tools.", releaseRequest, releaseResult)
	releaseTool("release_label", "Release labels", "Workspace release-label palette. List with release_id returns attached labels. Attach and detach change one release; delete removes the palette entry. sort_order=0 is valid.", releaseLabelRequest, releaseLabelResult)
	releaseTool("release_tag", "Release tags", "Workspace release version markers, with optional Git metadata. Version strings are unique in the workspace. A release uses the tag ID, not its version text.", releaseTagRequest, releaseTagResult)
}
