package main

import (
	"strings"
	"testing"
)

func TestAssetsPathConvertersAcceptStorageKeysAndValidateWorkspaceIDs(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000009701"
	for _, pattern := range []string{
		"/api/users/file-assets/{asset_key}/",
		"/api/workspaces/file-assets/{workspace_id}/{asset_key}/",
		"/api/workspaces/file-assets/{workspace_id}/{asset_key}/restore/",
	} {
		protocol := newReadServiceProtocol(pattern, "GET, DELETE, HEAD, OPTIONS", "Asset", "", true)
		for _, key := range []string{"assets-fixture.png", "report 2026.pdf", "résumé.png"} {
			path := strings.NewReplacer("{workspace_id}", id, "{asset_key}", key).Replace(pattern)
			if readServiceProtocolHasMalformedUUID(path, protocol) {
				t.Errorf("Django str converter must accept storage key in %q", path)
			}
			if strings.Contains(pattern, "{workspace_id}") {
				malformed := strings.Replace(path, id, "not-a-workspace-uuid", 1)
				if !readServiceProtocolHasMalformedUUID(malformed, protocol) {
					t.Errorf("Django uuid converter must reject workspace ID in %q", malformed)
				}
			}
		}
	}
}
