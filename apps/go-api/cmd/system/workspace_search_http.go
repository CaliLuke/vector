package main

import "encoding/json"

func workspaceSearchProtocols() []readServiceProtocol {
	global := newSessionServiceProtocol("/api/workspaces/{slug}/search/", "GET, HEAD, OPTIONS", "Global Search Endpoint", true)
	global.optionsMetadata = json.RawMessage(`{"description":"Endpoint to search across multiple fields in the workspace and\nalso show related workspace if found","name":"Global Search Endpoint","parses":["application/json","application/x-www-form-urlencoded","multipart/form-data"],"renders":["application/json"]}`)
	entity := newSessionServiceProtocol("/api/workspaces/{slug}/entity-search/", "GET, HEAD, OPTIONS", "Search Endpoint", true)
	entity.permissionBeforeTimezone = true
	return []readServiceProtocol{global, entity}
}

func isWorkspaceSearchProtocol(pattern string) bool {
	return pattern == "/api/workspaces/{slug}/search/" || pattern == "/api/workspaces/{slug}/entity-search/"
}
