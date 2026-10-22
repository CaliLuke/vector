//go:build integration

package proxy_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/routing"
)

func TestAdministrationBatchGoOnlyProductionProxyPolicy(t *testing.T) {
	inventory, err := routing.LoadFile("../routing/route-inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var operations []routing.Operation
	for _, op := range inventory.Operations {
		if op.Implementation == nil {
			continue
		}
		switch *op.Implementation {
		case "instance_management", "project_configuration", "workspace_webhooks":
			operations = append(operations, op)
		}
	}
	if len(operations) != 34 {
		t.Fatalf("selected operations=%d want34", len(operations))
	}
	const id = "11111111-1111-4111-8111-111111111111"
	replacements := strings.NewReplacer("{slug}", "demo", "{pk}", id, "{project_id}", id, "{estimate_id}", id, "{webhook_id}", id, "{estimate_point_id}", "legacy-point")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	for _, config := range []string{"../../../proxy/Caddyfile.ce", "../../../proxy/Caddyfile.aio.ce"} {
		t.Run(config, func(t *testing.T) {
			proxyURL := startCaddyWithConfig(t, ctx, config, django.port, goAPI.port, nil)
			paths := map[string]bool{}
			for _, op := range operations {
				path := replacements.Replace(op.Path)
				paths[path] = true
				for _, form := range []string{path, strings.TrimSuffix(path, "/")} {
					assertTarget(t, op.Method, proxyURL+form, "go")
				}
			}
			for path := range paths {
				assertHeadTarget(t, proxyURL+path, "go")
				assertTarget(t, http.MethodOptions, proxyURL+path, "go")
				assertTarget(t, http.MethodPut, proxyURL+path, "go")
			}
			for _, path := range []string{
				"/api/instances/admins/sign-in/", "/api/instances/admins/sign-up/", "/api/instances/admins/sign-out/",
				"/api/instances/admins/not-a-uuid/", "/api/workspaces/demo/webhooks/not-a-uuid/",
				"/api/workspaces/demo/projects/not-a-uuid/estimates/", "/api/workspaces/demo/webhooks/" + id + "/unknown-action/",
			} {
				assertTarget(t, http.MethodPost, proxyURL+path, "go")
			}
		})
	}
}
