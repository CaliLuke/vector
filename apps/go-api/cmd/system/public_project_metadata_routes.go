package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	publicprojectmetadatareadssvr "github.com/CaliLuke/go-plane/gen/http/public_project_metadata_reads/server"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
)

var publicMetadataCompatibilityPathPattern = regexp.MustCompile(`^(?:` + strings.TrimPrefix(strings.TrimSuffix(publicProjectMetadataAnyPathPattern.String(), "$"), "^") + `|` + strings.TrimPrefix(strings.TrimSuffix(publicFeedbackPathPattern.String(), "$"), "^") + `)$`)

func publicProjectMetadataMethodNotAllowed() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		writeStickyJSON(response, http.StatusMethodNotAllowed, map[string]string{
			"detail": fmt.Sprintf("Method %q not allowed.", request.Method),
		})
	})
}

var (
	publicProjectMetadataPathPattern          = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)/$`)
	publicProjectMetadataSlashlessPathPattern = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)$`)
	publicProjectMetadataAnyPathPattern       = regexp.MustCompile(`^(?:/api/public/anchor/[^/]+/(?:cycles|labels|members|meta|modules|settings|states)|/api/public/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/anchor)/?$`)
)

func publicProjectMetadataRouteCompatibilityMiddleware(
	compatibility func(http.Handler) http.Handler,
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	debugMode bool,
) func(http.Handler) http.Handler {
	fallback := compatibility(publicProjectMetadataMethodNotAllowed())
	slashless := publicProjectMetadataNotFoundHandler(
		throttle, allowedCORSOrigins, sessionCookie, activityLogger, activitySecret, bodySizeLimit, debugMode, true,
	)
	malformed := publicProjectMetadataNotFoundHandler(
		throttle, allowedCORSOrigins, sessionCookie, activityLogger, activitySecret, bodySizeLimit, debugMode, false,
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			trailingSlash := publicProjectMetadataPathPattern.MatchString(request.URL.Path)
			slashlessPath := publicProjectMetadataSlashlessPathPattern.MatchString(request.URL.Path)
			if !trailingSlash && !slashlessPath {
				cleanedPath := path.Clean(request.URL.Path)
				if publicProjectMetadataAnyPathPattern.MatchString(cleanedPath) {
					malformed.ServeHTTP(response, request)
					return
				}
				next.ServeHTTP(response, request)
				return
			}
			canonicalMethod := strings.ToUpper(request.Method)
			if canonicalMethod != request.Method {
				request = request.Clone(request.Context())
				request.Method = canonicalMethod
			}
			if slashlessPath {
				slashless.ServeHTTP(response, request)
				return
			}
			if canonicalMethod != http.MethodGet && canonicalMethod != http.MethodHead && canonicalMethod != http.MethodOptions {
				fallback.ServeHTTP(response, request)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func publicProjectMetadataNotFoundHandler(
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	debugMode bool,
	appendSlash bool,
) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		if publicProjectMetadataReadsCORSPreflight(headers, request, allowedCORSOrigins) {
			response.WriteHeader(http.StatusOK)
			return
		}
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
		requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
		if tooLarge {
			writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, throttle)
			return
		}
		if bodyErr != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
			cookieErr := expireUnaccessedDjangoSessionCookie(headers, request, sessionCookie)
			if cookieErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
				headers.Del("X-Frame-Options")
			}
			mergeVary(headers, "origin")
			headers.Set("Content-Type", "text/html; charset=utf-8")
			headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write(djangoServerErrorBody)
			return
		}
		mergeVary(headers, "Cookie", "origin")
		requestLogStarted := time.Now()
		writer := &publicProjectMetadataReadsResponseWriter{
			ResponseWriter: response,
			acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
		}
		expireSession, refreshSession, rotatedSession, userID, err := throttle.InspectSession(request.Context(), request)
		var sessionCookieErr error
		if expireSession {
			sessionCookieErr = expireDjangoSessionCookie(headers, sessionCookie)
		}
		if err != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), err)
			replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
			writeAPIActivityLog(
				request.Context(), activityLogger, activitySecret, request, requestBody,
				http.StatusInternalServerError, djangoServerErrorBody,
			)
			writer.finish()
			return
		}
		logPublicProjectMetadataInnerResponse(request, http.StatusNotFound, requestLogStarted, userID)
		writeAPIActivityLog(
			request.Context(), activityLogger, activitySecret, request, requestBody,
			http.StatusNotFound, djangoNotFoundBody,
		)
		if sessionCookieErr != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), sessionCookieErr)
			replacePublicProjectMetadataSessionError(writer, sessionCookieErr)
			writer.finish()
			return
		}
		if appendSlash && debugMode && (request.Method == http.MethodPost || request.Method == http.MethodPut ||
			request.Method == http.MethodPatch || request.Method == http.MethodDelete) {
			debugBody, debugContentType := djangoDebugAppendSlashResponse(request)
			// CommonMiddleware raises after the inner GZip and XFrame middleware
			// have returned, so neither one processes Django's technical 500.
			headers.Del("X-Frame-Options")
			writer.skipGzip = true
			writer.omitLength = true
			replacePublicProjectMetadataResponse(
				writer, http.StatusInternalServerError, debugContentType, debugBody,
			)
			writer.finish()
			return
		}
		var preparedSessionRefresh *djangosession.SessionRefresh
		if rotatedSession != nil {
			preparedSessionRefresh = rotatedSession
		} else if refreshSession {
			if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
				preparedSessionRefresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
			}
		}
		if preparedSessionRefresh != nil {
			refresh, refreshErr := throttle.RefreshSession(request.Context(), request, *preparedSessionRefresh)
			if refreshErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), refreshErr)
				replacePublicProjectMetadataSessionError(writer, refreshErr)
				writer.finish()
				return
			}
			if cookieErr := refreshDjangoSessionCookie(headers, sessionCookie, refresh); cookieErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
				replacePublicProjectMetadataSessionError(writer, cookieErr)
				writer.finish()
				return
			}
		}
		if !appendSlash {
			headers.Set("Content-Type", "text/html; charset=utf-8")
			writer.WriteHeader(http.StatusNotFound)
			_, _ = writer.Write(djangoNotFoundBody)
			writer.finish()
			return
		}
		location := djangoFullPath(request, true)
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Location", location)
		headers.Del("X-Frame-Options")
		writer.omitLength = true
		writer.WriteHeader(http.StatusMovedPermanently)
		writer.finish()
	})
}

