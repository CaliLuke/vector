package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/loom/clue/log"
	loom "github.com/CaliLuke/loom/pkg"
)

var (
	readServicesNotAcceptableBody  = []byte(`{"detail":"Could not satisfy the request Accept header."}`)
	readServicesFormatNotFoundBody = []byte(`{"detail":"Not found."}`)
)

type readServicesJSONIndentResponseWriter struct {
	http.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (writer *readServicesJSONIndentResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *readServicesJSONIndentResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *readServicesJSONIndentResponseWriter) finish(indent int) {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if len(body) > 0 && strings.HasPrefix(strings.ToLower(writer.Header().Get("Content-Type")), "application/json") {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", indent)); err == nil {
			body = indented.Bytes()
			body = bytes.ReplaceAll(body, []byte("\u2028"), []byte(`\u2028`))
			body = bytes.ReplaceAll(body, []byte("\u2029"), []byte(`\u2029`))
			writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
	}
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func readServicesContentNegotiationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		formats, formatErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
		if formatErr != nil {
			log.Error(request.Context(), formatErr, log.KV{K: "msg", V: "parse read-service renderer format"})
			writeReadServiceNegotiationResponse(response, request, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			return
		}
		if len(formats) > 0 {
			format := formats[len(formats)-1]
			if format != "" && format != "json" {
				writeReadServiceNegotiationResponse(response, request, http.StatusNotFound, "application/json", readServicesFormatNotFoundBody)
				return
			}
		}
		accept, present := djangoRequestAcceptHeader(request)
		acceptsJSON, jsonIndent, err := projectMembershipReadsJSONNegotiation(accept, present)
		if err != nil {
			log.Error(request.Context(), err, log.KV{K: "msg", V: "negotiate read-service response"})
			writeReadServiceNegotiationResponse(response, request, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			return
		}
		if !acceptsJSON {
			writeReadServiceNegotiationResponse(response, request, http.StatusNotAcceptable, "application/json", readServicesNotAcceptableBody)
			return
		}
		if jsonIndent == 0 {
			next.ServeHTTP(response, request)
			return
		}
		writer := &readServicesJSONIndentResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		writer.finish(jsonIndent)
	})
}

func writeReadServiceNegotiationResponse(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	contentType string,
	body []byte,
) {
	// Django accesses the session while handling renderer rejection even before API authentication.
	if strings.HasPrefix(request.URL.Path, "/api/v1/") && (status == http.StatusNotFound || status == http.StatusNotAcceptable) {
		removeVary(response.Header(), "Cookie", "origin")
		mergeVary(response.Header(), "Cookie", "origin")
	}
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Content-Length", strconv.Itoa(len(body)))
	response.WriteHeader(status)
	if request.Method != http.MethodHead || apiV1AuditDeferred(request) {
		_, _ = response.Write(body)
	}
}

type readServicesGzipResponseWriter struct {
	http.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (writer *readServicesGzipResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
}

func (writer *readServicesGzipResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *readServicesGzipResponseWriter) finish(request *http.Request) {
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	body := writer.body.Bytes()
	if protocol, matched := resolveReadServiceProtocol(request.URL.Path); matched &&
		protocol.pattern == "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/" &&
		(request.Method == http.MethodGet || request.Method == http.MethodHead) &&
		status == http.StatusOK && writer.Header().Get("Content-Encoding") == "" {
		prependAcceptEncodingVary(writer.Header())
		if acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")) {
			if compressed, err := djangoGzip(body, nil); err == nil {
				writer.Header().Del("Content-Length")
				writer.Header().Set("Content-Encoding", "gzip")
				if request.Method == http.MethodHead {
					body = nil
				} else {
					body = compressed
				}
			}
		}
		writer.ResponseWriter.WriteHeader(status)
		if len(body) > 0 {
			_, _ = writer.ResponseWriter.Write(body)
		}
		return
	}
	representationLength := len(body)
	streamingHead := false
	if representationLength == 0 && request.Method == http.MethodHead {
		if length, err := strconv.Atoi(writer.Header().Get("Content-Length")); err == nil && length >= 200 {
			representationLength = length
			streamingHead = true
		}
	}
	if representationLength >= 200 && writer.Header().Get("Content-Encoding") == "" {
		prependAcceptEncodingVary(writer.Header())
		if acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")) {
			if streamingHead {
				writer.Header().Del("Content-Length")
				writer.Header().Set("Content-Encoding", "gzip")
			} else if compressed, err := djangoGzip(body, nil); err == nil && len(compressed) < len(body) {
				body = compressed
				writer.Header().Set("Content-Encoding", "gzip")
				writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
			}
		}
	}
	writer.ResponseWriter.WriteHeader(status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

func readServicesGzipMiddleware(next http.Handler) http.Handler {
	return readServicesGzipMiddlewareWithAuthBypass(next, true)
}

func readServicesGzipMiddlewareWithAuthBypass(next http.Handler, bypassAuthentication bool) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// Django's plain authentication views do not run DRF's response gzip
		// negotiation. Keep their headers and cookie responses on that contract.
		if bypassAuthentication && authResponseSessionPath(request) {
			next.ServeHTTP(response, request)
			return
		}
		writer := &readServicesGzipResponseWriter{ResponseWriter: response}
		next.ServeHTTP(writer, request)
		writer.finish(request)
	})
}

type readServicesResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (writer *readServicesResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *readServicesResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

type readServicesSessionResponseWriter struct {
	http.ResponseWriter
	request       *http.Request
	sessions      readServicesSessionLifecycle
	sessionCookie djangoSessionCookieConfig
	refresh       *djangosession.SessionRefresh
	wroteHeader   bool
	failed        bool
}

func (writer *readServicesSessionResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	override, expire, explicit := djangosession.ResponseSessionActionFromContext(writer.request.Context())
	if expire || status < http.StatusInternalServerError && (explicit || writer.refresh != nil) {
		var err error
		switch {
		case expire:
			err = expireDjangoSessionCookie(writer.Header(), writer.sessionCookie)
		case explicit:
			err = refreshDjangoSessionCookie(writer.Header(), writer.sessionCookie, *override)
		default:
			var refresh djangosession.SessionRefresh
			refresh, err = writer.sessions.RefreshSession(writer.request.Context(), writer.request, *writer.refresh)
			if err == nil {
				err = refreshDjangoSessionCookie(writer.Header(), writer.sessionCookie, refresh)
			}
		}
		if err != nil {
			logPublicProjectMetadataCompatibilityError(writer.request.Context(), err)
			writer.failed = true
			writeReadServicesSessionError(writer.ResponseWriter, writer.request, err)
			return
		}
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *readServicesSessionResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

func readServicesQueryPresenceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if systemapi.ReadServiceQueryValues(request.Context()) == nil {
			request = request.WithContext(systemapi.WithReadServiceQueryPresence(request.Context(), request.URL.Query()))
		}
		next.ServeHTTP(response, request)
	})
}

func readServicesSessionMiddleware(
	sessions readServicesSessionLifecycle,
	sessionCookie djangoSessionCookieConfig,
) func(http.Handler) http.Handler {
	return djangoSessionLifecycleMiddleware(sessions, sessionCookie, func(request *http.Request) bool {
		_, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		return matched
	})
}

func writeReadServicesSessionInspectionError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, djangosession.ErrTimezoneNotFound):
		writeReadServiceJSON(response, http.StatusBadRequest, map[string]string{
			"error": "The required key does not exist.",
		})
	case errors.Is(err, djangosession.ErrInvalidTimezone):
		writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{
			"error": "Something went wrong please try again later",
		})
	default:
		writeReadServicesSessionError(response, request)
	}
}

func writeReadServicesSessionError(response http.ResponseWriter, request *http.Request, causes ...error) {
	status := http.StatusInternalServerError
	body := djangoServerErrorBody
	if len(causes) != 0 && (errors.Is(causes[0], djangosession.ErrSessionNotFound) || errors.Is(causes[0], djangosession.ErrSessionUpdate)) {
		status = http.StatusBadRequest
		body = djangoBadRequestBody
	}
	headers := response.Header()
	headers.Del("Allow")
	headers.Del("Content-Encoding")
	headers.Del("Cache-Control")
	headers.Del("X-Frame-Options")
	removeVary(headers, "Cookie", "Accept-Encoding")
	headers.Set("Content-Type", "text/html; charset=utf-8")
	if status == http.StatusInternalServerError {
		headers.Set("Content-Length", strconv.Itoa(len(body)))
	} else {
		headers.Del("Content-Length")
	}
	response.WriteHeader(status)
	if request.Method != http.MethodHead {
		_, _ = response.Write(body)
	}
}

