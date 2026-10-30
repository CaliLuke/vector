package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	stickysvr "github.com/CaliLuke/go-plane/gen/http/stickies/server"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

func stickyListOptionsHandler(middleware func(http.Handler) http.Handler) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": "Sticky List", "description": "", "renders": []string{"application/json"},
			"parses":  []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions": map[string]any{http.MethodPost: stickyOptionsFields},
		})
	}))
}

var stickyOptionsFields = map[string]any{
	"id":                   map[string]any{"type": "field", "required": false, "read_only": true, "label": "Id"},
	"created_at":           map[string]any{"type": "datetime", "required": false, "read_only": true, "label": "Created At"},
	"updated_at":           map[string]any{"type": "datetime", "required": false, "read_only": true, "label": "Last Modified At"},
	"deleted_at":           map[string]any{"type": "datetime", "required": false, "read_only": false, "label": "Deleted At"},
	"name":                 map[string]any{"type": "string", "required": false, "read_only": false, "label": "Name"},
	"description":          map[string]any{"type": "field", "required": false, "read_only": false, "label": "Description"},
	"description_html":     map[string]any{"type": "string", "required": false, "read_only": false, "label": "Description html"},
	"description_stripped": map[string]any{"type": "string", "required": false, "read_only": false, "label": "Description stripped"},
	"description_binary":   map[string]any{"type": "field", "required": false, "read_only": true, "label": "Description binary"},
	"logo_props":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Logo props"},
	"color":                map[string]any{"type": "string", "required": false, "read_only": false, "label": "Color", "max_length": 255},
	"background_color":     map[string]any{"type": "string", "required": false, "read_only": false, "label": "Background color", "max_length": 255},
	"sort_order":           map[string]any{"type": "float", "required": false, "read_only": false, "label": "Sort order"},
	"created_by":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Created By"},
	"updated_by":           map[string]any{"type": "field", "required": false, "read_only": false, "label": "Last Modified By"},
	"workspace":            map[string]any{"type": "field", "required": false, "read_only": true, "label": "Workspace"},
	"owner":                map[string]any{"type": "field", "required": false, "read_only": true, "label": "Owner"},
}

func stickyHeadPaginationAuthorizationMiddleware(mux loomhttp.Muxer, authenticate security.AuthAPIKeyFunc, authorizer systemapi.StickyWorkspaceAuthorizer, next http.Handler) http.Handler {
	invalidPagination := http.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setStickyProtocolHeaders(response.Header(), request.URL.Path)
		writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": validateStickyPagination(request)})
	}))
	if authorizer != nil {
		invalidPagination = stickyWorkspaceAuthorizationMiddleware(mux, authorizer, invalidPagination)
	}
	if authenticate != nil {
		invalidPagination = stickyAuthenticationMiddleware(authenticate)(invalidPagination)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if validateStickyPagination(request) == "" {
			next.ServeHTTP(response, request)
			return
		}
		invalidPagination.ServeHTTP(response, request)
	})
}

func stickyWorkspaceAuthorizationMiddleware(mux loomhttp.Muxer, authorizer systemapi.StickyWorkspaceAuthorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := authorizer.AuthorizeStickyWorkspace(request.Context(), mux.Vars(request)["slug"])
		if err == nil {
			next.ServeHTTP(response, request)
			return
		}
		var permission *sticky.StickyPermissionError
		if errors.As(err, &permission) {
			writeStickyJSON(response, http.StatusForbidden, map[string]string{"error": permission.Message})
			return
		}
		var unauthenticated *sticky.StickyUnauthenticatedError
		if errors.As(err, &unauthenticated) {
			writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": unauthenticated.Detail})
			return
		}
		writeStickyJSON(response, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
	})
}

func stickyDetailUUIDMiddleware(mux loomhttp.Muxer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if _, err := uuid.Parse(mux.Vars(request)["pk"]); err != nil {
			http.NotFound(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func stickyFrontendTrailingSlashMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if stickyDetailPathNeedsTrailingSlash(request.Method, request.URL.Path) {
			request = request.Clone(request.Context())
			requestURL := *request.URL
			requestURL.Path += "/"
			request.URL = &requestURL
		}
		next.ServeHTTP(response, request)
	})
}

