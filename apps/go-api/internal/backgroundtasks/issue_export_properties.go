package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type issueExportData struct {
	rows    []map[string]any
	columns []workitemtypes.PropertyExportColumn
	itemIDs json.RawMessage
}

func (store *issueExportStore) exportData(ctx context.Context, input IssueExportInput, projects []uuid.UUID) (issueExportData, error) {
	if !input.IncludeCustomProperties {
		rows, err := readIssueExportRows(ctx, store.pool, exportIdentityQuery(), input, projects)
		return issueExportData{rows: rows}, err
	}
	var actor, slug string
	err := store.pool.QueryRow(ctx, `SELECT e.initiated_by_id::text,w.slug FROM exporters e JOIN workspaces w ON w.id=e.workspace_id
 WHERE e.token=$1 AND e.workspace_id=$2::uuid AND e.type='issue_exports' AND e.deleted_at IS NULL
 AND e.rich_filters->>'include_custom_properties'='true'`, input.TokenID, input.WorkspaceID).Scan(&actor, &slug)
	if err != nil {
		return issueExportData{}, err
	}
	return workitemtypes.WithPropertyDiscoveryRead(ctx, store.pool, actor, slug, "", func(tx pgx.Tx, snapshot *workitemtypes.PropertyDiscoverySnapshot) (issueExportData, error) {
		var allowed bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM exporters e JOIN workspace_members wm ON wm.workspace_id=e.workspace_id AND wm.member_id=e.initiated_by_id
         WHERE e.token=$1 AND e.workspace_id=$2::uuid AND e.type='issue_exports' AND e.deleted_at IS NULL
         AND e.initiated_by_id=$3::uuid AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN (15,20)
         AND e.rich_filters->>'include_custom_properties'='true' AND e.project @> $4::uuid[] AND e.project <@ $4::uuid[])`, input.TokenID, input.WorkspaceID, actor, projects).Scan(&allowed)
		if err != nil {
			return issueExportData{}, err
		}
		if !allowed {
			return issueExportData{}, errors.New("export permissions or saved scope changed")
		}
		rows, err := readIssueExportRows(ctx, tx, exportIdentityQuery(), input, projects)
		if err != nil {
			return issueExportData{}, err
		}
		ids := make([]string, len(projects))
		for i, id := range projects {
			ids[i] = id.String()
		}
		columns, err := snapshot.ExportRows(ctx, rows, ids)
		if err != nil {
			return issueExportData{}, err
		}
		itemIDs, err := exportSourceIDs(ctx, tx, actor, input.WorkspaceID, rows)
		return issueExportData{rows: rows, columns: columns, itemIDs: itemIDs}, err
	})
}

type issueExportReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// Internal identities are retained only until partitioning and authorization
// metadata have been derived. They do not change the standard JSON format.
func exportIdentityQuery() string {
	return strings.Replace(issueExportQuery, "SELECT json_build_object(", "SELECT json_build_object('id',i.id,'project_id',i.project_id,'type_id',i.type_id,", 1)
}

type issueExportGroup struct {
	name string
	data issueExportData
}

func splitIssueExport(input IssueExportInput, projects []uuid.UUID, data issueExportData) []issueExportGroup {
	groups := []issueExportGroup{}
	if input.Multiple {
		for _, project := range projects {
			selected := issueExportData{rows: []map[string]any{}, columns: data.columns}
			for _, row := range data.rows {
				if row["project_id"] == project.String() {
					selected.rows = append(selected.rows, row)
				}
			}
			groups = append(groups, issueExportGroup{input.Slug + "-" + project.String(), selected})
		}
	} else {
		groups = append(groups, issueExportGroup{input.Slug + "-" + input.WorkspaceID, data})
	}
	for _, row := range data.rows {
		delete(row, "id")
		delete(row, "project_id")
		if !input.IncludeCustomProperties {
			delete(row, "type_id")
		}
	}
	return groups
}

// References contribute identifiers and counts even when their projects are not
// selected for export. Record the same visible sources for history authorization.
func exportSourceIDs(ctx context.Context, tx pgx.Tx, actor, workspace string, rows []map[string]any) (json.RawMessage, error) {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row["id"].(string)
	}
	references, err := tx.Query(ctx, `SELECT DISTINCT visible.id::text FROM (
 SELECT i.parent_id id FROM issues i WHERE i.id=ANY($1::uuid[])
 UNION SELECT child.id FROM issues child WHERE child.parent_id=ANY($1::uuid[]) AND child.archived_at IS NULL
 UNION SELECT CASE WHEN rel.issue_id=ANY($1::uuid[]) THEN rel.related_issue_id ELSE rel.issue_id END FROM issue_relations rel
 WHERE (rel.issue_id=ANY($1::uuid[]) OR rel.related_issue_id=ANY($1::uuid[])) AND rel.deleted_at IS NULL
 ) ref JOIN issues visible ON visible.id=ref.id WHERE `+djangoapi.IssueVisibleSQL("visible", "$3::uuid", "$2::uuid"), ids, workspace, actor)
	if err != nil {
		return nil, err
	}
	defer references.Close()
	for references.Next() {
		var id string
		if err = references.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = references.Err(); err != nil {
		return nil, err
	}
	return json.Marshal(ids)
}
