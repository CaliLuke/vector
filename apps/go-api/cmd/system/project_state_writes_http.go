package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
)

//go:embed project_state_options_metadata.json
var projectStateOptionsJSON []byte
var projectStateOptionsMetadata = func() map[string]json.RawMessage {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(projectStateOptionsJSON, &metadata); err != nil {
		panic(err)
	}
	return metadata
}()

func isProjectStateSessionProtocol(pattern string) bool {
	switch pattern {
	case "/api/workspaces/{slug}/projects/{project_id}/states/", "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/mark-default/":
		return true
	}
	return false
}

func newProjectStateSessionProtocol(pattern, writes, reads string) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, writes, "State", reads != "")
	if reads != "" {
		protocol.methodAllow = map[string]string{http.MethodGet: reads, http.MethodHead: reads, http.MethodOptions: reads}
	}
	name := "detail"
	if strings.HasSuffix(pattern, "/states/") {
		name = "collection"
	}
	if strings.HasSuffix(pattern, "/mark-default/") {
		name = "mark-default"
	}
	protocol.optionsMetadata = projectStateOptionsMetadata[name]
	return protocol
}

func newProjectStateAPIProtocol(pattern, reads, name string) readServiceProtocol {
	writes := "PATCH, DELETE"
	if pattern == "/api/v1/workspaces/{slug}/projects/{project_id}/states/" {
		writes = "POST"
	}
	protocol := newGeneratedAPIWriteProtocol(pattern, writes, name)
	protocol.methodAllow = map[string]string{http.MethodGet: reads}
	return protocol
}
