package main

import (
	"context"
	"encoding/json"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
)

func newLabelWriteProtocol(pattern, allow string) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, "Label", false)
	protocol.authoredOptions = true
	protocol.permissionBeforeTimezone = true
	return protocol
}

type labelSessionProtocolDispatch struct {
	exports            sessionResourceProtocolAuthorizer
	personal           sessionResourceProtocolAuthorizer
	search             sessionResourceProtocolAuthorizer
	analytics          sessionResourceProtocolAuthorizer
	userAccount        sessionResourceProtocolAuthorizer
	workspaceLifecycle systemapi.SessionResourceProtocolAuthorizer
	external           sessionResourceProtocolAuthorizer
	preferences        sessionResourceProtocolAuthorizer
	intakeWrites       systemapi.SessionResourceProtocolAuthorizer
	configuration      projectConfigurationProtocolAuthorizer
	webhooks           sessionResourceProtocolAuthorizer
	assets             systemapi.SessionResourceProtocolAuthorizer
	assetsMethods      map[assetsLifecycleMountedRoute]struct{}
	notifications      systemapi.SessionProtocolAuthorizer
	labels             systemapi.SessionResourceProtocolAuthorizer
	cycles             cycleLifecycleSessionProtocolAuthorizer
	membership         systemapi.MembershipLifecycleProtocolAuthorizer
	invitations        systemapi.InvitationsMembershipProtocolAuthorizer
	modules            moduleLifecycleSessionProtocolAuthorizer
	projects           projectLifecycleProtocolAuthorizer
	pages              systemapi.SessionResourceProtocolAuthorizer
	states             moduleLifecycleSessionProtocolAuthorizer
	userInsights       systemapi.SessionResourceProtocolAuthorizer
	customViews        systemapi.SessionResourceProtocolAuthorizer
}

type cycleLifecycleSessionProtocolAuthorizer interface {
	systemapi.SessionResourceProtocolAuthorizer
	systemapi.SessionProtocolAuthorizer
	OwnsSessionProtocolMethod(string, string) bool
}

type moduleLifecycleSessionProtocolAuthorizer interface {
	systemapi.SessionResourceProtocolAuthorizer
	systemapi.SessionProtocolAuthorizer
}

type cycleLifecycleProtocolResourceLookup interface {
	SessionProtocolResourceExists(context.Context, systemapi.SessionProtocolResource) (bool, error)
}

func (a labelSessionProtocolDispatch) SessionProtocolResourceExists(ctx context.Context, resource systemapi.SessionProtocolResource) (bool, error) {
	if isProjectLifecycleSessionProtocol(resource.Pattern) && a.projects != nil {
		if lookup, ok := a.projects.(interface {
			SessionProtocolResourceExists(context.Context, systemapi.SessionProtocolResource) (bool, error)
		}); ok {
			return lookup.SessionProtocolResourceExists(ctx, resource)
		}
	}

	if strings.HasPrefix(resource.Pattern, "/api/workspaces/") && (strings.Contains(resource.Pattern, "/modules") || strings.Contains(resource.Pattern, "/archived-modules")) && a.modules != nil {
		if lookup, ok := a.modules.(interface {
			SessionProtocolResourceExists(context.Context, systemapi.SessionProtocolResource) (bool, error)
		}); ok {
			return lookup.SessionProtocolResourceExists(ctx, resource)
		}
	}
	if a.cycles != nil {
		if lookup, ok := a.cycles.(cycleLifecycleProtocolResourceLookup); ok {
			return lookup.SessionProtocolResourceExists(ctx, resource)
		}
	}
	return true, nil
}

func (a labelSessionProtocolDispatch) SessionProtocolOptionsMetadata(ctx context.Context, key string, resource systemapi.SessionProtocolResource, metadata, guest json.RawMessage) (int, any, bool) {
	if isProjectStateSessionProtocol(resource.Pattern) && a.states != nil {
		if provider, ok := a.states.(sessionProtocolOptionsMetadataAuthorizer); ok {
			return provider.SessionProtocolOptionsMetadata(ctx, key, resource, metadata, guest)
		}
	}
	if strings.HasPrefix(resource.Pattern, "/api/workspaces/") && (strings.Contains(resource.Pattern, "/modules") || strings.Contains(resource.Pattern, "/archived-modules")) && a.modules != nil {
		if provider, ok := a.modules.(interface {
			SessionProtocolOptionsMetadata(context.Context, string, systemapi.SessionProtocolResource, json.RawMessage, json.RawMessage) (int, any, bool)
		}); ok {
			return provider.SessionProtocolOptionsMetadata(ctx, key, resource, metadata, guest)
		}
	}
	return 0, nil, false
}

