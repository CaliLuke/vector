package main

func isPersonalDashboardProtocol(pattern string) bool {
	switch pattern {
	case "/api/users/me/workspaces/{slug}/activity-graph/", "/api/users/me/workspaces/{slug}/dashboard/", "/api/users/me/workspaces/{slug}/issues-completed-graph/":
		return true
	default:
		return false
	}
}
