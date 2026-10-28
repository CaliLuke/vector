package workitemrelations

import (
	"context"
	"encoding/json"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const definitionProjection = `jsonb_build_object('id',d.id,'name',d.name,'outward',d.outward,'inward',d.inward,'is_default',d.is_default,'is_active',d.is_active,'color',d.color,'sort_order',d.sort_order,'created_at',d.created_at,'updated_at',d.updated_at,'workspace',d.workspace_id)`

func definitionRows(ctx context.Context, tx pgx.Tx, q Query, id uuid.UUID) ([]json.RawMessage, error) {
	rows, err := tx.Query(ctx, `SELECT `+definitionProjection+` FROM work_item_relation_definitions d WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR d.id=$2) AND ($3::bool IS NULL OR d.is_default=$3) AND ($4::bool IS NULL OR d.is_active=$4) ORDER BY d.sort_order,d.id`, q.WorkspaceID, id, q.IsDefault, q.IsActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var raw json.RawMessage
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, rows.Err()
}

// Projection references only owned live metadata. The surrounding query supplies
// item visibility and the current actor as $3; no unscoped related IDs escape.
const targetProjection = `jsonb_build_object('id',i.id,'name',i.name,'sequence_id',i.sequence_id,'project_id',i.project_id,
 'state_id',(SELECT s.id FROM states s WHERE s.id=i.state_id AND s.project_id=i.project_id AND s.workspace_id=i.workspace_id AND s.deleted_at IS NULL),
 'priority',i.priority,'type_id',(SELECT t.id FROM issue_types t WHERE t.id=i.type_id AND t.workspace_id=i.workspace_id AND t.deleted_at IS NULL),
 'is_epic',COALESCE((SELECT t.is_epic FROM issue_types t WHERE t.id=i.type_id AND t.workspace_id=i.workspace_id AND t.deleted_at IS NULL),false),
 'label_ids',COALESCE((SELECT jsonb_agg(l.id ORDER BY l.id) FROM issue_labels il JOIN labels l ON l.id=il.label_id AND l.project_id=i.project_id AND l.workspace_id=i.workspace_id AND l.deleted_at IS NULL WHERE il.issue_id=i.id AND il.project_id=i.project_id AND il.workspace_id=i.workspace_id AND il.deleted_at IS NULL),'[]'::jsonb),
 'assignee_ids',COALESCE((SELECT jsonb_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a JOIN users u ON u.id=a.assignee_id AND u.is_active JOIN project_members pm ON pm.member_id=u.id AND pm.project_id=i.project_id AND pm.workspace_id=i.workspace_id AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) JOIN workspace_members wm ON wm.member_id=u.id AND wm.workspace_id=i.workspace_id AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20) WHERE a.issue_id=i.id AND a.project_id=i.project_id AND a.workspace_id=i.workspace_id AND a.deleted_at IS NULL),'[]'::jsonb),
 'sort_order',i.sort_order,'created_at',r.created_at,'updated_at',r.updated_at,'created_by',r.created_by_id,'updated_by',r.updated_by_id)`

func relationRows(ctx context.Context, tx pgx.Tx, q Query) (map[string]any, error) {
	dependencies := map[string][]json.RawMessage{}
	for _, kind := range DependencyTypes {
		dependencies[kind] = []json.RawMessage{}
	}
	custom := map[string][]json.RawMessage{}
	for _, isCustom := range []bool{false, true} {
		table := "issue_relations r"
		label := `CASE WHEN r.issue_id=$1 THEN r.relation_type ELSE CASE r.relation_type WHEN 'blocked_by' THEN 'blocking' WHEN 'blocking' THEN 'blocked_by' WHEN 'start_before' THEN 'start_after' WHEN 'start_after' THEN 'start_before' WHEN 'finish_before' THEN 'finish_after' WHEN 'finish_after' THEN 'finish_before' END END`
		extra := ` AND r.relation_type=ANY($4::text[])`
		if isCustom {
			table = "work_item_custom_relations r JOIN work_item_relation_definitions d ON d.id=r.definition_id AND d.workspace_id=r.workspace_id"
			label = `CASE WHEN r.issue_id=$1 THEN d.outward ELSE d.inward END`
			extra = ` AND d.deleted_at IS NULL AND d.is_active AND cardinality($4::text[])>0`
		}
		rows, err := tx.Query(ctx, `SELECT `+label+`,`+targetProjection+`||jsonb_build_object('relation_type',`+label+`) FROM `+table+` JOIN issues i ON i.id=CASE WHEN r.issue_id=$1 THEN r.related_issue_id ELSE r.issue_id END JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE (r.issue_id=$1 OR r.related_issue_id=$1) AND r.workspace_id=$2 AND r.deleted_at IS NULL AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$2::uuid")+extra+` ORDER BY r.created_at,r.id`, q.IssueID, q.WorkspaceID, q.ActorID, DependencyTypes)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var label string
			var record json.RawMessage
			if err = rows.Scan(&label, &record); err != nil {
				rows.Close()
				return nil, err
			}
			if isCustom {
				custom[label] = append(custom[label], record)
			} else {
				dependencies[label] = append(dependencies[label], record)
			}
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
	}
	return map[string]any{"dependencies": dependencies, "custom": custom}, nil
}
