package systemapi

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGeneratedOpenAPIOperationIDsAreUnique(t *testing.T) {
	data, err := os.ReadFile("gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode generated OpenAPI document: %v", err)
	}

	operationRoutes := make(map[string]string)
	for path, methods := range document.Paths {
		for method, rawOperation := range methods {
			if !isOpenAPIOperationMethod(method) {
				continue
			}
			var operation struct {
				OperationID string `json:"operationId"`
			}
			if err := json.Unmarshal(rawOperation, &operation); err != nil {
				t.Errorf("decode %s %s operation: %v", method, path, err)
				continue
			}
			if operation.OperationID == "" {
				t.Errorf("%s %s has no operationId", method, path)
				continue
			}
			route := method + " " + path
			if previous, exists := operationRoutes[operation.OperationID]; exists {
				t.Errorf("operationId %q is shared by %s and %s", operation.OperationID, previous, route)
				continue
			}
			operationRoutes[operation.OperationID] = route
		}
	}
}

func isOpenAPIOperationMethod(method string) bool {
	switch method {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}
