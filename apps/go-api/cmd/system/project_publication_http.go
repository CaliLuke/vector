package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	projectpublicationsvr "github.com/CaliLuke/go-plane/projectpublication/gen/http/project_publication/server"
	projectpublication "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed project_publication_options.json
var projectPublicationOptionsJSON []byte

var projectPublicationOptions = func() map[string]json.RawMessage {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(projectPublicationOptionsJSON, &metadata); err != nil {
		panic(err)
	}
	return metadata
}()

func newProjectPublicationProtocol(pattern, allow, name, action string, derivedHead bool) readServiceProtocol {
	writes, key := "POST, OPTIONS", "collection"
	if strings.HasSuffix(pattern, "/{pk}/") {
		writes, key = "PATCH, DELETE, OPTIONS", "detail"
	}
	protocol := newProjectConfigurationSessionProtocol(pattern, writes, allow, "Deploy Board", true)
	protocol.optionsMetadata = projectPublicationOptions[key]
	return protocol
}

var projectPublicationProtocols = []readServiceProtocol{
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/", "POST, OPTIONS", "Project Deploy Board", false),
	newSessionServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/", "PATCH, DELETE, OPTIONS", "Project Deploy Board", false),
}

func mountProjectPublication(
	mux loomhttp.Muxer,
	endpoints *projectpublication.Endpoints,
	authorizer sessionResourceProtocolAuthorizer,
	cors loomhttp.RuntimeCORSPolicy,
	options apiV1CompatibilityOptions,
	sessions readServicesSessionLifecycle,
	cookie djangoSessionCookieConfig,
) {
	server := projectpublicationsvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, cors)
	stack := func(protocol readServiceProtocol, next http.Handler) http.Handler {
		next = projectPublicationAuthorizationMiddleware(authorizer, protocol)(next)
		if sessions != nil {
			next = sessionServiceMiddlewareStack(options, sessions, cookie)(next)
		}
		return next
	}
	server.Create = stack(projectPublicationProtocols[0], server.Create)
	server.Update = stack(projectPublicationProtocols[1], server.Update)
	server.Delete = stack(projectPublicationProtocols[1], server.Delete)

	projectpublicationsvr.MountCreateHandler(mux, server.Create, cors)
	projectpublicationsvr.MountUpdateHandler(mux, server.Update, cors)
	projectpublicationsvr.MountDeleteHandler(mux, server.Delete, cors)
}

func projectPublicationAuthorizationMiddleware(authorizer sessionResourceProtocolAuthorizer, protocol readServiceProtocol) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			request = withReadServicePathValues(request, protocol)
			status, body := authorizer.AuthorizeSessionResourceProtocol(request.Context(), "", sessionProtocolResource(request, protocol))
			if status != 0 {
				writeReadServiceJSON(response, status, body)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

type projectPublicationSessionProtocolDispatch struct {
	labelSessionProtocolDispatch
	publication sessionResourceProtocolAuthorizer
}

func (dispatch projectPublicationSessionProtocolDispatch) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	if isProjectPublicationProtocol(resource.Pattern) && dispatch.publication != nil {
		return dispatch.publication.AuthorizeSessionResourceProtocol(ctx, key, resource)
	}
	return dispatch.labelSessionProtocolDispatch.AuthorizeSessionResourceProtocol(ctx, key, resource)
}

func isProjectPublicationProtocol(pattern string) bool {
	for _, protocol := range projectPublicationProtocols {
		if protocol.pattern == pattern {
			return true
		}
	}
	return false
}