func readPublicProjectMetadataRequestBody(request *http.Request, limit int64) ([]byte, bool, error) {
	// Django's body property enforces DATA_UPLOAD_MAX_MEMORY_SIZE solely from
	// the declared CONTENT_LENGTH, treating a missing value as zero. Unknown-
	// length and under-declared streams are read in full without a second check.
	declaredLength := request.ContentLength
	if declaredLength < 0 {
		declaredLength = 0
	}
	if declaredLength > limit {
		return nil, true, nil
	}
	if request.Body == nil {
		return nil, false, nil
	}
	body, err := io.ReadAll(request.Body)
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("read request body: %w", err)
	}
	return body, false, nil
}

func logPublicProjectMetadataInnerResponse(request *http.Request, status int, started time.Time, userID string) {
	originalRequest := request.Clone(request.Context())
	originalRequest.URL.RawQuery = readServicesOriginalRawQuery(request)
	log.Print(
		request.Context(),
		log.KV{K: "msg", V: fmt.Sprintf("%s %s %d", request.Method, djangoFullPath(originalRequest, false), status)},
		log.KV{K: "path", V: request.URL.Path},
		log.KV{K: "method", V: request.Method},
		log.KV{K: "status_code", V: status},
		log.KV{K: "duration_ms", V: time.Since(started).Milliseconds()},
		log.KV{K: "remote_addr", V: apiActivityIPAddress(request)},
		log.KV{K: "user_agent", V: djangoJoinedHeader(request, "User-Agent")},
		log.KV{K: "user_id", V: nullableString(userID)},
	)
}

func logPublicProjectMetadataCompatibilityError(ctx context.Context, err error) {
	log.Error(ctx, err, log.KV{K: "msg", V: "public project metadata compatibility failure"})
}

type publicProjectMetadataReadsResponseWriter struct {
	http.ResponseWriter
	rawResponse      *systemapi.PublicProjectMetadataReadsResponseCapture
	capturedResponse func() json.RawMessage
	jsonIndent       int
	status           int
	wroteHeader      bool
	body             bytes.Buffer
	acceptsGzip      bool
	appendGzipVary   bool
	skipCookieVary   bool
	skipGzip         bool
	omitLength       bool
	gzipPadding      func() (int, error)
}

func (writer *publicProjectMetadataReadsResponseWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("loom-error")
	if !writer.skipCookieVary {
		mergeVary(writer.Header(), "Cookie")
	}
}

