package mcptransport

import (
	"net/http"
	"strings"
)

// RedactRejectedQueries runs outside HTTP telemetry. These routes reject every
// query string; retain that rejection while keeping accidental credentials out
// of URL attributes and request logs. Request bodies and headers remain intact.
func RedactRejectedQueries(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		sensitive := path == OAuthPath || strings.HasPrefix(path, OAuthPath+"/") ||
			path == MetadataPath || strings.HasPrefix(path, MetadataPath+"/") ||
			path == "/auth/mcp/callback/" || strings.HasPrefix(path, "/api/users/me/mcp-link/")
		if sensitive && r.URL.RawQuery != "" {
			r = r.Clone(r.Context())
			r.URL.RawQuery = "redacted"
			r.RequestURI = r.URL.RequestURI()
		}
		next.ServeHTTP(w, r)
	})
}
