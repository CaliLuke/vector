package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceFeatureKeys = []string{"project_grouping", "initiatives", "teams", "customers", "wiki", "pi"}
var projectFeatureKeys = []string{"modules", "cycles", "views", "pages", "intakes", "workitem_types", "epics", "parallel_cycles", "project_updates", "workflows"}
var workspaceGetFeatures = Type("WorkspaceGetFeatures", func() { pageTarget("get_features", false); Attribute("include_revision", Boolean) })
var workspaceUpdateFeatures = Type("WorkspaceUpdateFeatures", func() {
	pageTarget("update_features", false)
	for _, key := range workspaceFeatureKeys {
		Attribute(key, Boolean, func() { Nullable() })
	}
	mutationExtensions(true)
})
var projectGetFeatures = Type("ProjectGetFeatures", func() { pageTarget("get_features", false, "project_id"); Attribute("include_revision", Boolean) })
var projectUpdateFeatures = Type("ProjectUpdateFeatures", func() {
	pageTarget("update_features", false, "project_id")
	for _, key := range projectFeatureKeys {
		Attribute(key, Boolean, func() { Nullable() })
	}
	mutationExtensions(true)
})
var workspaceFeatureRecord = Type("WorkspaceFeatureRecord", func() {
	for _, key := range append(append([]string{}, workspaceFeatureKeys...), "work_item_types", "releases", "states_owned_by_workspace") {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var projectFeatureRecord = Type("ProjectFeatureRecord", func() {
	for _, key := range projectFeatureKeys {
		if key == "workitem_types" {
			key = "work_item_types"
		}
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