func (writer *publicProjectMetadataReadsResponseWriter) Write(body []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

func (writer *publicProjectMetadataReadsResponseWriter) renderedBody() []byte {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if writer.rawResponse != nil && writer.rawResponse.RenderError {
		writer.status = http.StatusInternalServerError
		writer.Header().Del("Allow")
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		return djangoServerErrorBody
	}
	body := writer.body.Bytes()
	if writer.status == http.StatusOK && writer.rawResponse != nil && len(writer.rawResponse.Data) > 0 {
		body = writer.rawResponse.Data
	}
	if writer.status == http.StatusOK && writer.capturedResponse != nil {
		if captured := writer.capturedResponse(); len(captured) > 0 {
			body = captured
		}
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, bytes.TrimSpace(body)); err == nil {
		body = compacted.Bytes()
	}
	if writer.jsonIndent > 0 {
		var indented bytes.Buffer
		if err := json.Indent(&indented, bytes.TrimSpace(body), "", strings.Repeat(" ", writer.jsonIndent)); err == nil {
			body = indented.Bytes()
		}
	}
	// DRF's JSONRenderer escapes JavaScript line and paragraph separators even
	// with ensure_ascii disabled.
	body = bytes.ReplaceAll(body, []byte("\u2028"), []byte(`\u2028`))
	body = bytes.ReplaceAll(body, []byte("\u2029"), []byte(`\u2029`))
	return body
}

func (writer *publicProjectMetadataReadsResponseWriter) finish() {
	body := writer.renderedBody()
	if !writer.skipGzip && len(body) >= 200 {
		if writer.appendGzipVary {
			mergeVary(writer.Header(), "Accept-Encoding")
		} else {
			prependAcceptEncodingVary(writer.Header())
		}
		if writer.acceptsGzip && writer.Header().Get("Content-Encoding") == "" {
			compressed, err := djangoGzip(body, writer.gzipPadding)
			if err == nil && len(compressed) < len(body) {
				body = compressed
				writer.Header().Set("Content-Encoding", "gzip")
			}
		}
	}
	if writer.omitLength {
		writer.Header().Del("Content-Length")
	} else {
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	writer.ResponseWriter.WriteHeader(writer.status)
	if len(body) > 0 {
		_, _ = writer.ResponseWriter.Write(body)
	}
}

// publicMetadataSessionEffects retains cookie changes until audit logging has
// captured the inner response. Refresh failures can then replace the response.
type publicMetadataSessionEffects struct {
	refresh   *djangosession.SessionRefresh
	userID    string
	cookieErr error
}

func (effects *publicMetadataSessionEffects) apply(headers http.Header, request *http.Request, sessionCookie djangoSessionCookieConfig, expireSession, refreshSession bool, rotatedSession *djangosession.SessionRefresh, resolvedUserID string) {
	if resolvedUserID != "" {
		effects.userID = resolvedUserID
	}
	if expireSession {
		effects.cookieErr = expireDjangoSessionCookie(headers, sessionCookie)
	}
	if rotatedSession != nil {
		effects.refresh = rotatedSession
	} else if refreshSession {
		if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
			effects.refresh = &djangosession.SessionRefresh{SessionKey: cookie.Value}
		}
	}
}

func (effects *publicMetadataSessionEffects) finish(writer *publicProjectMetadataReadsResponseWriter, request *http.Request, throttle publicProjectMetadataThrottle, sessionCookie djangoSessionCookieConfig) {
	headers := writer.Header()
	if effects.cookieErr != nil {
		logPublicProjectMetadataCompatibilityError(request.Context(), effects.cookieErr)
		replacePublicProjectMetadataSessionError(writer, effects.cookieErr)
	} else if effects.refresh != nil && writer.status < http.StatusInternalServerError {
		refresh, err := throttle.RefreshSession(request.Context(), request, *effects.refresh)
		if err != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), err)
			replacePublicProjectMetadataSessionError(writer, err)
		} else if cookieErr := refreshDjangoSessionCookie(headers, sessionCookie, refresh); cookieErr != nil {
			logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
			replacePublicProjectMetadataSessionError(writer, cookieErr)
		}
	}
	writer.finish()
}

