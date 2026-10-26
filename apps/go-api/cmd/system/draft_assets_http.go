package main

import (
	"net/http"
	"strings"
)

// The new workspace draft association route requires CSRF protection. Retain
// the frozen protocol of the older asset routes.
func draftAssetCSRFMiddleware(trustedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		protected := workItemTypesCSRFMiddleware(trustedOrigins)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			if len(parts) == 7 && strings.Join(parts[:4], "/") == "api/assets/v2/workspaces" && parts[6] == "bulk" {
				protected.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
