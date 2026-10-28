package design

import . "github.com/CaliLuke/loom/dsl"

func customerPropertyFields(create bool) {
	for _, key := range []string{"display_name", "description", "default_value", "options", "external_source", "external_id"} {
		Attribute(key, String)
	}
	for _, key := range []string{"is_required", "is_multi", "is_active"} {
		Attribute(key, Boolean)
	}
	Attribute("relation_type", String, func() { Enum("", "ISSUE", "USER") })
	if create {
		Attribute("property_type", String, func() { Enum("TEXT", "DATETIME", "DECIMAL", "BOOLEAN", "OPTION", "RELATION", "URL", "EMAIL", "FILE") })
		Attribute("display_format", String)
	}
}

var customerPropertyList = Type("CustomerPropertyList", func() { customerTarget("list"); pagePagination(); Attribute("include_revision", Boolean) })
var customerPropertyRetrieve = Type("CustomerPropertyRetrieve", func() { customerTarget("retrieve", "property_id"); Attribute("include_revision", Boolean) })
var customerPropertyCreate = Type("CustomerPropertyCreate", func() {
	customerTarget("create")
	customerPropertyFields(true)
	Required("display_name", "property_type")
	mutationExtensions(false)
})
var customerPropertyUpdate = Type("CustomerPropertyUpdate", func() {
	customerTarget("update", "property_id")
	customerPropertyFields(false)
	mutationExtensions(true)
})
var customerPropertyDelete = Type("CustomerPropertyDelete", func() { customerTarget("delete", "property_id"); mutationExtensions(true) })
var customerPropertyGetValues = Type("CustomerPropertyGetValues", func() {
	customerTarget("get_values", "customer_id")
	Attribute("property_id", String, func() { Format(FormatUUID) })
})
var customerPropertySetValues = Type("CustomerPropertySetValues", func() {
	customerTarget("set_values", "customer_id")
	Attribute("values", String, "JSON object mapping property UUIDs to string arrays.")
	Required("values")
	mutationExtensions(true)
})
var customerPropertyRequest = Type("CustomerPropertyRequest", OneOf(customerPropertyList, customerPropertyRetrieve, customerPropertyCreate, customerPropertyUpdate, customerPropertyDelete, customerPropertyGetValues, customerPropertySetValues), func() { Untagged() })

// A single root map preserves the SDK's bare UUID-to-string-array values result.
// Maps nested in untagged result branches fail generation (loom#1041). Restore the
// record/page/value-map union after https://github.com/CaliLuke/loom/issues/1041
// is released. Native projections own the bounded SDK fields for other actions.
var customerPropertyResult = Type("CustomerPropertyResult", MapOf(String, Any))

func customerProperties() {
	Method("customer_property", func() {
		Description("Customer property definitions and values. Display names are unique in the workspace. TEXT and DATETIME use default display formats unless specified at create. RELATION requires ISSUE or USER. Options are JSON arrays; existing option IDs retain identity. Defaults accept JSON string arrays or one plain string without comma splitting. Values are a JSON object of property UUID to string array; setting replaces only named properties. Inactive properties are omitted from value reads. Optional receipts/revisions support retries.")
		Payload(customerPropertyRequest)
		Result(customerPropertyResult, func() { Nullable() })
		customerAnnotations("customer_property", "Customer properties", "Manage customer property definitions and values.")
	})
}
