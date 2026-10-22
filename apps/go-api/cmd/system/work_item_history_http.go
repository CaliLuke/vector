package main

import (
	"context"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	workitemhistorysvr "github.com/CaliLuke/go-plane/workitemhistory/gen/http/work_item_history/server"
	workitemhistory "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	loomhttp "github.com/CaliLuke/loom/http"
)

var workItemHistoryPaths = []string{
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/",
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/",
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/",
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/{pk}/",
	"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/",
	"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/{pk}/",
}

func newWorkItemHistoryProtocol(pattern string) readServiceProtocol {
	name := "Issue Activity Endpoint"
	switch pattern {
	case workItemHistoryPaths[1]:
		name = "Issue Meta Endpoint"
	case workItemHistoryPaths[2], workItemHistoryPaths[3]:
		name = "Issue Version Endpoint"
	case workItemHistoryPaths[4], workItemHistoryPaths[5]:
		name = "Work Item Description Version Endpoint"
	}
	protocol := newSessionServiceProtocol(pattern, "GET, HEAD, OPTIONS", name, true)
	protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	if pattern == workItemHistoryPaths[0] {
		protocol.permissionBeforeTimezone = true
	}
	return protocol
}

func mountWorkItemHistory(
	mux loomhttp.Muxer,
	endpoints *workitemhistory.Endpoints,
	cors loomhttp.RuntimeCORSPolicy,
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) {
	server := workitemhistorysvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, cors)
	server.Use(sessionServiceMiddlewareStack(options, sessions, cookie))
	workitemhistorysvr.Mount(mux, server)
	mountReadServiceProtocolRoutes(mux, []readServiceProtocolMount{
		{workItemHistoryPaths[0], server.History},
		{workItemHistoryPaths[1], server.Meta},
		{workItemHistoryPaths[2], server.Versions},
		{workItemHistoryPaths[3], server.VersionDetail},
		{workItemHistoryPaths[4], server.DescriptionVersions},
		{workItemHistoryPaths[5], server.DescriptionVersionDetail},
	}, cors)
}

type workItemHistorySessionProtocolDispatch struct {
	projectPublicationSessionProtocolDispatch
	history sessionResourceProtocolAuthorizer
}

func (dispatch workItemHistorySessionProtocolDispatch) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if dispatch.history != nil && isWorkItemHistoryProtocol(resource.Pattern) {
		return dispatch.history.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	return dispatch.projectPublicationSessionProtocolDispatch.AuthorizeSessionResourceProtocol(ctx, key, resource)
}

func isWorkItemHistoryProtocol(pattern string) bool {
	for _, candidate := range workItemHistoryPaths {
		if pattern == candidate {
			return true
		}
	}
	return false
}
