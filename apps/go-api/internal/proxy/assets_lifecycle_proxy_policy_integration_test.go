//go:build integration

package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/routing"
)

const assetsLifecycleProxyOperationCount = 81

type assetsLifecycleOpenAPIOperation struct {
	ID string `json:"operationId"`
}

type assetsLifecycleProxyOperation struct {
	Method string
	Path   string
	ID     string
}

type assetsLifecycleProxyContract struct {
	ControlFixtureRefs []string `json:"control_fixture_refs"`
}

// This exercises the production Caddyfile against the exact family operations
// and rendered current inventory. The Go-only API catchall owns all API paths,
// including unsupported methods and neighboring paths, so Django remains a
// distinct stub that detects an accidental fallback.
func TestAssetsLifecycleGoOnlyProductionProxyPolicy(t *testing.T) {
	root := assetsLifecycleProxyRepositoryRoot(t)
	operations, err := assetsLifecycleProxyOpenAPIOperations(filepath.Join(root, "apps/go-api/assetslifecycle/gen/http/openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The migration inventory is frozen; Vector's workspace draft association
	// route is an additional operation served by the same Go-only API policy.
	if len(operations) != assetsLifecycleProxyOperationCount+1 {
		t.Fatalf("assets_lifecycle OpenAPI operation count=%d, want %d", len(operations), assetsLifecycleProxyOperationCount+1)
	}
	inventory, err := routing.LoadFile(filepath.Join(root, "apps/go-api/internal/routing/route-inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	family := make(map[string]routing.Operation, len(operations))
	byPath := make(map[string][]routing.Operation)
	draftAssociationFound := false
	for _, operation := range operations {
		key := operation.Method + " " + operation.Path
		if key == "POST /api/assets/v2/workspaces/{slug}/{entity_id}/bulk/" {
			if operation.ID != "post_api_assets_v2_workspaces_by_slug_by_entity_id_bulk" {
				t.Fatalf("unexpected draft association operation ID %q", operation.ID)
			}
			draftAssociationFound = true
			continue
		}
		for _, candidate := range inventory.Operations {
			if candidate.Method == operation.Method && candidate.Path == operation.Path {
				if candidate.OperationID != operation.ID {
					t.Fatalf("route inventory operation ID for %s=%q, want %q", key, candidate.OperationID, operation.ID)
				}
				family[key] = candidate
				byPath[operation.Path] = append(byPath[operation.Path], candidate)
				break
			}
		}
	}
	if !draftAssociationFound {
		t.Fatal("workspace draft association operation is absent")
	}
	if len(family) != assetsLifecycleProxyOperationCount {
		t.Fatalf("route inventory covers %d of %d assets_lifecycle operations", len(family), assetsLifecycleProxyOperationCount)
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
	var contract assetsLifecycleProxyContract
	contractPath := filepath.Join(root, "docs/go-backend-replacement/contracts/assets_lifecycle.json")
	readAssetsLifecycleProxyJSON(t, contractPath, &contract)
	if err := validateAssetsLifecycleControlFixtureRefs(contract.ControlFixtureRefs); err != nil {
		t.Fatalf("invalid assets lifecycle control fixture refs: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	django := newNamedUpstream(t, "django")
	goAPI := newNamedUpstream(t, "go")
	proxyURL := startCaddy(t, ctx, django.port, goAPI.port, nil)
	for _, operation := range operations {
		operation := operation
		t.Run(operation.Method+" "+operation.Path, func(t *testing.T) {
			assertGoProxyResponse(t, operation.Method, proxyURL+assetsLifecycleProxyConcretePath(operation.Path))
		})
	}

	var decision struct {
		IncludedOperationCount int `json:"included_operation_count"`
		Routes                 []struct {
			Path           string   `json:"path"`
			InvalidMethods []string `json:"invalid_methods"`
		} `json:"routes"`
	}
	decisionFile := filepath.Join(root, "docs/go-backend-replacement/decisions/ASSET-001-invalid-methods.json")
	readAssetsLifecycleProxyJSON(t, decisionFile, &decision)
	if decision.IncludedOperationCount != 23 {
		t.Fatalf("invalid-method decision includes %d operations, want 23", decision.IncludedOperationCount)
	}
	for _, route := range decision.Routes {
		pathOperations := byPath[route.Path]
		if len(pathOperations) == 0 {
			t.Errorf("invalid-method decision path %s is absent from selected OpenAPI routes", route.Path)
			continue
		}
		for _, method := range route.InvalidMethods {
			method := method
			t.Run("ASSET-001 "+method+" "+route.Path, func(t *testing.T) {
				assertGoProxyResponse(t, method, proxyURL+assetsLifecycleProxyConcretePath(route.Path))
			})
		}
	}

	for _, fixturePath := range contract.ControlFixtureRefs {
		fixturePath := fixturePath
		t.Run("control "+filepath.Base(fixturePath), func(t *testing.T) {
			var fixture struct {
				Request struct {
					Method string `json:"method"`
					Path   string `json:"path"`
				} `json:"request"`
			}
			readAssetsLifecycleProxyJSON(t, filepath.Join(root, fixturePath), &fixture)
			path := assetsLifecycleProxyFixturePath(fixture.Request.Path)
			var pathOperations []routing.Operation
			for template, candidates := range byPath {
				if assetsLifecycleProxyMatchesPath(template, path) {
					pathOperations = candidates
					break
				}
			}
			if len(pathOperations) == 0 {
				t.Fatalf("control request %s %s does not match an assets lifecycle path", fixture.Request.Method, path)
			}
			assertGoProxyResponse(t, fixture.Request.Method, proxyURL+path)
		})
	}

	for template, pathOperations := range byPath {
		path := assetsLifecycleProxyConcretePath(template)
		for _, method := range []string{http.MethodPut, http.MethodTrace} {
			if assetsLifecycleProxyOwnsMethod(pathOperations, method) {
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
		t.Run("slashless "+template, func(t *testing.T) {
			assertGoProxyResponse(t, http.MethodGet, proxyURL+strings.TrimSuffix(path, "/"))
		})
		t.Run("extra segment "+template, func(t *testing.T) {
			// Two extra segments avoid matching one-segment sibling routes such as asset details.
			assertGoProxyResponse(t, http.MethodGet, proxyURL+path+"unowned/extra/")
		})
	}
	t.Run("unknown API path routes to Go", func(t *testing.T) {
		assertGoProxyResponse(t, http.MethodGet, proxyURL+"/api/not-migrated/")
	})
}

func assetsLifecycleProxyFixturePath(path string) string {
	for index, value := range []string{
		"00000000-0000-4000-8000-000000000011", "00000000-0000-4000-8000-000000000012",
		"00000000-0000-4000-8000-000000000013", "00000000-0000-4000-8000-000000000014",
		"00000000-0000-4000-8000-000000000015", "00000000-0000-4000-8000-000000000016",
		"00000000-0000-4000-8000-000000000017", "00000000-0000-4000-8000-000000000018",
		"00000000-0000-4000-8000-000000000019",
	} {
		path = strings.ReplaceAll(path, "<uuid-"+strconv.Itoa(index+1)+">", value)
	}
	return path
}

func assetsLifecycleProxyMatchesPath(template, concrete string) bool {
	var pattern strings.Builder
	for len(template) > 0 {
		start := strings.IndexByte(template, '{')
		if start < 0 {
			pattern.WriteString(regexp.QuoteMeta(template))
			break
		}
		end := strings.IndexByte(template[start:], '}')
		if end < 0 {
			return false
		}
		end += start
		pattern.WriteString(regexp.QuoteMeta(template[:start]))
		pattern.WriteString(`[^/]+`)
		template = template[end+1:]
	}
	matched, err := regexp.MatchString("^"+pattern.String()+"$", concrete)
	return err == nil && matched
}

func assetsLifecycleProxyOpenAPIOperations(path string) ([]assetsLifecycleProxyOperation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document struct {
		Paths map[string]map[string]assetsLifecycleOpenAPIOperation `json:"paths"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	methods := map[string]string{"get": "GET", "post": "POST", "patch": "PATCH", "delete": "DELETE"}
	operations := make([]assetsLifecycleProxyOperation, 0, assetsLifecycleProxyOperationCount)
	for path, entries := range document.Paths {
		for method, entry := range entries {
			if normalized, ok := methods[strings.ToLower(method)]; ok {
				operations = append(operations, assetsLifecycleProxyOperation{Method: normalized, Path: path, ID: entry.ID})
			}
		}
	}
	return operations, nil
}

func assetsLifecycleProxyConcretePath(path string) string {
	values := map[string]string{
		"slug": "assets-lifecycle-workspace", "project_id": "00000000-0000-4000-8000-000000000001",
		"issue_id": "00000000-0000-4000-8000-000000000002", "asset_id": "00000000-0000-4000-8000-000000000003",
		"pk": "00000000-0000-4000-8000-000000000004", "anchor": "assets-lifecycle-board",
		"asset_key": "assets-lifecycle-key", "workspace_id": "00000000-0000-4000-8000-000000000005",
		"entity_id": "00000000-0000-4000-8000-000000000006",
	}
	for key, value := range values {
		path = strings.ReplaceAll(path, "{"+key+"}", value)
	}
	return path
}

func assetsLifecycleProxyOwnsMethod(operations []routing.Operation, method string) bool {
	for _, operation := range operations {
		if operation.Method == method {
			return true
		}
	}
	return false
}

func assetsLifecycleProxyRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate assets lifecycle proxy integration test")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	if _, err := os.Stat(filepath.Join(root, "apps/go-api/assetslifecycle/gen/http/openapi.json")); err != nil {
		t.Fatalf("resolve repository root %s: %v", root, err)
	}
	return root
}

func readAssetsLifecycleProxyJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
