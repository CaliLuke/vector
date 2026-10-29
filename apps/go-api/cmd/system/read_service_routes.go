package main

import (
	"encoding/json"
	"net/http"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
)

type readServiceProtocol struct {
	workItem                              bool
	generatedMethods                      bool
	passthroughMethods                    []string
	protocolOnlyMethods                   []string
	throttleBeforeTimezone                bool
	optionsMetadata                       json.RawMessage
	optionsMetadataMissing                json.RawMessage
	optionsMetadataGuest                  json.RawMessage
	optionsStatus                         int
	pattern                               string
	allow                                 string
	name                                  string
	action                                string
	segments                              []string
	stringPathParameters                  []string
	derivedHead                           bool
	authenticateMethod                    bool
	authoredOptions                       bool
	permissionBeforeTimezone              bool
	permissionBeforeTimezoneExceptMethods []string
	csrfForAuthenticatedSession           bool
	pageLifecycle                         bool
	methodAllow                           map[string]string
	resolveMalformedSlashless             bool
	apiAuthorization                      string
}

type readServiceProtocolMount struct {
	pattern string
	get     http.Handler
}

func newReadServiceProtocol(pattern, allow, name, action string, derivedHead bool, apiAuthorization ...string) readServiceProtocol {
	policy := "project"
	if len(apiAuthorization) > 0 {
		policy = apiAuthorization[0]
	}
	return readServiceProtocol{
		apiAuthorization: policy,
		pattern:          pattern,
		allow:            allow,
		name:             name,
		action:           action,
		segments:         strings.Split(strings.Trim(pattern, "/"), "/"),
		derivedHead:      derivedHead,
	}
}

func newGeneratedAPIWriteProtocol(pattern, allow, name string) readServiceProtocol {
	protocol := newReadServiceProtocol(pattern, allow, name, "", false)
	protocol.generatedMethods = true
	return protocol
}

