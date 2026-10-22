package design

import (
	"slices"
	"testing"
)

func TestMemberListSchemasMatchTheirDistinctWireShapes(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")

	aggregate := document.Components.Schemas["WorkspaceProjectMembershipRole"]
	if _, exists := aggregate.Properties["project"]; exists {
		t.Error("workspace project-members aggregate schema exposes an omitted project field")
	}
	if member := aggregate.Properties["member"].Resolved(); !member.Type.Nullable {
		t.Errorf("workspace project-members member = %#v, want nullable", member)
	}
	for _, required := range []string{"id", "role", "member", "original_role", "created_at"} {
		if !slices.Contains(aggregate.Required, required) {
			t.Errorf("workspace project-members schema does not require %q", required)
		}
	}

	projectList := document.Components.Schemas["ProjectMembershipRole"]
	for _, fieldName := range []string{"member", "project"} {
		field := projectList.Properties[fieldName].Resolved()
		if field.Type.Value != "string" || field.Format != "uuid" || field.Type.Nullable {
			t.Errorf("project member-list %s = %#v, want required non-null UUID", fieldName, field)
		}
		if !slices.Contains(projectList.Required, fieldName) {
			t.Errorf("project member-list schema does not require %q", fieldName)
		}
	}
}
