package design

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type openAPIDocument struct {
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components struct {
		Schemas    map[string]openAPISchema    `json:"schemas"`
		Parameters map[string]openAPIParameter `json:"parameters"`
	} `json:"components"`
}

type openAPIPathItem map[string]openAPIOperation

func (item *openAPIPathItem) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("decode OpenAPI path item: %w", err)
	}
	*item = make(openAPIPathItem)
	for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
		raw, ok := fields[method]
		if !ok {
			continue
		}
		var operation openAPIOperation
		if err := json.Unmarshal(raw, &operation); err != nil {
			return fmt.Errorf("decode OpenAPI %s operation: %w", method, err)
		}
		(*item)[method] = operation
	}
	return nil
}

type openAPIOperation struct {
	OperationID string                     `json:"operationId"`
	Parameters  []openAPIParameter         `json:"parameters"`
	Responses   map[string]json.RawMessage `json:"responses"`
	Security    []map[string][]string      `json:"security"`
	RequestBody struct {
		Content map[string]json.RawMessage `json:"content"`
	} `json:"requestBody"`
}

type openAPIParameter struct {
	Ref    string        `json:"$ref"`
	Name   string        `json:"name"`
	In     string        `json:"in"`
	Schema openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Type       openAPIType              `json:"type"`
	AnyOf      []openAPISchema          `json:"anyOf"`
	Enum       []any                    `json:"enum"`
	Format     string                   `json:"format"`
	ReadOnly   bool                     `json:"readOnly"`
	Nullable   bool                     `json:"x-nullable"`
	Maximum    *float64                 `json:"maximum"`
	MaxLength  *int                     `json:"maxLength"`
	Properties map[string]openAPISchema `json:"properties"`
	Required   []string                 `json:"required"`
}

type openAPIType struct {
	Value    string
	Nullable bool
}

func (value *openAPIType) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		value.Value = single
		return nil
	}
	var union []string
	if err := json.Unmarshal(data, &union); err != nil {
		return fmt.Errorf("decode OpenAPI schema type: %w", err)
	}
	for _, item := range union {
		if item == "null" {
			value.Nullable = true
			continue
		}
		if value.Value != "" {
			return fmt.Errorf("decode OpenAPI schema type: unsupported union %v", union)
		}
		value.Value = item
	}
	return nil
}

func TestGeneratedStickySurfaceMatchesDesignContract(t *testing.T) {
	generated := readOpenAPI(t)

	expected := map[string]map[string][]string{
		"/api/workspaces/{slug}/stickies/": {
			"get":  {"200", "400", "401", "403"},
			"post": {"201", "400", "401", "403", "413"},
		},
		"/api/workspaces/{slug}/stickies/{pk}/": {
			"delete": {"204", "401", "403", "404"},
			"get":    {"200", "401", "404"},
			"patch":  {"200", "400", "401", "403", "404"},
		},
	}
	for path, methods := range expected {
		if got := sortedKeys(generated.Paths[path]); !slices.Equal(got, sortedKeys(methods)) {
			t.Fatalf("generated methods for %s = %v, want %v", path, got, sortedKeys(methods))
		}
		for method, statuses := range methods {
			generatedOperation, ok := generated.Paths[path][method]
			if !ok {
				t.Fatalf("generated surface missing %s %s", method, path)
			}
			if got := sortedKeys(generatedOperation.Responses); !slices.Equal(got, statuses) {
				t.Fatalf("%s %s response statuses = %v, want %v", method, path, got, statuses)
			}
			if len(generatedOperation.Security) != 1 || len(generatedOperation.Security[0]["plane_session_cookie"]) != 0 {
				t.Fatalf("%s %s security = %#v, want required session cookie", method, path, generatedOperation.Security)
			}
		}
	}
	list := generated.Paths["/api/workspaces/{slug}/stickies/"]["get"]
	if got, want := parameterNames(generated, list.Parameters), []string{"cursor", "per_page", "query", "slug"}; !slices.Equal(got, want) {
		t.Fatalf("generated list parameters = %v, want %v", got, want)
	}
	for path := range generated.Paths {
		if len(path) >= 8 && path[:8] == "/api/v1/" {
			t.Fatalf("generated Go surface includes deferred API-v1 path %q", path)
		}
	}
	assertStickySchemas(t, generated)
}

func TestGeneratorLeavesNoTemporaryDesignPackage(t *testing.T) {
	matches, err := filepath.Glob("loom[0-9]*")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("generator left temporary design packages: %v", matches)
	}
}