var readServiceProtocols = append(append(append(append(append(append(invitationsMembershipProtocols(), accountAuthenticationProtocols()...), assetsReadServiceProtocols()...), customViewCreationProtocols()...), workspaceAnalyticsProtocols()...), workspaceSearchProtocols()...), []readServiceProtocol{
	newSessionServiceProtocol("/api/workspaces/{slug}/export-issues/", "GET, POST, HEAD, OPTIONS", "Export Issues Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/export-analytics/", "POST, OPTIONS", "Export Analytics Endpoint", false),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/activity-graph/", "GET, HEAD, OPTIONS", "User Activity Graph Endpoint", true),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/dashboard/", "GET, HEAD, OPTIONS", "User Workspace Dashboard Endpoint", true),
	newSessionServiceProtocol("/api/users/me/workspaces/{slug}/issues-completed-graph/", "GET, HEAD, OPTIONS", "User Issue Completed Graph Endpoint", true),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/{pk}/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/{pk}/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/property-history/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/"),
	newWorkItemHistoryProtocol("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/{pk}/"),
	userAccountMeWriteProtocol(),
	newSessionServiceProtocol("/api/users/me/email/", "PATCH, OPTIONS", "User Endpoint", false),
	userAccountEmailCodeProtocol(),
	newSessionServiceProtocol("/api/unsplash/", "GET, HEAD, OPTIONS", "Unsplash Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/ai-assistant/", "POST, OPTIONS", "Workspace Gpt Integration Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/ai-assistant/", "POST, OPTIONS", "Gpt Integration Endpoint", false),
	projectMemberPreferencesProtocol(),
	newInstanceManagementProtocol("/api/instances/", "GET, PATCH, HEAD, OPTIONS", "Instance Endpoint", true),
	newInstanceManagementProtocol("/api/instances/admins/", "GET, POST, DELETE, HEAD, OPTIONS", "Instance Admin Endpoint", true),
	newInstanceManagementProtocol("/api/instances/admins/sign-in/", "POST, OPTIONS", "Instance Admin Sign In Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-up/", "POST, OPTIONS", "Instance Admin Sign Up Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-out/", "POST, OPTIONS", "Instance Admin Sign Out Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/sign-up-screen-visited/", "POST, OPTIONS", "Sign Up Screen Visited Endpoint", false),
	newInstanceManagementProtocol("/api/instances/admins/{pk}/", "GET, POST, DELETE, HEAD, OPTIONS", "Instance Admin Endpoint", true),
	newInstanceManagementProtocol("/api/instances/configurations/", "GET, PATCH, HEAD, OPTIONS", "Instance Configuration Endpoint", true),
	newInstanceManagementProtocol("/api/instances/telemetry/", "GET, HEAD, OPTIONS", "Instance Telemetry Endpoint", true),
	newInstanceManagementProtocol("/api/instances/configurations/disable-email-feature/", "DELETE, OPTIONS", "Disable Email Feature Endpoint", false),
	newInstanceManagementProtocol("/api/instances/email-credentials-check/", "POST, OPTIONS", "Email Credential Check Endpoint", false),
	newInstanceManagementProtocol("/api/instances/workspaces/", "GET, POST, HEAD, OPTIONS", "Instance Work Space Endpoint", true),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/members/", "GET, POST"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/", "GET, POST"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/", "PATCH, DELETE, GET"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/", "PATCH, DELETE, GET"),
	newMembershipAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/project-members-lite/", "GET"),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/", "POST, OPTIONS", true),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/{pk}/", "PATCH, DELETE, OPTIONS", true),
	newMembershipSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/members/leave/", "POST, OPTIONS", false),

	newLabelWriteProtocol("/api/workspaces/{slug}/projects/{project_id}/issue-labels/", "GET, POST, HEAD, OPTIONS"),
	newLabelWriteProtocol("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"),
	newProjectConfigurationAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/labels/", "POST", "GET, POST", "Label List Create API Endpoint"),
	newProjectConfigurationAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/", "PATCH, DELETE", "GET, PATCH, DELETE", "Label Detail API Endpoint"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/", "POST, OPTIONS", "GET, POST, HEAD, OPTIONS"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", "PATCH, DELETE, OPTIONS", "GET, PATCH, DELETE, HEAD, OPTIONS"),
	newProjectStateSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/mark-default/", "POST, OPTIONS", ""),
	newProjectStateAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/states/", "GET, POST", "State List Create API Endpoint"),
	newProjectStateAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/", "GET, PATCH, DELETE", "State Detail API Endpoint"),
	newReadServiceProtocol("/api/v1/users/me/", "GET", "User Endpoint", "", false, "identity"),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/read/", "POST, DELETE, OPTIONS", "Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/{pk}/archive/", "POST, DELETE, OPTIONS", "Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/mark-all-read/", "POST, OPTIONS", "Mark All Read Notification", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/", "GET, HEAD, OPTIONS", "Notification", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/users/notifications/unread/", "GET, HEAD, OPTIONS", "Unread Notification Endpoint", true),
	notificationRemainingDetailProtocol(),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/", "GET", "Cycle API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles-lite/", "GET", "Cycle API Endpoint", "", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/", "POST", true),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", "POST", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-cycles/{cycle_id}/unarchive/", "DELETE", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "GET, POST", "Cycle API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "GET, DELETE", "Cycle API Endpoint", "", false),
	newCycleAPIProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", "PATCH, DELETE", true),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/", "GET", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules-lite/", "GET", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/", "GET, POST", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/", "GET, POST", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "GET, PATCH, DELETE", "Module API Endpoint", "", false),
	newReadServiceProtocol("/api/users/me/activities/", "GET, HEAD, OPTIONS", "User Activity Endpoint", "", true),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/", "GET", "Issue Activity List API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/{pk}/", "GET", "Issue Activity Detail API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/", "GET", "Issue Activity List API Endpoint", "", false),
	newReadServiceProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/{pk}/", "GET", "Issue Activity Detail API Endpoint", "", false),
	newReadServiceProtocol("/api/users/me/workspaces/", "GET, HEAD, OPTIONS", "User Work Spaces Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/", "GET, POST, HEAD, OPTIONS", "Work Space", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Work Space", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/project-stats/", "GET, HEAD, OPTIONS", "Project Stats Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/", "GET, POST, HEAD, OPTIONS", "Project", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/details/", "GET, HEAD, OPTIONS", "Project", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Project", http.MethodPut, true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-activity/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Activity Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-activity/{user_id}/export/", "POST, OPTIONS", "Export Workspace User Activity Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-profile/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Profile Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/user-stats/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Profile Stats Endpoint", true),

	newReadServiceProtocol("/api/workspaces/{slug}/cycles/", "GET, HEAD, OPTIONS", "Workspace Cycles Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/", "POST, OPTIONS", "Cycle", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/date-check/", "POST, OPTIONS", "Cycle Date Check Endpoint", false),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/", "GET, HEAD, OPTIONS", "Cycle Analytics Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/", "POST, DELETE, OPTIONS", "Cycle Archive Unarchive Endpoint", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "GET, POST, HEAD, OPTIONS", "Cycle Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Cycle Issue", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/", "GET, HEAD, OPTIONS", "Cycle Progress Endpoint", "", true),
	newCycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/", "PUT, PATCH, DELETE, OPTIONS", "Cycle", true),

	newReadServiceProtocol("/api/workspaces/{slug}/modules/", "GET, HEAD, OPTIONS", "Workspace Modules Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/", "GET, POST, HEAD, OPTIONS", "Module", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/", "GET, POST, DELETE, HEAD, OPTIONS", "Module Archive Unarchive Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/", "GET, POST, HEAD, OPTIONS", "Module Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module Issue", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/", "GET, POST, HEAD, OPTIONS", "Module Link", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module Link", http.MethodPut, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Module", http.MethodPut, true),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/", "GET, POST, HEAD, OPTIONS", "Sticky List Create API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Sticky Detail API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies.json/", "GET, POST, HEAD, OPTIONS", "Sticky List Create API Endpoint"),
	newAPIV1StickyProtocol("/api/v1/workspaces/{slug}/stickies/{pk}.json/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Sticky Detail API Endpoint"),

	newReadServiceProtocol("/api/workspaces/{slug}/estimates/", "GET, HEAD, OPTIONS", "Workspace Estimates Endpoint", "", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/", "POST, OPTIONS", "GET, POST, HEAD, OPTIONS", "Bulk Estimate Point Endpoint", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/", "PATCH, DELETE, OPTIONS", "GET, PATCH, DELETE, HEAD, OPTIONS", "Bulk Estimate Point Endpoint", true),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/", "POST, OPTIONS", "", "Estimate Point Endpoint", false),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/", "PATCH, DELETE, OPTIONS", "", "Estimate Point Endpoint", false),
	newProjectConfigurationSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/bulk-create-labels/", "POST, OPTIONS", "", "Bulk Create Issue Labels Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/project-estimates/", "GET, HEAD, OPTIONS", "Project Estimate Point Endpoint", true),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhooks/", "GET, POST, PATCH, DELETE, HEAD, OPTIONS", "Webhook Endpoint", true),
	newWebhookDetailProtocol(),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhooks/{pk}/regenerate/", "POST, OPTIONS", "Webhook Secret Regenerate Endpoint", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/webhook-logs/{webhook_id}/", "GET, HEAD, OPTIONS", "Webhook Logs Endpoint", true),

	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages-summary/", "GET, HEAD, OPTIONS", "Page", "", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/", "GET, POST, HEAD, OPTIONS", "Page", "collection", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Page", "detail", true),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", "GET, PATCH, HEAD, OPTIONS", "Pages Description", "description", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/", "POST, OPTIONS", "Page", "access", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/", "POST, DELETE, OPTIONS", "Page", "archive", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/", "POST, OPTIONS", "Page Duplicate Endpoint", "duplicate", false),
	pageLifecycleSessionProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/", "POST, DELETE, OPTIONS", "Page", "lock", false),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/", "GET, HEAD, OPTIONS", "Page Version Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/", "GET, HEAD, OPTIONS", "Page Version Endpoint", "", true),

	newReadServiceProtocol("/api/workspaces/{slug}/draft-issues/", "GET, POST, HEAD, OPTIONS", "Workspace Draft Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/draft-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Workspace Draft Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", "GET, POST, HEAD, OPTIONS", "Intake Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake Issue", "", true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/inboxes/", "GET, POST, HEAD, OPTIONS", http.MethodPost),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", ""),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/", "GET, POST, HEAD, OPTIONS", "Intake Issue", http.MethodPost, true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Intake Issue", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/", "GET, HEAD, OPTIONS", "Intake Work Item Description Version Endpoint", "", true),
	newReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/", "GET, HEAD, OPTIONS", "Intake Work Item Description Version Endpoint", "", true),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/intakes/", "GET, POST, HEAD, OPTIONS", http.MethodPost),
	newIntakeMixedProtocol("/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", ""),

	newProjectPublicationProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/", "GET, POST, HEAD, OPTIONS"),
	newProjectPublicationProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS"),
}...)

