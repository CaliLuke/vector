package routing

import (
	"regexp"
	"strings"
	"testing"
)

func TestRenderCaddyUsesOnlyGoActiveOperations(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{
		Response:        []string{"response"},
		DatabaseChanges: []string{"database"},
		EmittedEffects:  []string{"effects"},
	}
	inventory := Inventory{
		Version:       1,
		Source:        "test",
		DefaultState:  StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{
			{Method: "GET", Path: "/django/", OperationID: "django", State: StateDjangoOnly},
			{Method: "POST", Path: "/implemented/", OperationID: "implemented", State: StateGoImplemented, Implementation: &implementation},
			{Method: "PATCH", Path: "/api/workspaces/{slug}/items/{pk}/", OperationID: "active", State: StateGoActive, Implementation: &implementation, ParityEvidence: evidence},
		},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	if strings.Contains(policy, "/django/") || strings.Contains(policy, "/implemented/") {
		t.Fatalf("inactive route leaked into Caddy policy:\n%s", policy)
	}
	for _, want := range []string{"method PATCH", `path_regexp ^/api/workspaces/[^/]+/items/[^/]+/$`, "reverse_proxy @go_active_001"} {
		if !strings.Contains(policy, want) {
			t.Errorf("Caddy policy omits %q:\n%s", want, policy)
		}
	}
}

func TestRuntimeFallbackRequiresCompleteDjangoRetirement(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations:    []Operation{{Method: "GET", Path: "/api/example/", OperationID: "example", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")}},
	}
	for _, state := range []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved} {
		t.Run(state, func(t *testing.T) {
			inventory.Operations[0].State = state
			policy, err := RenderCaddy(inventory)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(policy), "reverse_proxy @go_runtime_fallback"); got != (state == StateDjangoRemoved) {
				t.Fatalf("runtime fallback = %v for %s", got, state)
			}
		})
	}
}

func TestStickyCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	implementation := "stickies"
	operation := Operation{
		Method: "GET", Path: "/api/workspaces/{slug}/stickies/{pk}/",
		Implementation: &implementation,
	}
	pattern := regexp.MustCompile(caddyPathPattern(operation))
	for _, valid := range []string{
		"/api/workspaces/acme/stickies/00000000-0000-4000-8000-000000000001/",
		"/api/workspaces/acme/stickies/550E8400-E29B-41D4-A716-446655440000/",
		"/api/workspaces/acme/stickies/550e8400e29b41d4a716446655440000/",
		"/api/workspaces/acme/stickies/{550e8400-e29b-41d4-a716-446655440000}/",
		"/api/workspaces/acme/stickies/urn:uuid:550e8400-e29b-41d4-a716-446655440000/",
	} {
		if !pattern.MatchString(valid) {
			t.Fatalf("valid UUID path %q does not match %s", valid, pattern)
		}
	}
	for _, invalid := range []string{
		"/api/workspaces/acme/stickies/not-a-uuid/",
		"/api/workspaces/acme/stickies/550e8400-e29b-41d4-0716-446655440000/",
		"/api/workspaces/acme/stickies/550e8400e29b41d4f716446655440000/",
		"/api/workspaces/acme/stickies/{550e8400-e29b-41d4-0716-446655440000}/",
		"/api/workspaces/acme/stickies/urn:uuid:550e8400-e29b-41d4-f716-446655440000/",
	} {
		if pattern.MatchString(invalid) {
			t.Fatalf("decoder-invalid UUID path %q unexpectedly matches %s", invalid, pattern)
		}
	}
}
func TestProjectMembershipCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	implementation := "project_membership_reads"
	operation := Operation{
		Method: "GET", Path: "/api/workspaces/{slug}/projects/{project_id}/members/{pk}/",
		Implementation: &implementation,
	}
	pattern := regexp.MustCompile(caddyPathPattern(operation))
	for _, valid := range []string{
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/members/00000000-0000-4000-8000-000000000002/",
		"/api/workspaces/acme/projects/00000000-0000-0000-0000-000000000001/members/00000000-0000-f000-f000-000000000002/",
	} {
		if !pattern.MatchString(valid) {
			t.Fatalf("Django-compatible UUID path %q does not match %s", valid, pattern)
		}
	}
	for _, invalid := range []string{
		"/api/workspaces/acme/projects/not-a-uuid/members/00000000-0000-4000-8000-000000000002/",
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/members/not-a-uuid/",
		"/api/workspaces/acme/projects/550E8400-E29B-41D4-A716-446655440000/members/00000000-0000-4000-8000-000000000002/",
		"/api/workspaces/acme/projects/00000000000040008000000000000001/members/00000000-0000-4000-8000-000000000002/",
	} {
		if pattern.MatchString(invalid) {
			t.Fatalf("Django-unsupported UUID path %q unexpectedly matches %s", invalid, pattern)
		}
	}
}

func TestProjectStateCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	implementation := "project_state_reads"
	operation := Operation{
		Method: "GET", Path: "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/",
		Implementation: &implementation,
	}
	pattern := regexp.MustCompile(caddyPathPattern(operation))
	valid := "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/states/00000000-0000-4000-8000-000000000002/"
	if !pattern.MatchString(valid) {
		t.Fatalf("valid UUID path does not match %s", pattern)
	}
	for _, invalid := range []string{
		"/api/workspaces/acme/projects/not-a-uuid/states/00000000-0000-4000-8000-000000000002/",
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/states/not-a-uuid/",
		"/api/workspaces/acme/projects/550E8400-E29B-41D4-A716-446655440000/states/00000000-0000-4000-8000-000000000002/",
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/states/00000000000040008000000000000002/",
	} {
		if pattern.MatchString(invalid) {
			t.Fatalf("Django-unsupported UUID path %q unexpectedly matches %s", invalid, pattern)
		}
	}
}

func TestProjectLabelCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	for _, implementation := range []string{"project_label_reads", "project_label_writes"} {
		t.Run(implementation, func(t *testing.T) {
			operation := Operation{
				Method: "GET", Path: "/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/",
				Implementation: &implementation,
			}
			pattern := regexp.MustCompile(caddyPathPattern(operation))
			valid := "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issue-labels/00000000-0000-4000-8000-000000000002/"
			if !pattern.MatchString(valid) {
				t.Fatalf("valid UUID path does not match %s", pattern)
			}
			for _, invalid := range []string{
				"/api/workspaces/acme/projects/not-a-uuid/issue-labels/00000000-0000-4000-8000-000000000002/",
				"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issue-labels/not-a-uuid/",
				"/api/workspaces/acme/projects/550E8400-E29B-41D4-A716-446655440000/issue-labels/00000000-0000-4000-8000-000000000002/",
				"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issue-labels/00000000000040008000000000000002/",
			} {
				if pattern.MatchString(invalid) {
					t.Fatalf("Django-unsupported UUID path %q unexpectedly matches %s", invalid, pattern)
				}
			}
		})
	}
}

func TestPublicProjectAnchorCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	implementation := "public_project_metadata_reads"
	operation := Operation{
		Method: "GET", Path: "/api/public/workspaces/{slug}/projects/{project_id}/anchor/",
		Implementation: &implementation,
	}
	pattern := regexp.MustCompile(caddyPathPattern(operation))
	for _, valid := range []string{
		"/api/public/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/anchor/",
		"/api/public/workspaces/acme/projects/00000000-0000-0000-0000-000000000000/anchor/",
		"/api/public/workspaces/acme/projects/00000000-0000-f000-f000-000000000000/anchor/",
	} {
		if !pattern.MatchString(valid) {
			t.Fatalf("Django-compatible UUID path %q does not match %s", valid, pattern)
		}
	}
	for _, invalid := range []string{
		"/api/public/workspaces/acme/projects/not-a-uuid/anchor/",
		"/api/public/workspaces/acme/projects/00000000000040008000000000000001/anchor/",
		"/api/public/workspaces/acme/projects/550E8400-E29B-41D4-A716-446655440000/anchor/",
		"/api/public/workspaces/acme/projects/{00000000-0000-4000-8000-000000000001}/anchor/",
		"/api/public/workspaces/acme/projects/urn:uuid:00000000-0000-4000-8000-000000000001/anchor/",
	} {
		if pattern.MatchString(invalid) {
			t.Fatalf("unsupported public project UUID path %q unexpectedly matches %s", invalid, pattern)
		}
	}
}
func TestActivityReadsCaddyPatternLeavesMalformedUUIDsWithDjango(t *testing.T) {
	implementation := "activity_reads"
	operation := Operation{
		Method:         "GET",
		Path:           "/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/{pk}/",
		Implementation: &implementation,
	}
	pattern := regexp.MustCompile(caddyPathPattern(operation))
	valid := "/api/v1/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/00000000-0000-4000-8000-000000000003/"
	if !pattern.MatchString(valid) {
		t.Fatalf("Django-compatible UUID path %q does not match %s", valid, pattern)
	}
	for _, invalid := range []string{
		"/api/v1/workspaces/acme/projects/not-a-uuid/issues/00000000-0000-4000-8000-000000000002/activities/00000000-0000-4000-8000-000000000003/",
		"/api/v1/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issues/not-a-uuid/activities/00000000-0000-4000-8000-000000000003/",
		"/api/v1/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/activities/not-a-uuid/",
		"/api/v1/workspaces/acme/projects/550E8400-E29B-41D4-A716-446655440000/issues/00000000-0000-4000-8000-000000000002/activities/00000000-0000-4000-8000-000000000003/",
	} {
		if pattern.MatchString(invalid) {
			t.Fatalf("Django-unsupported UUID path %q unexpectedly matches %s", invalid, pattern)
		}
	}
}

