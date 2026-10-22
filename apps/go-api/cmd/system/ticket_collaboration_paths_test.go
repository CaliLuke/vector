package main

import (
	"encoding/json"

	systemapi "github.com/CaliLuke/go-plane"
	"strings"
	"testing"
)

func TestCollaborationPathConvertersRetainStringAnchorAndReaction(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000030120"
	for _, item := range collaborationProtocols {
		p := item.readServiceProtocol
		path := strings.NewReplacer("{slug}", "workspace", "{project_id}", id, "{issue_id}", id, "{comment_id}", id, "{subscriber_id}", id, "{pk}", id, "{anchor}", "public-board-anchor", "{reaction_code}", "👍").Replace(p.pattern)
		if readServiceProtocolHasMalformedUUID(path, p) {
			t.Errorf("string converter rejected %s", path)
		}
		malformed := strings.Replace(path, id, "invalid-uuid", 1)
		if !readServiceProtocolHasMalformedUUID(malformed, p) {
			t.Errorf("UUID converter accepted %s", malformed)
		}
	}
}

func TestCollaborationOptionsMetadataRetainsPermittedPutAction(t *testing.T) {
	for _, kind := range []string{"comments", "issue-links"} {
		pattern := "/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/" + kind + "/{pk}/"
		protocol := workItemProtocolForPattern(pattern)
		_, body := workItemProtocolMetadata(protocol, systemapi.WorkItemProtocolMetadata{})
		var full map[string]any
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &full); err != nil {
			t.Fatal(err)
		}
		actions, ok := full["actions"].(map[string]any)
		if !ok || actions["PUT"] == nil {
			t.Fatalf("%s lost PUT metadata: %s", kind, raw)
		}
		_, missing := workItemProtocolMetadata(protocol, systemapi.WorkItemProtocolMetadata{OmitPutAction: true})
		missingRaw, _ := json.Marshal(missing)
		var missingFields map[string]any
		if err = json.Unmarshal(missingRaw, &missingFields); err != nil {
			t.Fatal(err)
		}
		if _, present := missingFields["actions"]; present {
			t.Fatalf("unavailable %s exposes writable fields", kind)
		}
		_, again := workItemProtocolMetadata(protocol, systemapi.WorkItemProtocolMetadata{})
		againRaw, _ := json.Marshal(again)
		if string(againRaw) != string(raw) {
			t.Fatal("request mutated shared OPTIONS metadata")
		}
	}
}