func (a labelSessionProtocolDispatch) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	if a.userInsights != nil {
		if authorizer, ok := a.userInsights.(systemapi.SessionProtocolAuthorizer); ok {
			return authorizer.AuthorizeSessionProtocol(ctx, key)
		}
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	if a.notifications == nil {
		if a.exports != nil {
			return a.exports.AuthorizeSessionProtocol(ctx, key)
		}
		if a.personal != nil {
			return a.personal.AuthorizeSessionProtocol(ctx, key)
		}
		if a.search != nil {
			return a.search.AuthorizeSessionProtocol(ctx, key)
		}
		if a.analytics != nil {
			return a.analytics.AuthorizeSessionProtocol(ctx, key)
		}
		if a.userAccount != nil {
			return a.userAccount.AuthorizeSessionProtocol(ctx, key)
		}
		if a.external != nil {
			return a.external.AuthorizeSessionProtocol(ctx, key)
		}
		if a.preferences != nil {
			return a.preferences.AuthorizeSessionProtocol(ctx, key)
		}
		if a.configuration != nil {
			return a.configuration.AuthorizeSessionProtocol(ctx, key)
		}
		if a.webhooks != nil {
			return a.webhooks.AuthorizeSessionProtocol(ctx, key)
		}
		if a.states != nil {
			return a.states.AuthorizeSessionProtocol(ctx, key)
		}
		if a.projects != nil {
			return a.projects.AuthorizeSessionProtocol(ctx, key)
		}
		if a.modules != nil {
			return a.modules.AuthorizeSessionProtocol(ctx, key)
		}
		if a.cycles != nil {
			return a.cycles.AuthorizeSessionProtocol(ctx, key)
		}
		return 401, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	return a.notifications.AuthorizeSessionProtocol(ctx, key)
}

func (a labelSessionProtocolDispatch) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if (resource.Pattern == "/api/workspaces/" || resource.Pattern == "/api/workspaces/{slug}/") && a.workspaceLifecycle != nil {
		return a.workspaceLifecycle.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isProjectIntakeWritesProtocol(resource.Pattern) && a.intakeWrites != nil {
		return a.intakeWrites.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isCustomViewCreationProtocol(resource.Pattern) && a.customViews != nil {
		return a.customViews.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if strings.Contains(resource.Pattern, "/user-activity/") || strings.Contains(resource.Pattern, "/user-profile/") || strings.Contains(resource.Pattern, "/user-stats/") {
		if a.userInsights != nil {
			return a.userInsights.AuthorizeSessionResourceProtocol(ctx, key, resource)
		}
	}
	if a.exports != nil && (resource.Pattern == "/api/workspaces/{slug}/export-issues/" || resource.Pattern == "/api/workspaces/{slug}/export-analytics/") {
		return a.exports.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if a.personal != nil && isPersonalDashboardProtocol(resource.Pattern) {
		return a.personal.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if a.search != nil && isWorkspaceSearchProtocol(resource.Pattern) {
		return a.search.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if a.analytics != nil && isWorkspaceAnalyticsProtocol(resource.Pattern) {
		return a.analytics.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if a.userAccount != nil && isUserAccountWriteProtocol(resource.Pattern) {
		return a.userAccount.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if a.external != nil && isExternalIntegrationProtocol(resource.Pattern) {
		return a.external.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if resource.Pattern == "/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/" && a.preferences != nil {
		return a.preferences.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isProjectConfigurationSessionProtocol(resource.Pattern) && a.configuration != nil {
		return a.configuration.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isWorkspaceWebhookProtocol(resource.Pattern) && a.webhooks != nil {
		return a.webhooks.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isProjectStateSessionProtocol(resource.Pattern) && a.states != nil {
		return a.states.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isPageLifecycleSessionProtocol(resource.Pattern) && a.pages != nil {
		return a.pages.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isProjectLifecycleSessionProtocol(resource.Pattern) && a.projects != nil {
		return a.projects.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if strings.HasPrefix(resource.Pattern, "/api/workspaces/") && (strings.Contains(resource.Pattern, "/modules") || strings.Contains(resource.Pattern, "/archived-modules")) && a.modules != nil {
		return a.modules.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if isInvitationsMembershipProtocol(resource.Pattern) && a.invitations != nil {
		return a.invitations.AuthorizeInvitationsMembershipProtocol(ctx, key, resource, false)
	}
	if _, ok := assetsProtocolMetadata[resource.Pattern]; ok && a.assets != nil {
		return a.assets.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if strings.Contains(resource.Pattern, "/members/") && a.membership != nil {
		return a.membership.AuthorizeMembershipLifecycleProtocol(ctx, key, resource, false)
	}
	if strings.Contains(resource.Pattern, "/issue-labels/") && a.labels != nil {
		return a.labels.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	if strings.Contains(resource.Pattern, "/cycles/") || strings.Contains(resource.Pattern, "/archived-cycles") {
		if a.cycles != nil {
			return a.cycles.AuthorizeSessionResourceProtocol(ctx, key, resource)
		}
	}
	return a.AuthorizeSessionProtocol(ctx, key)
}

func isPageLifecycleSessionProtocol(pattern string) bool {
	for _, protocol := range readServiceProtocols {
		if protocol.pattern == pattern {
			return protocol.pageLifecycle
		}
	}
	return false
}

func (a labelSessionProtocolDispatch) OwnsSessionProtocolMethod(pattern, method string) bool {
	for _, p := range projectLifecycleProtocols {
		if p.authenticateMethod && p.pattern == pattern {
			for _, owned := range p.methods {
				if strings.EqualFold(owned, method) {
					return true
				}
			}
		}
	}

	for _, protocol := range moduleLifecycleProtocols {
		if protocol.pattern != pattern || strings.HasPrefix(pattern, "/api/v1/") {
			continue
		}
		for _, owned := range protocol.methods {
			if strings.EqualFold(owned, method) {
				return true
			}
		}
	}
	if a.cycles != nil && a.cycles.OwnsSessionProtocolMethod(pattern, method) {
		return true
	}
	_, ok := a.assetsMethods[assetsLifecycleMountedRoute{Method: method, Path: pattern}]
	return ok
}

func isExternalIntegrationProtocol(pattern string) bool {
	switch pattern {
	case "/api/unsplash/", "/api/workspaces/{slug}/ai-assistant/", "/api/workspaces/{slug}/projects/{project_id}/ai-assistant/":
		return true
	}
	return false
}
