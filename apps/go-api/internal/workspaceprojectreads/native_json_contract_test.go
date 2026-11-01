package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func TestNativeProjectJSONPreservesExactNumbersAndAuditRequest(t *testing.T) {
	const raw = `{"icon_prop":{"number":9007199254740993.0,"large":1e300},"unused":{"label":"Zoë","large":1e309}}`
	input, err := requestinput.Parse(strings.NewReader(raw), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	if err != nil {
		t.Fatal(err)
	}
	values, err := repository.validateProjectFields(context.Background(), "workspace", "project", input, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	properties := values["icon_prop"].(map[string]any)
	if properties["number"] != json.Number("9007199254740993.0") || properties["large"] != json.Number("1e300") {
		t.Fatalf("JSON values changed: %#v", properties)
	}
	requested, err := lifecycleRequestedData(input)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(requested)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != raw {
		t.Fatalf("audit request=%s; want %s", encoded, raw)
	}
	if input.Fields["icon_prop"].(map[string]any)["number"] != json.Number("9007199254740993.0") {
		t.Fatal("normalizing the write mutated the request")
	}
	snapshot, err := projectCurrentInstance(json.RawMessage(raw))
	if err != nil || !json.Valid([]byte(snapshot)) {
		t.Fatalf("invalid native activity snapshot: %q, %v", snapshot, err)
	}
	if !strings.Contains(snapshot, "9007199254740993.0") || !strings.Contains(snapshot, "1e309") {
		t.Fatalf("activity snapshot changed numbers: %s", snapshot)
	}

}
