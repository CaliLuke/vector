package design

import . "github.com/CaliLuke/loom/dsl"

func customerRequestTarget(action string) {
	ids := []string{"customer_id"}
	if action == "retrieve" || action == "update" || action == "delete" {
		ids = append(ids, "request_id")
	}
	customerTarget(action, ids...)
}
func customerRequestExtensions() {
	Attribute("mutation_id", String, "Optional Vector retry UUID. request_id identifies the customer request.", func() { Format(FormatUUID) })
	Attribute("expected_revision", String, func() { MinLength(1); MaxLength(128) })
}
func customerRequestFields() {
	Attribute("name", String)
	Attribute("description_html", String)
	Attribute("link", String)
}

var customerRequestList = Type("CustomerRequestList", func() {
	customerRequestTarget("list")
	Attribute("query", String)
	pagePagination()
	Attribute("include_revision", Boolean)
})
var customerRequestRetrieve = Type("CustomerRequestRetrieve", func() { customerRequestTarget("retrieve"); Attribute("include_revision", Boolean) })
var customerRequestCreate = Type("CustomerRequestCreate", func() {
	customerRequestTarget("create")
	customerRequestFields()
	Attribute("workitem_ids", String)
	Required("name")
	customerRequestExtensions()
})
var customerRequestUpdate = Type("CustomerRequestUpdate", func() { customerRequestTarget("update"); customerRequestFields(); customerRequestExtensions() })
var customerRequestDelete = Type("CustomerRequestDelete", func() { customerRequestTarget("delete"); customerRequestExtensions() })
var customerRequestRecord = Type("CustomerRequestRecord", func() {
	Attribute("name", String)
	Required("name")
	nullableStrings("id", "description_html", "link", "created_at")
	Attribute("description", Any, func() { Nullable() })
	Attribute("work_item_ids", ArrayOf(String), func() { Nullable() })
	Attribute("attachment_count", Int, func() { Nullable() })
	Required("description", "work_item_ids", "attachment_count")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var customerRequestPage = Type("CustomerRequestPage", func() {
	customerPageFields()
	Attribute("results", ArrayOf(customerRequestRecord))
	Meta("openapi:additionalProperties", "false")
})
var customerRequestReceipt = Type("CustomerRequestReceipt", func() {
	Attribute("resource_id", String)
	Attribute("revision", String)
	Attribute("result", customerRequestRecord, func() { Nullable() })
	Attribute("mutation_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var customerRequestRequest = Type("CustomerRequestRequest", OneOf(customerRequestList, customerRequestRetrieve, customerRequestCreate, customerRequestUpdate, customerRequestDelete), func() { Untagged() })
var customerRequestResult = Type("CustomerRequestResult", OneOf(customerRequestRecord, customerRequestPage, customerRequestReceipt), func() { Untagged() })

func customerRequests() {
	Method("customer_request", func() {
		Description("Customer requests. request_id identifies a request under customer_id. Optional mutation_id is the separate Vector retry identity. Create can link workitem_ids; use customer manage_workitems for subsequent link changes. HTML is the request body; link is a URL. Work-item IDs are write-only and returned as null in the SDK record; list links using customer list_workitems. Delete removes request links and keeps work items.")
		Payload(customerRequestRequest)
		Result(customerRequestResult, func() { Nullable() })
		customerAnnotations("customer_request", "Customer requests", "Manage requests raised by customers.")
	})
}
