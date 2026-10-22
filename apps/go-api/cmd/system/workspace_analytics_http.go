package main

var workspaceAnalyticsProtocolDefinitions = [...]struct{ path, name string }{
	{"/api/workspaces/{slug}/advance-analytics/", "Advance Analytics Endpoint"},
	{"/api/workspaces/{slug}/advance-analytics-stats/", "Advance Analytics Stats Endpoint"},
	{"/api/workspaces/{slug}/advance-analytics-charts/", "Advance Analytics Chart Endpoint"},
	{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics/", "Project Advance Analytics Endpoint"},
	{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-stats/", "Project Advance Analytics Stats Endpoint"},
	{"/api/workspaces/{slug}/projects/{project_id}/advance-analytics-charts/", "Project Advance Analytics Chart Endpoint"},
	{"/api/workspaces/{slug}/analytics/", "Analytics Endpoint"},
	{"/api/workspaces/{slug}/default-analytics/", "Default Analytics Endpoint"},
	{"/api/workspaces/{slug}/saved-analytic-view/{analytic_id}/", "Saved Analytic Endpoint"},
}

func workspaceAnalyticsProtocols() []readServiceProtocol {
	protocols := make([]readServiceProtocol, 0, len(workspaceAnalyticsProtocolDefinitions))
	for _, definition := range workspaceAnalyticsProtocolDefinitions {
		protocols = append(protocols, newSessionServiceProtocol(definition.path, "GET, HEAD, OPTIONS", definition.name, true))
	}
	return protocols
}

func isWorkspaceAnalyticsProtocol(pattern string) bool {
	for _, definition := range workspaceAnalyticsProtocolDefinitions {
		if definition.path == pattern {
			return true
		}
	}
	return false
}
