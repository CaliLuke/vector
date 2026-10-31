//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000047_native_template_deletion.up.sql
var templateParentRepairUp string

//go:embed schema_migrations/000047_native_template_deletion.down.sql
var templateParentRepairDown string

func TestTemplateParentRepairPreservesDefinitionsAuditAndPriorTombstones(t *testing.T) {
	for _, parent := range []string{"project", "workspace", "both"} {
		t.Run(parent, func(t *testing.T) {
			f := newMilestoneSchemaFixture(t)
			projectTemplate, workspaceTemplate, priorTemplate := uuid.NewString(), uuid.NewString(), uuid.NewString()
			otherWorkspace, control := uuid.NewString(), uuid.NewString()
			f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) SELECT $1::uuid,name,owner_id,$1::text,timezone,background_color,created_at,updated_at FROM workspaces WHERE id=$2`, otherWorkspace, f.workspace)
			f.exec(t, `INSERT INTO resource_templates(id,workspace_id,kind,name) VALUES($1,$2,'project','Unrelated workspace')`, control, otherWorkspace)
			old := time.Now().UTC().Truncate(time.Second).Add(-72 * time.Hour)
			projectTime, workspaceTime := old.Add(time.Hour), old.Add(2*time.Hour)
			f.exec(t, `INSERT INTO resource_templates(id,workspace_id,project_id,kind,name,template_data,created_at,updated_at) VALUES($1,$2,$3,'page','Project definition',jsonb_build_object('project_id',$3::uuid),$4,$4),($5,$2,NULL,'project','Workspace definition',jsonb_build_object('project_id',$3::uuid),$4,$4),($6,$2,$3,'page','Prior deletion','{}',$4,$4)`, projectTemplate, f.workspace, f.project, old, workspaceTemplate, priorTemplate)
			f.exec(t, `UPDATE resource_templates SET deleted_at=$2 WHERE id=$1`, priorTemplate, old)
			f.exec(t, `INSERT INTO template_events(workspace_id,template_id,action,after_value) SELECT workspace_id,id,'create',to_jsonb(t) FROM resource_templates t`)
			f.exec(t, `UPDATE workspace_template_settings SET owned_by_workspace=true WHERE workspace_id=$1`, f.workspace)
			if parent != "workspace" {
				f.exec(t, `UPDATE projects SET deleted_at=$2 WHERE id=$1`, f.project, projectTime)
			}
			if parent != "project" {
				f.exec(t, `UPDATE workspaces SET deleted_at=$2 WHERE id=$1`, f.workspace, workspaceTime)
			}
			assertTemplateParentRepair(t, f, parent, projectTemplate, workspaceTemplate, priorTemplate, old, projectTime, workspaceTime)
			var controlLive bool
			require.NoError(t, f.db.QueryRow(f.ctx, `SELECT deleted_at IS NULL FROM resource_templates WHERE id=$1`, control).Scan(&controlLive))
			require.True(t, controlLive)
		})
	}
}

func assertTemplateParentRepair(t *testing.T, f milestoneSchemaFixture, parent, projectTemplate, workspaceTemplate, priorTemplate string, old, projectTime, workspaceTime time.Time) {
	t.Helper()
	var beforeDefinitions, beforeAudit string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_agg(to_jsonb(t)-'deleted_at'-'updated_at' ORDER BY id)::text FROM resource_templates t`).Scan(&beforeDefinitions))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_agg(to_jsonb(e) ORDER BY id)::text FROM template_events e`).Scan(&beforeAudit))
	var revision int64
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision FROM workspace_template_settings WHERE workspace_id=$1`, f.workspace).Scan(&revision))
	f.exec(t, templateParentRepairUp)
	var deleted, updated time.Time
	expected := projectTime
	if parent == "workspace" {
		expected = workspaceTime
	}
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT deleted_at,updated_at FROM resource_templates WHERE id=$1`, projectTemplate).Scan(&deleted, &updated))
	require.Equal(t, expected, deleted.UTC())
	require.True(t, updated.After(old))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT deleted_at,updated_at FROM resource_templates WHERE id=$1`, priorTemplate).Scan(&deleted, &updated))
	require.Equal(t, old, deleted.UTC())
	require.Equal(t, old, updated.UTC(), "existing tombstones must not change")
	var workspaceDeleted *time.Time
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT deleted_at FROM resource_templates WHERE id=$1`, workspaceTemplate).Scan(&workspaceDeleted))
	changed := int64(1)
	if parent == "project" {
		require.Nil(t, workspaceDeleted)
	} else {
		require.NotNil(t, workspaceDeleted)
		require.Equal(t, workspaceTime, workspaceDeleted.UTC())
		changed++
	}
	var afterDefinitions, afterAudit string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_agg(to_jsonb(t)-'deleted_at'-'updated_at' ORDER BY id)::text FROM resource_templates t`).Scan(&afterDefinitions))
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT jsonb_agg(to_jsonb(e) ORDER BY id)::text FROM template_events e`).Scan(&afterAudit))
	require.Equal(t, beforeDefinitions, afterDefinitions)
	require.Equal(t, beforeAudit, afterAudit)
	var afterRevision int64
	var owned bool
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT revision,owned_by_workspace FROM workspace_template_settings WHERE workspace_id=$1`, f.workspace).Scan(&afterRevision, &owned))
	require.Equal(t, revision+changed, afterRevision)
	require.True(t, owned)
	repaired := templateMigrationSnapshot(t, f)
	f.exec(t, templateParentRepairUp)
	require.Equal(t, repaired, templateMigrationSnapshot(t, f), "repair must be idempotent")
	f.exec(t, templateParentRepairDown)
	require.Equal(t, repaired, templateMigrationSnapshot(t, f), "rollback must not resurrect children of deleted parents")
}
