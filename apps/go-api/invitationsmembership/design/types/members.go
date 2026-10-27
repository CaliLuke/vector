package types

import . "github.com/CaliLuke/loom/dsl"

var WorkspaceMember = Type("PlaneApiSerializersMemberWorkspacememberliteapiserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Minimal WorkspaceMember representation for paginated member pickers/directories.")
	Attribute("avatar", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("avatar_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		Format(FormatEmail)
		Meta("openapi:readOnly", "true")
	})
	Attribute("first_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_active", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", MemberRole, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("role_slug", String, func() { Nullable(); Meta("openapi:readOnly", "true") })
	Required("role_slug", "display_name", "email", "first_name", "id", "last_name")
})

var WorkspaceMembersPage = Type("RestFrameworkSerializersPaginatedworkspacememberliteResponse", func() {
	Meta("openapi:typename:canonical", "true")
	MemberPageFields()
	Attribute("results", ArrayOf(WorkspaceMember))
})

// MemberPageFields is the authored cursor envelope shared by HTTP and MCP.
// The caller supplies the results array type.
func MemberPageFields() {
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String, func() {})
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String, func() {})
	Attribute("prev_page_results", Boolean)
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
}

var MemberRole = Type("RoleC37Enum", Int, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `20` - Admin\n* `15` - Member\n* `5` - Guest")
	Meta("openapi:format", "")
	Enum(20, 15, 5)
})
