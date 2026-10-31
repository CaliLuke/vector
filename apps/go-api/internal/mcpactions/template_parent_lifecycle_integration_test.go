//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type templateParentCase struct {
	project, pageTemplate, deletedTemplate, workspaceTemplate, event string
	deletion                                                         map[string]any
}

func queueTemplateParentDeletion(t *testing.T, f milestoneFixture, call catalogueCall) templateParentCase {
	t.Helper()
	project := milestoneObject(t, call("project", map[string]any{"action": "create", "name": "Template parent", "identifier": strings.ToUpper(uuid.NewString()[:8])}))["id"].(string)
	item := f.itemInScope(t, testsupport.LabelWorkspace, project)
	data, err := json.Marshal(map[string]any{"workitem_id": item, "project_id": project})
	require.NoError(t, err)
	create := func(kind, scope string) string {
		input := templateScopedInput(templateScope{kind, scope}, "create")
		input["name"], input["template_data"] = "Reusable definition", string(data)
		return milestoneObject(t, call("template", input))["id"].(string)
	}
	value := templateParentCase{project: project, pageTemplate: create("page", project), deletedTemplate: create("workitem", project), workspaceTemplate: create("workitem", "")}
	remove := templateScopedInput(templateScope{"workitem", project}, "delete")
	remove["template_id"] = value.deletedTemplate
	catalogueNull(t, call("template", remove))
	var itemLive bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, item).Scan(&itemLive))
	require.True(t, itemLive, "deleting a definition must not delete the resource named inside its JSON")
	value.deletion = map[string]any{"action": "delete", "project_id": project, "request_id": uuid.NewString()}
	require.Equal(t, false, milestoneObject(t, call("project", value.deletion))["replayed"])
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id FROM work_item_effects WHERE kind='soft_delete' AND payload->>'model'='project' AND payload->>'id'=$1`, value.project).Scan(&value.event))
	for _, kind := range []string{"workitem", "page"} {
		rows := catalogueArray(t, call("template", templateScopedInput(templateScope{kind, ""}, "list")))
		ids := templateIDs(rows)
		require.NotContains(t, ids, value.pageTemplate)
		require.NotContains(t, ids, value.deletedTemplate)
	}
	return value
}

func waitTemplateParentDeletion(t *testing.T, f milestoneFixture, runtime *backgroundtasks.Runtime, value templateParentCase) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	event := value.event
	require.Eventually(t, func() bool {
		var admitted bool
		err := f.pool.QueryRow(ctx, `SELECT admitted_at IS NOT NULL AND last_error='' FROM work_item_effects WHERE id=$1`, event).Scan(&admitted)
		return err == nil && admitted
	}, 30*time.Second, 25*time.Millisecond)
	instance, err := runtime.Deletion.Admit(ctx, backgroundtasks.DeletionInput{EventID: event, App: "db", Model: "project", ID: value.project})
	require.NoError(t, err)
	outcome, err := selvedge.WaitWorkflow[struct{}](ctx, runtime.App, instance)
	require.NoError(t, err)
	require.Equal(t, "completed", outcome.Status, outcome.Error)
	var tombstones int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM resource_templates WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL`, []string{value.pageTemplate, value.deletedTemplate}).Scan(&tombstones))
	require.Equal(t, 2, tombstones, "the native project cascade must retain template tombstones")
	var controlLive bool
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM resource_templates WHERE id=$1`, value.workspaceTemplate).Scan(&controlLive))
	require.True(t, controlLive, "workspace definitions are independent of the project IDs stored inside template JSON")
}

func TestMCPTemplateParentDeletionUsesDurableCascadeAndPreservesDefinitions(t *testing.T) {
	f := templateFixture(t)
	configureCatalogueWorker(t)
	controlItem := f.item(t)
	var cases []templateParentCase
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, call catalogueCall) {
		cases = append(cases, queueTemplateParentDeletion(t, f, call))
	})
	require.Len(t, cases, 4)
	runtime := startCatalogueWorker(t, f)
	for _, value := range cases {
		waitTemplateParentDeletion(t, f, runtime, value)
	}
	before := templateStorageSnapshot(t, f)
	index := 0
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		require.Equal(t, true, milestoneObject(t, call("project", cases[index].deletion))["replayed"])
		index++
	})
	require.Equal(t, before, templateStorageSnapshot(t, f))
	var deletionEffects int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='soft_delete'`).Scan(&deletionEffects))
	require.Equal(t, len(cases), deletionEffects, "project receipt replay must not enqueue deletion again")
	for _, value := range cases {
		var effects int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM work_item_effects WHERE kind='soft_delete' AND id=$1`, value.event).Scan(&effects))
		require.Equal(t, 1, effects)
		_, err := f.pool.Exec(t.Context(), `DELETE FROM projects WHERE id=$1`, value.project)
		require.NoError(t, err)
		var templates, audits int
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM resource_templates WHERE project_id=$1),(SELECT count(*) FROM template_events WHERE template_id=ANY($2::uuid[]))`, value.project, []string{value.pageTemplate, value.deletedTemplate}).Scan(&templates, &audits))
		require.Zero(t, templates, "physical parent deletion cascades its stored definitions")
		require.Equal(t, 3, audits, "project removal retains immutable template audit history")
	}
	var controlLive bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM issues WHERE id=$1`, controlItem).Scan(&controlLive))
	require.True(t, controlLive)
}
