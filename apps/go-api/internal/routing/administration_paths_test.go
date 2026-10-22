package routing

import (
	"regexp"
	"strings"
	"testing"
)

func TestAdministrationConvertersPreserveLiteralAndMalformedRoutes(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct {
		family, template string
		matches, rejects []string
	}{
		{family: "instance_management", template: "/api/instances/admins/{pk}/", matches: []string{"/api/instances/admins/" + id + "/"}, rejects: []string{"/api/instances/admins/sign-in/", "/api/instances/admins/sign-out/", "/api/instances/admins/sign-up/", "/api/instances/admins/me/", "/api/instances/admins/not-a-uuid/"}},
		{family: "workspace_webhooks", template: "/api/workspaces/{slug}/webhooks/{pk}/", matches: []string{"/api/workspaces/demo/webhooks/" + id + "/"}, rejects: []string{"/api/workspaces/demo/webhooks/not-a-uuid/"}},
		{family: "workspace_webhooks", template: "/api/workspaces/{slug}/webhook-logs/{webhook_id}/", matches: []string{"/api/workspaces/demo/webhook-logs/" + id + "/"}, rejects: []string{"/api/workspaces/demo/webhook-logs/not-a-uuid/"}},
		{family: "project_configuration", template: "/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/", matches: []string{"/api/workspaces/demo/projects/" + id + "/estimates/" + id + "/estimate-points/legacy-point/"}, rejects: []string{"/api/workspaces/demo/projects/not-a-uuid/estimates/" + id + "/estimate-points/legacy-point/", "/api/workspaces/demo/projects/" + id + "/estimates/not-a-uuid/estimate-points/legacy-point/"}},
	} {
		t.Run(tc.template, func(t *testing.T) {
			pattern := regexp.MustCompile(caddyPathPattern(Operation{Path: tc.template, Implementation: &tc.family}))
			for _, path := range tc.matches {
				if !pattern.MatchString(path) {
					t.Errorf("source-valid path not routed: %s", path)
				}
			}
			for _, path := range tc.rejects {
				if pattern.MatchString(path) {
					t.Errorf("source-excluded path routed: %s", path)
				}
			}
			if pattern.MatchString(strings.TrimSuffix(tc.matches[0], "/") + "/extra/") {
				t.Error("neighboring route matched")
			}
		})
	}
}