func resolveReadServiceProtocol(requestPath string) (readServiceProtocol, bool) {
	// DRF format aliases resolve with or without a trailing slash. Resolve the
	// same policy at both the outer dispatcher and its authorization handler.
	if !strings.HasSuffix(requestPath, "/") {
		if format, _, valid := apiV1StickyFormatSuffix(requestPath); valid && format == "json" {
			requestPath += "/"
		}
	}
	if !strings.HasPrefix(requestPath, "/") || !strings.HasSuffix(requestPath, "/") {
		return readServiceProtocol{}, false
	}
	trimmed := strings.TrimSuffix(strings.TrimPrefix(requestPath, "/"), "/")
	if trimmed == "" || strings.HasSuffix(trimmed, "/") {
		return readServiceProtocol{}, false
	}
	segments := strings.Split(trimmed, "/")
	for _, protocol := range readServiceProtocols {
		if readServiceProtocolMatchesSegments(protocol, segments) {
			return protocol, true
		}
	}
	return readServiceProtocol{}, false
}

func readServiceProtocolMatchesPath(requestPath string, protocol readServiceProtocol) bool {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(requestPath, "/"), "/")
	if trimmed == "" || strings.HasSuffix(trimmed, "/") {
		return false
	}
	return readServiceProtocolMatchesSegments(protocol, strings.Split(trimmed, "/"))
}

