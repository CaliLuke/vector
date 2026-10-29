package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	apistatereadssvr "github.com/CaliLuke/go-plane/apistatereads/gen/http/api_state_reads/server"
	projectstatereadssvr "github.com/CaliLuke/go-plane/gen/http/project_state_reads/server"
	projectstatereads "github.com/CaliLuke/go-plane/gen/project_state_reads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProjectStateReadsProtocolRoutes(
	mux loomhttp.Muxer,
	server *projectstatereadssvr.Server,
	authorizer systemapi.ProjectStateReadsMetadataAuthorizer,
) {
	routes := []struct {
		path string
		get  http.Handler
	}{
		{path: "/api/workspaces/{slug}/projects/{project_id}/states/", get: server.List},
		{path: "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", get: server.Detail},
		{path: "/api/workspaces/{slug}/projects/{project_id}/intake-state/", get: server.Intake},
	}
	for _, route := range routes {
		mountProtocolHandler(
			mux,
			http.MethodHead,
			route.path,
			djangoProtocolHeadHandler(projectStateReadsCompatibilityMiddleware, route.get),
		)
	}
	mountProtocolHandler(
		mux,
		http.MethodOptions,
		"/api/workspaces/{slug}/projects/{project_id}/intake-state/",
		projectStateReadsOptionsHandler(authorizer),
	)
}

func projectStateReadsOptionsHandler(
	authorizer systemapi.ProjectStateReadsMetadataAuthorizer,
) http.Handler {
	return projectStateReadsCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var sessionID string
		if cookie, err := request.Cookie("session-id"); err == nil {
			sessionID = cookie.Value
		}
		err := authorizer.AuthorizeMetadata(request.Context(), sessionID)
		if err != nil {
			var unauthenticated *projectstatereads.ProjectStateReadsUnauthenticated
			var permissionDenied *projectstatereads.ProjectStateReadsPermissionDenied
			var internalError *projectstatereads.ProjectStateReadsInternalError
			switch {
			case errors.As(err, &unauthenticated):
				writeStickyJSON(response, http.StatusUnauthorized, unauthenticated)
			case errors.As(err, &permissionDenied):
				writeStickyJSON(response, http.StatusForbidden, permissionDenied)
			case errors.As(err, &internalError):
				writeStickyJSON(response, http.StatusInternalServerError, internalError)
			default:
				writeStickyJSON(response, http.StatusInternalServerError, map[string]string{
					"error": projectStateReadsInternalErrorDetail,
				})
			}
			return
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": "Intake State Endpoint", "description": "", "renders": []string{"application/json"},
			"parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

const projectStateReadsInternalErrorDetail = "Something went wrong please try again later"

type projectStateReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse *systemapi.ProjectStateReadsResponseCapture
	jsonIndent  int
	status      int
	wroteHeader bool
}

func (writer *projectStateReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	removeVary(writer.Header(), "Cookie", "origin")
	mergeVary(writer.Header(), "Cookie", "origin")
	if status == http.StatusOK {
		prependAcceptEncodingVary(writer.Header())
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectStateReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	inputLength := len(body)
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 {
		body = writer.rawResponse.Data
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
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

func projectStateReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		allow := projectStateReadsAllow(request.URL.Path)
		switch request.Method {
		case http.MethodPost:
			allow = "POST, OPTIONS"
		case http.MethodPatch, http.MethodDelete:
			allow = "PATCH, DELETE, OPTIONS"
		}
		headers.Set("Allow", allow)
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectStateReadsJSON(response, request, http.StatusNotFound, map[string]string{"detail": "Not found."})
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
			writeProjectStateReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		if values := request.URL.Query()["grouped"]; len(values) > 1 {
			query := request.URL.Query()
			query.Set("grouped", values[len(values)-1])
			request = request.Clone(request.Context())
			clonedURL := *request.URL
			request.URL = &clonedURL
			request.URL.RawQuery = query.Encode()
		}
		rawResponse := &systemapi.ProjectStateReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectStateReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectStateReadsResponseWriter{
			ResponseWriter: response,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectStateReadsJSON(response http.ResponseWriter, request *http.Request, status int, value any) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func projectStateReadsAllow(path string) string {
	if strings.Contains(path, "/intake-state/") {
		return "GET, HEAD, OPTIONS"
	}
	trimmed := strings.TrimSuffix(path, "/")
	if strings.HasSuffix(trimmed, "/states") {
		return "GET, POST, HEAD, OPTIONS"
	}
	return "GET, PATCH, DELETE, HEAD, OPTIONS"
}

func mountProjectStateReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.projectStateReads != nil {
		server := projectstatereadssvr.New(pending.projectStateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectStateReadsCompatibilityMiddleware)
		options := apiV1CompatibilityOptions{bodySizeLimit: pending.activityReadsBodyLimit, activityLogger: pending.activityReadsActivity, activitySecret: pending.activityReadsSecret}
		writes := sessionServiceMiddlewareStack(options, pending.readServicesSession, pending.readServicesCookie)
		server.Create = writes(server.Create)
		server.Update = writes(server.Update)
		server.Delete = writes(server.Delete)
		server.MarkDefault = writes(server.MarkDefault)
		projectstatereadssvr.Mount(mux, server)
		mountProjectStateReadsProtocolRoutes(mux, server, pending.projectStateMetadata)
	}
}

func mountAPIStateReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.apiStateReads != nil {
		states := apistatereadssvr.New(pending.apiStateReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		states.Use(stack)
		apistatereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDStatesHandler(mux, states.GetAPIV1WorkspacesBySlugProjectsByProjectIDStates, readServicesCORSPolicy)
		apistatereadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
		apistatereadssvr.MountCreateAPIV1WorkspacesBySlugProjectsByProjectIDStatesHandler(mux, states.CreateAPIV1WorkspacesBySlugProjectsByProjectIDStates, readServicesCORSPolicy)
		apistatereadssvr.MountUpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
		apistatereadssvr.MountDeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateIDHandler(mux, states.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID, readServicesCORSPolicy)
	}
}
