package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	apilabelreadssvr "github.com/CaliLuke/go-plane/apilabelreads/gen/http/api_label_reads/server"
	projectlabelreadssvr "github.com/CaliLuke/go-plane/gen/http/project_label_reads/server"
	labelwritessvr "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type projectLabelReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse *systemapi.ProjectLabelReadsResponseCapture
	jsonIndent  int
	status      int
	wroteHeader bool
}

func (writer *projectLabelReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	mergeVary(writer.Header(), "Cookie")
	if status == http.StatusOK {
		mergeVary(writer.Header(), "Accept-Encoding")
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *projectLabelReadsResponseWriter) Write(body []byte) (int, error) {
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

func projectLabelReadsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", projectLabelReadsAllow(request.URL.Path))
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		if formats := request.URL.Query()["format"]; len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				headers.Set("Content-Type", "application/json")
				writeProjectLabelReadsJSON(response, request, http.StatusNotFound, map[string]string{"detail": "Not found."})
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
			writeProjectLabelReadsJSON(response, request, http.StatusNotAcceptable, map[string]string{
				"detail": "Could not satisfy the request Accept header.",
			})
			return
		}
		rawResponse := &systemapi.ProjectLabelReadsResponseCapture{}
		request = request.WithContext(systemapi.WithProjectLabelReadsResponseCapture(request.Context(), rawResponse))
		next.ServeHTTP(&projectLabelReadsResponseWriter{
			ResponseWriter: response,
			rawResponse:    rawResponse,
			jsonIndent:     jsonIndent,
		}, request)
	})
}

func writeProjectLabelReadsJSON(response http.ResponseWriter, request *http.Request, status int, value any) {
	if request.Method == http.MethodHead {
		writeStickyJSON(workspaceSlugHeadWriter{response}, status, value)
		return
	}
	writeStickyJSON(response, status, value)
}

func projectLabelReadsAllow(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	if strings.HasSuffix(trimmed, "/issue-labels") {
		return "GET, POST, HEAD, OPTIONS"
	}
	return "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
}

func mountProjectLabelReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.projectLabelReads != nil {
		server := projectlabelreadssvr.New(pending.projectLabelReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(projectLabelReadsCompatibilityMiddleware)
		projectlabelreadssvr.Mount(mux, server)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/issue-labels/", server.List)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/", server.Detail)
	}
}

func mountProjectLabelWritesRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.projectLabelWrites != nil {
		server := labelwritessvr.New(pending.projectLabelWrites, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		server.Use(stack)
		labelwritessvr.Mount(mux, server)
	}
}

func mountAPILabelReadsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler) {
	if pending.apiLabelReads != nil {
		labels := apilabelreadssvr.New(pending.apiLabelReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		labels.Use(stack)
		apilabelreadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsHandler(mux, labels.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels, readServicesCORSPolicy)
		apilabelreadssvr.MountGetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPkHandler(mux, labels.GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPk, readServicesCORSPolicy)
	}
}
