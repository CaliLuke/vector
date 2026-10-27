package main

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"

	linkserver "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/http/mcp_oauth_links/server"
	linkapi "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	loomhttp "github.com/CaliLuke/loom/http"
)

const mcpLinkBodyLimit = 16 << 10

func mountMCPOAuthLinks(mux loomhttp.Muxer, pending pendingReplacementEndpoints, cors loomhttp.RuntimeCORSPolicy, onError func(context.Context, http.ResponseWriter, error)) {
	if pending.mcpLinks == nil {
		return
	}
	var safeErrorHandler func(context.Context, http.ResponseWriter, error)
	if onError != nil {
		safeErrorHandler = func(ctx context.Context, w http.ResponseWriter, _ error) {
			onError(ctx, w, errors.New("MCP account linking response failed"))
		}
	}
	server := linkserver.New(pending.mcpLinks, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, safeErrorHandler, mcpLinkErrorFormatter, cors)
	server.Use(mcpOAuthLinkBoundary(pending.mcpResourceURL, pending.accountAuthenticationCSRFTrustedOrigins))
	linkserver.Mount(mux, server)
}

// The provider callback consumes a form protected by state, nonce and PKCE.
// Every browser mutation uses the existing session CSRF policy instead.
func mcpOAuthLinkBoundary(resource string, trustedOrigins []string) func(http.Handler) http.Handler {
	canonical, _ := url.Parse(resource) // Configuration is validated before mounting.
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			if r.Host != canonical.Host || r.URL.RawQuery != "" || r.URL.ForceQuery {
				writeReadServiceJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid account linking request."})
				return
			}
			if r.ContentLength > mcpLinkBodyLimit {
				writeReadServiceJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"message": "Account linking request is too large."})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, mcpLinkBodyLimit)
			if r.Method == http.MethodPost && r.URL.Path == "/auth/mcp/callback/" {
				mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || mediaType != "application/x-www-form-urlencoded" {
					writeReadServiceJSON(w, http.StatusUnsupportedMediaType, map[string]string{"message": "The identity provider must return a form response."})
					return
				}
				if err := loomhttp.ParseFormWithLimit(r, mcpLinkBodyLimit); err != nil {
					status := http.StatusBadRequest
					if loomhttp.NewErrorResponse(r.Context(), err).StatusCode() == http.StatusRequestEntityTooLarge {
						status = http.StatusRequestEntityTooLarge
					}
					writeReadServiceJSON(w, status, map[string]string{"message": "Invalid identity provider response."})
					return
				}
				for _, values := range r.PostForm {
					if len(values) != 1 {
						writeReadServiceJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid identity provider response."})
						return
					}
				}
			} else if accountAuthenticationUnsafeMethod(r.Method) {
				if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRFToken")) > 1 || accountAuthenticationCSRFReason(r, trustedOrigins) != "" {
					writeReadServiceJSON(w, http.StatusForbidden, map[string]string{"message": "Refresh this page and try again."})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

type mcpLinkFailure struct {
	Message string `json:"message"`
	status  int
}

func (e mcpLinkFailure) StatusCode() int { return e.status }

// Generated validation errors can contain submitted values. Retain their
// classification without returning callback codes, state or provider messages.
func mcpLinkErrorFormatter(ctx context.Context, err error) loomhttp.StatusCoder {
	var declared *linkapi.LinkError
	if errors.As(err, &declared) {
		switch declared.Kind {
		case "status_400":
			return mcpLinkFailure{Message: "This connection attempt is invalid or expired. Start again.", status: 400}
		case "status_401":
			return mcpLinkFailure{Message: "Sign in again to manage MCP access.", status: 401}
		case "status_403":
			return mcpLinkFailure{Message: "Refresh this page and try again.", status: 403}
		case "status_409":
			return mcpLinkFailure{Message: "This account connection cannot be confirmed. Check its current status.", status: 409}
		case "status_503":
			return mcpLinkFailure{Message: "MCP account linking is temporarily unavailable.", status: 503}
		}
	}
	status := loomhttp.NewErrorResponse(ctx, err).StatusCode()
	message := "Invalid account linking request."
	if status >= 500 {
		message = "MCP account linking is temporarily unavailable."
	}
	return mcpLinkFailure{Message: message, status: status}
}
