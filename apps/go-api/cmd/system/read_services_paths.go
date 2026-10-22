package main

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	loomhttp "github.com/CaliLuke/loom/http"
)

func readServiceProtocolHasMalformedUUID(requestPath string, protocol readServiceProtocol) bool {
	// DRF routers accept arbitrary non-dot lookup strings, unlike Django UUID converters.
	// Sticky lookup conversion runs in the service after API-key and workspace
	// permissions, so this middleware must not preempt malformed IDs.
	if strings.HasPrefix(protocol.pattern, "/api/v1/workspaces/{slug}/stickies/") {
		return false
	}
	segments := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(segments) != len(protocol.segments) {
		return false
	}
	for index, patternSegment := range protocol.segments {
		if slices.Contains(protocol.stringPathParameters, strings.Trim(patternSegment, "{}")) {
			continue
		}
		if !strings.HasPrefix(patternSegment, "{") || patternSegment == "{slug}" || patternSegment == "{anchor}" || patternSegment == "{reaction_code}" || patternSegment == "{asset_key}" {
			continue
		}
		if protocol.apiAuthorization == "invitations" && activityReadsAPIProtocol(protocol.pattern) {
			if strings.Contains(segments[index], ".") {
				return true
			}
			continue
		}
		if patternSegment == "{identifier}" {
			value := segments[index]
			valid := false
			for i := len(value) - 2; i > 0; i-- {
				if value[i] == '-' {
					valid = true
					break
				}
			}
			if !valid {
				return true
			}
			continue
		}
		if !djangoUUIDPathSegment(segments[index]) {
			return true
		}
	}
	return false
}

func apiV1StickyFormatSuffix(requestPath string) (string, bool, bool) {
	segments := strings.Split(strings.Trim(requestPath, "/"), "/")
	if len(segments) < 5 || segments[0] != "api" || segments[1] != "v1" || segments[2] != "workspaces" || segments[3] == "" {
		return "", false, false
	}
	if len(segments) == 5 {
		const prefix = "stickies."
		if !strings.HasPrefix(segments[4], prefix) {
			return "", false, false
		}
		format := strings.TrimPrefix(segments[4], prefix)
		return format, false, stickyFormatToken(format)
	}
	if len(segments) != 6 || segments[4] != "stickies" {
		return "", false, false
	}
	pk, format, found := strings.Cut(segments[5], ".")
	if !found || pk == "" || strings.Contains(format, ".") {
		return "", true, false
	}
	return format, true, stickyFormatToken(format)
}

func stickyFormatToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < 'a' || character > 'z' {
			if character < '0' || character > '9' {
				return false
			}
		}
	}
	return true
}