func assertStickySchemas(t *testing.T, generated openAPIDocument) {
	t.Helper()
	response := generated.Components.Schemas["StickyResponse"]
	if response.Type.Value != "object" {
		t.Fatalf("StickyResponse type = %q, want object", response.Type.Value)
	}
	wantRequired := []string{"created_at", "description", "description_html", "id", "logo_props", "owner", "sort_order", "updated_at", "workspace"}
	gotRequired := slices.Clone(response.Required)
	slices.Sort(gotRequired)
	if !slices.Equal(gotRequired, wantRequired) {
		t.Fatalf("StickyResponse required fields = %v, want %v", gotRequired, wantRequired)
	}
	wantFields := map[string]struct {
		typeName string
		format   string
		nullable bool
	}{
		"id":                   {typeName: "string", format: "uuid"},
		"workspace":            {typeName: "string", format: "uuid"},
		"owner":                {typeName: "string", format: "uuid"},
		"created_by":           {typeName: "string", format: "uuid", nullable: true},
		"updated_by":           {typeName: "string", format: "uuid", nullable: true},
		"created_at":           {typeName: "string", format: "date-time"},
		"updated_at":           {typeName: "string", format: "date-time"},
		"deleted_at":           {typeName: "string", format: "date-time", nullable: true},
		"name":                 {typeName: "string", nullable: true},
		"color":                {typeName: "string", nullable: true},
		"background_color":     {typeName: "string", nullable: true},
		"description_html":     {typeName: "string"},
		"description_stripped": {typeName: "string", nullable: true},
		"sort_order":           {typeName: "number", format: "double"},
	}
	for field, want := range wantFields {
		got, exists := response.Properties[field]
		if !exists || got.Type.Value != want.typeName || got.Format != want.format || got.Nullable != want.nullable {
			t.Fatalf("StickyResponse.%s = %#v, want type=%q format=%q nullable=%t", field, got, want.typeName, want.format, want.nullable)
		}
	}
	binary := response.Properties["description_binary"]
	binary = binary.Resolved()
	if binary.Type.Value != "string" || !binary.Type.Nullable || binary.Format != "binary" || !binary.ReadOnly {
		t.Fatalf("generated description_binary = %#v, want typed read-only binary", binary)
	}
	generatedOperation := generated.Paths["/api/workspaces/{slug}/stickies/"]["post"]
	if got, want := sortedKeys(generatedOperation.RequestBody.Content), []string{"application/json"}; !slices.Equal(got, want) {
		t.Fatalf("generated request media types = %v, want %v", got, want)
	}
	request := operationRequestSchema(t, generated, generatedOperation, "application/json")
	for _, forbidden := range []string{"id", "workspace", "owner", "created_at", "updated_at", "description_binary"} {
		if _, exists := request.Properties[forbidden]; exists {
			t.Fatalf("generated request exposes read-only field %q", forbidden)
		}
	}
	wantRequestFields := []string{"background_color", "color", "created_by", "deleted_at", "description", "description_html", "description_stripped", "logo_props", "name", "sort_order", "updated_by"}
	if got := sortedKeys(request.Properties); !slices.Equal(got, wantRequestFields) {
		t.Fatalf("generated Sticky request fields = %v, want %v", got, wantRequestFields)
	}
	for _, field := range []string{"background_color", "color", "created_by", "deleted_at", "description_stripped", "name", "updated_by"} {
		if !request.Properties[field].Resolved().Type.Nullable {
			t.Fatalf("generated request %s is not nullable", field)
		}
	}
	for _, field := range []string{"color", "background_color"} {
		if got := request.Properties[field].Resolved().MaxLengthValue(); got != 255 {
			t.Fatalf("generated Sticky request %s maxLength = %d, want 255", field, got)
		}
	}
	page := generated.Components.Schemas["StickyPage"]
	if page.Properties["results"].Type.Value != "array" {
		t.Fatalf("generated list results schema = %#v, want array", page.Properties["results"])
	}
	for _, parameter := range generated.Paths["/api/workspaces/{slug}/stickies/"]["get"].Parameters {
		if parameter.Name == "per_page" {
			schema := parameter.Schema
			if schema.Type.Value != "integer" || schema.Maximum == nil || *schema.Maximum != 1000 {
				t.Fatalf("generated per_page schema = %#v", schema)
			}
		}
	}
}

func operationRequestSchema(t *testing.T, document openAPIDocument, operation openAPIOperation, contentType string) openAPISchema {
	t.Helper()
	media, exists := operation.RequestBody.Content[contentType]
	if !exists {
		t.Fatalf("request body does not define %s", contentType)
	}
	var reference struct {
		Schema struct {
			Ref string `json:"$ref"`
		} `json:"schema"`
	}
	if err := json.Unmarshal(media, &reference); err != nil {
		t.Fatalf("decode request schema reference: %v", err)
	}
	name := strings.TrimPrefix(reference.Schema.Ref, "#/components/schemas/")
	if name == reference.Schema.Ref || name == "" {
		t.Fatalf("request schema reference = %q", reference.Schema.Ref)
	}
	schema, exists := document.Components.Schemas[name]
	if !exists {
		t.Fatalf("request schema %q does not exist", name)
	}
	return schema
}

func (schema openAPISchema) Resolved() openAPISchema {
	if len(schema.AnyOf) == 0 {
		return schema
	}
	resolved := schema
	for _, branch := range schema.AnyOf {
		if branch.Type.Value == "null" {
			resolved.Type.Nullable = true
			continue
		}
		resolved.Type.Value = branch.Type.Value
		resolved.Type.Nullable = resolved.Type.Nullable || branch.Type.Nullable
		resolved.Enum = branch.Enum
		resolved.Format = branch.Format
		resolved.Maximum = branch.Maximum
		resolved.MaxLength = branch.MaxLength
	}
	return resolved
}

func (schema openAPISchema) MaxLengthValue() int {
	if schema.MaxLength == nil {
		return 0
	}
	return *schema.MaxLength
}

func readOpenAPI(t *testing.T) openAPIDocument {
	t.Helper()
	const path = "../gen/http/openapi.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI %s: %v", path, err)
	}
	var document openAPIDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode OpenAPI %s: %v", path, err)
	}
	return document
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func parameterNames(document openAPIDocument, parameters []openAPIParameter) []string {
	names := make([]string, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter.Ref != "" {
			parameter = document.Components.Parameters[filepath.Base(parameter.Ref)]
		}
		names = append(names, parameter.Name)
	}
	slices.Sort(names)
	return names
}
