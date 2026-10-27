package types

import . "github.com/CaliLuke/loom/dsl"

var UserLite = Type("PlaneApiSerializersUserUserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight user serializer for minimal data transfer.\n\nProvides essential user information including names, avatar, and contact details\noptimized for member lists, assignee displays, and user references.")
	Attribute("avatar", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("avatar_url", String, func() {
		// Source fixtures include null and omission for users without a workspace.
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("first_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
})
