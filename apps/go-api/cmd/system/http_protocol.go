package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	systemapi "github.com/CaliLuke/go-plane"
	projectmembershipreads "github.com/CaliLuke/go-plane/gen/project_membership_reads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func mountProtocolHandler(mux loomhttp.Muxer, method, pattern string, handler http.Handler) {
	//loom:vet ignore route-outside-design -- Django protocol compatibility, not a separate application operation
	loomhttp.MountHandler(mux, method, pattern, handler)
}

type nativePreflightResponseWriter struct {
	http.ResponseWriter
}

func (writer nativePreflightResponseWriter) WriteHeader(status int) {
	if status == http.StatusNoContent {
		removeVary(writer.Header(), "Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers")
		mergeVary(writer.Header(), "origin")
		writer.Header().Set("Content-Length", "0")
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		status = http.StatusOK
	}
	writer.ResponseWriter.WriteHeader(status)
}

func nativeProtocolCORSOptionsHandler(policy loomhttp.RuntimeCORSPolicy, methods []string, next http.Handler) http.Handler {
	options := policy.OptionsHandler(loomhttp.AsHandlerFunc(next), methods)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0 {
			request = request.Clone(request.Context())
			request.Header = request.Header.Clone()
			request.Header.Set("Access-Control-Request-Method", http.MethodGet)
			options.ServeHTTP(nativePreflightResponseWriter{response}, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func nativeProtocolCompatibilityMiddleware(allow func(string) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			setNativeProtocolCompatibilityHeaders(response.Header(), allow(request.URL.Path))
			next.ServeHTTP(&workspaceContentResponseWriter{ResponseWriter: response}, request)
		})
	}
}

func nativeProtocolHeadHandler(middleware func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(workspaceSlugHeadWriter{response}, request)
	}))
}

func nativeBufferedProtocolHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func nativeProtocolOptionsHandler(middleware func(http.Handler) http.Handler, name string, patch bool) http.Handler {
	if patch {
		return nativeProtocolOptionsActionsHandler(middleware, name, http.MethodPatch)
	}
	return nativeProtocolOptionsActionsHandler(middleware, name)
}

func nativeProtocolOptionsActionsHandler(middleware func(http.Handler) http.Handler, name string, actionMethods ...string) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		actions := make(map[string]any, len(actionMethods))
		for _, method := range actionMethods {
			actions[method] = map[string]any{}
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": name, "description": "", "renders": []string{"application/json"},
			"parses":  []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
			"actions": actions,
		})
	}))
}

func nativeProtocolOptionsWithoutActionsHandler(middleware func(http.Handler) http.Handler, name string) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		writeStickyJSON(response, http.StatusOK, struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Renders     []string `json:"renders"`
			Parses      []string `json:"parses"`
		}{
			Name:        name,
			Description: "",
			Renders:     []string{"application/json"},
			Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func nativeProtocolOptionsJSONNegotiationMiddleware(next http.Handler) http.Handler {
	notAcceptable := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		mergeVary(response.Header(), "Accept")
		writeStickyJSON(response, http.StatusNotAcceptable, map[string]string{
			"detail": "Could not satisfy the request Accept header.",
		})
	})
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		accept, present := nativeRequestAcceptHeader(request)
		acceptsJSON, _ := projectMembershipReadsJSONNegotiation(accept, present)

		if !acceptsJSON {
			notAcceptable.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func nativeReadOnlyOptionsHandler(
	mux loomhttp.Muxer,
	middleware func(http.Handler) http.Handler,
	name string,
	authorizer systemapi.ProjectMembershipReadsMetadataAuthorizer,
	requireWorkspaceMembership bool,
) http.Handler {
	return middleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var sessionID string
		if cookie, err := request.Cookie("session-id"); err == nil {
			sessionID = cookie.Value
		}
		err := authorizer.AuthorizeMetadata(
			request.Context(),
			sessionID,
			mux.Vars(request)["slug"],
			requireWorkspaceMembership,
		)
		if err != nil {
			var unauthenticated *projectmembershipreads.ProjectMembershipReadsUnauthenticated
			var forbidden *projectmembershipreads.ProjectMembershipReadsForbidden
			var internalError *projectmembershipreads.ProjectMembershipReadsInternalError
			switch {
			case errors.As(err, &unauthenticated):
				writeStickyJSON(response, http.StatusUnauthorized, unauthenticated)
			case errors.As(err, &forbidden):
				writeStickyJSON(response, http.StatusForbidden, forbidden)
			case errors.As(err, &internalError):
				writeStickyJSON(response, http.StatusInternalServerError, internalError)
			default:
				writeStickyJSON(response, http.StatusInternalServerError, map[string]string{
					"message": "Something went wrong please try again later",
				})
			}
			return
		}
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name": name, "description": "", "renders": []string{"application/json"},
			"parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	}))
}

func nativeJSONCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		headers := response.Header()
		headers.Set("Allow", "GET, HEAD, OPTIONS")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "same-origin")
		mergeVary(headers, "Cookie", "origin")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func nativeJSONHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		nativeJSONCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func nativeBooleanValue(raw json.RawMessage) (bool, int, map[string]string) {
	switch string(bytes.TrimSpace(raw)) {
	case "true", "1":
		return true, 0, nil
	case "false", "0":
		return false, 0, nil
	case "null":
		return false, http.StatusBadRequest, map[string]string{"error": "The payload is not valid"}
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		switch value {
		case "t", "True", "1":
			return true, 0, nil
		case "f", "False", "0":
			return false, 0, nil
		}
	}
	return false, http.StatusBadRequest, map[string]string{"error": "Please provide valid detail"}
}
