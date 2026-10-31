//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000037_templates.down.sql
var templateStorageDown string

//go:embed schema_migrations/000037_templates.up.sql
var templateStorageUp string

func templateMigrationSnapshot(t *testing.T, f milestoneSchemaFixture) string {
	t.Helper()
	var snapshot string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_build_object(
 'templates',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM resource_templates t),
 'events',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM template_events t),
 'settings',(SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM workspace_template_settings t)
 )::text`).Scan(&snapshot))
	return snapshot
}

func TestTemplateRollbackRefusesRecordsAuditAndGovernance(t *testing.T) {
	for _, state := range []string{"live record", "tombstone", "audit after parent deletion", "workspace governance"} {
		t.Run(state, func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			if state == "workspace governance" {
				f.exec(t, `UPDATE workspace_template_settings SET owned_by_workspace=true WHERE workspace_id=$1`, f.workspace)
			} else {
				id := uuid.NewString()
				f.exec(t, `INSERT INTO resource_templates(id,workspace_id,project_id,kind,name,template_data) VALUES($1,$2,$3,'page','Retained','{"name":"Reusable page"}')`, id, f.workspace, f.project)
				f.exec(t, `INSERT INTO template_events(workspace_id,template_id,action,after_value) SELECT workspace_id,id,'create',to_jsonb(t) FROM resource_templates t WHERE id=$1`, id)
				switch state {
				case "tombstone":
					f.exec(t, `UPDATE resource_templates SET deleted_at=now() WHERE id=$1`, id)
				case "audit after parent deletion":
					f.exec(t, `DELETE FROM projects WHERE id=$1`, f.project)
				}
			}
			before := templateMigrationSnapshot(t, f)
			_, err := f.db.Exec(f.ctx, templateStorageDown)
			require.ErrorContains(t, err, "cannot roll back template storage containing data")
			f.exec(t, `ROLLBACK`)
			require.Equal(t, before, templateMigrationSnapshot(t, f), "rollback refusal must retain records, audit and governance")
		})
	}
}

func TestEmptyTemplateStorageCanRollBackAndUpgrade(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	before := templateMigrationSnapshot(t, f)
	f.exec(t, templateStorageDown)
	var gone bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT to_regclass('public.resource_templates') IS NULL AND to_regclass('public.template_events') IS NULL AND to_regclass('public.workspace_template_settings') IS NULL`).Scan(&gone))
	require.True(t, gone)
	f.exec(t, templateStorageUp)
	require.Equal(t, before, templateMigrationSnapshot(t, f))
}
