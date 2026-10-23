package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"

	systemapi "github.com/CaliLuke/go-plane"
)

// The protocol descriptions are source observations. They include native DRF
// methods absent from the generated business-operation manifest.
//
//go:embed assets_protocol_metadata.json
var assetsProtocolMetadataJSON []byte

var assetsProtocolMetadata = func() map[string]struct {
	Allow    string          `json:"allow"`
	Status   int             `json:"status"`
	Metadata json.RawMessage `json:"metadata"`
} {
	var result map[string]struct {
		Allow    string          `json:"allow"`
		Status   int             `json:"status"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(assetsProtocolMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

func assetsReadServiceProtocols() []readServiceProtocol {
	paths := make([]string, 0, len(assetsProtocolMetadata))
	for path := range assetsProtocolMetadata {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	result := make([]readServiceProtocol, 0, len(paths))
	for _, path := range paths {
		metadata := assetsProtocolMetadata[path]
		protocol := newSessionServiceProtocol(path, metadata.Allow, "", readServiceMethodAllowed(metadata.Allow, http.MethodHead))
		protocol.optionsMetadata = metadata.Metadata
		protocol.optionsStatus = metadata.Status
		protocol.permissionBeforeTimezone = path == "/api/workspaces/{slug}/file-assets/" || path == "/api/workspaces/file-assets/{workspace_id}/{asset_key}/" || path == "/api/workspaces/file-assets/{workspace_id}/{asset_key}/restore/"
		protocol.throttleBeforeTimezone = path == duplicateAssetThrottlePath
		if strings.HasPrefix(path, "/api/v1/") {
			protocol.authenticateMethod = false
			protocol.generatedMethods = true
			protocol.apiAuthorization = "assets"
		}
		result = append(result, protocol)
	}
	return result
}

func sessionProtocolResource(request *http.Request, protocol readServiceProtocol) systemapi.SessionProtocolResource {
	request = withReadServicePathValues(request, protocol)
	parameters := make(map[string]string)
	for _, part := range protocol.segments {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			name := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			parameters[name] = request.PathValue(name)
		}
	}
	return systemapi.SessionProtocolResource{
		Method: request.Method, Pattern: protocol.pattern, Slug: request.PathValue("slug"),
		ProjectID: request.PathValue("project_id"), ID: request.PathValue("pk"), PathParameters: parameters,
	}
}

func assetsProtocolTimezone(request *http.Request, protocol readServiceProtocol) (int, map[string]any) {
	if _, ok := assetsProtocolMetadata[protocol.pattern]; !ok || !protocol.permissionBeforeTimezone {
		return 0, nil
	}
	timezone, ok := systemapi.ReadServiceSessionTimezone(request.Context())
	if ok && djangosession.ValidateUserTimezone(timezone) != nil {
		return 400, map[string]any{"error": "The required key does not exist."}
	}
	return 0, nil
}

func assetsSessionPermissionMiddleware(authorizer systemapi.SessionResourceProtocolAuthorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if authorizer != nil {
				protocol, matched := resolveReadServiceProtocol(request.URL.Path)
				if !matched {
					next.ServeHTTP(response, request)
					return
				}
				status, body := authorizer.AuthorizeSessionResourceProtocol(request.Context(), "", sessionProtocolResource(request, protocol))
				if status == 0 {
					status, body = assetsProtocolTimezone(request, protocol)
				}
				if status != 0 {
					writeReadServiceJSON(response, status, body)
					return
				}
			}
			next.ServeHTTP(response, request)
		})
	}
}