func TestRenderCaddyRoutesGETProtocolCompanionsToGo(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations:    []Operation{{Method: "GET", Path: "/api/example/", OperationID: "example", State: StateGoActive, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")}},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rendered), "method GET HEAD OPTIONS") {
		t.Fatalf("GET protocol companions are not routed to Go:\n%s", rendered)
	}
}

func TestRenderCaddyHonorsEmptyGETProtocolCompanions(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations:    []Operation{{Method: "GET", Path: "/api/v1/example/", OperationID: "example", State: StateGoActive, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods()}},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if policy := string(rendered); !strings.Contains(policy, "method GET\n") || strings.Contains(policy, "HEAD") || strings.Contains(policy, "OPTIONS") {
		t.Fatalf("exact-method GET routing includes an undeclared companion:\n%s", rendered)
	}
}

func TestRenderCaddyRoutesEveryMethodForRetiredPublicMetadataPaths(t *testing.T) {
	implementation := "public_project_metadata_reads"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{{
			Method: "GET", Path: "/api/public/anchor/{anchor}/labels/", OperationID: "labels",
			State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence,
			ProtocolCompanions: companionMethods("HEAD", "OPTIONS"),
		}},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	if strings.Contains(policy, "@go_active_redirect") {
		t.Fatalf("public metadata slashless requests must reach Go middleware before redirect:\n%s", policy)
	}
	start := strings.Index(policy, "@go_active_001 {")
	end := strings.Index(policy[start:], "\n}")
	block := policy[start : start+end]
	if strings.Contains(block, "method ") || !strings.Contains(block, `path_regexp ^/api/public/anchor/[^/]+/labels/?$`) {
		t.Fatalf("public metadata matcher must proxy every method with or without a slash:\n%s", block)
	}
}

func TestRenderCaddyRoutesEverySlashFormAndMethodForRetiredPaths(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{{
			Method: "GET", Path: "/api/workspaces/{slug}/members/", OperationID: "members",
			State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence,
			ProtocolCompanions: companionMethods("HEAD", "OPTIONS"),
		}},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	if !strings.Contains(policy, "path_regexp ^/api/workspaces/[^/]+/members/?$") {
		t.Fatalf("retired route does not match canonical and slashless paths: %s", policy)
	}
	if strings.Contains(policy, "\tmethod ") || strings.Contains(policy, "go_active_redirect") {
		t.Fatalf("retired route must proxy every method and slash form without redirecting: %s", policy)
	}
}

func TestRenderCaddyPreservesAllowForFullyRetiredSplitPaths(t *testing.T) {
	readsImplementation := "project_label_reads"
	writesImplementation := "project_label_writes"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	path := "/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/"
	operations := []Operation{
		{Method: "GET", Path: path, OperationID: "label-read", State: StateDjangoRemoved, Implementation: &readsImplementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
		{Method: "DELETE", Path: path, OperationID: "label-delete", State: StateDjangoOnly, Implementation: &writesImplementation},
		{Method: "PATCH", Path: path, OperationID: "label-patch", State: StateDjangoOnly, Implementation: &writesImplementation},
		{Method: "PUT", Path: path, OperationID: "label-put", State: StateDjangoOnly, Implementation: &writesImplementation},
	}
	makeInventory := func(operations []Operation) Inventory {
		return Inventory{
			Version: 1, Source: "test", DefaultState: StateDjangoOnly,
			AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
			Operations:    operations,
		}
	}

	t.Run("mixed path baseline", func(t *testing.T) {
		rendered, err := RenderCaddy(makeInventory(operations))
		if err != nil {
			t.Fatal(err)
		}
		policy := string(rendered)
		if !strings.Contains(policy, `>Allow "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"`) {
			t.Fatalf("mixed-path Allow does not include each owner and protocol method:\n%s", policy)
		}
		if !strings.Contains(policy, "not method GET HEAD\n\tpath_regexp ^/api/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/issue-labels/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/$") {
			t.Fatalf("mixed-path Allow repair lost its narrow canonical UUID path matcher:\n%s", policy)
		}
	})

	t.Run("all methods retired", func(t *testing.T) {
		retired := append([]Operation(nil), operations...)
		for index := range retired {
			retired[index].State = StateDjangoRemoved
			retired[index].ParityEvidence = evidence
		}
		rendered, err := RenderCaddy(makeInventory(retired))
		if err != nil {
			t.Fatal(err)
		}
		policy := string(rendered)
		if !strings.Contains(policy, `>Allow "GET, PUT, PATCH, DELETE, HEAD, OPTIONS"`) || !strings.Contains(policy, "match header Allow *") {
			t.Fatalf("fully retired split path must restore the complete Allow only when upstream supplied it:\n%s", policy)
		}
		allowStart := strings.Index(policy, "@go_active_allow_001 {")
		if allowStart < 0 {
			t.Fatalf("fully retired split path has no Allow repair matcher:\n%s", policy)
		}
		allowBlockEnd := strings.Index(policy[allowStart:], "\n}")
		if allowBlockEnd < 0 {
			t.Fatalf("could not isolate fully retired Allow matcher:\n%s", policy)
		}
		allowBlock := policy[allowStart : allowStart+allowBlockEnd+2]
		if !strings.Contains(allowBlock, `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/$`) || strings.Contains(allowBlock, `/?$`) {
			t.Fatalf("Allow repair must match only the canonical UUID route with a trailing slash:\n%s", allowBlock)
		}
		if strings.Contains(allowBlock, "method ") {
			t.Fatalf("fully retired Allow repair must cover upstream Allow errors for every method:\n%s", allowBlock)
		}
		if !strings.Contains(policy, "path_regexp ^/api/workspaces/[^/]+/projects/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/issue-labels/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/?$") {
			t.Fatalf("retired Go route must still handle slashless requests:\n%s", policy)
		}
	})
}

func TestRenderCaddyLeavesSlashlessMixedPathsWithDjango(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{
			{Method: "GET", Path: "/api/example/", OperationID: "read", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
			{Method: "PATCH", Path: "/api/example/", OperationID: "write", State: StateDjangoOnly},
		},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered), "go_active_redirect") {
		t.Fatalf("mixed path must retain Django's slash handling:\n%s", rendered)
	}
}

func TestRenderCaddyLeavesOptionsWithDjangoForMixedPaths(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{
			{Method: "GET", Path: "/api/example/", OperationID: "read", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
			{Method: "PATCH", Path: "/api/example/", OperationID: "write", State: StateDjangoOnly},
		},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	if !strings.Contains(policy, "method GET HEAD\n") || strings.Contains(policy, "method GET HEAD OPTIONS") {
		t.Fatalf("mixed-path OPTIONS must stay with Django:\n%s", policy)
	}
	if !strings.Contains(policy, `header @go_active_allow_001 >Allow "GET, PATCH, HEAD, OPTIONS"`) {
		t.Fatalf("mixed-path OPTIONS does not merge the Go read methods into Allow:\n%s", policy)
	}
	if !strings.Contains(policy, "match header Allow *") {
		t.Fatalf("mixed-path Allow repair must only replace an upstream Allow header:\n%s", policy)
	}
	if !strings.Contains(policy, "not method GET HEAD\n\tpath_regexp ^/api/example/$") {
		t.Fatalf("mixed-path Allow repair must apply to all methods not owned by Go:\n%s", policy)
	}
}

func TestRenderCaddyRoutesUnsupportedMethodsForFullyRetiredPathsToGo(t *testing.T) {
	implementation := "activity_reads"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{{
			Method: "GET", Path: "/api/users/me/activities/", OperationID: "activities", State: StateDjangoRemoved,
			Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS"),
		}},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	start := strings.Index(policy, "@go_active_001 {")
	if start < 0 {
		t.Fatalf("Caddy policy omits retired route matcher:\n%s", policy)
	}
	end := strings.Index(policy[start:], "reverse_proxy @go_active_001")
	if end < 0 {
		t.Fatalf("Caddy policy omits retired route proxy:\n%s", policy)
	}
	matcher := policy[start : start+end]
	if strings.Contains(matcher, "\tmethod ") {
		t.Fatalf("fully retired canonical path still filters unsupported methods:\n%s", matcher)
	}
	if strings.Contains(policy, "@go_active_redirect_001 {\n\tmethod ") {
		t.Fatalf("fully retired slashless path still filters unsupported methods:\n%s", policy)
	}
}

func TestRenderCaddyKeepsUnsupportedMethodsOnDjangoForMixedPaths(t *testing.T) {
	implementation := "example"
	evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
	inventory := Inventory{
		Version: 1, Source: "test", DefaultState: StateDjangoOnly,
		AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved},
		Operations: []Operation{
			{Method: "GET", Path: "/api/example/", OperationID: "read", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
			{Method: "POST", Path: "/api/example/", OperationID: "write", State: StateDjangoOnly},
		},
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rendered), "@go_active_001 {\n\tmethod GET HEAD") {
		t.Fatalf("mixed path no longer leaves unsupported methods on Django:\n%s", rendered)
	}
}
func TestCommittedCaddyPolicyRoutesRetiredOperations(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	policy := string(rendered)
	if got, want := strings.Count(policy, "reverse_proxy @go_active_"), len(inventory.ActiveOperations()); got != want {
		t.Fatalf("Caddy policy Go route count = %d, want inventory count %d:\n%s", got, want, policy)
	}
	activityImplementation := "activity_reads"
	activityPattern := strings.TrimSuffix(caddyPathPattern(Operation{
		Path:           "/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/",
		Implementation: &activityImplementation,
	}), "/$") + "/?$"
	if !strings.Contains(policy, "path_regexp "+activityPattern) {
		t.Fatalf("Caddy policy omits retired activity routes:\n%s", policy)
	}
	for _, path := range []string{
		"path_regexp ^/api/timezones/?$",
		"path_regexp ^/api/workspace-slug-check/?$",
		"path_regexp ^/api/instances/workspace-slug-check/?$",
		"path_regexp ^/api/users/me/instance-admin/?$",
		"path_regexp ^/api/users/me/onboard/?$",
		"path_regexp ^/api/users/me/tour-completed/?$",
		"path_regexp ^/api/users/me/notification-preferences/?$",
		"path_regexp ^/api/users/last-visited-workspace/?$",
		"path_regexp ^/api/workspaces/[^/]+/user-favorite-projects/?$",
		"path_regexp ^/api/users/me/accounts/?$",
		"path_regexp ^/api/users/me/accounts/[^/]+/?$",
		"path_regexp ^/api/users/me/settings/?$",
		"path_regexp ^/api/workspaces/[^/]+/project-identifiers/?$",
		"path_regexp ^/api/workspaces/[^/]+/sidebar-preferences/?$",
		"path_regexp ^/api/workspaces/[^/]+/views/?$",
		"path_regexp ^/api/workspaces/[^/]+/views/[^/]+/?$",
		"path_regexp ^/api/workspaces/[^/]+/projects/[^/]+/views/?$",
		"path_regexp ^/api/workspaces/[^/]+/projects/[^/]+/views/[^/]+/?$",
		"path_regexp ^/api/workspaces/[^/]+/workspace-members/me/?$",
		"path_regexp ^/api/workspaces/[^/]+/recent-visits/?$",
		"path_regexp ^/api/workspaces/[^/]+/analytic-view/?$",
		"path_regexp ^/api/workspaces/[^/]+/analytic-view/[^/]+/?$",
	} {
		if !strings.Contains(policy, path) {
			t.Fatalf("Caddy policy omits retired route %s:\n%s", path, policy)
		}
	}
}

func TestNotificationsCaddyPreservesDjangoUUIDConverter(t *testing.T) {
	for _, implementation := range []string{"notification_reads", "notification_actions"} {
		operation := Operation{Method: "GET", Path: "/api/workspaces/{slug}/users/notifications/{pk}/", Implementation: &implementation}
		pattern := regexp.MustCompile(caddyPathPattern(operation))
		for _, id := range []string{"00000000-0000-0000-0000-000000000002", "00000000-0000-4000-8000-000000000002"} {
			if !pattern.MatchString("/api/workspaces/acme/users/notifications/" + id + "/") {
				t.Fatalf("valid Django UUID %s rejected", id)
			}
		}
		for _, id := range []string{"not-a-uuid", "550E8400-E29B-41D4-A716-446655440000", "00000000000040008000000000000002"} {
			if pattern.MatchString("/api/workspaces/acme/users/notifications/" + id + "/") {
				t.Fatalf("malformed UUID %s reached Go", id)
			}
		}
	}
}

func TestRetiredWritesPreserveDjangoMethodsOnMixedPaths(t *testing.T) {
	for _, implementation := range []string{"notification_actions", "public_project_metadata_reads"} {
		t.Run(implementation, func(t *testing.T) {
			evidence := &ParityEvidence{Response: []string{"response"}, DatabaseChanges: []string{"database"}, EmittedEffects: []string{"effects"}}
			inventory := Inventory{Version: 1, Source: "test", DefaultState: StateDjangoOnly, AllowedStates: []string{StateDjangoOnly, StateGoImplemented, StateGoParity, StateGoActive, StateDjangoRemoved}, Operations: []Operation{
				{Method: "GET", Path: "/api/example/", OperationID: "get", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence, ProtocolCompanions: companionMethods("HEAD", "OPTIONS")},
				{Method: "DELETE", Path: "/api/example/", OperationID: "delete", State: StateDjangoRemoved, Implementation: &implementation, ParityEvidence: evidence},
				{Method: "PATCH", Path: "/api/example/", OperationID: "patch", State: StateDjangoOnly},
			}}
			rendered, err := RenderCaddy(inventory)
			if err != nil {
				t.Fatal(err)
			}
			policy := string(rendered)
			// Every reverse proxy matcher must be method-limited. In particular the
			// retired DELETE must not claim PATCH, OPTIONS or slashless fallback.
			proxies := regexp.MustCompile(`(?m)^reverse_proxy (@\S+) `).FindAllStringSubmatch(policy, -1)
			if len(proxies) != 2 {
				t.Fatalf("Go proxy count = %d, want 2", len(proxies))
			}
			for _, proxy := range proxies {
				matcher := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(proxy[1]) + ` \{\n(.*?)^\}`)
				block := matcher.FindString(policy)
				if !strings.Contains(block, "\tmethod ") || strings.Contains(block, "PATCH") || strings.Contains(block, "OPTIONS") || strings.Contains(block, "/?$") {
					t.Fatalf("retired write stole retained Django dispatch via %s:\n%s", proxy[1], block)
				}
			}
			if !strings.Contains(policy, "\tmethod DELETE\n") || !strings.Contains(policy, "\tmethod GET HEAD\n") {
				t.Fatalf("Go application methods missing:\n%s", policy)
			}
			if strings.Count(policy, "header @go_active_allow_") != 1 || !strings.Contains(policy, `>Allow "GET, PATCH, DELETE, HEAD, OPTIONS"`) {
				t.Fatalf("expected one complete mixed-path Allow repair:\n%s", policy)
			}
		})
	}
}

func TestRejectedProtocolCompanionsAreRoutedButNotAllowed(t *testing.T) {
	inventory, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, operation := range inventory.Operations {
		if operation.Implementation == nil || *operation.Implementation != "api_cycle_reads" {
			continue
		}
		found++
		if operation.AllowedProtocolCompanions.IsZero() {
			t.Fatal("cycle source explicitly rejects protocol companions")
		}
		if got := operationProxyMethods(operation, false); got != "GET HEAD OPTIONS" {
			t.Fatalf("routing lost rejected companions: %q", got)
		}
		allow := pathAllowHeader(inventory, operation.Path)
		if strings.Contains(allow, "HEAD") || strings.Contains(allow, "OPTIONS") {
			t.Fatalf("source-rejected method advertised: %s %q", operation.Path, allow)
		}
		if djangoOwnsPathMethod(inventory, operation.Path) && allow == "GET" {
			t.Fatalf("retained writes missing from Allow: %s", operation.Path)
		}
	}
	if found != 6 {
		t.Fatalf("checked %d cycle operations, want 6", found)
	}
	rendered, err := RenderCaddy(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rendered), `>Allow "GET, POST"`) {
		t.Fatal("mixed cycle Allow lost retained POST")
	}
}
