package design

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestGeneratedEstimateReadSurfaceHasThreeDistinctOperations(t *testing.T) {
	document := readOpenAPI(t)
	expected := map[string][]string{
		"/api/workspaces/{slug}/estimates/":                                     {"search", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/estimates/":               {"project_id", "search", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/": {"estimate_id", "project_id", "slug"},
	}
	for path, wantParameters := range expected {
		operation, exists := document.Paths[path]["get"]
		if !exists {
			t.Fatalf("generated surface missing GET %s", path)
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Fatalf("GET %s security=%#v, want required session cookie", path, operation.Security)
		}
		if got := parameterNames(document, operation.Parameters); !slices.Equal(got, wantParameters) {
			t.Fatalf("GET %s parameters=%v, want %v", path, got, wantParameters)
		}
		for _, status := range []string{"200", "401", "403", "500"} {
			if _, exists := operation.Responses[status]; !exists {
				t.Errorf("GET %s missing %s response", path, status)
			}
		}
	}
	detail := document.Paths["/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/"]["get"]
	if _, exists := detail.Responses["404"]; !exists {
		t.Fatal("project estimate detail is missing its 404 response")
	}
}

func TestGeneratedEstimateSchemasIncludeRequiredOrderedPointsAndNullability(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document estimateOpenAPIDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	estimate, exists := document.Components.Schemas["EstimateRead"]
	if !exists {
		t.Fatal("EstimateRead schema is missing")
	}
	for _, field := range []string{
		"id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at",
		"workspace", "project", "name", "description", "type", "last_used", "points",
	} {
		if _, exists := estimate.Properties[field]; !exists {
			t.Errorf("EstimateRead schema missing %q", field)
		}
		if !slices.Contains(estimate.Required, field) {
			t.Errorf("EstimateRead schema does not require %q", field)
		}
	}
	points := estimate.Properties["points"]
	if points.Type != "array" || points.Items == nil || points.Items.Ref != "#/components/schemas/EstimatePointRead" {
		t.Fatalf("EstimateRead.points=%#v", points)
	}
	point, exists := document.Components.Schemas["EstimatePointRead"]
	if !exists {
		t.Fatal("EstimatePointRead schema is missing")
	}
	for _, field := range []string{
		"id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at",
		"estimate", "workspace", "project", "key", "value", "description",
	} {
		if _, exists := point.Properties[field]; !exists {
			t.Errorf("EstimatePointRead schema missing %q", field)
		}
		if !slices.Contains(point.Required, field) {
			t.Errorf("EstimatePointRead schema does not require %q", field)
		}
	}
	for _, schema := range []estimateOpenAPISchema{estimate, point} {
		for _, field := range []string{"created_by", "updated_by", "deleted_at"} {
			if !schema.Properties[field].nullable() {
				t.Errorf("%s is not nullable in schema %#v", field, schema)
			}
		}
	}
}

func TestEstimateDetailNotFoundContractUsesExactDjangoBody(t *testing.T) {
	document := readOpenAPI(t)
	notFound, exists := document.Components.Schemas["EstimateReadsNotFound"]
	if !exists {
		t.Fatal("EstimateReadsNotFound schema is missing")
	}
	if _, exists := notFound.Properties["error"]; !exists || !slices.Contains(notFound.Required, "error") {
		t.Fatalf("EstimateReadsNotFound=%#v, want required error field", notFound)
	}
}

type estimateOpenAPIDocument struct {
	Components struct {
		Schemas map[string]estimateOpenAPISchema `json:"schemas"`
	} `json:"components"`
}

type estimateOpenAPISchema struct {
	Type       string                           `json:"type"`
	Ref        string                           `json:"$ref"`
	AnyOf      []estimateOpenAPISchema          `json:"anyOf"`
	Items      *estimateOpenAPISchema           `json:"items"`
	Properties map[string]estimateOpenAPISchema `json:"properties"`
	Required   []string                         `json:"required"`
}

func (schema estimateOpenAPISchema) nullable() bool {
	if schema.Type == "null" {
		return true
	}
	for _, branch := range schema.AnyOf {
		if branch.nullable() {
			return true
		}
	}
	return false
}
