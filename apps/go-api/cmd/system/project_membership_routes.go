package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	projectmembershipreadssvr "github.com/CaliLuke/go-plane/gen/http/project_membership_reads/server"
	membershiplifecyclesvr "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectMembershipReadsProtocolRoutes(
	mux loomhttp.Muxer,
	server *projectmembershipreadssvr.Server,
	authorizer systemapi.ProjectMembershipReadsMetadataAuthorizer,
) {
	routes := []struct {
		path                       string
		name                       string
		get                        http.Handler
		mountOptions               bool
		requireWorkspaceMembership bool
	}{
		{
			path:         "/api/workspaces/{slug}/projects/{project_id}/project-members/me/",
			name:         "Project Member User Endpoint",
			get:          server.Membership,
			mountOptions: true,
		},
		{
			path:                       "/api/users/me/workspaces/{slug}/project-roles/",
			name:                       "User Project Roles Endpoint",
			get:                        server.Roles,
			mountOptions:               true,
			requireWorkspaceMembership: true,
		},
		{
			path:         "/api/workspaces/{slug}/members/",
			name:         "Work Space Member",
			get:          server.WorkspaceMembers,
			mountOptions: true,
		},
		{
			path: "/api/workspaces/{slug}/members/{pk}/",
			name: "Workspace Member Endpoint",
			get:  server.WorkspaceMember,
		},
		{
			path:                       "/api/workspaces/{slug}/project-members/",
			name:                       "Workspace Project Member Endpoint",
			get:                        server.WorkspaceProjectMembers,
			mountOptions:               true,
			requireWorkspaceMembership: true,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/members/",
			name: "Project Member Endpoint",
			get:  server.ProjectMembers,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/members/{pk}/",
			name: "Project Member Endpoint",
			get:  server.ProjectMember,
		},
		{
			path: "/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/",
			name: "Project Member Preference Endpoint",
			get:  server.Preference,
		},
	}
	for _, route := range routes {
		mountProtocolHandler(
			mux,
			http.MethodHead,
			route.path,
			djangoProtocolHeadHandler(projectMembershipReadsCompatibilityMiddleware, route.get),
		)
		if route.mountOptions {
			mountProtocolHandler(
				mux,
				http.MethodOptions,
				route.path,
				djangoReadOnlyOptionsHandler(
					mux,
					projectMembershipReadsCompatibilityMiddleware,
					route.name,
					authorizer,
					route.requireWorkspaceMembership,
				),
			)
		}
	}
}

type projectMembershipReadsResponseWriter struct {
	http.ResponseWriter
	requestPath   string
	requestMethod string
	rawResponse   *systemapi.ProjectMembershipReadsResponseCapture
	jsonIndent    int
	status        int
	wroteHeader   bool
}

func (writer *projectMembershipReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	mergeVary(writer.Header(), "Cookie")
	if status == http.StatusOK && writer.requestMethod != http.MethodOptions &&
		!strings.Contains(writer.requestPath, "/project-roles/") {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectMembershipReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	inputLength := len(body)
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 && writer.requestMethod != http.MethodOptions {
		body = writer.rawResponse.Data
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, djangoJSONColonSpacing(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	if len(body) != inputLength {
		written, err := writer.ResponseWriter.Write(body)
		if err != nil {
			return 0, err
		}
		if written != len(body) {
			return 0, io.ErrShortWrite
		}
		return inputLength, nil
	}
	return writer.ResponseWriter.Write(body)
}

func projectMembershipReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", projectMembershipReadsAllow(request.URL.Path))
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectMembershipReadsJSON(
					response,
					request,
					http.StatusNotFound,
					map[string]string{"detail": "Not found."},
				)
				return
			}
		}
		accept, acceptPresent := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
		if negotiationErr != nil {
			writeDjangoAPIViewServerError(response, request)
			return
		}
		if !acceptsJSON {
			headers.Set("Content-Type", "application/json")
			writeProjectMembershipReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		if values := request.URL.Query()["search"]; len(values) > 1 {
			query := request.URL.Query()
			query.Set("search", values[len(values)-1])
			request = request.Clone(request.Context())
			clonedURL := *request.URL
			request.URL = &clonedURL
			request.URL.RawQuery = query.Encode()
		}
		rawResponse := &systemapi.ProjectMembershipReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectMembershipReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectMembershipReadsResponseWriter{
			ResponseWriter: response,
			requestPath:    request.URL.Path,
			requestMethod:  request.Method,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectMembershipReadsJSON(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	value any,
) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func projectMembershipReadsAllow(path string) string {
	switch {
	case strings.Contains(path, "/preferences/member/"):
		return "GET, PATCH, HEAD, OPTIONS"
	case strings.Contains(path, "/projects/") && strings.Contains(path, "/members/"):
		trimmed := strings.TrimSuffix(path, "/")
		if strings.HasSuffix(trimmed, "/members") {
			return "GET, POST, HEAD, OPTIONS"
		}
		return "GET, PATCH, DELETE, HEAD, OPTIONS"
	case strings.Contains(path, "/members/") && !strings.Contains(path, "/project-members/"):
		trimmed := strings.TrimSuffix(path, "/")
		if strings.HasSuffix(trimmed, "/members") {
			return "GET, HEAD, OPTIONS"
		}
		return "GET, PATCH, DELETE, HEAD, OPTIONS"
	default:
		return "GET, HEAD, OPTIONS"
	}
}

func mountProjectMembershipReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.projectMembershipReads != nil {
		server := projectmembershipreadssvr.New(pending.projectMembershipReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectMembershipReadsCompatibilityMiddleware)
		projectmembershipreadssvr.Mount(mux, server)
		mountProjectMembershipReadsProtocolRoutes(mux, server, pending.projectMembershipMetadata)
	}
}

func mountMembershipLifecycleRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions, stack func(http.Handler) http.Handler) {
	if pending.membershipLifecycle != nil {
		membership := membershiplifecyclesvr.New(pending.membershipLifecycle, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		apiStack := stack
		sessionStack := sessionServiceMiddlewareStack(*activityOptions, pending.readServicesSession, pending.readServicesCookie)
		membership.Use(func(next http.Handler) http.Handler {
			api := apiStack(next)
			session := sessionStack(next)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/v1/") {
					api.ServeHTTP(w, r)
				} else {
					session.ServeHTTP(w, r)
				}
			})
		})
		membershiplifecyclesvr.Mount(mux, membership)
	}
}