func stickyDetailPathNeedsTrailingSlash(method, path string) bool {
	if method != http.MethodGet && method != http.MethodDelete {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 5 && parts[0] == "api" && parts[1] == "workspaces" && parts[2] != "" && parts[3] == "stickies" && parts[4] != "" && !strings.HasSuffix(path, "/")
}

func stickyAuthenticationMiddleware(authenticate security.AuthAPIKeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			setStickyHeaders(response.Header())
			cookie, err := request.Cookie("session-id")
			if err != nil || cookie.Value == "" {
				writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			ctx, err := authenticate(request.Context(), cookie.Value, &security.APIKeyScheme{Name: "plane_session_cookie"})
			if err != nil {
				writeStickyJSON(response, http.StatusUnauthorized, map[string]string{"detail": "Authentication credentials were not provided."})
				return
			}
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

const stickyRequestLimit = 10 * 1024 * 1024

type stickyResponseWriter struct{ http.ResponseWriter }

func (writer stickyResponseWriter) WriteHeader(status int) {
	writer.Header().Del("loom-error")
	writer.ResponseWriter.WriteHeader(status)
}

func (writer stickyResponseWriter) Write(body []byte) (int, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if decoder.Decode(&value) == nil {
		addStickyNulls(value)
		encoded, err := json.Marshal(value)
		if err == nil {
			encoded = append(encoded, '\n')
			_, writeErr := writer.ResponseWriter.Write(encoded)
			return len(body), writeErr
		}
	}
	return writer.ResponseWriter.Write(body)
}

func addStickyNulls(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	if results, ok := object["results"].([]any); ok {
		for _, field := range []string{"grouped_by", "sub_grouped_by", "extra_stats"} {
			if _, exists := object[field]; !exists {
				object[field] = nil
			}
		}
		for _, result := range results {
			addStickyNulls(result)
		}
	}
	if _, isSticky := object["id"]; !isSticky {
		return
	}
	for _, field := range []string{"created_by", "updated_by", "deleted_at", "name", "color", "background_color", "description_stripped", "description_binary"} {
		if _, exists := object[field]; !exists {
			object[field] = nil
		}
	}
}

func stickyCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		setStickyProtocolHeaders(headers, request.URL.Path)
		if (request.Method == http.MethodGet || request.Method == http.MethodHead) && strings.HasSuffix(request.URL.Path, "/stickies/") {
			if detail := validateStickyPagination(request); detail != "" {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": detail})
				return
			}
		}
		if request.Method == http.MethodPost || request.Method == http.MethodPatch {
			body, err := io.ReadAll(io.LimitReader(request.Body, stickyRequestLimit+1))
			if err != nil {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
				return
			}
			if len(body) > stickyRequestLimit {
				headers.Set("Content-Type", "application/json")
				response.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			body, err = normalizeStickyRequestBody(request.Header.Get("Content-Type"), body)
			if err != nil {
				writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Body = io.NopCloser(bytes.NewReader(body))
			if validation := validateStickyRequest(body); validation != nil {
				writeStickyJSON(response, http.StatusBadRequest, validation)
				return
			}
		}
		next.ServeHTTP(stickyResponseWriter{response}, request)
	})
}

func stickyProtocolHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setStickyProtocolHeaders(response.Header(), request.URL.Path)
		next.ServeHTTP(response, request)
	})
}

func setStickyProtocolHeaders(headers http.Header, path string) {
	setStickyHeaders(headers)
	if strings.HasSuffix(path, "/stickies/") {
		headers.Set("Allow", "GET, POST, HEAD, OPTIONS")
		return
	}
	headers.Set("Allow", "GET, PATCH, DELETE, HEAD, OPTIONS")
}

func normalizeStickyRequestBody(contentType string, body []byte) ([]byte, error) {
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		return nil, err
	}
	switch mediaType {
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		return encodeStickyForm(values)
	case "multipart/form-data":
		boundary := parameters["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart boundary is required")
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		values := url.Values{}
		for {
			part, nextErr := reader.NextPart()
			// Only bare EOF ends iteration; wrapped EOF remains a parse failure.
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				return nil, nextErr
			}
			value, err := io.ReadAll(io.LimitReader(part, stickyRequestLimit+1))
			_ = part.Close()
			if err != nil || len(value) > stickyRequestLimit {
				return nil, fmt.Errorf("read multipart field")
			}
			if name := part.FormName(); name != "" && part.FileName() == "" {
				values.Add(name, string(value))
			}
		}
		return encodeStickyForm(values)
	default:
		return body, nil
	}
}

func encodeStickyForm(values url.Values) ([]byte, error) {
	result := make(map[string]any, len(values))
	for name, items := range values {
		if len(items) == 0 {
			continue
		}
		value := items[len(items)-1]
		switch name {
		case "description", "logo_props":
			if json.Valid([]byte(value)) {
				result[name] = json.RawMessage(value)
			} else {
				result[name] = value
			}
		case "sort_order":
			if decoded, err := strconv.ParseFloat(value, 64); err == nil {
				result[name] = decoded
			} else {
				result[name] = value
			}
		default:
			result[name] = value
		}
	}
	return json.Marshal(result)
}

