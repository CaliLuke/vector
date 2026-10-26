package workitemtypes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

type hierarchyNode struct {
	ID, Parent, Type, Project, Updated, Name string
	Draft, Visible                           bool
}
type hierarchyPlan struct {
	review      HierarchyReview
	fingerprint string
	invalid     []hierarchyNode
}

func (r *Repository) PreviewHierarchy(ctx context.Context, actor, slug string, input HierarchyInput) (HierarchyReview, error) {
	command, err := normalizeHierarchy(input)
	if err != nil {
		return HierarchyReview{}, err
	}
	return withSchemaWrite(ctx, r, actor, slug, func(tx pgx.Tx, workspace string) (HierarchyReview, error) {
		catalog, err := loadHierarchyCatalog(ctx, tx, workspace, true)
		if err != nil {
			return HierarchyReview{}, err
		}
		if err = validateHierarchyCommand(command, catalog); err != nil {
			return HierarchyReview{}, err
		}
		plan, err := planHierarchy(ctx, tx, workspace, actor, command)
		if err != nil {
			return HierarchyReview{}, err
		}
		plan.review.ID, err = saveSchemaReview(ctx, tx, workspace, actor, command, plan.fingerprint)
		return plan.review, err
	})
}
func (r *Repository) CommitHierarchy(ctx context.Context, actor, slug string, input HierarchyInput) (HierarchyCatalog, error) {
	command, err := normalizeHierarchy(input)
	if err != nil {
		return HierarchyCatalog{}, err
	}
	return withSchemaWrite(ctx, r, actor, slug, func(tx pgx.Tx, workspace string) (HierarchyCatalog, error) {
		receipt, err := loadSchemaReview(ctx, tx, workspace, actor, input.ReviewID, command)
		if err != nil {
			return HierarchyCatalog{}, err
		}
		catalog, err := loadHierarchyCatalog(ctx, tx, workspace, true)
		if err != nil || receipt.Committed {
			return catalog, err
		}
		if receipt.Expired {
			return HierarchyCatalog{}, reject(409, "This review is out of date. Refresh it before confirming.")
		}
		if err = validateHierarchyCommand(command, catalog); err != nil {
			return HierarchyCatalog{}, err
		}
		plan, err := planHierarchy(ctx, tx, workspace, actor, command)
		if err != nil {
			return HierarchyCatalog{}, err
		}
		if plan.fingerprint != receipt.Fingerprint {
			return HierarchyCatalog{}, reject(409, "Parent relationships changed. Refresh the review before confirming.")
		}
		if plan.review.Links > 0 && input.Confirm != "confirm" {
			return HierarchyCatalog{}, reject(422, "Type confirm to unlink the reviewed relationships.")
		}
		if err = applyHierarchy(ctx, tx, workspace, actor, command, plan); err != nil {
			return HierarchyCatalog{}, err
		}
		if err = finishSchemaReview(ctx, tx, workspace, input.ReviewID); err != nil {
			return HierarchyCatalog{}, err
		}
		return loadHierarchyCatalog(ctx, tx, workspace, true)
	})
}

func planHierarchy(ctx context.Context, tx pgx.Tx, workspace, actor string, command hierarchyCommand) (hierarchyPlan, error) {
	plan := hierarchyPlan{review: HierarchyReview{Pairs: []HierarchyPair{}}}
	// Drafts and trashed records participate in integrity and impact, but private
	// draft identities never leave this planner. Visibility is rechecked each time.
	rows, err := tx.Query(ctx, `SELECT i.id::text,COALESCE(i.parent_id::text,''),COALESCE(i.type_id::text,''),i.project_id::text,i.updated_at::text,i.name,false,
 i.deleted_at IS NULL AND p.deleted_at IS NULL AND EXISTS(SELECT 1 FROM project_members m JOIN workspace_members wm ON wm.workspace_id=m.workspace_id AND wm.member_id=m.member_id WHERE m.project_id=i.project_id AND m.workspace_id=i.workspace_id AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL AND wm.is_active AND wm.deleted_at IS NULL AND (m.role>=15 OR m.role=5 AND (p.guest_view_all_features OR i.created_by_id=$2)))
 FROM issues i JOIN projects p ON p.id=i.project_id WHERE i.workspace_id=$1
 UNION ALL SELECT id::text,COALESCE(parent_id::text,''),COALESCE(type_id::text,''),COALESCE(project_id::text,''),updated_at::text,COALESCE(name,''),true,false FROM draft_issues WHERE workspace_id=$1 ORDER BY 7,1`, workspace, actor)
	if err != nil {
		return plan, err
	}
	nodes := []hierarchyNode{}
	parents := map[string]hierarchyNode{}
	for rows.Next() {
		var node hierarchyNode
		if err = rows.Scan(&node.ID, &node.Parent, &node.Type, &node.Project, &node.Updated, &node.Name, &node.Draft, &node.Visible); err != nil {
			rows.Close()
			return plan, err
		}
		nodes = append(nodes, node)
		if !node.Draft {
			parents[node.ID] = node
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return plan, err
	}
	levels := map[string]int{}
	for _, placement := range command.Placements {
		levels[placement.TypeID] = placement.Level
	}
	for _, child := range nodes {
		if child.Parent == "" {
			continue
		}
		parent, exists := parents[child.Parent]
		if exists && parent.ID != child.ID && (child.Draft && child.Type == "" || levels[parent.Type] == levels[child.Type]+1) {
			continue
		}
		plan.invalid = append(plan.invalid, child)
		plan.review.Links++
		if child.Visible && exists && parent.Visible {
			plan.review.Pairs = append(plan.review.Pairs, HierarchyPair{ChildID: child.ID, ChildName: child.Name, ChildProjectID: child.Project, ParentID: parent.ID, ParentName: parent.Name, ParentProjectID: parent.Project})
		} else {
			plan.review.RestrictedLinks++
		}
	}
	// Include every graph node, even currently valid/hidden neighbours, so a new
	// child or changed type cannot reuse an obsolete review.
	raw, err := json.Marshal(struct {
		Command hierarchyCommand
		Nodes   []hierarchyNode
	}{command, nodes})
	if err != nil {
		return plan, err
	}
	sum := sha256.Sum256(raw)
	plan.fingerprint = hex.EncodeToString(sum[:])
	return plan, nil
}
