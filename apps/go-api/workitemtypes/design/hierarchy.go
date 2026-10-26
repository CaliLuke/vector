package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var hierarchyCatalog = Type("HierarchyCatalog", func() {
	Extend(catalog)
	Attribute("hierarchy_enabled", Boolean)
	Attribute("default_level", Int)
	Required("hierarchy_enabled", "default_level")
})
var hierarchyPlacement = Type("HierarchyPlacement", func() {
	Attribute("type_id", String, func() { Format(FormatUUID) })
	Attribute("level", Int, func() { Minimum(0); Maximum(20) })
	Required("type_id", "level")
})
var hierarchyInput = Type("HierarchyInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("default_level", Int, func() { Minimum(0); Maximum(20) })
	Attribute("placements", ArrayOf(hierarchyPlacement))
	Attribute("review_id", String, func() { Format(FormatUUID) })
	Attribute("confirm", String, func() { MaxLength(7) })
	Required("revision", "default_level", "placements")
})
var hierarchyPair = Type("HierarchyPair", func() {
	for _, name := range []string{"child_id", "child_name", "child_project_id", "parent_id", "parent_name", "parent_project_id"} {
		Attribute(name, String)
	}
	Required("child_id", "child_name", "child_project_id", "parent_id", "parent_name", "parent_project_id")
})
var hierarchyReview = Type("HierarchyReview", func() {
	Attribute("id", String)
	Attribute("links", Int)
	Attribute("restricted_links", Int)
	Attribute("pairs", ArrayOf(hierarchyPair))
	Required("id", "links", "restricted_links", "pairs")
})

func hierarchyMethods() {
	for _, action := range []string{"HierarchyCatalog", "PreviewHierarchy", "CommitHierarchy"} {
		Method(action, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Required("slug")
				if action != "HierarchyCatalog" {
					Attribute("data", hierarchyInput)
					Required("data")
				}
			})
			if action == "PreviewHierarchy" {
				Result(hierarchyReview)
			} else {
				Result(hierarchyCatalog)
			}
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(code), failure)
			}
			HTTP(func() {
				base := "/api/workspaces/{slug}/work-item-hierarchy/"
				switch action {
				case "HierarchyCatalog":
					GET(base)
				case "PreviewHierarchy":
					POST(base + "preview/")
				case "CommitHierarchy":
					POST(base)
				}
				Param("slug")
				if action != "HierarchyCatalog" {
					Body("data")
				}
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
}
