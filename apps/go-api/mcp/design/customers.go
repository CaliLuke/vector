package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func customerTarget(action string, ids ...string) { pageTarget(action, false, ids...) }
func customerFields() {
	for _, key := range []string{"name", "description_html", "email", "website_url", "domain", "stage", "contract_status", "revenue", "external_source", "external_id"} {
		Attribute(key, String)
	}
	Attribute("employees", Int64, func() { Minimum(0) })
}

var customerList = Type("CustomerList", func() {
	customerTarget("list")
	Attribute("query", String)
	pagePagination()
	Attribute("include_revision", Boolean)
})
var customerRetrieve = Type("CustomerRetrieve", func() { customerTarget("retrieve", "customer_id"); Attribute("include_revision", Boolean) })
var customerCreate = Type("CustomerCreate", func() { customerTarget("create"); customerFields(); Required("name"); mutationExtensions(false) })
var customerUpdate = Type("CustomerUpdate", func() { customerTarget("update", "customer_id"); customerFields(); mutationExtensions(true) })
var customerDelete = Type("CustomerDelete", func() {
	customerTarget("delete")
	Attribute("customer_id", String, func() { Format(FormatUUID) })
	Attribute("external_source", String)
	Attribute("external_id", String)
	mutationExtensions(true)
})
var customerListWorkitems = Type("CustomerListWorkitems", func() {
	customerTarget("list_workitems", "customer_id")
	Attribute("customer_request_id", String, func() { Format(FormatUUID) })
	Attribute("search", String)
	Attribute("include_revision", Boolean)
})
var customerManageWorkitems = Type("CustomerManageWorkitems", func() {
	customerTarget("manage_workitems", "customer_id")
	Attribute("customer_request_id", String, func() { Format(FormatUUID) })
	Attribute("link_ids", String)
	Attribute("unlink_ids", String)
	mutationExtensions(true)
})
var customerRecord = Type("CustomerRecord", func() {
	Attribute("name", String)
	Required("name")
	nullableStrings("id", "deleted_at", "logo_url", "created_at", "updated_at", "description_html", "description_stripped", "description_binary", "email", "website_url", "domain", "stage", "contract_status", "revenue", "external_source", "external_id", "archived_at", "created_by", "updated_by", "logo_asset", "workspace")
	Attribute("customer_request_count", Int, func() { Nullable() })
	Attribute("employees", Int64, func() { Nullable() })
	Attribute("description", Any, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Required("customer_request_count", "employees", "description", "logo_props")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})

func customerPageFields() {
	toolkitPageFields()
	nullableStrings("grouped_by", "sub_grouped_by", "extra_stats")
	Attribute("total_pages", Int)
	Attribute("total_results", Int)
	Required("total_pages", "total_results")
	Attribute("revision", String)
}

var customerPage = Type("CustomerPage", func() {
	customerPageFields()
	Attribute("results", ArrayOf(customerRecord))
	Meta("openapi:additionalProperties", "false")
})
var customerWorkitem = Type("CustomerWorkitem", func() {
	Attribute("id", String)
	Required("id")
	nullableStrings("name", "state_id", "priority", "project_id", "project_identifier", "created_at", "updated_at")
	Attribute("sequence_id", Int64, func() { Nullable() })
	Required("sequence_id")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var customerWorkitems = Type("CustomerWorkitems", ArrayOf(customerWorkitem))
var customerRequest = Type("CustomerRequest", OneOf(customerList, customerRetrieve, customerCreate, customerUpdate, customerDelete, customerListWorkitems, customerManageWorkitems), func() { Untagged() })
var customerResult = Type("CustomerResult", OneOf(customerRecord, customerPage, customerWorkitems, pageMutationReceipt), func() { Untagged() })

func customerAnnotations(name, title, description string) {
	readErrors()
	writeErrors()
	Error("feature_disabled", func() {
		Remedy(func() {
			SafeMessage("Customers are disabled. A workspace administrator can enable customers with workspace update_features.")
		})
	})
	Meta("mcp:annotation:readOnlyHint", "false")
	Meta("mcp:annotation:destructiveHint", "true")
	Meta("mcp:annotation:openWorldHint", "false")
	mcp.Tool(name, description, mcp.ToolTitle(title))
}
func customers() {
	Method("customer", func() {
		Description("Workspace customers. Create upserts by a complete external_source/external_id pair, otherwise exact name. Stage and contract_status accept free-form text; domain is industry and revenue is a string. Delete accepts customer_id or an external identity; an unmatched external identity succeeds. Work-item links can belong to a customer request. Unfiltered unlink removes all customer links to that item; request-filtered unlink removes only that request link. Unlink follows link. Optional receipts/revisions support retries.")
		Payload(customerRequest)
		Result(customerResult, func() { Nullable() })
		customerAnnotations("customer", "Customers", "Manage customers and their work-item associations.")
	})
	customerProperties()
	customerRequests()
}
