//go:build integration

package bootstrap_test

import (
	"strconv"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

type numericHierarchyFixture struct {
	milestoneSchemaFixture
	childType, parentType, parent, draft string
}

func seedNumericHierarchy(t *testing.T, f milestoneSchemaFixture) numericHierarchyFixture {
	t.Helper()
	h := numericHierarchyFixture{f, uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()}
	for index, id := range []string{h.childType, h.parentType} {
		f.exec(t, insertNumericType, id, f.workspace, strconv.Itoa(index), nil)
		f.exec(t, `INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,is_default,level,is_active,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,true,'2020-01-02','2020-01-03')`, f.workspace, f.project, id, index == 0, index)
	}
	f.exec(t, `INSERT INTO work_item_type_settings(workspace_id,enabled,hierarchy_enabled,default_level,revision) VALUES($1,true,false,0,7)`, f.workspace)
	f.exec(t, `UPDATE projects SET is_issue_type_enabled=true WHERE id=$1`, f.project)
	f.exec(t, `INSERT INTO issues SELECT (jsonb_populate_record(NULL::issues,to_jsonb(i)||jsonb_build_object('id',$2::uuid,'type_id',$3::uuid,'sequence_id',2))).* FROM issues i WHERE id=$1`, f.item, h.parent, h.parentType)
	f.exec(t, `UPDATE issues SET type_id=$2,parent_id=$3 WHERE id=$1`, f.item, h.childType, h.parent)
	f.exec(t, `INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id,type_id,parent_id,description_json,description_html,priority,sort_order,created_at,updated_at)
 SELECT $1,$2,$3,owner_id,$4,$5,'{}','<p>kept</p>','none',0,'2020-01-02','2020-01-03' FROM workspaces WHERE id=$2`, h.draft, f.workspace, f.project, h.childType, h.parent)
	f.exec(t, `UPDATE work_item_type_settings SET hierarchy_enabled=true WHERE workspace_id=$1`, f.workspace)
	return h
}

type numericHierarchyState struct {
	metadata, childLevel, parentLevel string
	nativeRevision, apiRevision       int64
}

func readNumericHierarchy(t *testing.T, h numericHierarchyFixture) numericHierarchyState {
	t.Helper()
	var state numericHierarchyState
	require.NoError(t, h.db.QueryRow(h.ctx, `SELECT jsonb_build_object(
 'types',(SELECT jsonb_agg(to_jsonb(t)-'level' ORDER BY t.id) FROM issue_types t WHERE workspace_id=$1),
 'project_types',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id) FROM project_issue_types p WHERE workspace_id=$1),
 'issues',(SELECT jsonb_agg(to_jsonb(i) ORDER BY i.id) FROM issues i WHERE workspace_id=$1),
 'drafts',(SELECT jsonb_agg(to_jsonb(d) ORDER BY d.id) FROM draft_issues d WHERE workspace_id=$1),
 'settings',(SELECT to_jsonb(s)-'revision' FROM work_item_type_settings s WHERE workspace_id=$1))::text,
 (SELECT level::text FROM issue_types WHERE id=$2),(SELECT level::text FROM issue_types WHERE id=$3),
 (SELECT revision FROM work_item_type_settings WHERE workspace_id=$1),(SELECT revision FROM work_item_type_api_revisions WHERE workspace_id=$1)`, h.workspace, h.childType, h.parentType).Scan(&state.metadata, &state.childLevel, &state.parentLevel, &state.nativeRevision, &state.apiRevision))
	return state
}

func corruptHierarchyLevels(t *testing.T, h numericHierarchyFixture, value string) {
	t.Helper()
	h.exec(t, `BEGIN; SET CONSTRAINTS type_hierarchy_guard DEFERRED`)
	h.exec(t, `UPDATE issue_types SET level=$2::text::double precision WHERE workspace_id=$1`, h.workspace, value)
	h.exec(t, `COMMIT`)
}
