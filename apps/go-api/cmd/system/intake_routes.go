package main

import (
	"net/http"

	intakeinboxreadssvr "github.com/CaliLuke/go-plane/gen/http/intake_inbox_reads/server"
	intakewritessvr "github.com/CaliLuke/go-plane/projectintakewrites/gen/http/project_intake_writes/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountIntakeInboxReadsProtocolRoutes(mux loomhttp.Muxer, server *intakeinboxreadssvr.Server, corsPolicy loomhttp.RuntimeCORSPolicy) {
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{"/api/workspaces/{slug}/draft-issues/", server.DraftIssueList},
		{"/api/workspaces/{slug}/draft-issues/{pk}/", server.DraftIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", server.InboxIssueList},
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", server.InboxIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/", server.InboxList},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/", server.InboxDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-issues/", server.IntakeIssueList},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/", server.IntakeIssueDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/", server.DescriptionVersionList},
		{"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/", server.DescriptionVersionDetail},
		{"/api/workspaces/{slug}/projects/{project_id}/intakes/", server.IntakeList},
		{"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/", server.IntakeDetail},
	}, corsPolicy)
}

func mountProjectIntakeWritesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.projectIntakeWrites != nil {
		server := intakewritessvr.New(pending.projectIntakeWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		intakewritessvr.Mount(mux, server)
	}
}

func mountIntakeInboxReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if pending.intakeInboxReads != nil {
		server := intakeinboxreadssvr.New(pending.intakeInboxReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(readServicesQueryPresenceMiddleware)
		server.Use(readServicesSessionMiddleware(pending.readServicesSession, pending.readServicesCookie))
		server.Use(readServicesContentNegotiationMiddleware)
		server.Use(readServicesGzipMiddleware)
		intakeinboxreadssvr.Mount(mux, server)
		mountIntakeInboxReadsProtocolRoutes(mux, server, readServicesCORSPolicy)
	}
}
