package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	cyclelifecycleapi "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	cyclelifecyclesvr "github.com/CaliLuke/go-plane/cyclelifecycle/gen/http/cycle_lifecycle_api/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed cycle_lifecycle_options_metadata.json
var cycleLifecycleOptionsMetadataJSON []byte

var cycleLifecycleOptionsMetadata = func() map[string]struct {
	Status          int             `json:"status"`
	Metadata        json.RawMessage `json:"metadata"`
	MissingMetadata json.RawMessage `json:"missing_metadata"`
} {
	var result map[string]struct {
		Status          int             `json:"status"`
		Metadata        json.RawMessage `json:"metadata"`
		MissingMetadata json.RawMessage `json:"missing_metadata"`
	}
	if err := json.Unmarshal(cycleLifecycleOptionsMetadataJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

func newCycleAPIProtocol(pattern, allow string, retainedRead bool) readServiceProtocol {
	protocol := newReadServiceProtocol(pattern, allow, "Cycle API Endpoint", "", false)
	protocol.generatedMethods = true
	if retainedRead {
		protocol.passthroughMethods = []string{http.MethodGet}
	}
	return protocol
}

func newCycleSessionProtocol(pattern, allow, name string, head bool) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, name, head)
	protocol.generatedMethods = true
	if metadata, ok := cycleLifecycleOptionsMetadata[pattern]; ok {
		protocol.optionsStatus = metadata.Status
		protocol.optionsMetadata = metadata.Metadata
		protocol.optionsMetadataMissing = metadata.MissingMetadata
	}
	if head {
		protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	}
	return protocol
}

func mountCycleLifecycle(mux loomhttp.Muxer, endpoints *cyclelifecycleapi.Endpoints, cors loomhttp.RuntimeCORSPolicy, options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) {
	server := cyclelifecyclesvr.New(endpoints, mux, djangoRequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
	apiStack := activityReadsMiddlewareStack(options, sessions, cookie)
	sessionStack := sessionServiceMiddlewareStack(options, sessions, cookie)
	server.Use(func(next http.Handler) http.Handler {
		api := apiStack(next)
		session := sessionStack(next)
		return cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/") {
				api.ServeHTTP(w, r)
				return
			}
			session.ServeHTTP(w, r)
		}))
	})
	server.Mount(mux)
}
