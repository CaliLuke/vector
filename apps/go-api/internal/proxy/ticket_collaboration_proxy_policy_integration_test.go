//go:build integration

package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/routing"
)

const ticketCollaborationProxyOperations = 55

type ticketCollaborationOpenAPIOperation struct {
	ID string `json:"operationId"`
}

// This integration test uses the production Caddyfile and generated route
// policy and checks all 55 operations through the Go-only API catchall. The
// distinct Django upstream remains a detector for accidental fallback.
func TestTicketCollaborationGoOnlyProductionProxyPolicy(t *testing.T) {
	root := ticketCollaborationRepositoryRoot(t)
	operations, err := ticketCollaborationOpenAPIOperations(filepath.Join(root, "apps/go-api/ticket_collaboration/gen/http/openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != ticketCollaborationProxyOperations {
		t.Fatalf("ticket_collaboration OpenAPI operation count=%d, want %d", len(operations), ticketCollaborationProxyOperations)
	}

	inventory, err := routing.LoadFile(filepath.Join(root, "apps/go-api/internal/routing/route-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	family := make(map[string]routing.Operation, len(operations))
	for _, operation := range operations {
		key := operation.Method + " " + operation.Path
		for _, candidate := range inventory.Operations {
			if candidate.Method == operation.Method && candidate.Path == operation.Path {
				if candidate.OperationID != operation.ID {
					t.Fatalf("inventory operation ID for %s=%q, want %q", key, candidate.OperationID, operation.ID)
				}
				family[key] = candidate
				break
			}
		}
	}
	if len(family) != ticketCollaborationProxyOperations {
		t.Fatalf("route inventory covers %d of %d ticket_collaboration operations", len(family), ticketCollaborationProxyOperations)
	}

	generated, err := routing.RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(root, "apps/proxy/go-active-routes.caddy")
	actual, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(generated) {
		t.Fatalf("generated Caddy policy is stale; regenerate %s", policyPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)

	paths := map[string][]routing.Operation{}
	for _, operation := range operations {
		policy := family[operation.Method+" "+operation.Path]
		path := ticketCollaborationConcretePath(operation.Path)
		paths[operation.Path] = append(paths[operation.Path], policy)
		t.Run(operation.Method+" "+operation.Path, func(t *testing.T) {
			assertGoProxyResponse(t, operation.Method, proxyURL+path)
		})
	}

	for template, pathOperations := range paths {
		path := ticketCollaborationConcretePath(template)
		for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodTrace} {
			if ticketCollaborationOwnsMethod(pathOperations, method) {
				continue
			}
			t.Run("unsupported "+method+" "+template, func(t *testing.T) {
				assertGoProxyResponse(t, method, proxyURL+path)
			})
		}
		for _, method := range []string{http.MethodHead, http.MethodOptions} {
			t.Run("protocol "+method+" "+template, func(t *testing.T) {
				assertGoProxyResponse(t, method, proxyURL+path)
			})
		}
		withoutSlash := strings.TrimSuffix(path, "/")
		t.Run("slashless "+template, func(t *testing.T) {
			assertGoProxyResponse(t, http.MethodGet, proxyURL+withoutSlash)
		})
	}

	t.Run("unknown API paths route to Go", func(t *testing.T) {
		assertGoProxyResponse(t, http.MethodGet, proxyURL+"/api/not-migrated/")
	})
	t.Run("query string does not change owner", func(t *testing.T) {
		var read routing.Operation
		for _, operation := range operations {
			if operation.Method == http.MethodGet && strings.HasSuffix(operation.Path, "/comments/") {
				read = family[operation.Method+" "+operation.Path]
				break
			}
		}
		if read.Path == "" {
			t.Fatal("ticket_collaboration has no comments collection GET")
		}
		assertGoProxyResponse(t, http.MethodGet, proxyURL+ticketCollaborationConcretePath(read.Path)+"?per_page=1&fields=id")
	})
}

func ticketCollaborationOpenAPIOperations(path string) ([]struct {
	Method, Path, ID string
}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document struct {
		Paths map[string]map[string]ticketCollaborationOpenAPIOperation `json:"paths"`
	}
	if err = json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	methods := map[string]string{"get": "GET", "post": "POST", "put": "PUT", "patch": "PATCH", "delete": "DELETE"}
	operations := make([]struct{ Method, Path, ID string }, 0, ticketCollaborationProxyOperations)
	for path, entries := range document.Paths {
		for method, operation := range entries {
			if normalized, ok := methods[strings.ToLower(method)]; ok {
				operations = append(operations, struct{ Method, Path, ID string }{normalized, path, operation.ID})
			}
		}
	}
	return operations, nil
}

func ticketCollaborationConcretePath(path string) string {
	values := map[string]string{
		"anchor": "ticket-collaboration-board", "slug": "ticket-collaboration-workspace",
		"project_id": "00000000-0000-4000-8000-000000000001", "issue_id": "00000000-0000-4000-8000-000000000002",
		"pk": "00000000-0000-4000-8000-000000000003", "comment_id": "00000000-0000-4000-8000-000000000003",
		"subscriber_id": "00000000-0000-4000-8000-000000000004", "reaction_code": "%F0%9F%91%8D",
	}
	for key, value := range values {
		path = strings.ReplaceAll(path, "{"+key+"}", value)
	}
	return path
}

func ticketCollaborationOwnsMethod(operations []routing.Operation, method string) bool {
	for _, operation := range operations {
		if operation.Method == method {
			return true
		}
	}
	return false
}

func ticketCollaborationRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate ticket-collaboration proxy test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	if _, err := os.Stat(filepath.Join(root, "apps/go-api/ticket_collaboration/gen/http/openapi.json")); err != nil {
		t.Fatalf("resolve repository root %s: %v", root, err)
	}
	return root
}