func publicProjectMetadataReadsCompatibilityMiddleware(
	throttle publicProjectMetadataThrottle,
	allowedCORSOrigins []string,
	sessionCookie djangoSessionCookieConfig,
	activityLogger publicprojectmetadatareadsstore.ActivityLogger,
	activitySecret string,
	bodySizeLimit int64,
	manageCORS bool,
	activateTimezone bool,
	routePolicies ...publicProjectMetadataRoutePolicy,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			if manageCORS && publicProjectMetadataReadsCORSPreflight(headers, request, allowedCORSOrigins) {
				response.WriteHeader(http.StatusOK)
				return
			}
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			if manageCORS {
				publicProjectMetadataReadsCORSHeaders(headers, request, allowedCORSOrigins)
			}
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			requestBody, tooLarge, bodyErr := readPublicProjectMetadataRequestBody(request, bodySizeLimit)
			if tooLarge {
				writePublicProjectMetadataRequestTooLarge(response, request, sessionCookie, throttle)
				return
			}
			if bodyErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), bodyErr)
				cookieErr := expireUnaccessedDjangoSessionCookie(headers, request, sessionCookie)
				if cookieErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
					headers.Del("X-Frame-Options")
				}
				mergeVary(headers, "origin")
				headers.Set("Content-Type", "text/html; charset=utf-8")
				headers.Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write(djangoServerErrorBody)
				return
			}
			mergeVary(headers, "Cookie", "origin")
			requestLogStarted := time.Now()
			allow := "GET, HEAD, OPTIONS"
			var routePolicy *publicProjectMetadataRoutePolicy
			if len(routePolicies) > 0 {
				routePolicy = &routePolicies[0]
				if routePolicy.allowMethods != nil {
					allow = routePolicy.allowMethods(request)
				}
			}
			headers.Set("Allow", allow)
			rawResponse := &systemapi.PublicProjectMetadataReadsResponseCapture{}
			writer := &publicProjectMetadataReadsResponseWriter{
				ResponseWriter: response,
				rawResponse:    rawResponse,
				acceptsGzip:    acceptsGzipEncoding(djangoJoinedHeader(request, "Accept-Encoding")),
			}
			var effects publicMetadataSessionEffects
			inspectSessionAfterResponse := false
			skipInnerRequestLog := false
			applySessionEffects := func(expireSession bool, refreshSession bool, rotatedSession *djangosession.SessionRefresh, resolvedUserID string) {
				effects.apply(headers, request, sessionCookie, expireSession, refreshSession, rotatedSession, resolvedUserID)
			}
			defer func() {
				if inspectSessionAfterResponse {
					expireSession, refreshSession, rotatedSession, resolvedUserID, err := throttle.InspectSession(request.Context(), request)
					applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
					if err != nil {
						logPublicProjectMetadataCompatibilityError(request.Context(), err)
						skipInnerRequestLog = true
						writePublicProjectMetadataInternalError(writer, err, false, allow)
					}
				}
				innerBody := writer.renderedBody()
				if !skipInnerRequestLog {
					logPublicProjectMetadataInnerResponse(request, writer.status, requestLogStarted, effects.userID)
				}
				writeAPIActivityLog(
					request.Context(), activityLogger, activitySecret, request, requestBody,
					writer.status, innerBody,
				)
				effects.finish(writer, request, throttle, sessionCookie)
			}()
			formats, queryErr := djangoQueryValues(readServicesOriginalRawQuery(request), "format")
			if queryErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), queryErr)
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if len(formats) > 0 {
				format := formats[len(formats)-1]
				if format != "" && format != "json" {
					inspectSessionAfterResponse = true
					headers.Set("Content-Type", "application/json")
					writeStickyJSON(writer, http.StatusNotFound, map[string]string{"detail": "Not found."})
					return
				}
			}
			accept, acceptPresent := djangoRequestAcceptHeader(request)
			acceptsJSON, jsonIndent, negotiationErr := projectMembershipReadsJSONNegotiation(accept, acceptPresent)
			if negotiationErr != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), negotiationErr)
				inspectSessionAfterResponse = true
				writeStickyJSON(writer, http.StatusInternalServerError, map[string]string{
					"error": "Something went wrong please try again later",
				})
				return
			}
			if !acceptsJSON {
				inspectSessionAfterResponse = true
				headers.Set("Content-Type", "application/json")
				writeStickyJSON(writer, http.StatusNotAcceptable, map[string]string{
					"detail": "Could not satisfy the request Accept header.",
				})
				return
			}
			writer.jsonIndent = jsonIndent
			if routePolicy != nil && routePolicy.requiresSession != nil && routePolicy.requiresSession(request) {
				lifecycle, ok := throttle.(userIdentitySessionLifecycle)
				if !ok {
					err := errors.New("public feedback required-session route needs session inspection")
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				expireSession, refreshSession, rotatedSession, resolvedUserID, err := lifecycle.InspectSessionWithTimezone(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					skipInnerRequestLog = publicProjectMetadataLazyAuthenticationError(err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				if resolvedUserID == "" {
					writeStickyJSON(writer, http.StatusUnauthorized, map[string]string{
						"detail": "Authentication credentials were not provided.",
					})
					return
				}
				request = request.WithContext(publicprojectmetadatareadsstore.WithUserID(request.Context(), resolvedUserID))
			} else {
				allowRequest := throttle.AllowRequestWithoutTimezone
				if activateTimezone {
					allowRequest = throttle.AllowRequest
				}
				allowed, wait, expireSession, refreshSession, rotatedSession, resolvedUserID, err := allowRequest(request.Context(), request)
				applySessionEffects(expireSession, refreshSession, rotatedSession, resolvedUserID)
				if err != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), err)
					skipInnerRequestLog = publicProjectMetadataLazyAuthenticationError(err)
					writePublicProjectMetadataInternalError(writer, err, true, allow)
					return
				}
				if !allowed {
					seconds := int(math.Ceil(wait.Seconds()))
					writer.Header().Set("Retry-After", strconv.Itoa(seconds))
					writeStickyJSON(writer, http.StatusTooManyRequests, struct {
						ErrorCode    int    `json:"error_code"`
						ErrorMessage string `json:"error_message"`
					}{
						ErrorCode: 5900, ErrorMessage: "RATE_LIMIT_EXCEEDED",
					})
					return
				}
				request = request.WithContext(publicprojectmetadatareadsstore.WithUserID(request.Context(), resolvedUserID))
			}
			request = request.WithContext(systemapi.WithPublicProjectMetadataReadsResponseCapture(request.Context(), rawResponse))
			next.ServeHTTP(writer, request)
		})
	}
}

