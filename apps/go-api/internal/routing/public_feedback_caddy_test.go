package routing

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestPublicFeedbackRetirementPolicyRoutesEveryMethodAndSlashForm(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	policy, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}

	blocks := regexp.MustCompile(`(?s)@go_active_[0-9]+ \{([^}]*)\}`).FindAllStringSubmatch(string(policy), -1)
	paths := make(map[string]Operation)
	feedbackPaths := make(map[string]struct{})
	var collection *Operation
	for _, operation := range inventory.Operations {
		if operation.Implementation == nil || (*operation.Implementation != "public_feedback" && *operation.Implementation != "public_issue_collection") {
			continue
		}
		if operation.State != StateDjangoRemoved {
			t.Fatalf("public route is not retired: %#v", operation)
		}
		switch *operation.Implementation {
		case "public_feedback":
			feedbackPaths[operation.Path] = struct{}{}
			paths[operation.Path] = operation
		case "public_issue_collection":
			collection = &operation
			paths[operation.Path] = operation
		}
	}
	if len(feedbackPaths) != 6 || len(paths) != 7 {
		t.Fatalf("retired public routes=%d (%d public feedback paths, want 6 plus collection)", len(paths), len(feedbackPaths))
	}
	if collection == nil || collection.OperationID != "get_api_public_anchor_by_anchor_issues" || collection.Implementation == nil || *collection.Implementation != "public_issue_collection" {
		t.Fatalf("public issue collection must be owned by its Go family: %#v", collection)
	}

	for path, operation := range paths {
		pattern := strings.TrimSuffix(caddyPathPattern(operation), "/$") + "/?$"
		want := fmt.Sprintf("path_regexp %s", pattern)
		found := false
		for _, block := range blocks {
			if !strings.Contains(block[1], want) {
				continue
			}
			found = true
			if strings.Contains(block[1], "\tmethod ") {
				t.Errorf("retired path %s restricts methods in proxy policy: %s", path, block[1])
			}
		}
		if !found {
			t.Errorf("retired path %s has no all-method slash-form matcher %q", path, want)
		}
	}
}