func readServiceProtocolMatchesSegments(protocol readServiceProtocol, segments []string) bool {
	if len(segments) != len(protocol.segments) {
		return false
	}
	if protocol.pattern == "/api/instances/admins/{pk}/" && instanceAdminLiteralSegment(segments[len(segments)-1]) {
		return false
	}
	for index, patternSegment := range protocol.segments {
		if strings.HasPrefix(protocol.pattern, "/api/v1/workspaces/{slug}/stickies/") && patternSegment == "{pk}" && strings.Contains(segments[index], ".") {
			return false
		}
		if strings.HasPrefix(protocol.pattern, "/api/v1/workspaces/{slug}/stickies/") && patternSegment == "{pk}.json" {
			pk, suffix, found := strings.Cut(segments[index], ".")
			if !found || pk == "" || suffix != "json" {
				return false
			}
		}
		if strings.HasPrefix(patternSegment, "{") {
			if segments[index] == "" {
				return false
			}
			if strings.HasSuffix(patternSegment, ".json") && !strings.HasSuffix(segments[index], ".json") {
				return false
			}
			continue
		}
		if segments[index] != patternSegment {
			return false
		}
	}
	return true
}

func mustReadServiceProtocol(pattern string) readServiceProtocol {
	for _, protocol := range readServiceProtocols {
		if protocol.pattern == pattern {
			return protocol
		}
	}
	panic("missing read-service protocol for " + pattern)
}

func mountReadServiceProtocolRoutes(
	mux loomhttp.Muxer,
	routes []readServiceProtocolMount,
	corsPolicy loomhttp.RuntimeCORSPolicy,
) {
	for _, route := range routes {
		protocol := mustReadServiceProtocol(route.pattern)
		if protocol.derivedHead {
			loomhttp.MountDerivedHead(mux, protocol.pattern, corsPolicy.Handler(loomhttp.AsHandlerFunc(route.get)))
		}
	}
}

func withReadServicePathValues(request *http.Request, protocol readServiceProtocol) *http.Request {
	segments := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	clone := request.Clone(request.Context())
	for index, patternSegment := range protocol.segments {
		if strings.HasPrefix(patternSegment, "{") {
			end := strings.IndexByte(patternSegment, '}')
			if end > 1 {
				value := segments[index]
				suffix := patternSegment[end+1:]
				value = strings.TrimSuffix(value, suffix)
				clone.SetPathValue(patternSegment[1:end], value)
			}
		}
	}
	return clone
}
