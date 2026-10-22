package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
)

//go:embed page_lifecycle_options_metadata.json
var pageLifecycleProtocolOptionsJSON []byte

var pageLifecycleProtocolOptions = func() map[string]json.RawMessage {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(pageLifecycleProtocolOptionsJSON, &result); err != nil {
		panic(err)
	}
	return result
}()

func pageLifecycleSessionProtocol(pattern, allow, name, action string, derivedHead bool) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, name, derivedHead)
	protocol.action = action
	protocol.permissionBeforeTimezone = true
	protocol.pageLifecycle = true
	if readServiceMethodAllowed(allow, http.MethodGet) {
		protocol.permissionBeforeTimezoneExceptMethods = []string{http.MethodGet, http.MethodHead}
	}
	protocol.authoredOptions = true
	protocol.optionsMetadata = pageLifecycleProtocolOptions[action]
	switch action {
	case "collection":
		protocol.methodAllow = map[string]string{
			http.MethodGet: "GET, POST, HEAD, OPTIONS", http.MethodHead: "GET, POST, HEAD, OPTIONS",
			http.MethodPost: "POST, OPTIONS", http.MethodOptions: "POST, OPTIONS",
		}
	case "detail":
		protocol.methodAllow = map[string]string{
			http.MethodGet: "GET, PATCH, DELETE, HEAD, OPTIONS", http.MethodHead: "GET, PATCH, DELETE, HEAD, OPTIONS",
			http.MethodPatch: "PATCH, DELETE, OPTIONS", http.MethodDelete: "PATCH, DELETE, OPTIONS", http.MethodOptions: "PATCH, DELETE, OPTIONS",
		}
	case "description":
		protocol.methodAllow = map[string]string{
			http.MethodGet: "GET, PATCH, HEAD, OPTIONS", http.MethodHead: "GET, PATCH, HEAD, OPTIONS",
			http.MethodPatch: "PATCH, OPTIONS", http.MethodOptions: "PATCH, OPTIONS",
		}
	case "access", "duplicate":
		protocol.methodAllow = map[string]string{http.MethodPost: "POST, OPTIONS", http.MethodOptions: "POST, OPTIONS"}
	case "archive", "lock":
		protocol.methodAllow = map[string]string{http.MethodPost: "POST, DELETE, OPTIONS", http.MethodDelete: "POST, DELETE, OPTIONS", http.MethodOptions: "POST, DELETE, OPTIONS"}
	}
	if derivedHead {
		protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	}
	return protocol
}
