package main

import (
	"context"
	_ "embed"
	"encoding/json"
	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	loomhttp "github.com/CaliLuke/loom/http"
	"net/http"
	"strings"
)

//go:embed project_membership_lifecycle_metadata.json
var membershipLifecyclePostMetadata json.RawMessage

func newMembershipAPIProtocol(pattern, allow string) readServiceProtocol {
	protocol := newReadServiceProtocol(pattern, allow, "Project Member", "", false, "membership")
	// These generated endpoints own all authored methods on the path.
	protocol.generatedMethods = true
	return protocol
}
func newMembershipSessionProtocol(pattern, allow string, retainedRead bool) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, "Project Member", false)
	// GET is already retired and retains its existing serializer/middleware.
	if retainedRead {
		protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	}
	if readServiceMethodAllowed(allow, http.MethodPost) {
		protocol.optionsMetadata = membershipLifecyclePostMetadata
	}
	return protocol
}

type membershipActivityProtocolDispatch struct {
	configuration projectConfigurationProtocolAuthorizer
	systemapi.ActivityReadsProtocolAuthorizer
	membership  systemapi.MembershipLifecycleProtocolAuthorizer
	assets      systemapi.AssetsAPIProtocolAuthorizer
	invitations systemapi.InvitationsMembershipProtocolAuthorizer
	projects    projectLifecycleProtocolAuthorizer
	stickies    systemapi.APIV1StickyProtocolAuthorizer
}

func (a membershipActivityProtocolDispatch) AuthorizeMembershipLifecycleProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource, api bool) (int, map[string]any) {
	if a.membership == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.membership.AuthorizeMembershipLifecycleProtocol(ctx, key, resource, api)
}
func (a membershipActivityProtocolDispatch) AuthorizeAPIIdentityProtocol(ctx context.Context, key string) (int, map[string]any) {
	if identity, ok := a.ActivityReadsProtocolAuthorizer.(systemapi.APIIdentityProtocolAuthorizer); ok {
		return identity.AuthorizeAPIIdentityProtocol(ctx, key)
	}
	return 500, map[string]any{"error": "Something went wrong please try again later"}
}

type membershipAPIKeyValidator interface {
	Validate(context.Context, string) (apikey.Principal, error)
}
type membershipAPIKeyRouter struct{ reader, writer membershipAPIKeyValidator }

func (r membershipAPIKeyRouter) Validate(ctx context.Context, key string) (apikey.Principal, error) {
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		parts := strings.Split(strings.Trim(metadata.Path, "/"), "/")
		// Invitation ViewSets use primary reads, including authentication. Roster
		// BaseAPIViews retain replica reads, as do existing membership endpoints.
		invitation := (len(parts) == 5 || len(parts) == 6) && parts[0] == "api" && parts[1] == "v1" && parts[2] == "workspaces" && parts[4] == "invitations"
		if invitation || (metadata.Method != http.MethodGet && metadata.Method != http.MethodHead && metadata.Method != http.MethodOptions) {
			return r.writer.Validate(ctx, key)
		}
	}
	return r.reader.Validate(ctx, key)
}

func (a membershipActivityProtocolDispatch) AuthorizeAssetsAPIProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if a.assets == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.assets.AuthorizeAssetsAPIProtocol(ctx, key, resource)
}

func (a membershipActivityProtocolDispatch) AuthorizeProjectLifecycleAPIProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if a.projects == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.projects.AuthorizeProjectLifecycleAPIProtocol(ctx, key, resource)
}

func (a membershipActivityProtocolDispatch) AuthorizeAPIV1StickyProtocol(ctx context.Context, key, slug string) (int, map[string]any) {
	if a.stickies == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.stickies.AuthorizeAPIV1StickyProtocol(ctx, key, slug)
}

func newAPIV1StickyProtocol(pattern, allow, name string) readServiceProtocol {
	protocol := newGeneratedAPIWriteProtocol(pattern, allow, name)
	protocol.apiAuthorization = "stickies"
	return protocol
}

func (a membershipActivityProtocolDispatch) AuthorizeAPIV1LabelProtocol(ctx context.Context, key, slug, projectID string) (int, map[string]any) {
	if a.configuration == nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return a.configuration.AuthorizeAPIV1LabelProtocol(ctx, key, slug, projectID)
}