type publicProjectMetadataRoutePolicy struct {
	allowMethods    func(*http.Request) string
	requiresSession func(*http.Request) bool
}

func writePublicProjectMetadataInternalError(writer *publicProjectMetadataReadsResponseWriter, err error, viewHandled bool, allowMethods ...string) {
	if !viewHandled || publicProjectMetadataLazyAuthenticationError(err) {
		replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "text/html; charset=utf-8", djangoServerErrorBody)
		return
	}
	if errors.Is(err, djangosession.ErrTimezoneNotFound) {
		replacePublicProjectMetadataResponse(writer, http.StatusBadRequest, "application/json", []byte(`{"error":"The required key does not exist."}`))
		writer.Header().Set("Allow", publicProjectMetadataErrorAllow(allowMethods))
		return
	}
	replacePublicProjectMetadataResponse(writer, http.StatusInternalServerError, "application/json", []byte(`{"error":"Something went wrong please try again later"}`))
	writer.Header().Set("Allow", publicProjectMetadataErrorAllow(allowMethods))
}

func publicProjectMetadataErrorAllow(allowMethods []string) string {
	if len(allowMethods) > 0 && allowMethods[0] != "" {
		return allowMethods[0]
	}
	return "GET, HEAD, OPTIONS"
}

func publicProjectMetadataLazyAuthenticationError(err error) bool {
	return djangosession.IsReadFailure(err)
}

func replacePublicProjectMetadataResponse(writer *publicProjectMetadataReadsResponseWriter, status int, contentType string, body []byte) {
	writer.body.Reset()
	writer.rawResponse = nil
	writer.wroteHeader = true
	writer.status = status
	writer.Header().Del("Allow")
	writer.Header().Del("Content-Encoding")
	writer.Header().Del("Cache-Control")
	writer.Header().Set("Content-Type", contentType)
	_, _ = writer.body.Write(body)
}

func replacePublicProjectMetadataSessionError(writer *publicProjectMetadataReadsResponseWriter, err error) {
	status := http.StatusInternalServerError
	body := djangoServerErrorBody
	if errors.Is(err, djangosession.ErrSessionNotFound) || errors.Is(err, djangosession.ErrSessionUpdate) {
		status = http.StatusBadRequest
		body = djangoBadRequestBody
	}
	replacePublicProjectMetadataResponse(writer, status, "text/html; charset=utf-8", body)
	// SessionMiddleware raises outside CommonMiddleware, XFrameOptionsMiddleware,
	// and GZipMiddleware. Their headers and body processing do not apply to the
	// replacement error response.
	writer.Header().Del("X-Frame-Options")
	writer.Header().Del("Location")
	writer.Header().Del("Retry-After")
	removeVary(writer.Header(), "Cookie", "Accept-Encoding")
	writer.skipGzip = true
	writer.omitLength = true
}

