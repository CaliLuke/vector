package design

import (
	"slices"
	"testing"
)

func TestWorkspaceMemberMeSchemaUsesConcreteWireTypes(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	schema, exists := document.Components.Schemas["WorkspaceMemberMe"]
	if !exists {
		t.Fatal("generated OpenAPI does not define WorkspaceMemberMe")
	}

	tests := []struct {
		field    string
		typeName string
		format   string
		nullable bool
	}{
		{field: "company_role", typeName: "string", nullable: true},
		{field: "created_at", typeName: "string", format: "date-time"},
		{field: "created_by", typeName: "string", format: "uuid", nullable: true},
		{field: "deleted_at", typeName: "string", format: "date-time", nullable: true},
		{field: "id", typeName: "string", format: "uuid"},
		{field: "member", typeName: "string", format: "uuid", nullable: true},
		{field: "role", typeName: "integer", format: "int64", nullable: true},
		{field: "updated_at", typeName: "string", format: "date-time"},
		{field: "updated_by", typeName: "string", format: "uuid", nullable: true},
		{field: "workspace", typeName: "string", format: "uuid", nullable: true},
	}
	for _, test := range tests {
		field, exists := schema.Properties[test.field]
		if !exists {
			t.Errorf("WorkspaceMemberMe does not define %s", test.field)
			continue
		}
		field = field.Resolved()
		if field.Type.Value != test.typeName || field.Format != test.format || field.Type.Nullable != test.nullable {
			t.Errorf("WorkspaceMemberMe.%s = %#v, want type=%q format=%q nullable=%t", test.field, field, test.typeName, test.format, test.nullable)
		}
	}

	role := schema.Properties["role"].Resolved()
	if !slices.Equal(role.Enum, []any{float64(5), float64(15), float64(20)}) {
		t.Errorf("WorkspaceMemberMe.role enum = %#v, want [5 15 20]", role.Enum)
	}

	wantRequired := []string{
		"company_role", "created_by", "default_props", "deleted_at", "explored_features",
		"getting_started_checklist", "is_active", "issue_props", "member", "role", "tips",
		"updated_by", "view_props", "workspace",
	}
	if !slices.Equal(schema.Required, wantRequired) {
		t.Errorf("WorkspaceMemberMe required fields = %v, want %v", schema.Required, wantRequired)
	}
}
