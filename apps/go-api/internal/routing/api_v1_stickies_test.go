package routing

import (
	"regexp"
	"strings"
	"testing"
)

func TestAPIV1StickyPolicyOwnsDRFFormatSuffixesWithoutCapturingInvalidPaths(t *testing.T) {
	implementation := "api_v1_stickies"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{
			{Method: "GET", Path: "/api/not-yet-retired/", OperationID: "remaining", State: StateDjangoOnly},
			{Method: "GET", Path: "/api/v1/workspaces/{slug}/stickies/", OperationID: "list", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
			{Method: "GET", Path: "/api/v1/workspaces/{slug}/stickies/{pk}/", OperationID: "detail", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
		},
	}
	policy, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	var patterns []*regexp.Regexp
	for line := range strings.SplitSeq(string(policy), "\n") {
		if expression, ok := strings.CutPrefix(strings.TrimSpace(line), "path_regexp "); ok {
			patterns = append(patterns, regexp.MustCompile(expression))
		}
	}
	for _, test := range []struct {
		path string
		want bool
	}{
		{"/api/v1/workspaces/example/stickies", true},
		{"/api/v1/workspaces/example/stickies/", true},
		{"/api/v1/workspaces/example/stickies.json", true},
		{"/api/v1/workspaces/example/stickies.json/", true},
		{"/api/v1/workspaces/example/stickies.xml", true},
		{"/api/v1/workspaces/example/stickies/item", true},
		{"/api/v1/workspaces/example/stickies/item/", true},
		{"/api/v1/workspaces/example/stickies/item.json", true},
		{"/api/v1/workspaces/example/stickies/item.json/", true},
		{"/api/v1/workspaces/example/stickies/item.xml", true},
		{"/api/v1/workspaces/example/stickies/.json", false},
		{"/api/v1/workspaces/example/stickies/item.JSON", false},
		{"/api/v1/workspaces/example/stickies/item.json.more", false},
		{"/api/v1/workspaces/example/stickies/item/extra/", false},
		{"/api/v1/workspaces/example/sticky/", false},
	} {
		matched := false
		for _, pattern := range patterns {
			matched = matched || pattern.MatchString(test.path)
		}
		if matched != test.want {
			t.Errorf("proxy owns %q = %v, want %v", test.path, matched, test.want)
		}
	}
}