func writePublicProjectMetadataRequestTooLarge(
	response http.ResponseWriter,
	request *http.Request,
	sessionCookie djangoSessionCookieConfig,
	throttle publicProjectMetadataThrottle,
) {
	headers := response.Header()
	expireSession, refreshSession, err := throttle.FinalizeUnaccessedSession(
		request.Context(), request, http.StatusRequestEntityTooLarge,
	)
	if expireSession {
		if cookieErr := expireDjangoSessionCookie(headers, sessionCookie); cookieErr != nil {
			mergeVary(headers, "origin")
			logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
			writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
			replacePublicProjectMetadataSessionError(writer, cookieErr)
			writer.finish()
			return
		}
	}
	if err != nil {
		mergeVary(headers, "origin")
		logPublicProjectMetadataCompatibilityError(request.Context(), err)
		writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
		replacePublicProjectMetadataSessionError(writer, err)
		writer.finish()
		return
	}
	if refreshSession != nil {
		if cookieErr := refreshUnaccessedDjangoSessionCookie(headers, sessionCookie, *refreshSession); cookieErr != nil {
			mergeVary(headers, "origin")
			logPublicProjectMetadataCompatibilityError(request.Context(), cookieErr)
			writer := &publicProjectMetadataReadsResponseWriter{ResponseWriter: response}
			replacePublicProjectMetadataSessionError(writer, cookieErr)
			writer.finish()
			return
		}
	}
	mergeVary(headers, "origin")
	headers.Set("Content-Type", "application/json")
	headers.Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
	response.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = response.Write(djangoRequestTooLargeBody)
}

func mountPublicProjectMetadataReadsRoutes(dbg bool, pending pendingReplacementEndpoints, mux loomhttp.Muxer) func(http.Handler) http.Handler {
	var publicProjectMetadataRouteCompatibility func(http.Handler) http.Handler
	if pending.publicProjectMetadataReads != nil {
		if pending.publicProjectMetadataThrottle == nil {
			panic("public project metadata throttle is required")
		}
		bodySizeLimit := pending.publicProjectMetadataBodyLimit
		if !pending.publicProjectMetadataBodySet {
			bodySizeLimit = defaultFileSizeLimit
		}
		compatibilityMiddleware := publicProjectMetadataReadsCompatibilityMiddleware(
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			true,
			true,
		)
		server := publicprojectmetadatareadssvr.New(pending.publicProjectMetadataReads, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(compatibilityMiddleware)
		publicprojectmetadatareadssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/cycles/", djangoBufferedProtocolHeadHandler(server.Cycles))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/labels/", djangoBufferedProtocolHeadHandler(server.Labels))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/members/", djangoBufferedProtocolHeadHandler(server.Members))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/meta/", djangoBufferedProtocolHeadHandler(server.Meta))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/modules/", djangoBufferedProtocolHeadHandler(server.Modules))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/settings/", djangoBufferedProtocolHeadHandler(server.Settings))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/anchor/{anchor}/states/", djangoBufferedProtocolHeadHandler(server.States))
		mountProtocolHandler(mux, http.MethodHead, "/api/public/workspaces/{slug}/projects/{project_id}/anchor/", djangoBufferedProtocolHeadHandler(server.ProjectAnchor))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/cycles/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Cycles Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/labels/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Labels Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/members/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Members Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/meta/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Meta Data Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/modules/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Modules Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/settings/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project Deploy Board Public Settings Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/anchor/{anchor}/states/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Project States Endpoint"))
		mountProtocolHandler(mux, http.MethodOptions, "/api/public/workspaces/{slug}/projects/{project_id}/anchor/", djangoProtocolOptionsWithoutActionsHandler(compatibilityMiddleware, "Workspace Project Anchor Endpoint"))
		publicProjectMetadataRouteCompatibility = publicProjectMetadataRouteCompatibilityMiddleware(
			compatibilityMiddleware,
			pending.publicProjectMetadataThrottle,
			pending.publicProjectMetadataCORS,
			pending.publicProjectMetadataCookie,
			pending.publicProjectMetadataActivity,
			pending.publicProjectMetadataSecret,
			bodySizeLimit,
			dbg,
		)
	}
	return publicProjectMetadataRouteCompatibility
}
