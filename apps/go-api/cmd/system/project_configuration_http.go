package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
)

//go:embed project_configuration_estimate_metadata.json
var projectConfigurationEstimateMetadata json.RawMessage

//go:embed project_configuration_estimate_point_metadata.json
var projectConfigurationEstimatePointMetadata json.RawMessage

type sessionResourceProtocolAuthorizer interface {
	systemapi.SessionProtocolAuthorizer
	systemapi.SessionResourceProtocolAuthorizer
}

type projectConfigurationProtocolAuthorizer interface {
	sessionResourceProtocolAuthorizer
	AuthorizeAPIV1LabelProtocol(context.Context, string, string, string) (int, map[string]any)
}

func newProjectConfigurationSessionProtocol(pattern, writes, reads, name string, permissionBeforeTimezone bool) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, writes, name, strings.Contains(reads, "HEAD"))
	protocol.permissionBeforeTimezone = permissionBeforeTimezone
	if reads != "" {
		protocol.methodAllow = map[string]string{http.MethodGet: reads, http.MethodHead: reads}
		protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
		protocol.permissionBeforeTimezoneExceptMethods = []string{http.MethodGet, http.MethodHead}
	}
	if pattern == "/api/workspaces/{slug}/projects/{project_id}/estimates/" {
		protocol.optionsMetadata = projectConfigurationEstimateMetadata
	}
	if pattern == "/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/" {
		protocol.optionsMetadata = projectConfigurationEstimatePointMetadata
	}
	// Django's point lookup is a str converter; conversion belongs to the view.
	if strings.HasSuffix(pattern, "/{estimate_point_id}/") {
		protocol.stringPathParameters = []string{"estimate_point_id"}
	}
	return protocol
}

func newProjectConfigurationAPIProtocol(pattern, writes, reads, name string) readServiceProtocol {
	protocol := newGeneratedAPIWriteProtocol(pattern, writes, name)
	protocol.apiAuthorization = "project_configuration"
	protocol.methodAllow = map[string]string{http.MethodGet: reads}
	protocol.passthroughMethods = []string{http.MethodGet}
	return protocol
}

func isProjectConfigurationSessionProtocol(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/estimates/",
		"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/",
		"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/project-estimates/",
		"/api/workspaces/{slug}/projects/{project_id}/bulk-create-labels/":
		return true
	}
	return false
}

func isWorkspaceWebhookProtocol(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/webhooks/", "/api/workspaces/{slug}/webhooks/{pk}/",
		"/api/workspaces/{slug}/webhooks/{pk}/regenerate/", "/api/workspaces/{slug}/webhook-logs/{webhook_id}/":
		return true
	}
	return false
}
