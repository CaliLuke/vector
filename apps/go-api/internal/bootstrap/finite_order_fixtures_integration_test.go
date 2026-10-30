//go:build integration

package bootstrap_test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type finiteOrderTable struct {
	table, column, columns, values string
}

// Explicit fixture fields keep the test data independent of database defaults.
var finiteOrderTables = []finiteOrderTable{
	{"cycles", "sort_order", "name,description,owned_by_id,project_id,view_props,progress_snapshot,logo_props,timezone,version", "@id::uuid::text,'keep',@owner,@project,'{}','{}','{}','UTC',1"},
	{"draft_issues", "sort_order", "description_json,description_html,priority,project_id", "'{}','<p>keep</p>','none',@project"},
	{"issue_versions", "sort_order", "name,priority,sequence_id,is_draft,last_saved_at,owned_by_id,assignees,labels,modules,properties,meta,issue_id,project_id", "@id::uuid::text,'none',1,false,'2020-02-03',@owner,'{}','{}','{}','{}','{}',@item,@project"},
	{"issue_views", "sort_order", "name,description,query,access,filters,display_filters,display_properties,logo_props,is_locked,owned_by_id,rich_filters,project_id", "@id::uuid::text,'keep','{}',0,'{}','{}','{}','{}',false,@owner,'{}',@project"},
	{"issues", "sort_order", "name,description_json,priority,sequence_id,project_id,description_html,is_draft", "@id::uuid::text,'{}','none',2,@project,'<p>keep</p>',false"},
	{"modules", "sort_order", "name,description,status,project_id,view_props,logo_props", "@id::uuid::text,'keep','backlog',@project,'{}','{}'"},
	{"pages", "sort_order", "name,description_json,description_html,access,owned_by_id,color,is_locked,view_props,logo_props,is_global", "@id::uuid::text,'{}','<p>keep</p>',0,@owner,'#abc',false,'{}','{}',true"},
	{"project_members", "sort_order", "role,project_id,member_id,view_props,default_props,preferences,is_active", "20,@project,@owner,'{}','{}','{}',true"},
	{"project_user_properties", "sort_order", "display_properties,project_id,user_id,display_filters,filters,rich_filters,preferences", "'{}',@project,@owner,'{}','{}','{}','{}'"},
	{"states", "sequence", "name,description,color,slug,project_id,\"group\",\"default\",is_triage", "@id::uuid::text,'keep','#abc',@id::uuid::text,@project,'backlog',false,false"},
	{"stickies", "sort_order", "description,description_html,logo_props,owner_id", "'{}','<p>keep</p>','{}',@owner"},
	{"user_favorites", "sequence", "entity_type,is_folder,user_id,name", "'folder',true,@owner,'keep'"},
	{"workspace_home_preferences", "sort_order", "key,is_enabled,config,user_id", "'keep',true,'{}',@owner"},
	{"workspace_user_preferences", "sort_order", "key,is_pinned,user_id", "'keep',true,@owner"},
	{"work_item_relation_definitions", "sort_order", "name,outward,inward", "@id::uuid::text,'depends on','blocks'"},
	{"release_labels", "sort_order", "name,color", "@id::uuid::text,'#abc'"},
	{"page_collections", "sort_order", "name,owned_by_id", "@id::uuid::text,@owner"},
	{"page_collection_pages", "sort_order", "collection_id,page_id", "@page_collections,@pages"},
	{"customer_properties", "sort_order", "name,display_name,property_type", "@id::uuid::text,@id::uuid::text,'OPTION'"},
	{"customer_property_options", "sort_order", "property_id,name", "@customer_properties,@id::uuid::text"},
}

type finiteOrderRow struct {
	finiteOrderTable
	id string
}

func seedFiniteOrders(t *testing.T, f milestoneSchemaFixture) []finiteOrderRow {
	t.Helper()
	var owner string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT owner_id FROM workspaces WHERE id=$1`, f.workspace).Scan(&owner))
	args := pgx.NamedArgs{"workspace": f.workspace, "project": f.project, "item": f.item, "owner": owner, "order": "12.5"}
	rows := make([]finiteOrderRow, 0, len(finiteOrderTables))
	for _, spec := range finiteOrderTables {
		id := uuid.New().String()
		args["id"] = id
		query := fmt.Sprintf(`INSERT INTO %s (id,workspace_id,created_at,updated_at,%s,%s) VALUES(@id::uuid,@workspace,'2020-01-01','2020-01-02',@order::text::double precision,%s)`, pgx.Identifier{"public", spec.table}.Sanitize(), pgx.Identifier{spec.column}.Sanitize(), spec.columns, spec.values)
		_, err := f.db.Exec(f.ctx, query, args)
		require.NoError(t, err, spec.table)
		args[spec.table] = id
		rows = append(rows, finiteOrderRow{spec, id})
	}
	return rows
}
