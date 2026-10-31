//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/stretchr/testify/require"
)

func enableTypeCatalogue(t *testing.T, f milestoneFixture, slug string) *workitemtypes.Repository {
	t.Helper()
	repository := workitemtypes.NewRepository(f.pool)
	catalogue, err := repository.Catalog(t.Context(), testsupport.LabelActor, slug)
	require.NoError(t, err)
	if !catalogue.Enabled {
		_, err = repository.Mutate(t.Context(), testsupport.LabelActor, slug, "enable", "", workitemtypes.Input{Revision: catalogue.Revision})
		require.NoError(t, err)
	}
	return repository
}

func typeStorageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	var snapshot []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'types',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_types t),
 'links',(SELECT jsonb_agg(to_jsonb(l) ORDER BY id) FROM project_issue_types l),
 'settings',(SELECT jsonb_agg(to_jsonb(s) ORDER BY workspace_id) FROM work_item_type_settings s),
 'revisions',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id) FROM work_item_type_api_revisions r),
 'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY workspace_id,project_id,tool,action,request_id) FROM mutation_receipts r WHERE tool='workitem_type'),
 'effects',(SELECT jsonb_agg(to_jsonb(e) ORDER BY id) FROM work_item_effects e))`).Scan(&snapshot))
	return snapshot
}
