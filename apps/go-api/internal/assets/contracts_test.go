package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type assetsOperation struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id"`
}

type assetsFamilyManifest struct {
	OperationCount int                        `json:"operation_count"`
	Operations     map[string]assetsOperation `json:"operations"`
}

type assetsGeneratedOpenAPI struct {
	Paths map[string]map[string]struct {
		OperationID string `json:"operationId"`
	} `json:"paths"`
}

type assetsInvalidMethodDecision struct {
	IncludedOperationCount int      `json:"included_operation_count"`
	Operations             []string `json:"operations"`
	Routes                 []struct {
		Path           string   `json:"path"`
		InvalidMethods []string `json:"invalid_methods"`
	} `json:"routes"`
}

type assetsRoute struct {
	method string
	path   string
}

func TestAssetsGeneratedOperationsMatchFamilyManifest(t *testing.T) {
	root := assetsRepositoryRoot(t)
	var manifest assetsFamilyManifest
	var generated assetsGeneratedOpenAPI
	readAssetsContractJSON(t, root, "docs/go-backend-replacement/assets-lifecycle-manifest.json", &manifest)
	readAssetsContractJSON(t, root, "apps/go-api/assetslifecycle/gen/http/openapi.json", &generated)

	if manifest.OperationCount != 81 || len(manifest.Operations) != 81 {
		t.Fatalf("assets manifest has operation_count=%d and %d operation entries; want 81", manifest.OperationCount, len(manifest.Operations))
	}
	generation := make(map[string]assetsRoute, len(manifest.Operations))
	for path, methods := range generated.Paths {
		for method, operation := range methods {
			if operation.OperationID == "" {
				continue
			}
			if _, exists := generation[operation.OperationID]; exists {
				t.Fatalf("generated OpenAPI repeats operation ID %q", operation.OperationID)
			}
			generation[operation.OperationID] = assetsRoute{method: strings.ToUpper(method), path: path}
		}
	}
	if len(generation) != 81 {
		t.Fatalf("generated assets OpenAPI has %d operations; want 81", len(generation))
	}
	for operationID, operation := range manifest.Operations {
		if operation.OperationID != operationID {
			t.Errorf("manifest key %q disagrees with embedded operation ID %q", operationID, operation.OperationID)
		}
		generatedRoute, ok := generation[operationID]
		if !ok {
			t.Errorf("manifest operation %q is absent from generated OpenAPI", operationID)
			continue
		}
		if generatedRoute.method != strings.ToUpper(operation.Method) || generatedRoute.path != operation.Path {
			t.Errorf("generated route for %q is %s %s; manifest declares %s %s", operationID, generatedRoute.method, generatedRoute.path, strings.ToUpper(operation.Method), operation.Path)
		}
	}
	for operationID := range generation {
		if _, ok := manifest.Operations[operationID]; !ok {
			t.Errorf("generated OpenAPI operation %q is absent from assets manifest", operationID)
		}
	}
}

func TestAssetsInvalidMethodDecisionOperationsMatchDeclaredRoutes(t *testing.T) {
	root := assetsRepositoryRoot(t)
	var manifest assetsFamilyManifest
	var generated assetsGeneratedOpenAPI
	var decision assetsInvalidMethodDecision
	readAssetsContractJSON(t, root, "docs/go-backend-replacement/assets-lifecycle-manifest.json", &manifest)
	readAssetsContractJSON(t, root, "apps/go-api/assetslifecycle/gen/http/openapi.json", &generated)
	readAssetsContractJSON(t, root, "docs/go-backend-replacement/decisions/ASSET-001-invalid-methods.json", &decision)

	if decision.IncludedOperationCount != 23 || len(decision.Operations) != 23 {
		t.Fatalf("invalid-method decision includes count=%d and %d operation IDs; want 23", decision.IncludedOperationCount, len(decision.Operations))
	}
	generation := make(map[assetsRoute]string)
	for path, methods := range generated.Paths {
		for method, operation := range methods {
			if operation.OperationID != "" {
				generation[assetsRoute{method: strings.ToUpper(method), path: path}] = operation.OperationID
			}
		}
	}
	declaredRoutes := make(map[assetsRoute]struct{})
	for _, route := range decision.Routes {
		for _, method := range route.InvalidMethods {
			declaredRoutes[assetsRoute{method: strings.ToUpper(method), path: route.Path}] = struct{}{}
		}
	}

	seen := make(map[string]struct{}, len(decision.Operations))
	for _, operationID := range decision.Operations {
		if _, duplicate := seen[operationID]; duplicate {
			t.Errorf("invalid-method decision repeats operation ID %q", operationID)
			continue
		}
		seen[operationID] = struct{}{}
		operation, ok := manifest.Operations[operationID]
		if !ok {
			t.Errorf("invalid-method operation %q is absent from the assets manifest", operationID)
			continue
		}
		route := assetsRoute{method: strings.ToUpper(operation.Method), path: operation.Path}
		if _, ok := declaredRoutes[route]; !ok {
			t.Errorf("invalid-method operation %q (%s %s) has no matching decision route", operationID, route.method, route.path)
		}
		if generatedID, ok := generation[route]; !ok || generatedID != operationID {
			t.Errorf("invalid-method decision operation %q does not map to its generated route %s %s (got %q)", operationID, route.method, route.path, generatedID)
		}
	}
}

func assetsRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve assets contract test source path")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "../../../.."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func readAssetsContractJSON(t *testing.T, root, relativePath string, destination any) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relativePath))
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read contract artifact %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		t.Fatalf("decode contract artifact %s: %v", filename, err)
	}
}
