package design

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

type deployBoardResponseEnvelope struct {
	Content map[string]struct {
		Schema struct {
			Ref   string `json:"$ref"`
			Type  string `json:"type"`
			Items any    `json:"items"`
		} `json:"schema"`
	} `json:"content"`
}

func TestDeployBoardReadRoutesHaveDistinctObjectContracts(t *testing.T) {
	document := readOpenAPI(t)
	collectionPath := "/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/"
	detailPath := "/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/"
	collection, exists := document.Paths[collectionPath]["get"]
	if !exists {
		t.Fatalf("generated surface missing GET %s", collectionPath)
	}
	detail, exists := document.Paths[detailPath]["get"]
	if !exists {
		t.Fatalf("generated surface missing GET %s", detailPath)
	}

	if got, want := parameterNames(document, collection.Parameters), []string{"project_id", "search", "slug"}; !slices.Equal(got, want) {
		t.Fatalf("collection parameters = %v, want %v", got, want)
	}
	if got, want := parameterNames(document, detail.Parameters), []string{"pk", "project_id", "slug"}; !slices.Equal(got, want) {
		t.Fatalf("detail parameters = %v, want %v", got, want)
	}
	for name, operation := range map[string]openAPIOperation{"collection": collection, "detail": detail} {
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("%s security = %#v, want required session cookie", name, operation.Security)
		}
	}
	if got, want := sortedKeys(collection.Responses), []string{"200", "401", "403", "500"}; !slices.Equal(got, want) {
		t.Errorf("collection statuses = %v, want %v", got, want)
	}
	if got, want := sortedKeys(detail.Responses), []string{"200", "401", "403", "404", "500"}; !slices.Equal(got, want) {
		t.Errorf("detail statuses = %v, want %v", got, want)
	}

	assertDeployBoardObjectResponse(t, collection.Responses["200"], "DeployBoardCollectionRead")
	assertDeployBoardObjectResponse(t, detail.Responses["200"], "DeployBoardRead")
}

func TestDeployBoardSchemasPreserveUnsavedCollectionDefaults(t *testing.T) {
	document := readOpenAPI(t)
	collection := document.Components.Schemas["DeployBoardCollectionRead"]
	detail := document.Components.Schemas["DeployBoardRead"]
	if collection.Type.Value != "object" || detail.Type.Value != "object" {
		t.Fatalf("deploy-board schemas = collection %#v detail %#v, want objects", collection.Type, detail.Type)
	}
	collectionDefaults := []string{
		"created_by", "deleted_at", "entity_identifier", "entity_name", "intake",
		"is_activity_enabled", "is_comments_enabled", "is_disabled",
		"is_reactions_enabled", "is_votes_enabled", "updated_by", "view_props",
	}
	if len(collection.Required) != len(collectionDefaults) {
		t.Fatalf("collection required fields = %v, want serializer defaults %v", collection.Required, collectionDefaults)
	}
	for _, field := range collectionDefaults {
		if !slices.Contains(collection.Required, field) {
			t.Errorf("collection schema does not require serializer default %q", field)
		}
	}
	if !collection.Properties["view_props"].Resolved().Type.Nullable {
		t.Error("collection view_props must admit the unsaved serializer's null default")
	}
	wantRequired := []string{
		"anchor", "created_at", "created_by", "deleted_at", "entity_identifier", "entity_name",
		"id", "intake", "is_activity_enabled", "is_comments_enabled", "is_disabled",
		"is_reactions_enabled", "is_votes_enabled", "project", "project_details", "updated_at",
		"updated_by", "view_props", "workspace", "workspace_detail",
	}
	for _, field := range wantRequired {
		if !slices.Contains(detail.Required, field) {
			t.Errorf("detail schema does not require %q", field)
		}
		if _, exists := collection.Properties[field]; !exists {
			t.Errorf("collection schema is missing %q", field)
		}
	}
	for _, field := range []string{"created_by", "deleted_at", "entity_identifier", "entity_name", "intake", "project", "project_details", "updated_by"} {
		if !detail.Properties[field].Resolved().Type.Nullable {
			t.Errorf("detail %s is not nullable", field)
		}
	}
	for _, field := range []string{"cover_image", "cover_image_url"} {
		if !document.Components.Schemas["DeployBoardProjectDetail"].Properties[field].Resolved().Type.Nullable {
			t.Errorf("project detail %s is not nullable", field)
		}
	}
	if !document.Components.Schemas["DeployBoardWorkspaceDetail"].Properties["logo_url"].Resolved().Type.Nullable {
		t.Error("workspace detail logo_url is not nullable")
	}
}

func assertDeployBoardObjectResponse(t *testing.T, raw json.RawMessage, schemaName string) {
	t.Helper()
	var response deployBoardResponseEnvelope
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	schema := response.Content["application/json"].Schema
	if schema.Type == "array" || schema.Items != nil {
		t.Fatalf("%s response is an array: %#v", schemaName, schema)
	}
	if got, want := strings.TrimPrefix(schema.Ref, "#/components/schemas/"), schemaName; got != want {
		t.Fatalf("response schema = %q, want %q", got, want)
	}
}
