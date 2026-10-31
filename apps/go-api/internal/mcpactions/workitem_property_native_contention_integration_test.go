//go:build integration

package mcpactions_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/stretchr/testify/require"
)

func TestPropertyNativeDefinitionEditInvalidatesQueuedMCPMutations(t *testing.T) {
	for _, action := range []string{"create", "update", "delete", "manage_type_properties", "create_option", "update_option", "delete_option"} {
		t.Run(action, func(t *testing.T) {
			h := newPropertyRecoveryCase(t, action)
			repository := workitemtypes.NewRepository(h.pool)
			catalogue, err := repository.PropertyCatalog(t.Context(), testsupport.LabelActor, "workspace-one")
			require.NoError(t, err)
			index := slices.IndexFunc(catalogue.Properties, func(record workitemtypes.PropertyRecord) bool { return record.Definition.ID == h.resourceID.String() })
			require.NotEqual(t, -1, index)
			definition := catalogue.Properties[index].Definition
			definition.Title = "Native property edit"
			native, mcp := contendWithNativeCatalogueWriter(t, h.milestoneFixture, `CREATE FUNCTION pause_property_definition() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
CREATE TRIGGER pause_property_definition BEFORE UPDATE ON work_item_properties FOR EACH ROW WHEN (NEW.definition->>'title'='Native property edit') EXECUTE FUNCTION pause_property_definition()`,
				func(ctx context.Context) (workitemtypes.PropertyCatalog, error) {
					return repository.MutateProperty(ctx, testsupport.LabelActor, "workspace-one", workitemtypes.PropertyInput{Revision: catalogue.Revision, Action: "update", Definition: definition})
				}, func(ctx context.Context) (projectmutations.Result, error) { return h.mutate(ctx, h.key) })
			require.NoError(t, native.err)
			require.ErrorIs(t, mcp.err, projectmutations.ErrRevision)
			require.Equal(t, projectmutations.Result{}, mcp.value)
			var title string
			var receipts int
			require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT definition->>'title',(SELECT count(*) FROM mutation_receipts WHERE tool='workitem_property') FROM work_item_properties WHERE id=$1`, h.resourceID).Scan(&title, &receipts))
			require.Equal(t, "Native property edit", title)
			require.Zero(t, receipts)
		})
	}
}

func TestPropertyNativeItemEditInvalidatesQueuedValueMutations(t *testing.T) {
	for _, action := range []string{"set_value", "delete_value"} {
		t.Run(action, func(t *testing.T) {
			h := newPropertyRecoveryCase(t, action)
			repository, err := workitems.NewRepository(h.pool, rosterPublisher{}, "https://vector.example")
			require.NoError(t, err)
			var before []byte
			require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT custom_properties FROM issues WHERE id=$1`, h.resourceID).Scan(&before))
			native, mcp := contendWithNativeCatalogueWriter(t, h.milestoneFixture, `CREATE FUNCTION pause_property_item() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
CREATE TRIGGER pause_property_item BEFORE UPDATE ON issues FOR EACH ROW WHEN (NEW.name='Native item edit') EXECUTE FUNCTION pause_property_item()`,
				func(ctx context.Context) (workitems.Result, error) {
					return repository.Execute(ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_issues_by_pk", Method: "PATCH", API: true, ActorID: testsupport.LabelActor, Slug: "workspace-one", ProjectID: testsupport.LabelProject, IssueID: h.resourceID.String(), Timezone: "UTC", Params: map[string]string{}, Present: map[string]bool{}}, strings.NewReader(`{"name":"Native item edit"}`), "application/json")
				}, func(ctx context.Context) (projectmutations.Result, error) { return h.mutate(ctx, h.key) })
			require.NoError(t, native.err)
			require.ErrorIs(t, mcp.err, projectmutations.ErrRevision)
			require.Equal(t, projectmutations.Result{}, mcp.value)
			var after []byte
			var name string
			var receipts int
			require.NoError(t, h.pool.QueryRow(t.Context(), `SELECT name,custom_properties,(SELECT count(*) FROM mutation_receipts WHERE tool='workitem_property_value') FROM issues WHERE id=$1`, h.resourceID).Scan(&name, &after, &receipts))
			require.Equal(t, "Native item edit", name)
			require.JSONEq(t, string(before), string(after), "stale MCP mutation cannot overwrite or delete the retained value")
			require.Zero(t, receipts)
		})
	}
}
