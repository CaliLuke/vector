package workitems

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

// All eligibility predicates run before LIMIT, including the complete graph
// traversal. Hidden nodes may prohibit a cycle but never become search results.
func (r *Repository) searchRelationships(ctx context.Context, q Request, add func(any, string), where *[]string) error {
	parent, child := q.Params["parent"] == "true", q.Params["sub_issue"] == "true"
	relation := q.Params["issue_relation"] == "true"
	if !parent && !child && !relation {
		return nil
	}
	var anchor map[string]any
	if raw := q.Params["issue_id"]; raw != "" {
		id, err := djangoinput.UUID(raw)
		if err != nil {
			return failure(400, "error", "Please provide valid detail")
		}
		rows, err := r.rows(ctx, `SELECT to_jsonb(i) FROM issues i JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1 AND w.slug=$2 AND `+liveIssue+` AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "w.id"), id.String(), q.Slug, q.ActorID)
		if err != nil {
			return err
		}
		if len(rows) != 1 {
			return requiredObject()
		}
		anchor = rows[0]
		add(id.String(), `i.id<>$%d`)
		if parent {
			if anchor["parent_id"] != nil {
				add(anchor["parent_id"], `i.id<>$%d`)
			}
			add(id.String(), `NOT EXISTS(WITH RECURSIVE descendants(id) AS (SELECT $%[1]d::uuid UNION SELECT node.id FROM issues node JOIN descendants ON node.parent_id=descendants.id WHERE node.workspace_id=i.workspace_id) SELECT 1 FROM descendants WHERE id=i.id)`)
		}
		if child {
			*where = append(*where, `i.parent_id IS NULL`, `EXISTS(SELECT 1 FROM project_members child_member JOIN workspace_members child_workspace_member ON child_workspace_member.workspace_id=child_member.workspace_id AND child_workspace_member.member_id=child_member.member_id WHERE child_member.project_id=i.project_id AND child_member.workspace_id=i.workspace_id AND child_member.member_id=$2 AND child_member.is_active AND child_member.deleted_at IS NULL AND child_workspace_member.is_active AND child_workspace_member.deleted_at IS NULL AND (child_member.role IN (15,20) OR child_workspace_member.role=20))`)
			add(id.String(), `NOT EXISTS(WITH RECURSIVE ancestors(id,parent_id) AS (SELECT node.id,node.parent_id FROM issues node WHERE node.id=$%[1]d UNION SELECT node.id,node.parent_id FROM issues node JOIN ancestors ON node.id=ancestors.parent_id WHERE node.workspace_id=i.workspace_id) SELECT 1 FROM ancestors WHERE id=i.id)`)
		}
		if relation {
			add(id.String(), `NOT EXISTS(SELECT 1 FROM issue_relations rel WHERE rel.deleted_at IS NULL AND ((rel.issue_id=$%[1]d AND rel.related_issue_id=i.id) OR (rel.related_issue_id=$%[1]d AND rel.issue_id=i.id)))`)
		}
	} else if child {
		return requiredObject()
	}
	if parent || child {
		return r.searchHierarchyLevel(ctx, q, anchor, parent, add, where)
	}
	return nil
}

func (r *Repository) searchHierarchyLevel(ctx context.Context, q Request, anchor map[string]any, parent bool, add func(any, string), where *[]string) error {
	typeID := stringValue(anchor["type_id"])
	if parent && q.Params["child_type_id"] != "" {
		typeID = q.Params["child_type_id"]
	}
	if typeID == "" {
		// Unsaved items without a selected type cannot choose a configured
		// hierarchy parent yet. Unconfigured workspaces keep their existing flow.
		*where = append(*where, `NOT EXISTS(SELECT 1 FROM work_item_type_settings h WHERE h.workspace_id=i.workspace_id AND h.hierarchy_enabled)`)
		return nil
	}
	id, err := djangoinput.UUID(typeID)
	if err != nil {
		return failure(400, "error", "Please provide a valid work item type")
	}
	var exists bool
	if err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_types t JOIN workspaces w ON w.id=t.workspace_id WHERE t.id=$1 AND w.slug=$2 AND t.deleted_at IS NULL)`, id.String(), q.Slug).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return failure(400, "error", "Please provide a valid work item type")
	}
	delta := "-1"
	if parent {
		delta = "+1"
	}
	add(id.String(), `(NOT EXISTS(SELECT 1 FROM work_item_type_settings h WHERE h.workspace_id=i.workspace_id AND h.hierarchy_enabled) OR COALESCE((SELECT level FROM issue_types WHERE id=i.type_id),0)=(SELECT COALESCE(level,0)`+delta+` FROM issue_types WHERE id=$%d))`)
	return nil
}
