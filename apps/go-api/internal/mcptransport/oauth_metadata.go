package mcptransport

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const MetadataPath = "/.well-known/oauth-protected-resource/mcp"

func oauthMetadata(config Config, issuer string) http.Handler {
	resource, _ := url.Parse(config.ResourceURL)
	metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               config.ResourceURL,
		AuthorizationServers:   []string{issuer},
		ScopesSupported:        []string{"vector:read", "vector:write"},
		BearerMethodsSupported: []string{"header"},
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Host != resource.Host || r.URL.RawQuery != "" || r.URL.ForceQuery {
			http.Error(w, "invalid metadata request target", http.StatusBadRequest)
			return
		}
		metadata.ServeHTTP(w, r)
	})
}

func writeOAuthChallenge(w http.ResponseWriter, config Config, invalid, insufficient bool) {
	resource, _ := url.Parse(config.ResourceURL)
	challenge := "Bearer resource_metadata=" + strconv.Quote("https://"+resource.Host+MetadataPath) + `, scope="vector:read"`
	if invalid {
		challenge += `, error="invalid_token"`
	}
	if insufficient {
		challenge += `, error="insufficient_scope"`
	}
	w.Header().Set("WWW-Authenticate", challenge)
}
