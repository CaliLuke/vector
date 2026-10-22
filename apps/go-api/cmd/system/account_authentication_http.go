package main

import (
	"context"
	"encoding/json"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
)

var accountAuthenticationProtocolPaths = []string{
	"/auth/email-check/",
	"/auth/spaces/email-check/",
	"/auth/forgot-password/",
	"/auth/spaces/forgot-password/",
	"/auth/magic-generate/",
	"/auth/spaces/magic-generate/",
	"/auth/change-password/",
	"/auth/set-password/",
}

func accountAuthenticationProtocols() []readServiceProtocol {
	names := []string{
		"Email Check Endpoint",
		"Email Check Space Endpoint",
		"Forgot Password Endpoint",
		"Forgot Password Space Endpoint",
		"Magic Generate Endpoint",
		"Magic Generate Space Endpoint",
		"Change Password Endpoint",
		"Set User Password Endpoint",
	}
	protocols := make([]readServiceProtocol, len(accountAuthenticationProtocolPaths))
	for index, pattern := range accountAuthenticationProtocolPaths {
		protocols[index] = newSessionServiceProtocol(pattern, "POST, OPTIONS", names[index], false)
		// These DRF APIViews do not use TimezoneMixin. Their initial() path
		// still authenticates and throttles before dispatching OPTIONS or a
		// method error, but it never validates the user's timezone.
		protocols[index].throttleBeforeTimezone = true
		protocols[index].csrfForAuthenticatedSession = true
	}
	return protocols
}

type accountAuthenticationSessionProtocolDispatch struct {
	account  systemapi.SessionProtocolAuthorizer
	fallback sessionResourceProtocolAuthorizer
}

func (dispatch accountAuthenticationSessionProtocolDispatch) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	if dispatch.fallback == nil {
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
	}
	return dispatch.fallback.AuthorizeSessionProtocol(ctx, key)
}

func (dispatch accountAuthenticationSessionProtocolDispatch) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if isAccountAuthenticationProtocol(resource.Pattern) {
		if !accountAuthenticationProtocolRequiresSession(resource.Pattern) {
			return 0, nil
		}
		if dispatch.account == nil {
			return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
		}
		return dispatch.account.AuthorizeSessionProtocol(ctx, key)
	}
	if dispatch.fallback == nil {
		return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
	}
	return dispatch.fallback.AuthorizeSessionResourceProtocol(ctx, key, resource)
}

func (dispatch accountAuthenticationSessionProtocolDispatch) AuthorizeSessionResourceProtocolWithHeaders(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any, http.Header) {
	if isAccountAuthenticationProtocol(resource.Pattern) && !accountAuthenticationProtocolRequiresSession(resource.Pattern) {
		if throttler, ok := dispatch.account.(interface {
			AuthorizePublicProtocolRequest(context.Context) (int, map[string]any, http.Header)
		}); ok {
			return throttler.AuthorizePublicProtocolRequest(ctx)
		}
	}
	status, body := dispatch.AuthorizeSessionResourceProtocol(ctx, key, resource)
	return status, body, nil
}

func (dispatch accountAuthenticationSessionProtocolDispatch) SessionProtocolResourceExists(ctx context.Context, resource systemapi.SessionProtocolResource) (bool, error) {
	if lookup, ok := dispatch.fallback.(interface {
		SessionProtocolResourceExists(context.Context, systemapi.SessionProtocolResource) (bool, error)
	}); ok {
		return lookup.SessionProtocolResourceExists(ctx, resource)
	}
	return true, nil
}

func (dispatch accountAuthenticationSessionProtocolDispatch) SessionProtocolOptionsMetadata(ctx context.Context, key string, resource systemapi.SessionProtocolResource, metadata, guest json.RawMessage) (int, any, bool) {
	// These APIViews expose their own OPTIONS metadata and do not run the
	// session-backed resource metadata hooks used by read-service endpoints.
	// In particular, source OPTIONS responses do not validate the user's
	// timezone even when the request carries an authenticated session.
	if isAccountAuthenticationProtocol(resource.Pattern) {
		return 0, nil, false
	}
	if provider, ok := dispatch.fallback.(interface {
		SessionProtocolOptionsMetadata(context.Context, string, systemapi.SessionProtocolResource, json.RawMessage, json.RawMessage) (int, any, bool)
	}); ok {
		return provider.SessionProtocolOptionsMetadata(ctx, key, resource, metadata, guest)
	}
	return 0, nil, false
}

func isAccountAuthenticationProtocol(pattern string) bool {
	for _, candidate := range accountAuthenticationProtocolPaths {
		if pattern == candidate {
			return true
		}
	}
	return false
}

func accountAuthenticationProtocolRequiresSession(pattern string) bool {
	return pattern == "/auth/change-password/" || pattern == "/auth/set-password/"
}
