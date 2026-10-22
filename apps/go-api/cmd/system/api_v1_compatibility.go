package main

import (
	"bytes"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/platform"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
)

type apiV1CompatibilityOptions struct {
	activityLogger publicprojectmetadatareadsstore.ActivityLogger
	activitySecret string
	bodySizeLimit  int64
}

type apiV1BufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newAPIV1BufferedResponse() *apiV1BufferedResponse {
	return &apiV1BufferedResponse{header: make(http.Header)}
}

func (response *apiV1BufferedResponse) Header() http.Header { return response.header }

func (response *apiV1BufferedResponse) WriteHeader(status int) {
	if response.status == 0 {
		response.status = status
	}
}

func (response *apiV1BufferedResponse) Write(body []byte) (int, error) {
	if response.status == 0 {
		response.status = http.StatusOK
	}
	return response.body.Write(body)
}

// apiV1CompatibilityMiddleware owns behavior shared by every replacement for
// Django's external API: the browser-visible request endpoint, rate-limit
// response metadata, and the global X-Api-Key activity-log effect.
func apiV1CompatibilityMiddleware(options apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			mergeVary(response.Header(), "origin")
			request = request.WithContext(platform.WithPublicRequestEndpoint(request.Context(), publicRequestEndpoint(request)))

			requestBody, tooLarge, err := readPublicProjectMetadataRequestBody(request, options.bodySizeLimit)
			if tooLarge {
				headers := response.Header()
				headers.Set("Content-Type", "application/json")
				headers.Set("Content-Length", strconv.Itoa(len(djangoRequestTooLargeBody)))
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = response.Write(djangoRequestTooLargeBody)
				return
			}
			if err != nil {
				response.Header().Set("Content-Type", "text/html; charset=utf-8")
				response.Header().Set("Content-Length", strconv.Itoa(len(djangoServerErrorBody)))
				response.WriteHeader(http.StatusInternalServerError)
				if request.Method != http.MethodHead {
					_, _ = response.Write(djangoServerErrorBody)
				}
				return
			}
			if apiKey := djangoJoinedHeader(request, "X-Api-Key"); apiKey != "" {
				request.Header.Set("X-Api-Key", apiKey)
			}
			effects := &apikey.Effects{}
			request = request.WithContext(apikey.WithEffects(request.Context(), effects))
			captured := newAPIV1BufferedResponse()
			next.ServeHTTP(captured, request)
			if captured.status == 0 {
				captured.status = http.StatusOK
			}

			if effects.Rate != nil && !effects.Rate.Allowed && captured.status == http.StatusTooManyRequests {
				captured.body.Reset()
				_, _ = captured.body.WriteString(`{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}`)
				captured.header.Set("Content-Type", "application/json")
				captured.header.Del("Content-Length")
				captured.header.Del("Retry-After")
				if effects.Rate.Wait > 0 {
					captured.header.Set("Retry-After", strconv.Itoa(int(math.Ceil(effects.Rate.Wait.Seconds()))))
				}
			}
			if effects.Rate != nil && effects.Rate.Allowed {
				captured.header.Set("X-RateLimit-Remaining", effects.Rate.Remaining)
				captured.header.Set("X-RateLimit-Reset", strconv.FormatInt(effects.Rate.Reset, 10))
			}
			captured.header.Del("Loom-Error")
			removeVary(captured.header, "Cookie", "Origin")
			if effects.Authenticated || captured.status == http.StatusNotFound || activityReadsSlashlessProtocol(request.URL.Path) {
				mergeVary(captured.header, "Cookie", "origin")
			} else {
				mergeVary(captured.header, "origin")
			}
			if captured.status >= http.StatusMultipleChoices && captured.status < http.StatusBadRequest {
				captured.header.Set("Content-Type", "text/html; charset=utf-8")
			}
			if captured.header.Get("Content-Length") == "" {
				captured.header.Set("Content-Length", strconv.Itoa(captured.body.Len()))
			}
			// The activity boundary audits after content negotiation has rendered
			// the final body. Standalone callers still own their audit here.
			deferredAudit := apiV1AuditDeferred(request)
			if !deferredAudit {
				writeAPIActivityLog(request.Context(), options.activityLogger, options.activitySecret, request, requestBody,
					captured.status, captured.body.Bytes())
			}

			for name, values := range captured.header {
				response.Header()[name] = append([]string(nil), values...)
			}
			response.WriteHeader(captured.status)
			if request.Method != http.MethodHead || deferredAudit {
				_, _ = response.Write(captured.body.Bytes())
			}
		})
	}
}

func publicRequestEndpoint(request *http.Request) string {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		if forwarded == "https" {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + request.Host
}

// The outer audit capture consumes a rendered HEAD body, then suppresses it on
// the wire. Other API-v1 mounts retain their standalone compatibility behavior.
func apiV1AuditDeferred(request *http.Request) bool {
	_, captured := request.Context().Value(activityReadsAuditStateKey{}).(*activityReadsAuditState)
	return captured && strings.HasPrefix(request.URL.Path, "/api/v1/")
}