func markDjangoPageInternalError(endpoint loom.Endpoint) loom.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		result, err := endpoint(ctx, request)
		if errors.Is(err, pagereadsstore.ErrInvalidOrder) || errors.Is(err, pagereadsstore.ErrDjangoPageNil) {
			systemapi.MarkReadServiceDjangoHTML500(ctx)
		}
		return result, err
	}
}

type djangoReadServiceHTML500ResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	wroteHeader bool
	failed      bool
	omitAllow   bool
}

func (writer *djangoReadServiceHTML500ResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	if status == http.StatusInternalServerError && systemapi.ReadServiceDjangoHTML500(writer.request.Context()) {
		writer.failed = true
		headers := writer.Header()
		if writer.omitAllow {
			headers.Del("Allow")
		}
		headers.Del("Content-Encoding")
		headers.Del("loom-error")
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
		writer.ResponseWriter.WriteHeader(status)
		if writer.request.Method != http.MethodHead {
			_, _ = writer.ResponseWriter.Write(djangoServerErrorBody)
		}
		return
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *djangoReadServiceHTML500ResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.failed {
		return len(body), nil
	}
	return writer.ResponseWriter.Write(body)
}

type readServicesOriginalRawQueryKey struct{}

func readServicesOriginalRawQuery(request *http.Request) string {
	if rawQuery, ok := request.Context().Value(readServicesOriginalRawQueryKey{}).(string); ok {
		return rawQuery
	}
	return request.URL.RawQuery
}

// Capture once: outer activity compatibility may retain integer syntax before
// replacing it with transport-safe values, and audit/limit readers need that input.
func captureReadServicesOriginalRawQuery(request *http.Request) *http.Request {
	if _, captured := request.Context().Value(readServicesOriginalRawQueryKey{}).(string); captured {
		return request
	}
	return request.WithContext(context.WithValue(request.Context(), readServicesOriginalRawQueryKey{}, request.URL.RawQuery))
}

func readServicesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		if !matched {
			next.ServeHTTP(response, request)
			return
		}
		activityAPI := activityReadsAPIProtocol(protocol.pattern)
		activityRoute := activityAPI || protocol.pattern == "/api/users/me/activities/"
		preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
		if (activityRoute || protocol.authenticateMethod) && preflight {
			next.ServeHTTP(response, request)
			return
		}
		request = captureReadServicesOriginalRawQuery(request)
		request = normalizeDjangoTolerantQuery(request)
		request = normalizeReadServiceScalarQueryValues(request)
		setDjangoProtocolCompatibilityHeaders(response.Header(), protocol.allow)
		if isInstanceManagementPlainRedirect(protocol.pattern) && readServiceMethodAllowed(protocol.allow, request.Method) {
			// These three admin routes are plain Django Views. Their successful
			// form responses do not carry DRF's Allow header; unsupported-method
			// responses still retain the source-derived Allow value.
			response.Header().Del("Allow")
		}
		if activityAPI {
			removeVary(response.Header(), "Cookie")
		}
		if !activityRoute && !protocol.authenticateMethod && !readServiceMethodAllowed(protocol.allow, request.Method) {
			writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{
				"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
			})
			return
		}
		compatibleResponse := http.ResponseWriter(&readServicesResponseWriter{ResponseWriter: response})
		if (request.Method == http.MethodGet || request.Method == http.MethodHead) &&
			readServiceUsesDjangoHTML500(protocol.pattern) {
			request = request.WithContext(systemapi.WithReadServiceDjangoHTML500(request.Context()))
			compatibleResponse = &djangoReadServiceHTML500ResponseWriter{
				ResponseWriter: compatibleResponse,
				request:        request,
			}
		}
		next.ServeHTTP(compatibleResponse, request)
	})
}

func activityReadsAPIProtocol(pattern string) bool {
	return strings.HasPrefix(pattern, "/api/v1/")
}

func readServiceMethodAllowed(allow, method string) bool {
	for _, allowed := range strings.Split(allow, ",") {
		if strings.TrimSpace(allowed) == method {
			return true
		}
	}
	return false
}

func writeReadServiceJSON(response http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("marshal read-service protocol response: %w", err))
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_, _ = response.Write(body)
}

func readServiceUsesDjangoHTML500(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/":
		return true
	default:
		return false
	}
}

func setDjangoProtocolCompatibilityHeaders(headers http.Header, allow string) {
	headers.Set("Allow", allow)
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}
