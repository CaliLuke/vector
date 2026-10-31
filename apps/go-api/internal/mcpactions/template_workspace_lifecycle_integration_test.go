//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTemplateWorkspaceDeletionPreservesAuditAndOtherWorkspaces(t *testing.T) {
	f := templateFixture(t)
	configureCatalogueWorker(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	repository := templates.NewRepository(f.pool)
	var ids []string
	for _, scope := range templateScopes() {
		query := templates.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Kind: scope.kind}
		if scope.project != "" {
			query.ProjectID = uuid.MustParse(scope.project)
		}
		name := "Parent lifecycle"
		created, err := repository.Mutate(t.Context(), templates.Command{Query: query, Action: "create", Name: &name, Data: map[string]any{}})
		require.NoError(t, err)
		ids = append(ids, created.ID)
	}
	control := templateRevisionAnchor(t, repository, otherWorkspace, "other")
	runtime := startCatalogueWorker(t, f)
	instance, err := runtime.Deletion.Admit(t.Context(), backgroundtasks.DeletionInput{EventID: uuid.NewString(), App: "db", Model: "workspace", ID: testsupport.LabelWorkspace})
	require.NoError(t, err)
	outcome, err := selvedge.WaitWorkflow[struct{}](t.Context(), runtime.App, instance)
	require.NoError(t, err)
	require.Equal(t, "completed", outcome.Status, outcome.Error)
	var tombstones, events int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM resource_templates WHERE id=ANY($1::uuid[]) AND deleted_at IS NOT NULL),(SELECT count(*) FROM template_events WHERE template_id=ANY($1::uuid[]))`, ids).Scan(&tombstones, &events))
	require.Equal(t, len(ids), tombstones)
	require.Equal(t, len(ids), events, "soft parent deletion retains template audit history")
	var controlLive bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NULL FROM resource_templates WHERE id=$1`, control.ID).Scan(&controlLive))
	require.True(t, controlLive)
}