func setStickyHeaders(headers http.Header) {
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func validateStickyPagination(request *http.Request) string {
	if raw := request.URL.Query().Get("per_page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return "Invalid per_page parameter."
		}
		if value > 1000 {
			return "Invalid per_page value. Cannot exceed 1000."
		}
	}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		parts := strings.Split(raw, ":")
		if len(parts) != 3 {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.ParseFloat(parts[0], 64); err != nil {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return "Invalid cursor parameter."
		}
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return "Invalid cursor parameter."
		}
	}
	return ""
}

func validateStickyRequest(body []byte) map[string][]string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil
	}
	if raw, ok := fields["sort_order"]; ok && string(raw) != "null" {
		var value float64
		if json.Unmarshal(raw, &value) != nil {
			return map[string][]string{"sort_order": {"A valid number is required."}}
		}
	}
	return nil
}

func writeStickyJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func mountStickyRoutes(ctx context.Context, stickyEndpoints *sticky.Endpoints, authenticate security.AuthAPIKeyFunc, pending pendingReplacementEndpoints, mux loomhttp.Muxer, readServicesCORSPolicy loomhttp.RuntimeCORSPolicy) {
	if stickyEndpoints != nil {
		stickyServer := stickysvr.New(stickyEndpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil, readServicesCORSPolicy)
		stickyServer.Use(stickyCompatibilityMiddleware)
		if authenticate != nil {
			stickyServer.Use(stickyAuthenticationMiddleware(authenticate))
		}
		stickyListHEAD := stickyHeadPaginationAuthorizationMiddleware(mux, authenticate, pending.stickyWorkspaceAuthorizer, loomhttp.AsHandlerFunc(stickyServer.List))
		stickyListHEAD = djangoProtocolOptionsJSONNegotiationMiddleware(stickyListHEAD)
		stickyListHEAD = stickyProtocolHeadersMiddleware(stickyListHEAD)
		stickyListHEAD = readServicesCORSPolicy.Handler(stickyListHEAD.ServeHTTP)
		stickyRetrieveHEAD := djangoProtocolOptionsJSONNegotiationMiddleware(loomhttp.AsHandlerFunc(stickyServer.Retrieve))
		stickyRetrieveHEAD = stickyProtocolHeadersMiddleware(stickyRetrieveHEAD)
		stickyRetrieveHEAD = readServicesCORSPolicy.Handler(stickyRetrieveHEAD.ServeHTTP)
		stickysvr.MountListHandler(mux, stickyServer.List, readServicesCORSPolicy)
		stickysvr.MountCreateHandler(mux, stickyServer.Create, readServicesCORSPolicy)
		stickysvr.MountRetrieveHandler(mux, stickyServer.Retrieve, readServicesCORSPolicy)
		stickysvr.MountUpdateHandler(mux, stickyServer.Update, readServicesCORSPolicy)
		stickysvr.MountDeleteHandler(mux, stickyServer.Delete, readServicesCORSPolicy)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/stickies/", stickyListHEAD)
		loomhttp.MountDerivedHead(mux, "/api/workspaces/{slug}/stickies/{pk}/", stickyRetrieveHEAD)
		listOptions := stickyListOptionsHandler(stickyCompatibilityMiddleware)
		detailOptions := djangoProtocolOptionsWithoutActionsHandler(stickyCompatibilityMiddleware, "Sticky Instance")
		if pending.stickyWorkspaceAuthorizer != nil {
			listOptions = stickyWorkspaceAuthorizationMiddleware(mux, pending.stickyWorkspaceAuthorizer, listOptions)
			detailOptions = stickyWorkspaceAuthorizationMiddleware(mux, pending.stickyWorkspaceAuthorizer, detailOptions)
		}
		if authenticate != nil {
			listOptions = stickyAuthenticationMiddleware(authenticate)(listOptions)
			detailOptions = stickyAuthenticationMiddleware(authenticate)(detailOptions)
		}
		listOptions = djangoProtocolOptionsJSONNegotiationMiddleware(listOptions)
		detailOptions = djangoProtocolOptionsJSONNegotiationMiddleware(detailOptions)
		detailOptions = stickyDetailUUIDMiddleware(mux, detailOptions)
		listOptions = stickyProtocolHeadersMiddleware(listOptions)
		detailOptions = stickyProtocolHeadersMiddleware(detailOptions)
		listOptions = djangoProtocolCORSOptionsHandler(readServicesCORSPolicy, []string{http.MethodGet, http.MethodPost}, listOptions)
		detailOptions = djangoProtocolCORSOptionsHandler(readServicesCORSPolicy, []string{http.MethodGet, http.MethodPatch, http.MethodDelete}, detailOptions)
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/stickies/", listOptions)
		mountProtocolHandler(mux, http.MethodOptions, "/api/workspaces/{slug}/stickies/{pk}/", detailOptions)
		for _, mount := range stickyServer.Mounts {
			logHTTPRouteMount(ctx, "stickies", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
