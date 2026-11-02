//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPTemplateInvalidWritesLeaveNoReceiptOrAudit(t *testing.T) {
	runCatalogue(t, newMilestoneFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		created := milestoneObject(t, call("template", map[string]any{"action": "create", "kind": "workitem", "name": "Validation target", "template_data": `{}`}))
		for _, invalid := range []struct {
			name   string
			fields map[string]any
			code   string
		}{
			{"unknown kind", map[string]any{"kind": "unknown"}, "invalid_params"},
			{"project kind within project", map[string]any{"kind": "project", "project_id": testsupport.LabelProject}, "invalid_params"},
			{"blank name", map[string]any{"name": "  "}, "invalid_params"},
			{"missing object", map[string]any{"template_data": ""}, "invalid_params"},
			{"array body", map[string]any{"template_data": `[]`}, "invalid_params"},
			{"null body", map[string]any{"template_data": `null`}, "invalid_params"},
			{"scalar body", map[string]any{"template_data": `42`}, "invalid_params"},
			{"malformed body", map[string]any{"template_data": `{"unterminated"`}, "invalid_params"},
			{"empty update object", map[string]any{"action": "update", "template_id": created["id"], "template_data": `{}`}, "invalid_params"},
			{"wrong project selector", map[string]any{"action": "update", "template_id": created["id"], "project_id": testsupport.LabelProject, "template_data": `{"x":true}`}, "not_found"},
			{"wrong kind selector", map[string]any{"action": "update", "template_id": created["id"], "kind": "page", "template_data": `{"x":true}`}, "not_found"},
		} {
			t.Run(invalid.name, func(t *testing.T) {
				input := map[string]any{"action": "create", "kind": "workitem", "name": "Valid name", "template_data": `{}`, "request_id": uuid.NewString()}
				maps.Copy(input, invalid.fields)
				before := templateStorageSnapshot(t, f)
				milestoneFailure(t, call("template", input), invalid.code)
				require.Equal(t, before, templateStorageSnapshot(t, f))
				corrected := map[string]any{"action": "create", "kind": "workitem", "name": "Corrected template", "template_data": `{}`, "request_id": input["request_id"]}
				accepted := milestoneObject(t, call("template", corrected))
				require.Equal(t, false, accepted["replayed"])
			})
		}
	})
}
