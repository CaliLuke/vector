package routing

import (
	"regexp"
	"strings"
	"testing"
)

func TestAssetsCaddyPatternKeepsDjangoUUIDAndStringConverters(t *testing.T) {
	implementation := "assets_lifecycle"
	const id = "00000000-0000-0000-0000-000000009701"
	for _, path := range []string{
		"/api/assets/v2/static/{asset_id}/",
		"/api/assets/v2/workspaces/{slug}/projects/{project_id}/{pk}/",
		"/api/assets/v2/workspaces/{slug}/projects/{project_id}/{entity_id}/bulk/",
		"/api/assets/v2/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/attachments/{pk}/",
		"/api/workspaces/file-assets/{workspace_id}/{asset_key}/",
		"/api/public/assets/v2/anchor/{anchor}/{pk}/",
	} {
		pattern := regexp.MustCompile(caddyPathPattern(Operation{Path: path, Implementation: &implementation}))
		valid := strings.NewReplacer(
			"{slug}", "workspace", "{anchor}", "public-board", "{asset_key}", "résumé 2026.pdf",
			"{asset_id}", id, "{workspace_id}", id, "{project_id}", id, "{issue_id}", id, "{entity_id}", id, "{pk}", id,
		).Replace(path)
		if !pattern.MatchString(valid) {
			t.Errorf("source-valid path %q rejected by %s", valid, pattern)
		}
		for _, parameter := range []string{"asset_id", "workspace_id", "project_id", "issue_id", "entity_id", "pk"} {
			if !strings.Contains(path, "{"+parameter+"}") {
				continue
			}
			for _, invalidID := range []string{"not-a-uuid", "00000000000000000000000000009701", "00000000-0000-0000-0000-00000000ABCD"} {
				invalidPath := strings.Replace(path, "{"+parameter+"}", invalidID, 1)
				invalidPath = strings.NewReplacer(
					"{slug}", "workspace", "{anchor}", "public-board", "{asset_key}", "photo.png",
					"{asset_id}", id, "{workspace_id}", id, "{project_id}", id, "{issue_id}", id, "{entity_id}", id, "{pk}", id,
				).Replace(invalidPath)
				if pattern.MatchString(invalidPath) {
					t.Errorf("source-invalid %s in %q must remain with Django", parameter, invalidPath)
				}
			}
		}
	}
}
