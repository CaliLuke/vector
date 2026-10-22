package systemapi

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This checks the authored delegation against the source contract, the Loom
// design, and generated HTTP metadata without maintaining a second route list.
func TestWorkItemCoreMethodAdaptersMatchContract(t *testing.T) {
	const family = "work_item_core"
	var contract struct {
		Operations map[string]struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"operations"`
	}
	readJSON(t, filepath.Join("..", "..", "docs", "go-backend-replacement", "contracts", family+".json"), &contract)
	if len(contract.Operations) != 72 {
		t.Fatalf("contract has %d operations, want 72", len(contract.Operations))
	}

	// Loom must preserve the canonical source route in x-plane-contract-id,
	// while these three routes use a single generated transport segment.
	identifierTransportRoutes := map[string]string{
		"GET /api/v1/workspaces/{slug}/issues/{project_identifier}-{issue_identifier}/":     "GET /api/v1/workspaces/{slug}/issues/{identifier}/",
		"GET /api/v1/workspaces/{slug}/work-items/{project_identifier}-{issue_identifier}/": "GET /api/v1/workspaces/{slug}/work-items/{identifier}/",
		"GET /api/workspaces/{slug}/work-items/{project_identifier}-{issue_identifier}/":    "GET /api/workspaces/{slug}/work-items/{identifier}/",
	}

	var openAPI struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	readJSON(t, filepath.Join(family, "gen", "http", "openapi.json"), &openAPI)
	metadata := make(map[string]string)
	contractIDs := make(map[string]string)
	for path, methods := range openAPI.Paths {
		for method, data := range methods {
			if !isHTTPMethod(method) {
				continue
			}
			var operation struct {
				ID         string `json:"operationId"`
				ContractID string `json:"x-plane-contract-id"`
			}
			if err := json.Unmarshal(data, &operation); err != nil {
				t.Fatalf("%s %s metadata: %v", method, path, err)
			}
			if operation.ID == "" || metadata[operation.ID] != "" {
				t.Fatalf("missing or duplicate generated operation ID %q", operation.ID)
			}
			metadata[operation.ID] = strings.ToUpper(method) + " " + path
			contractIDs[operation.ID] = operation.ContractID
		}
	}

	design := parseGoFile(t, filepath.Join(family, "design", "design.go"))
	designIDs := make(map[string]string)
	ast.Inspect(design, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || callName(call.Fun) != "Method" || len(call.Args) != 2 {
			return true
		}
		methodName, ok := stringLiteral(call.Args[0])
		if !ok {
			return true
		}
		body, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			t.Errorf("Method(%q) has no design body", methodName)
			return false
		}
		ast.Inspect(body, func(child ast.Node) bool {
			meta, ok := child.(*ast.CallExpr)
			if !ok || callName(meta.Fun) != "Meta" || len(meta.Args) < 2 {
				return true
			}
			key, _ := stringLiteral(meta.Args[0])
			if key == "openapi:operationId" {
				id, _ := stringLiteral(meta.Args[1])
				designIDs[methodName] = id
			}
			return true
		})
		return false
	})

	adapters := parseGoFile(t, "work_item_core_methods.go")
	seen := make(map[string]string)
	for _, declaration := range adapters.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil {
			continue
		}
		if len(method.Body.List) != 1 {
			t.Errorf("%s must only delegate to execute", method.Name.Name)
			continue
		}
		result, ok := method.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(result.Results) != 1 {
			t.Errorf("%s must return execute result", method.Name.Name)
			continue
		}
		call, ok := result.Results[0].(*ast.CallExpr)
		if !ok || callName(call.Fun) != "execute" || len(call.Args) != 4 {
			t.Errorf("%s must call execute with context, ID, payload, and body", method.Name.Name)
			continue
		}
		id, ok := stringLiteral(call.Args[1])
		if !ok {
			t.Errorf("%s has nonliteral operation ID", method.Name.Name)
			continue
		}
		if !identNamed(call.Args[0], "ctx") || !identNamed(call.Args[2], "payload") {
			t.Errorf("%s forwards different context or payload", method.Name.Name)
		}
		bodyArgument := "nil"
		if len(method.Type.Params.List) == 3 {
			bodyArgument = "requestBody"
		}
		if !identNamed(call.Args[3], bodyArgument) {
			t.Errorf("%s does not forward %s as its body", method.Name.Name, bodyArgument)
		}
		if expected := designIDs[method.Name.Name]; expected != id {
			t.Errorf("%s delegates to %q, design selects %q", method.Name.Name, id, expected)
		}
		if previous := seen[id]; previous != "" {
			t.Errorf("%q delegated by both %s and %s", id, previous, method.Name.Name)
		}
		seen[id] = method.Name.Name
	}

	if len(seen) != len(contract.Operations) || len(metadata) != len(contract.Operations) || len(designIDs) != len(contract.Operations) {
		t.Errorf("operation counts: adapters=%d design=%d generated=%d contract=%d", len(seen), len(designIDs), len(metadata), len(contract.Operations))
	}
	for id, operation := range contract.Operations {
		if seen[id] == "" {
			t.Errorf("missing adapter for %s", id)
		}
		canonicalID := operation.Method + " " + operation.Path
		if contractIDs[id] != canonicalID {
			t.Errorf("%s generated contract ID = %q, canonical source contract = %q", id, contractIDs[id], canonicalID)
		}
		wantTransportRoute := canonicalID
		if mapped, ok := identifierTransportRoutes[canonicalID]; ok {
			wantTransportRoute = mapped
		}
		if metadata[id] != wantTransportRoute {
			t.Errorf("%s generated transport route = %q, want %q", id, metadata[id], wantTransportRoute)
		}
	}
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func parseGoFile(t *testing.T, path string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func callName(expr ast.Expr) string {
	switch call := expr.(type) {
	case *ast.Ident:
		return call.Name
	case *ast.SelectorExpr:
		return call.Sel.Name
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	value, ok := expr.(*ast.BasicLit)
	if !ok || value.Kind != token.STRING {
		return "", false
	}
	unquoted, err := strconv.Unquote(value.Value)
	return unquoted, err == nil
}

func identNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func isHTTPMethod(method string) bool {
	switch method {
	case "get", "post", "put", "patch", "delete", "head", "options":
		return true
	default:
		return false
	}
}
