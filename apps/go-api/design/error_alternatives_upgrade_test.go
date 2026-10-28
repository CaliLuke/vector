package design

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// These endpoints return distinct source-backed errors with the same HTTP
// status. The published contract must describe every variant, so a client can
// distinguish a missing preference from field validation, or an invalid intake
// lookup from a bad request. Older Loom generation silently kept only one.
func TestOpenAPIKeepsSameStatusErrorAlternatives(t *testing.T) {
	document := readOpenAPI(t)
	for _, tc := range []struct {
		name, path, method string
		schemas            []string
	}{
		{
			name: "home preference lookup and validation",
			path: "/api/workspaces/{slug}/home-preferences/{key}/", method: "patch",
			schemas: []string{"WorkspaceHomePreferenceInvalid", "WorkspaceHomePreferenceNotFound"},
		},
		{
			name: "intake request and lookup",
			path: "/api/workspaces/{slug}/draft-issues/", method: "get",
			schemas: []string{"IntakeInboxReadsBadRequest", "IntakeInboxReadsInvalidDetail"},
		},
		{
			name: "cycle query errors",
			path: "/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/", method: "get",
			schemas: []string{"CycleReadsBadRequest", "CycleReadsFilterError", "CycleReadsInvalidGrouping", "CycleReadsInvalidPagination", "CycleReadsParseError"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			alternatives := responseAlternativeSchemaNames(t, document.Paths[tc.path][tc.method].Responses["400"])
			if len(alternatives) != len(tc.schemas) {
				t.Fatalf("400 response has %d variants, want %d: %+v", len(alternatives), len(tc.schemas), alternatives)
			}
			for _, name := range tc.schemas {
				if !slices.Contains(alternatives, name) {
					t.Errorf("400 response omitted %s", name)
				}
				if _, exists := document.Components.Schemas[name]; !exists {
					t.Errorf("400 response references missing schema %s", name)
				}
			}
		})
	}
}

func responseAlternativeSchemaNames(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var response struct {
		Content map[string]struct {
			Schema struct {
				AnyOf []struct {
					Ref string `json:"$ref"`
				} `json:"anyOf"`
			}
		}
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, branch := range response.Content["application/json"].Schema.AnyOf {
		name, valid := strings.CutPrefix(branch.Ref, "#/components/schemas/")
		if !valid || name == "" {
			t.Fatalf("response alternative schema ref=%q", branch.Ref)
		}
		names = append(names, name)
	}
	return names
}
