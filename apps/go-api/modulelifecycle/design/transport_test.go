package design_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modulelifecycleclient "github.com/CaliLuke/go-plane/modulelifecycle/gen/http/module_lifecycle_api/client"
	modulelifecycleapi "github.com/CaliLuke/go-plane/modulelifecycle/gen/module_lifecycle_api"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedArchiveClientAcceptsPythonDateTimeRepresentation(t *testing.T) {
	decode := modulelifecycleclient.DecodePostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchiveResponse(loomhttp.ResponseDecoder, false)
	for _, timestamp := range []string{
		"2026-10-20 10:11:12+00:00",
		"2026-10-20 10:11:12.123456+00:00",
	} {
		t.Run(timestamp, func(t *testing.T) {
			response := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"archived_at":"` + timestamp + `"}`)),
			}
			value, err := decode(response)
			if err != nil {
				t.Fatalf("decode Python datetime archive response: %v", err)
			}
			result, ok := value.(*modulelifecycleapi.RestFrameworkSerializersArchiveresponseResponse)
			if !ok {
				t.Fatalf("decoded response type=%T", value)
			}
			if got := string(result.ArchivedAt); got != timestamp {
				t.Errorf("archived_at=%q, want %q", got, timestamp)
			}
		})
	}
}

func TestGeneratedAppOperationsRequireSessionCookieSecurity(t *testing.T) {
	openapi, err := os.ReadFile(filepath.Join("..", "gen", "http", "openapi.json"))
	if err != nil {
		t.Fatalf("read generated OpenAPI contract: %v", err)
	}
	var document struct {
		Components struct {
			SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
		} `json:"components"`
		Paths map[string]map[string]struct {
			Security []map[string][]string `json:"security"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(openapi, &document); err != nil {
		t.Fatalf("decode generated OpenAPI contract: %v", err)
	}
	if _, exists := document.Components.SecuritySchemes["PlaneSessionCookie"]; !exists {
		t.Fatal("generated OpenAPI has no PlaneSessionCookie security scheme")
	}
	count := 0
	for path, methods := range document.Paths {
		if !strings.HasPrefix(path, "/api/workspaces/") {
			continue
		}
		for method, operation := range methods {
			if method != "post" && method != "put" && method != "patch" && method != "delete" {
				continue
			}
			count++
			if len(operation.Security) != 1 {
				t.Errorf("%s %s security=%v, want only PlaneSessionCookie", strings.ToUpper(method), path, operation.Security)
				continue
			}
			if _, secured := operation.Security[0]["PlaneSessionCookie"]; !secured {
				t.Errorf("%s %s security=%v, want PlaneSessionCookie", strings.ToUpper(method), path, operation.Security)
			}
		}
	}
	if count != 14 {
		t.Errorf("found %d app lifecycle operations, want 14", count)
	}
}
