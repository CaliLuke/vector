// Source-backed work_item_core HTTP compatibility metadata; manually maintained.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	workitemsvr "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/server"
	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
)

type workItemProtocol struct {
	readServiceProtocol
	methods []string
}

func newWorkItemProtocol(pattern, allow, name string, methods []string, derivedHead, api bool) workItemProtocol {
	protocol := newReadServiceProtocol(pattern, allow, name, "", derivedHead)
	protocol.workItem = true
	protocol.authenticateMethod = !api
	return workItemProtocol{readServiceProtocol: protocol, methods: methods}
}

var workItemProtocols = []workItemProtocol{
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/{issue_id}/", "DELETE", "Module Issue Detail Api Endpoint", []string{"DELETE"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "DELETE", "Cycle Issue Detail Api Endpoint", []string{"DELETE"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/", "POST", "Transfer Cycle Issue Api Endpoint", []string{"POST"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/", "POST", "Module Issue List Create Api Endpoint", []string{"POST"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/relations/", "GET, POST", "Issue Relation List Create Api Endpoint", []string{"GET", "POST"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "POST", "Cycle Issue List Create Api Endpoint", []string{"POST"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/", "PUT, PATCH, DELETE, OPTIONS", "Cycle Issue", []string{"DELETE", "PATCH", "PUT"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/", "PUT, PATCH, DELETE, OPTIONS", "Module Issue", []string{"DELETE", "PATCH", "PUT"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/", "POST, OPTIONS", "Transfer Cycle Issue Endpoint", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/remove-relation/", "POST, OPTIONS", "Issue Relation", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/", "GET, POST, HEAD, OPTIONS", "Issue Relation", []string{"GET", "POST", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", "POST, OPTIONS", "Cycle Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/", "GET, POST, HEAD, OPTIONS", "Sub Issues Endpoint", []string{"GET", "POST", "HEAD"}, true, false),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/{issue_id}/", "GET, PATCH, DELETE", "Intake Issue Detail Api Endpoint", []string{"DELETE", "GET", "PATCH"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/", "POST, OPTIONS", "Module Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/modules/", "POST, OPTIONS", "Module Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/", "GET, POST, DELETE, HEAD, OPTIONS", "Issue Archive", []string{"DELETE", "GET", "POST", "HEAD"}, true, false),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{pk}/", "GET, PATCH, DELETE", "Issue Detail Api Endpoint", []string{"DELETE", "GET", "PATCH"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/intake-issues/", "GET, POST", "Intake Issue List Create Api Endpoint", []string{"GET", "POST"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{pk}/", "GET, PATCH, DELETE", "Issue Detail Api Endpoint", []string{"DELETE", "GET", "PATCH"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/", "GET, POST", "Issue List Create Api Endpoint", []string{"GET", "POST"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/list/", "GET, HEAD, OPTIONS", "Issue List Endpoint", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/projects/{project_id}/issues/", "GET, POST", "Issue List Create Api Endpoint", []string{"GET", "POST"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/v2/issues/", "GET, HEAD, OPTIONS", "Issue Paginated", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/work-items/search/", "GET", "Issue Search Endpoint", []string{"GET"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/issues/search/", "GET", "Issue Search Endpoint", []string{"GET"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/bulk-archive-issues/", "POST, OPTIONS", "Bulk Archive Issues Endpoint", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/bulk-delete-issues/", "DELETE, OPTIONS", "Bulk Delete Issues Endpoint", []string{"DELETE"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/", "PATCH, DELETE, OPTIONS", "Intake Issue", []string{"DELETE", "PATCH"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", "PATCH, DELETE, OPTIONS", "Intake Issue", []string{"DELETE", "PATCH"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/archived-issues/", "GET, HEAD, OPTIONS", "Issue Archive", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/deleted-issues/", "GET, HEAD, OPTIONS", "Deleted Issues List", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/intake-issues/", "POST, OPTIONS", "Intake Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues-detail/", "GET, HEAD, OPTIONS", "Issue Detail Endpoint", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/search-issues/", "GET, HEAD, OPTIONS", "Issue Search Endpoint", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", "POST, OPTIONS", "Intake Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issue-dates/", "POST, OPTIONS", "Issue Bulk Update Date Endpoint", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS", "Issue", []string{"DELETE", "GET", "PATCH", "PUT", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/projects/{project_id}/issues/", "GET, POST, HEAD, OPTIONS", "Issue", []string{"GET", "POST", "HEAD"}, true, false),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/work-items/{identifier}/", "GET", "Workspace Issue Api Endpoint", []string{"GET"}, false, true),
	newWorkItemProtocol("/api/v1/workspaces/{slug}/issues/{identifier}/", "GET", "Workspace Issue Api Endpoint", []string{"GET"}, false, true),
	newWorkItemProtocol("/api/workspaces/{slug}/draft-to-issue/{draft_id}/", "POST, OPTIONS", "Workspace Draft Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/work-items/{identifier}/", "GET, HEAD, OPTIONS", "Issue Detail Identifier Endpoint", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/user-issues/{user_id}/", "GET, HEAD, OPTIONS", "Workspace User Profile Issues Endpoint", []string{"GET", "HEAD"}, true, false),
	newWorkItemProtocol("/api/workspaces/{slug}/draft-issues/{pk}/", "PATCH, DELETE, OPTIONS", "Workspace Draft Issue", []string{"DELETE", "PATCH"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/draft-issues/", "POST, OPTIONS", "Workspace Draft Issue", []string{"POST"}, false, false),
	newWorkItemProtocol("/api/workspaces/{slug}/issues/", "GET, HEAD, OPTIONS", "Workspace View Issues", []string{"GET", "HEAD"}, true, false),
}

func init() {
	workItemProtocols = append(workItemProtocols, collaborationProtocols...)
	sort.SliceStable(workItemProtocols, func(i, j int) bool {
		return readServiceProtocolSpecificity(workItemProtocols[i].pattern) > readServiceProtocolSpecificity(workItemProtocols[j].pattern)
	})
	for _, workItemProtocol := range workItemProtocols {
		merged := false
		for index := range readServiceProtocols {
			if readServiceProtocols[index].pattern != workItemProtocol.pattern {
				continue
			}
			// Preserve the existing method contract. The resolver below selects the
			// work-item variant only for methods owned by this family.
			readServiceProtocols[index].workItem = true
			// Full-path retirement brings slashless requests and unsupported verbs
			// through Go; retain the session boundary for these former mixed paths.
			readServiceProtocols[index].authenticateMethod = workItemProtocol.authenticateMethod
			merged = true
			break
		}
		if !merged {
			readServiceProtocols = append(readServiceProtocols, workItemProtocol.readServiceProtocol)
		}
	}
	sort.SliceStable(readServiceProtocols, func(i, j int) bool {
		return readServiceProtocolSpecificity(readServiceProtocols[i].pattern) > readServiceProtocolSpecificity(readServiceProtocols[j].pattern)
	})
}

func workItemProtocolForMethod(existing readServiceProtocol, method string) readServiceProtocol {
	for _, protocol := range workItemProtocols {
		if protocol.pattern != existing.pattern {
			continue
		}
		// Select the core companion only when OPTIONS actually supplies metadata.
		// API-v1 rejects OPTIONS; on shared paths that rejection retains the
		// existing resource contract, as other unsupported methods do.
		if strings.EqualFold(method, http.MethodOptions) && readServiceMethodAllowed(protocol.allow, http.MethodOptions) {
			return protocol.readServiceProtocol
		}
		for _, ownedMethod := range protocol.methods {
			if strings.EqualFold(ownedMethod, method) {
				return protocol.readServiceProtocol
			}
		}
	}
	return existing
}

func resolveReadServiceProtocolForMethod(requestPath, method string) (readServiceProtocol, bool) {
	protocol, matched := resolveReadServiceProtocol(requestPath)
	// User identity reads already own their complete transport boundary. The
	// account write registration must not add generic headers or negotiation to it.
	if matched && protocol.pattern == "/api/users/me/" && slices.Contains(protocol.passthroughMethods, method) {
		return readServiceProtocol{}, false
	}
	if !matched {
		// API-v1 lifecycle action aliases (archive and unarchive) are generated
		// write routes rather than read-service routes. Resolve them through the
		// shared method boundary so unsupported verbs get the normal API-key
		// authorization and DRF response handling.

		if lifecycle, ok := projectLifecycleProtocolForPath(requestPath); ok && strings.HasSuffix(requestPath, "/") {
			if allowed := lifecycle.methodAllow[method]; allowed != "" {
				lifecycle.allow = allowed
			}
			return lifecycle, true
		}
		if lifecycle, ok := moduleLifecycleProtocolForPath(requestPath); ok && strings.HasSuffix(requestPath, "/") && strings.HasPrefix(lifecycle.pattern, "/api/v1/") {
			return lifecycle, true
		}
		return readServiceProtocol{}, false
	}
	if protocol.generatedMethods && !readServiceMethodAllowed(protocol.allow, method) && !slices.Contains(protocol.passthroughMethods, method) && (method == http.MethodGet || method == http.MethodHead) {
		for _, candidate := range readServiceProtocols {
			if candidate.pattern == protocol.pattern || !readServiceProtocolMatchesPath(requestPath, candidate) {
				continue
			}
			retainedRead := slices.Contains(candidate.passthroughMethods, method) || readServiceMethodAllowed(candidate.allow, method) || (method == http.MethodHead && candidate.derivedHead && readServiceMethodAllowed(candidate.allow, http.MethodGet))
			if !retainedRead {
				continue
			}
			if readServiceProtocolHasMalformedUUID(requestPath, candidate) {
				return candidate, true
			}
		}
	}
	protocol = workspaceLifecycleProtocolForMethod(protocol, method)
	protocol = projectLifecycleProtocolForMethod(moduleLifecycleProtocolForMethod(workItemProtocolForMethod(protocol, method), method), method)
	if allowed, ok := protocol.methodAllow[method]; ok {
		protocol.allow = allowed
	}
	return protocol, true
}

func readServiceProtocolSpecificity(pattern string) int {
	specificity := 0
	for _, segment := range strings.Split(strings.Trim(pattern, "/"), "/") {
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
			specificity++
		}
	}
	return specificity
}

func mountWorkItemHeads(mux loomhttp.Muxer, server *workitemsvr.Server, policy loomhttp.RuntimeCORSPolicy) {
	for _, mount := range server.Mounts {
		if mount.Verb != http.MethodGet {
			continue
		}
		protocol := workItemProtocolForMethod(mustReadServiceProtocol(mount.Pattern), http.MethodGet)
		if !protocol.workItem || !protocol.derivedHead {
			continue
		}
		handlerField := reflect.ValueOf(server).Elem().FieldByName(mount.Method)
		if !handlerField.IsValid() || !handlerField.CanInterface() {
			panic("missing generated work item handler " + mount.Method)
		}
		handler, ok := handlerField.Interface().(http.Handler)
		if !ok || handler == nil {
			panic(fmt.Sprintf("generated work item handler %s is not an HTTP handler", mount.Method))
		}
		loomhttp.MountDerivedHead(mux, protocol.pattern, policy.Handler(loomhttp.AsHandlerFunc(handler)))
	}
}

func workItemResponseContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		if !matched || !workItemMethodOwned(protocol.pattern, request.Method) {
			next.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(&workItemResponseWriter{ResponseWriter: response}, request)
	})
}

type workItemResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (writer *workItemResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	if status == http.StatusNoContent {
		writer.Header().Del("Content-Type")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *workItemResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func workItemProtocolMiddleware(
	authorizer systemapi.WorkItemProtocolAuthorizer,
	policy loomhttp.RuntimeCORSPolicy,
	stack func(http.Handler) http.Handler,
	cookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	ordinary := policy.Handler(loomhttp.AsHandlerFunc(stack(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		if !matched || !protocol.workItem {
			writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
			return
		}
		if request.Method == http.MethodOptions {
			protocol = workItemProtocolForPattern(protocol.pattern)
		}
		request = withReadServicePathValues(request, protocol)
		params := make(map[string]string)
		for _, name := range []string{"filters", "issue__id", "workspace__id"} {
			values := request.URL.Query()[name]
			if len(values) > 0 {
				params[name] = values[len(values)-1]
			}
		}
		for index, segment := range protocol.segments {
			if index >= len(strings.Split(strings.Trim(request.URL.Path, "/"), "/")) || !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
				continue
			}
			name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
			params[name] = request.PathValue(name)
		}
		sessionID := ""
		if session, err := request.Cookie(sessionCookieName(cookie)); err == nil {
			sessionID = session.Value
		}
		decision, err := authorizer.AuthorizeWorkItemProtocol(request.Context(), systemapi.WorkItemProtocolResource{
			Method: request.Method, Pattern: protocol.pattern, Params: params,
			APIKey: djangoJoinedHeader(request, "X-Api-Key"), SessionID: sessionID,
		})
		if err != nil {
			writeWorkItemProtocolError(response, request, err)
			return
		}
		if request.Method == http.MethodOptions && !strings.HasPrefix(protocol.pattern, "/api/v1/") {
			status, body := workItemProtocolMetadata(protocol, decision)
			writeWorkItemProtocolJSON(response, request, status, body)
			return
		}
		writeWorkItemProtocolJSON(response, request, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	}))))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
			if !matched || !protocol.workItem || workItemMethodOwned(protocol.pattern, request.Method) ||
				(request.Method != http.MethodOptions && readServiceMethodAllowed(protocol.allow, request.Method)) ||
				(request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0) {
				next.ServeHTTP(response, request)
				return
			}
			// Global CORS middleware handles preflight before this compatibility layer.
			ordinary.ServeHTTP(response, request)
		})
	}
}

func workItemIdentifierResolver404Middleware(next http.Handler, activityOptions *apiV1CompatibilityOptions, corsPolicy loomhttp.RuntimeCORSPolicy) http.Handler {
	resolver404 := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Content-Type", "application/json")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "Cookie", "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		writeReadServiceJSON(response, http.StatusNotFound, map[string]string{"error": "Page not found."})
	})
	auditedResolver404 := http.Handler(resolver404)
	if activityOptions != nil {
		// Django emits the API-key activity log for a resolver 404 without
		// invoking endpoint authentication. Keep that audit boundary around
		// the response while the resolver remains outside session/auth stacks.
		auditedResolver404 = apiV1CompatibilityMiddleware(*activityOptions)(resolver404)
	}
	// CORS runs before Django's API-key audit middleware on both app and API
	// paths. In particular, a preflight returns before any audit or auth work.
	corsResolver404 := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, corsPolicy.Handler(loomhttp.AsHandlerFunc(auditedResolver404)))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if workItemIdentifierHasEncodedSlash(request.URL) {
			// Django resolves the decoded path before it invokes the identifier
			// view. Its string converter rejects a slash, including one encoded
			// as %2F, so this is a resolver 404 and must precede auth/session work.
			corsResolver404.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func workItemIdentifierHasEncodedSlash(requestURL *url.URL) bool {
	if requestURL == nil {
		return false
	}
	escapedSegments := strings.Split(strings.Trim(requestURL.EscapedPath(), "/"), "/")
protocolLoop:
	for _, protocol := range workItemProtocols {
		patternSegments := protocol.segments
		if len(patternSegments) != len(escapedSegments) {
			continue
		}
		hasEncodedSlash := false
		for index, patternSegment := range patternSegments {
			value, err := url.PathUnescape(escapedSegments[index])
			if err != nil {
				continue protocolLoop
			}
			if strings.Contains(patternSegment, "{") && strings.Contains(patternSegment, "}") {
				// Django's ordinary string converters reject an encoded slash in
				// every captured segment, including API routes whose generated
				// parameter name is pk or issue_id rather than identifier.
				if strings.Contains(strings.ToLower(escapedSegments[index]), "%2f") {
					hasEncodedSlash = true
				}
				continue
			}
			if value != patternSegment {
				continue protocolLoop
			}
		}
		if hasEncodedSlash {
			return true
		}
	}
	return false
}

func workItemProtocolForPattern(pattern string) readServiceProtocol {
	for _, protocol := range workItemProtocols {
		if protocol.pattern == pattern {
			return protocol.readServiceProtocol
		}
	}
	return readServiceProtocol{}
}

func workItemMethodOwned(pattern, method string) bool {
	for _, protocol := range workItemProtocols {
		if protocol.pattern != pattern {
			continue
		}
		for _, owned := range protocol.methods {
			if strings.EqualFold(owned, method) {
				return true
			}
		}
		return false
	}
	return false
}

type workItemOptionsResponse struct {
	status int
	body   json.RawMessage
}

var workItemOptionsResponses = map[string]workItemOptionsResponse{
	"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/": {status: 500, body: json.RawMessage("{\"error\":\"Something went wrong please try again later\"}")},
	"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/":     {status: 500, body: json.RawMessage("{\"error\":\"Something went wrong please try again later\"}")},
	"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/transfer-issues/":         {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Transfer Cycle Issue Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/remove-relation/":         {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"assignee_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"field\"},\"label\":\"Assignee ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"name\":{\"label\":\"Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"priority\":{\"label\":\"Priority\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"relation_type\":{\"label\":\"Relation type\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"sequence_id\":{\"label\":\"Sequence id\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"state_id\":{\"label\":\"State id\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Issue Relation\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/issue-relation/":          {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"assignee_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"field\"},\"label\":\"Assignee ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"name\":{\"label\":\"Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"priority\":{\"label\":\"Priority\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"relation_type\":{\"label\":\"Relation type\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"sequence_id\":{\"label\":\"Sequence id\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"state_id\":{\"label\":\"State id\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Issue Relation\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/sub-issues/":              {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Sub Issues Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/archive/":                       {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"description_html\":{\"label\":\"Description html\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"description_json\":{\"label\":\"Description json\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"is_draft\":{\"label\":\"Is draft\",\"read_only\":false,\"required\":false,\"type\":\"boolean\"},\"name\":{\"label\":\"Issue Name\",\"max_length\":255,\"read_only\":false,\"required\":true,\"type\":\"string\"},\"priority\":{\"choices\":[{\"display_name\":\"Urgent\",\"value\":\"urgent\"},{\"display_name\":\"High\",\"value\":\"high\"},{\"display_name\":\"Medium\",\"value\":\"medium\"},{\"display_name\":\"Low\",\"value\":\"low\"},{\"display_name\":\"None\",\"value\":\"none\"}],\"label\":\"Issue Priority\",\"read_only\":false,\"required\":false,\"type\":\"choice\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"max_value\":2147483647,\"min_value\":-2147483648,\"read_only\":false,\"required\":false,\"type\":\"integer\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":false,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":false,\"required\":false,\"type\":\"date\"}}},\"description\":\"\",\"name\":\"Issue Archive\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/":            {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"cycle\":{\"label\":\"Cycle\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"issue\":{\"label\":\"Issue\",\"read_only\":false,\"required\":true,\"type\":\"field\"},\"issue_detail\":{\"children\":{\"archived_at\":{\"label\":\"Archived at\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"assignee_details\":{\"child\":{\"children\":{\"avatar\":{\"label\":\"Avatar\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"avatar_url\":{\"label\":\"Avatar url\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"display_name\":{\"label\":\"Display name\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"first_name\":{\"label\":\"First name\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"is_bot\":{\"label\":\"Is bot\",\"read_only\":true,\"required\":false,\"type\":\"boolean\"},\"last_name\":{\"label\":\"Last name\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"}},\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"label\":\"Assignee details\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"assignees\":{\"label\":\"Assignees\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"attachment_count\":{\"label\":\"Attachment count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"completed_at\":{\"label\":\"Completed at\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"description_binary\":{\"label\":\"Description binary\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"description_html\":{\"label\":\"Description html\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"description_json\":{\"label\":\"Description json\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"description_stripped\":{\"label\":\"Description stripped\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"estimate_point\":{\"label\":\"Estimate point\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"external_id\":{\"label\":\"External id\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"external_source\":{\"label\":\"External source\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"is_draft\":{\"label\":\"Is draft\",\"read_only\":false,\"required\":false,\"type\":\"boolean\"},\"label_details\":{\"child\":{\"children\":{\"color\":{\"label\":\"Color\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Name\",\"max_length\":255,\"read_only\":false,\"required\":true,\"type\":\"string\"}},\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"label\":\"Label details\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"labels\":{\"label\":\"Labels\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"link_count\":{\"label\":\"Link count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"name\":{\"label\":\"Issue Name\",\"max_length\":255,\"read_only\":false,\"required\":true,\"type\":\"string\"},\"parent\":{\"label\":\"Parent\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"point\":{\"label\":\"Point\",\"max_value\":12,\"min_value\":0,\"read_only\":false,\"required\":false,\"type\":\"integer\"},\"priority\":{\"choices\":[{\"display_name\":\"Urgent\",\"value\":\"urgent\"},{\"display_name\":\"High\",\"value\":\"high\"},{\"display_name\":\"Medium\",\"value\":\"medium\"},{\"display_name\":\"Low\",\"value\":\"low\"},{\"display_name\":\"None\",\"value\":\"none\"}],\"label\":\"Issue Priority\",\"read_only\":false,\"required\":false,\"type\":\"choice\"},\"project\":{\"label\":\"Project\",\"read_only\":false,\"required\":true,\"type\":\"field\"},\"project_detail\":{\"children\":{\"cover_image\":{\"label\":\"Cover image\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"cover_image_url\":{\"label\":\"Cover image url\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"description\":{\"label\":\"Project Description\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"identifier\":{\"label\":\"Project Identifier\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"logo_props\":{\"label\":\"Logo props\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Project Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"}},\"label\":\"Project detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"max_value\":2147483647,\"min_value\":-2147483648,\"read_only\":false,\"required\":false,\"type\":\"integer\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":false,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"state\":{\"label\":\"State\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"state_detail\":{\"children\":{\"color\":{\"label\":\"State Color\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"group\":{\"label\":\"Group\",\"read_only\":true,\"required\":false,\"type\":\"choice\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"State Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"}},\"label\":\"State detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"type\":{\"label\":\"Type\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":false,\"required\":true,\"type\":\"field\"}},\"label\":\"Issue detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"project\":{\"label\":\"Project\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Cycle Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/":                        {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Intake Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/":                         {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Intake Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/":                {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"issue\":{\"label\":\"Issue\",\"read_only\":false,\"required\":true,\"type\":\"field\"},\"issue_detail\":{\"children\":{\"cover_image\":{\"label\":\"Cover image\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"cover_image_url\":{\"label\":\"Cover image url\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"description\":{\"label\":\"Project Description\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"identifier\":{\"label\":\"Project Identifier\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"logo_props\":{\"label\":\"Logo props\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Project Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"}},\"label\":\"Issue detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"module\":{\"label\":\"Module\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"module_detail\":{\"children\":{\"archived_at\":{\"label\":\"Archived at\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"description\":{\"label\":\"Module Description\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"description_html\":{\"label\":\"Module Description HTML\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"description_text\":{\"label\":\"Module Description RT\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"external_id\":{\"label\":\"External id\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"external_source\":{\"label\":\"External source\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"lead\":{\"label\":\"Lead\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"logo_props\":{\"label\":\"Logo props\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"members\":{\"label\":\"Members\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Module Name\",\"max_length\":255,\"read_only\":false,\"required\":true,\"type\":\"string\"},\"project\":{\"label\":\"Project\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":false,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"status\":{\"choices\":[{\"display_name\":\"Backlog\",\"value\":\"backlog\"},{\"display_name\":\"Planned\",\"value\":\"planned\"},{\"display_name\":\"In Progress\",\"value\":\"in-progress\"},{\"display_name\":\"Paused\",\"value\":\"paused\"},{\"display_name\":\"Completed\",\"value\":\"completed\"},{\"display_name\":\"Cancelled\",\"value\":\"cancelled\"}],\"label\":\"Status\",\"read_only\":false,\"required\":false,\"type\":\"choice\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"view_props\":{\"label\":\"View props\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":true,\"required\":false,\"type\":\"field\"}},\"label\":\"Module detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"project\":{\"label\":\"Project\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Module Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/modules/":                 {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"issue\":{\"label\":\"Issue\",\"read_only\":false,\"required\":true,\"type\":\"field\"},\"issue_detail\":{\"children\":{\"cover_image\":{\"label\":\"Cover image\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"cover_image_url\":{\"label\":\"Cover image url\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"description\":{\"label\":\"Project Description\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"identifier\":{\"label\":\"Project Identifier\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"logo_props\":{\"label\":\"Logo props\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Project Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"}},\"label\":\"Issue detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"module\":{\"label\":\"Module\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"module_detail\":{\"children\":{\"archived_at\":{\"label\":\"Archived at\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"deleted_at\":{\"label\":\"Deleted At\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"description\":{\"label\":\"Module Description\",\"read_only\":false,\"required\":false,\"type\":\"string\"},\"description_html\":{\"label\":\"Module Description HTML\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"description_text\":{\"label\":\"Module Description RT\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"external_id\":{\"label\":\"External id\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"external_source\":{\"label\":\"External source\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"lead\":{\"label\":\"Lead\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"logo_props\":{\"label\":\"Logo props\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"members\":{\"label\":\"Members\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"name\":{\"label\":\"Module Name\",\"max_length\":255,\"read_only\":false,\"required\":true,\"type\":\"string\"},\"project\":{\"label\":\"Project\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":false,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"status\":{\"choices\":[{\"display_name\":\"Backlog\",\"value\":\"backlog\"},{\"display_name\":\"Planned\",\"value\":\"planned\"},{\"display_name\":\"In Progress\",\"value\":\"in-progress\"},{\"display_name\":\"Paused\",\"value\":\"paused\"},{\"display_name\":\"Completed\",\"value\":\"completed\"},{\"display_name\":\"Cancelled\",\"value\":\"cancelled\"}],\"label\":\"Status\",\"read_only\":false,\"required\":false,\"type\":\"choice\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":false,\"required\":false,\"type\":\"date\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"view_props\":{\"label\":\"View props\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":true,\"required\":false,\"type\":\"field\"}},\"label\":\"Module detail\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"project\":{\"label\":\"Project\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"workspace\":{\"label\":\"Workspace\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Module Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/bulk-archive-issues/":                       {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Bulk Archive Issues Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/bulk-delete-issues/":                        {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Bulk Delete Issues Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issue-dates/":                               {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Bulk Update Date Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/intake-issues/":                             {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"created_by\":{\"label\":\"Created By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"duplicate_to\":{\"label\":\"Duplicate to\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"issue\":{\"children\":{\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"label_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Label ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"name\":{\"label\":\"Issue Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"priority\":{\"label\":\"Issue Priority\",\"read_only\":true,\"required\":false,\"type\":\"choice\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"read_only\":true,\"required\":false,\"type\":\"integer\"}},\"label\":\"Issue\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"snoozed_till\":{\"label\":\"Snoozed till\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"source\":{\"label\":\"Source\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"status\":{\"choices\":[{\"display_name\":\"Pending\",\"value\":-2},{\"display_name\":\"Rejected\",\"value\":-1},{\"display_name\":\"Snoozed\",\"value\":0},{\"display_name\":\"Accepted\",\"value\":1},{\"display_name\":\"Duplicate\",\"value\":2}],\"label\":\"Status\",\"read_only\":false,\"required\":false,\"type\":\"choice\"}}},\"description\":\"\",\"name\":\"Intake Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/":                              {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"created_by\":{\"label\":\"Created By\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"duplicate_to\":{\"label\":\"Duplicate to\",\"read_only\":false,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"issue\":{\"children\":{\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"label_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Label ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"name\":{\"label\":\"Issue Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"priority\":{\"label\":\"Issue Priority\",\"read_only\":true,\"required\":false,\"type\":\"choice\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"read_only\":true,\"required\":false,\"type\":\"integer\"}},\"label\":\"Issue\",\"read_only\":true,\"required\":false,\"type\":\"nested object\"},\"snoozed_till\":{\"label\":\"Snoozed till\",\"read_only\":false,\"required\":false,\"type\":\"datetime\"},\"source\":{\"label\":\"Source\",\"max_length\":255,\"read_only\":false,\"required\":false,\"type\":\"string\"},\"status\":{\"choices\":[{\"display_name\":\"Pending\",\"value\":-2},{\"display_name\":\"Rejected\",\"value\":-1},{\"display_name\":\"Snoozed\",\"value\":0},{\"display_name\":\"Accepted\",\"value\":1},{\"display_name\":\"Duplicate\",\"value\":2}],\"label\":\"Status\",\"read_only\":false,\"required\":false,\"type\":\"choice\"}}},\"description\":\"\",\"name\":\"Intake Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/archived-issues/":                           {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Archive\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/deleted-issues/":                            {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Deleted Issues List\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues-detail/":                             {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Detail Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/search-issues/":                             {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Search Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/":                               {status: 200, body: json.RawMessage("{\"actions\":{\"PUT\":{\"archived_at\":{\"label\":\"Archived at\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"assignee_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Assignee ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"attachment_count\":{\"label\":\"Attachment count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"completed_at\":{\"label\":\"Completed at\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"cycle_id\":{\"label\":\"Cycle id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"estimate_point\":{\"label\":\"Estimate point\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"is_draft\":{\"label\":\"Is draft\",\"read_only\":true,\"required\":false,\"type\":\"boolean\"},\"label_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Label ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"link_count\":{\"label\":\"Link count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"module_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Module ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"name\":{\"label\":\"Issue Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"parent_id\":{\"label\":\"Parent id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"priority\":{\"label\":\"Issue Priority\",\"read_only\":true,\"required\":false,\"type\":\"choice\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":true,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"state_id\":{\"label\":\"State id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/":                                    {status: 200, body: json.RawMessage("{\"actions\":{\"POST\":{\"archived_at\":{\"label\":\"Archived at\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"assignee_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Assignee ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"attachment_count\":{\"label\":\"Attachment count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"completed_at\":{\"label\":\"Completed at\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_at\":{\"label\":\"Created At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"created_by\":{\"label\":\"Created By\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"cycle_id\":{\"label\":\"Cycle id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"estimate_point\":{\"label\":\"Estimate point\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"id\":{\"label\":\"Id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"is_draft\":{\"label\":\"Is draft\",\"read_only\":true,\"required\":false,\"type\":\"boolean\"},\"label_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Label ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"link_count\":{\"label\":\"Link count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"module_ids\":{\"child\":{\"read_only\":false,\"required\":true,\"type\":\"string\"},\"label\":\"Module ids\",\"read_only\":false,\"required\":false,\"type\":\"list\"},\"name\":{\"label\":\"Issue Name\",\"read_only\":true,\"required\":false,\"type\":\"string\"},\"parent_id\":{\"label\":\"Parent id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"priority\":{\"label\":\"Issue Priority\",\"read_only\":true,\"required\":false,\"type\":\"choice\"},\"project_id\":{\"label\":\"Project id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sequence_id\":{\"label\":\"Issue Sequence ID\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"sort_order\":{\"label\":\"Sort order\",\"read_only\":true,\"required\":false,\"type\":\"float\"},\"start_date\":{\"label\":\"Start date\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"state_id\":{\"label\":\"State id\",\"read_only\":true,\"required\":false,\"type\":\"field\"},\"sub_issues_count\":{\"label\":\"Sub issues count\",\"read_only\":true,\"required\":false,\"type\":\"integer\"},\"target_date\":{\"label\":\"Target date\",\"read_only\":true,\"required\":false,\"type\":\"date\"},\"updated_at\":{\"label\":\"Last Modified At\",\"read_only\":true,\"required\":false,\"type\":\"datetime\"},\"updated_by\":{\"label\":\"Last Modified By\",\"read_only\":true,\"required\":false,\"type\":\"field\"}}},\"description\":\"\",\"name\":\"Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/v2/issues/":                                 {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Paginated\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/projects/{project_id}/issues/list/":                               {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue List Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/draft-to-issue/{draft_id}/":                                       {status: 500, body: json.RawMessage("{\"error\":\"Something went wrong please try again later\"}")},
	"/api/workspaces/{slug}/work-items/{identifier}/":                                         {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Issue Detail Identifier Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/user-issues/{user_id}/":                                           {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Workspace User Profile Issues Endpoint\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/draft-issues/{pk}/":                                               {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Workspace Draft Issue\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
	"/api/workspaces/{slug}/draft-issues/":                                                    {status: 500, body: json.RawMessage("{\"error\":\"Something went wrong please try again later\"}")},
	"/api/workspaces/{slug}/issues/":                                                          {status: 200, body: json.RawMessage("{\"description\":\"\",\"name\":\"Workspace View Issues\",\"parses\":[\"application/json\",\"application/x-www-form-urlencoded\",\"multipart/form-data\"],\"renders\":[\"application/json\"]}")},
}

func workItemProtocolMetadata(protocol readServiceProtocol, decision systemapi.WorkItemProtocolMetadata) (int, any) {
	metadata, ok := workItemOptionsResponses[protocol.pattern]
	if !ok {
		panic("missing source-backed work-item OPTIONS metadata for " + protocol.pattern)
	}
	if decision.OmitPutAction {
		// Decode a request-local copy: the source metadata map is shared by
		// concurrent requests, including OPTIONS for existing objects.
		var body map[string]json.RawMessage
		if err := json.Unmarshal(metadata.body, &body); err != nil {
			panic(err)
		}
		delete(body, "actions")
		return metadata.status, body
	}
	return metadata.status, metadata.body
}

func writeWorkItemProtocolError(response http.ResponseWriter, request *http.Request, err error) {
	var apiError *workitemapi.WorkItemError
	if !errors.As(err, &apiError) {
		writeWorkItemProtocolJSON(response, request, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
		return
	}
	status, parseErr := strconv.Atoi(strings.TrimPrefix(apiError.Kind, "status_"))
	if parseErr != nil || status < 100 || status > 599 {
		status = http.StatusInternalServerError
	}
	body := json.RawMessage(apiError.Body)
	if !json.Valid(body) {
		body = json.RawMessage(`{"error":"Something went wrong please try again later"}`)
		status = http.StatusInternalServerError
	}
	writeWorkItemProtocolJSON(response, request, status, body)
}

func writeWorkItemProtocolJSON(response http.ResponseWriter, request *http.Request, status int, body any) {
	encoded, err := json.Marshal(body)
	if err != nil {
		panic(fmt.Errorf("marshal work-item protocol response: %w", err))
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	// API-key auditing also applies to session endpoints. Preserve the rendered
	// body until that boundary records it and suppresses HEAD bytes on the wire.
	_, auditing := request.Context().Value(activityReadsAuditStateKey{}).(*activityReadsAuditState)
	if request.Method != http.MethodHead || auditing {
		_, _ = response.Write(encoded)
	}
}
