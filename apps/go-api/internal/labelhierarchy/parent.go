// Package labelhierarchy validates and removes project-scoped label trees.
package labelhierarchy

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ErrOutOfScopeChild prevents a recursive worker from crossing project scope.
var ErrOutOfScopeChild = errors.New("label tree has an out-of-scope child")

// LockProject serializes label-tree mutations in one project.
func LockProject(ctx context.Context, tx pgx.Tx, projectID, workspaceSlug string) (string, error) {
	var workspaceID string
	err := tx.QueryRow(ctx, `SELECT project.workspace_id::text FROM projects project
 JOIN workspaces workspace ON workspace.id=project.workspace_id
 WHERE project.id=$1 AND workspace.slug=$2 FOR UPDATE OF project`, projectID, workspaceSlug).Scan(&workspaceID)
	return workspaceID, err
}

// ValidateParent must run in the same transaction as LockProject and the write.
func ValidateParent(ctx context.Context, tx pgx.Tx, labelID, parentID, projectID, workspaceID string) (bool, error) {
	if labelID == parentID {
		return false, nil
	}
	var found string
	err := tx.QueryRow(ctx, `SELECT id::text FROM labels
 WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL
 FOR KEY SHARE`, parentID, projectID, workspaceID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var descendant, cycle, invalidScope, missingAncestor bool
	err = tx.QueryRow(ctx, `WITH RECURSIVE ancestry(id,parent_id,path,cycle) AS (
 SELECT id,parent_id,ARRAY[id],false FROM labels
 WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL
 UNION ALL
 SELECT parent.id,parent.parent_id,ancestry.path || parent.id,parent.id=ANY(ancestry.path)
 FROM ancestry JOIN labels parent ON parent.id=ancestry.parent_id
 WHERE NOT ancestry.cycle
)

SELECT COALESCE(bool_or(id=$4::uuid),false),
 COALESCE(bool_or(cycle),false),
 COALESCE(bool_or(EXISTS (
  SELECT 1 FROM labels parent WHERE parent.id=ancestry.id
   AND (parent.deleted_at IS NOT NULL OR parent.project_id IS DISTINCT FROM $2::uuid OR parent.workspace_id IS DISTINCT FROM $3::uuid)
 )),false),
 COALESCE(bool_or(parent_id IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM labels parent WHERE parent.id=ancestry.parent_id
 )),false)
FROM ancestry`, parentID, projectID, workspaceID, labelID).Scan(&descendant, &cycle, &invalidScope, &missingAncestor)
	if err != nil {
		return false, err
	}
	return !descendant && !cycle && !invalidScope && !missingAncestor, nil
}

// SoftDeleteTree marks a label and all live descendants in scope, then records
// one durable recursive-cleanup effect for each newly deleted row.
// SoftDeleteTree must run in the same transaction as LockProject.
func SoftDeleteTree(ctx context.Context, tx pgx.Tx, rootID, actorID, projectID, workspaceID string) (bool, error) {
	var hasOutOfScopeChild bool
	err := tx.QueryRow(ctx, `WITH RECURSIVE tree(id) AS (
 SELECT id FROM labels WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members WHERE project_id=labels.project_id AND member_id=$4)
 UNION
 SELECT child.id FROM labels child JOIN tree parent ON child.parent_id=parent.id
 WHERE child.project_id=$2 AND child.workspace_id=$3
)
SELECT EXISTS(SELECT 1 FROM labels child JOIN tree parent ON child.parent_id=parent.id
 WHERE child.project_id IS DISTINCT FROM $2::uuid OR child.workspace_id IS DISTINCT FROM $3::uuid)`, rootID, projectID, workspaceID, actorID).Scan(&hasOutOfScopeChild)
	if err != nil {
		return false, err
	}
	if hasOutOfScopeChild {
		return false, ErrOutOfScopeChild
	}
	var ids []string
	err = tx.QueryRow(ctx, `WITH RECURSIVE tree(id) AS (
 SELECT id FROM labels WHERE id=$1 AND project_id=$2 AND workspace_id=$3 AND deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members WHERE project_id=labels.project_id AND member_id=$4)
 UNION
 SELECT child.id FROM labels child JOIN tree parent ON child.parent_id=parent.id
 WHERE child.project_id=$2 AND child.workspace_id=$3
)
SELECT COALESCE(array_agg(label.id::text),ARRAY[]::text[])
FROM tree JOIN labels label ON label.id=tree.id WHERE label.deleted_at IS NULL`, rootID, projectID, workspaceID, actorID).Scan(&ids)
	if err != nil {
		return false, err
	}
	if len(ids) == 0 {
		return false, nil
	}
	effectIDs := make([]string, len(ids))
	for index := range effectIDs {
		effectIDs[index] = uuid.NewString()
	}
	var queued int
	err = tx.QueryRow(ctx, `WITH requested AS (
	 SELECT input.id,input.effect_id FROM unnest($1::uuid[],$2::uuid[]) AS input(id,effect_id)
), deleted AS (
 UPDATE labels label SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$3
 WHERE label.id=ANY($1::uuid[]) AND label.project_id=$4 AND label.workspace_id=$5 AND label.deleted_at IS NULL
 RETURNING label.id
), queued AS (
 INSERT INTO work_item_effects(id,kind,payload)
	 SELECT requested.effect_id,'soft_delete',jsonb_build_object('app','db','model','label','id',deleted.id::text)
	 FROM deleted JOIN requested ON requested.id=deleted.id
 RETURNING id
)
SELECT count(*)::int FROM queued`, ids, effectIDs, actorID, projectID, workspaceID).Scan(&queued)
	return queued == len(ids), err
}
