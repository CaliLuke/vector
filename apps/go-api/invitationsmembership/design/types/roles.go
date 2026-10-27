package types

import . "github.com/CaliLuke/loom/dsl"

var RoleDefinition = Type("MemberRoleDefinition", func() {
	Description("An existing Vector membership role in this workspace.")
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("slug", String)
	Attribute("namespace", String, func() { Enum("workspace", "project") })
	Required("id", "name", "slug", "namespace")
})

var RolesPage = Type("MemberRolesPage", func() {
	MemberPageFields()
	Attribute("results", ArrayOf(RoleDefinition))
})
