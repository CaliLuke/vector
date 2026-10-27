package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	server "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed invitations_membership_metadata.json
var invitationsMembershipMetadataJSON []byte

var invitationsMembershipMetadata = func() map[string]struct {
	Fixture  string          `json:"fixture"`
	Metadata json.RawMessage `json:"metadata"`
} {
	var result map[string]struct {
		Fixture  string          `json:"fixture"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(invitationsMembershipMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

// Invitation ViewSets support HEAD and OPTIONS; the API roster views explicitly
// restrict their http_method_names to GET. Keep that distinction in the shared
// protocol boundary, including authentication before unsupported methods.
func invitationsMembershipProtocols() []readServiceProtocol {
	type route struct{ path, allow, name string }
	routes := []route{
		{"/api/users/me/workspaces/invitations/", "GET, POST, HEAD, OPTIONS", "User Workspace Invitations"},
		{"/api/users/me/workspaces/{slug}/projects/invitations/", "GET, POST, HEAD, OPTIONS", "User Project Invitations"},
		{"/api/v1/workspaces/{slug}/invitations/", "GET, POST, HEAD, OPTIONS", "Workspace Invite"},
		{"/api/v1/workspaces/{slug}/invitations/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Workspace Invite"},
		{"/api/v1/workspaces/{slug}/members-lite/", "GET", "Workspace Members Lite Endpoint"},
		{"/api/v1/workspaces/{slug}/roles/", "GET", "Member Roles"},
		{"/api/v1/workspaces/{slug}/roles/{role_id}/", "GET", "Member Role"},
		{"/api/v1/workspaces/{slug}/members/", "GET", "Workspace Members Endpoint"},
		{"/api/workspaces/{slug}/invitations/", "GET, POST, HEAD, OPTIONS", "Workspace Invitations"},
		{"/api/workspaces/{slug}/invitations/{pk}/", "GET, PATCH, DELETE, HEAD, OPTIONS", "Workspace Invitations"},
		{"/api/workspaces/{slug}/invitations/{pk}/join/", "GET, POST, HEAD, OPTIONS", "Workspace Join Endpoint"},
		{"/api/workspaces/{slug}/members/leave/", "POST, OPTIONS", "Work Space Member"},
		{"/api/workspaces/{slug}/members/{pk}/", "PATCH, DELETE, OPTIONS", "Work Space Member"},
		{"/api/workspaces/{slug}/projects/{project_id}/invitations/", "GET, POST, HEAD, OPTIONS", "Project Invitations"},
		{"/api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/", "GET, DELETE, HEAD, OPTIONS", "Project Invitations"},
		{"/api/workspaces/{slug}/projects/{project_id}/join/{pk}/", "GET, POST, HEAD, OPTIONS", "Project Join Endpoint"},
	}
	protocols := make([]readServiceProtocol, 0, len(routes))
	for _, route := range routes {
		protocol := newReadServiceProtocol(route.path, route.allow, route.name, "", readServiceMethodAllowed(route.allow, http.MethodHead), "invitations")
		protocol.generatedMethods = true
		if route.path == "/api/workspaces/{slug}/members/{pk}/" {
			// Previously retired reads keep their own transport and Allow header.
			protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
		}
		protocol.authenticateMethod = !strings.HasPrefix(route.path, "/api/v1/")
		protocol.permissionBeforeTimezone = true
		protocol.resolveMalformedSlashless = true
		if metadata, ok := invitationsMembershipMetadata[route.path]; ok {
			protocol.optionsMetadata = metadata.Metadata
		}
		protocols = append(protocols, protocol)
	}
	return protocols
}

func isInvitationsMembershipProtocol(pattern string) bool {
	for _, protocol := range invitationsMembershipProtocols() {
		if protocol.pattern == pattern {
			return true
		}
	}
	return false
}

func mountInvitationsMembershipHeads(mux loomhttp.Muxer, service *server.Server, policy loomhttp.RuntimeCORSPolicy) {
	for _, mount := range service.Mounts {
		if mount.Verb != http.MethodGet || !mustReadServiceProtocol(mount.Pattern).derivedHead {
			continue
		}
		handler := reflect.ValueOf(service).Elem().FieldByName(mount.Method).Interface().(http.Handler)
		loomhttp.MountDerivedHead(mux, mount.Pattern, policy.Handler(loomhttp.AsHandlerFunc(handler)))
	}
}

func (a membershipActivityProtocolDispatch) AuthorizeInvitationsMembershipProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource, api bool) (int, map[string]any) {
	if a.invitations == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.invitations.AuthorizeInvitationsMembershipProtocol(ctx, key, resource, api)
}