func djangoUUIDPathSegment(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for index := 0; index < len(value); index++ {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		character := value[index]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

var readServiceScalarQueryParameters = map[string]struct{}{
	"assignee_id": {}, "assignee_id__in": {}, "assignees": {}, "completed_at": {},
	"created_at": {}, "created_at__exact": {}, "created_at__range": {}, "created_by": {},
	"created_by_id": {}, "created_by_id__in": {}, "cursor": {}, "cycle": {}, "cycle_id__in": {},
	"cycle_view": {}, "entity_name": {}, "estimate_point": {}, "expand": {}, "fields": {}, "filters": {}, "format": {}, "group_by": {}, "grouped": {}, "inbox_status": {},
	"intake_status": {}, "is_archived": {}, "is_draft": {}, "label_id": {}, "label_id__in": {},
	"labels": {}, "logged_by": {}, "mention_id": {}, "mention_id__in": {}, "mentions": {},
	"module": {}, "module_id": {}, "module_id__in": {}, "name": {}, "order_by": {}, "owner": {},
	"parent": {}, "per_page": {}, "priority": {}, "priority__in": {}, "project": {},
	"project_id__in": {}, "project_ids": {}, "query": {}, "search": {}, "slug": {}, "start_date": {},
	"start_date__exact": {}, "start_date__range": {}, "start_target_date": {}, "state": {},
	"state_group": {}, "state_group__in": {}, "state_id": {}, "state_id__in": {}, "status": {},
	"sub_group_by": {}, "sub_issue": {}, "subscriber": {}, "subscriber_id": {},
	"subscriber_id__in": {}, "target_date": {}, "target_date__exact": {}, "target_date__range": {},
	"snoozed": {}, "archived": {}, "read": {}, "mentioned": {},
	"track_visit": {}, "type": {}, "updated_at": {}, "updated_at__exact": {}, "updated_at__range": {},
}

func normalizeReadServiceScalarQueryValues(request *http.Request) *http.Request {
	values := request.URL.Query()
	changed := false
	for key, entries := range values {
		if len(entries) < 2 {
			continue
		}
		if _, scalar := readServiceScalarQueryParameters[key]; !scalar {
			continue
		}
		values[key] = entries[len(entries)-1:]
		changed = true
	}
	if !changed {
		return request
	}
	clone := request.Clone(request.Context())
	clonedURL := *request.URL
	clone.URL = &clonedURL
	clone.URL.RawQuery = values.Encode()
	return clone
}

func readServicesDjangoPathConverterMiddleware(policy loomhttp.RuntimeCORSPolicy, activityOptions *apiV1CompatibilityOptions) func(http.Handler) http.Handler {
	notFound := policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Content-Type", "text/html; charset=utf-8")
		headers.Set("Content-Length", strconv.Itoa(len(djangoNotFoundBody)))
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		removeVary(headers, "Origin")
		mergeVary(headers, "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		response.WriteHeader(http.StatusNotFound)
		if request.Method != http.MethodHead {
			_, _ = response.Write(djangoNotFoundBody)
		}
	}))
	activityNotFound := http.Handler(notFound)
	if activityOptions != nil {
		activityNotFound = apiV1CompatibilityMiddleware(*activityOptions)(policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			headers := response.Header()
			headers.Set("Cross-Origin-Opener-Policy", "same-origin")
			headers.Set("Referrer-Policy", "same-origin")
			mergeVary(headers, "Cookie", "origin")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			writeReadServiceJSON(response, http.StatusNotFound, map[string]string{"error": "Page not found."})
		})))
	}
	return func(next http.Handler) http.Handler {
		stickyFormatNotFound := policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			_, detail, _ := apiV1StickyFormatSuffix(request.URL.Path)
			allow := "GET, POST, HEAD, OPTIONS"
			if detail {
				allow = "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"
			}
			setDjangoProtocolCompatibilityHeaders(response.Header(), allow)
			writeReadServiceNegotiationResponse(response, request, http.StatusNotFound, "application/json", readServicesFormatNotFoundBody)
		}))
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if format, _, valid := apiV1StickyFormatSuffix(request.URL.Path); valid && format != "json" {
				stickyFormatNotFound.ServeHTTP(response, request)
				return
			}
			protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
			converterPath := request.URL.Path
			if !matched && !strings.HasSuffix(converterPath, "/") {
				// APPEND_SLASH cannot redirect a path whose converter would still
				// reject the appended form. Preserve the resolver's 404 boundary.
				converterPath += "/"
				protocol, matched = resolveReadServiceProtocolForMethod(converterPath, request.Method)
				matched = matched && (protocol.resolveMalformedSlashless || (protocol.generatedMethods && activityReadsAPIProtocol(protocol.pattern)))
			}
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if !matched || preflight || !readServiceProtocolHasMalformedUUID(converterPath, protocol) {
				next.ServeHTTP(response, request)
				return
			}
			if activityReadsAPIProtocol(protocol.pattern) || protocol.workItem || protocol.apiAuthorization == "invitations" {
				activityNotFound.ServeHTTP(response, request)
				return
			}
			if protocol.pattern == "/api/workspaces/{slug}/webhook-logs/{webhook_id}/" {
				headers := response.Header()
				headers.Set("Cross-Origin-Opener-Policy", "same-origin")
				headers.Set("Referrer-Policy", "same-origin")
				mergeVary(headers, "Cookie", "origin")
				headers.Set("X-Content-Type-Options", "nosniff")
				headers.Set("X-Frame-Options", "DENY")
				body := map[string]string{"error": "Page not found."}
				if request.Method == http.MethodHead {
					headers.Set("Content-Type", "application/json")
					response.WriteHeader(http.StatusNotFound)
					return
				}
				writeReadServiceJSON(response, http.StatusNotFound, body)
				return
			}
			notFound.ServeHTTP(response, request)
		})
	}
}
