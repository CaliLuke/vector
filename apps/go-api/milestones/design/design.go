// Package design defines the native milestone HTTP contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/milestones/design --output ..
package design

import (
	pagetypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	milestonetypes "github.com/CaliLuke/go-plane/milestones/design/types"
	. "github.com/CaliLuke/loom/dsl"
)

var key = APIKeySecurity("MilestoneAPIKey")
var _ = API("Vector Milestones", func() {
	Title("Vector Milestones")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	HTTP(func() { Consumes("application/json"); Produces("application/json") })
})

var fields = Type("MilestoneFields", func() {
	for _, field := range []string{"title", "target_date", "external_source", "external_id"} {
		Attribute(field, String, func() { Nullable() })
	}
	milestonetypes.MutationExtensions()
	Meta("openapi:additionalProperties", "false")
})
var createFields = Type("MilestoneCreateFields", func() { Extend(fields); Required("title"); Meta("openapi:additionalProperties", "false") })
var issues = Type("MilestoneIssues", func() {
	Attribute("issues", ArrayOf(String, func() { Format(FormatUUID) }))
	milestonetypes.MutationExtensions()
	Required("issues")
	Meta("openapi:additionalProperties", "false")
})
var recordPage = Type("MilestonePage", func() {
	pagetypes.MemberPageFields()
	Attribute("results", ArrayOf(milestonetypes.Record))
	Meta("openapi:additionalProperties", "false")
})
var linkPage = Type("MilestoneLinkPage", func() {
	pagetypes.MemberPageFields()
	Attribute("results", ArrayOf(milestonetypes.Association))
	Meta("openapi:additionalProperties", "false")
})
var writeResult = Type("MilestoneWriteResult", OneOf(milestonetypes.Record, milestonetypes.MutationReceipt), func() { Untagged() })
var emptyResult = Type("MilestoneEmptyResult", func() {
	Attribute("outcome", String)
	Attribute("receipt", milestonetypes.MutationReceipt)
})
var failure = Type("MilestoneError", func() {
	Attribute("name", String, func() { Meta("struct:error:name") })
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("name", "error")
})

const base = "/api/v1/workspaces/{slug}/projects/{project_id}/milestones/"

var _ = Service("Milestones", func() {
	HTTP(func() { RuntimeCORS() })
	for _, name := range []string{"unauthorized", "forbidden", "invalid_input", "not_found", "conflict", "request_conflict", "revision_conflict", "indeterminate", "rate_limited", "internal_error"} {
		Error(name, failure)
	}
	Method("list", func() {
		Security(key)
		Payload(func() { scopeFields(); pageFields() })
		Result(recordPage)
		HTTP(func() { GET(base); scopeBindings(); pageBindings(); Response(StatusOK); errorResponses() })
	})
	Method("retrieve", func() {
		Security(key)
		Payload(func() { scopeFields(); targetField(); Attribute("include_revision", Boolean) })
		Result(milestonetypes.Record)
		HTTP(func() {
			GET(base + "{milestone_id}/")
			scopeBindings()
			Param("milestone_id")
			Param("include_revision")
			Response(StatusOK)
			errorResponses()
		})
	})
	Method("list_workitems", func() {
		Security(key)
		Payload(func() { scopeFields(); targetField(); pageFields() })
		Result(linkPage)
		HTTP(func() {
			GET(base + "{milestone_id}/work-items/")
			scopeBindings()
			Param("milestone_id")
			pageBindings()
			Response(StatusOK)
			errorResponses()
		})
	})
	recordWrite("create", createFields, StatusCreated)
	recordWrite("update", fields, StatusOK)
	Method("delete", func() {
		Security(key)
		Payload(func() { scopeFields(); targetField(); milestonetypes.MutationExtensions() })
		Result(emptyResult)
		HTTP(func() {
			DELETE(base + "{milestone_id}/")
			scopeBindings()
			Param("milestone_id")
			Param("request_id")
			Param("expected_revision")
			emptyResponses()
			errorResponses()
		})
	})
	associationWrite("add_workitems")
	associationWrite("remove_workitems")
})

func scopeFields() {
	Attribute("slug", String, func() { MaxLength(255) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	APIKey("MilestoneAPIKey", "credential", String)
	Required("slug", "project_id")
}
func targetField() {
	Attribute("milestone_id", String, func() { Format(FormatUUID) })
	Required("milestone_id")
}
func pageFields()    { Attribute("cursor", String); Attribute("per_page", String) }
func scopeBindings() { Param("slug"); Param("project_id"); Header("credential:X-Api-Key") }
func pageBindings()  { Param("cursor"); Param("per_page") }
func recordWrite(name string, body any, status int) {
	Method(name, func() {
		Security(key)
		Payload(func() {
			scopeFields()
			if name == "update" {
				targetField()
			}
			Attribute("body", body)
			Required("body")
		})
		Result(writeResult)
		HTTP(func() {
			path := base
			if name == "update" {
				path += "{milestone_id}/"
			}
			if name == "create" {
				POST(path)
			} else {
				PATCH(path)
			}
			scopeBindings()
			if name == "update" {
				Param("milestone_id")
			}
			Body("body")
			Response(status)
			errorResponses()
		})
	})
}
func associationWrite(name string) {
	Method(name, func() {
		Security(key)
		Payload(func() { scopeFields(); targetField(); Attribute("body", issues); Required("body") })
		Result(emptyResult)
		HTTP(func() {
			if name == "add_workitems" {
				POST(base + "{milestone_id}/work-items/")
			} else {
				DELETE(base + "{milestone_id}/work-items/")
			}
			scopeBindings()
			Param("milestone_id")
			Body("body")
			emptyResponses()
			errorResponses()
		})
	})
}
func emptyResponses() {
	Response(StatusOK, func() { Tag("outcome", "extended"); Body("receipt") })
	Response(StatusNoContent, func() { Body(Empty) })
}
func errorResponses() {
	for _, item := range []struct {
		name   string
		status int
	}{
		{"unauthorized", 401}, {"forbidden", 403}, {"invalid_input", 400}, {"not_found", 404}, {"conflict", 409}, {"request_conflict", 409}, {"revision_conflict", 409}, {"indeterminate", 503}, {"rate_limited", 429}, {"internal_error", 500},
	} {
		Response(item.name, item.status)
	}
}
