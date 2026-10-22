package design_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loom "github.com/CaliLuke/loom/pkg"
)

// Target decisions deliberately differ from immutable Django fixtures. This
// checks only their generated error transport; it does not claim the domain
// enforces the selected tenant boundary.
func TestWorkItemGeneratedErrorsPreserveSelectedTargetBodies(t *testing.T) {
	data, err := os.ReadFile("../../../../docs/go-backend-replacement/work-item-core/target-contract-decisions.json")
	if err != nil {
		t.Fatal(err)
	}
	var decisions struct {
		Decisions []struct {
			ID           string         `json:"id"`
			TargetResult map[string]any `json:"target_result"`
		}
	}
	if err := json.Unmarshal(data, &decisions); err != nil {
		t.Fatal(err)
	}
	checked := 0
	var checkBody func(string, int, map[string]any)
	checkBody = func(name string, status int, body map[string]any) {
		if status < 400 {
			return
		}
		checked++
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			failure := &api.WorkItemError{Kind: fmt.Sprintf("status_%d", status), Body: loom.JSONValue(raw)}
			handler, service := transport(t)
			service.failure = failure
			request := httptest.NewRequest("POST", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != status {
				t.Fatalf("status=%d; target=%d", response.Code, status)
			}
			var actual map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, body) {
				t.Fatalf("generated error=%s; selected target=%s", response.Body, raw)
			}
		})
	}
	var walk func(string, int, map[string]any)
	walk = func(name string, status int, value map[string]any) {
		if code, ok := value["status"].(float64); ok {
			status = int(code)
		}
		for key, child := range value {
			object, ok := child.(map[string]any)
			if !ok {
				continue
			}
			switch key {
			case "body":
				checkBody(name+"/body", status, object)
			case "body_by_input", "body_by_invalid_field", "body_by_effective_input":
				for field, raw := range object {
					// These decision maps associate each input field with its JSON
					// error value (often an array of messages). The response body
					// is the object that wraps that value under the field name.
					body := map[string]any{field: raw}
					checkBody(name+"/"+field, status, body)
				}
			default:
				walk(name+"/"+key, status, object)
			}
		}
	}
	for _, decision := range decisions.Decisions {
		walk(decision.ID, 0, decision.TargetResult)
	}
	if checked == 0 {
		t.Fatal("no selected error responses checked")
	}
	t.Logf("checked %d selected target error bodies through generated transport", checked)
}
