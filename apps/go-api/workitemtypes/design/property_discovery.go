package design

import . "github.com/CaliLuke/loom/dsl"

var discoveryProperty = Type("DiscoveryProperty", func() {
	for _, name := range []string{"key", "property_id", "name", "icon", "kind", "result_kind", "level_id", "catalog_id"} {
		Attribute(name, String)
	}
	Attribute("multiple", Boolean)
	Attribute("groupable", Boolean)
	Attribute("operators", ArrayOf(String))
	Attribute("options", ArrayOf(propertyChoice))
	Required("key", "property_id", "name", "icon", "kind", "result_kind", "level_id", "catalog_id", "multiple", "groupable", "operators", "options")
})
var propertyDiscoveryCatalog = Type("PropertyDiscoveryCatalog", func() {
	Attribute("enabled", Boolean)
	Attribute("revision", Int64)
	Attribute("fields", ArrayOf(discoveryProperty))
	Required("enabled", "revision", "fields")
})

func propertyDiscoveryMethod() {
	discoveryMethod("DiscoveryProperties", "work-item-property-identities", propertyDiscoveryCatalog)
}
