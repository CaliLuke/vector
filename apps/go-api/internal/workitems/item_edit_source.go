package workitems

import (
	"context"
	"maps"
	"slices"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func (r *Repository) lockItemEditSource(ctx context.Context, q Request, command itemEditCommand, workspace string) (map[string]any, error) {
	var source map[string]any
	var err error
	if command.Mode == "edit" {
		if err = r.allowProject(ctx, q, []int{15, 20}, true); err != nil {
			return nil, err
		}
		source, err = r.row(ctx, `SELECT to_jsonb(i) FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND i.workspace_id=$3 AND `+liveIssue+` FOR UPDATE`, q.IssueID, q.ProjectID, workspace)
	} else {
		source, err = r.row(ctx, `SELECT to_jsonb(d) FROM draft_issues d WHERE d.id=$1 AND d.workspace_id=$2 AND d.created_by_id=$3 AND d.deleted_at IS NULL FOR UPDATE`, q.IssueID, workspace, q.ActorID)
	}
	if err != nil {
		return nil, err
	}
	if command.ProjectID != "" && (command.Mode != "draft" || command.ProjectID != stringValue(source["project_id"])) {
		roles := []int{15, 20}
		if command.Mode == "draft" {
			roles = []int{5, 15, 20}
		}
		if err = r.allowProject(ctx, q, roles, command.Mode == "edit"); err != nil {
			return nil, err
		}
		if _, err = r.row(ctx, `SELECT to_jsonb(p) FROM projects p WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND archived_at IS NULL FOR SHARE`, q.ProjectID, workspace); err != nil {
			return nil, err
		}
	}
	relations, err := r.lockItemEditRelations(ctx, command)
	if err != nil {
		return nil, err
	}
	return map[string]any{"item": source, "relations": relations}, nil
}

// Container metadata writers lock their container before updating links. Follow
// that order, then reload the links so a completed concurrent deletion is visible.
func (r *Repository) lockItemEditRelations(ctx context.Context, command itemEditCommand) (map[string]any, error) {
	relations := map[string]any{}
	for _, kind := range []string{"cycle", "module", "assignee", "label"} {
		table, column := itemEditRelationTable(command.Mode, kind)
		if kind == "cycle" || kind == "module" {
			ids, err := r.filteredIDs(ctx, `SELECT DISTINCT `+kind+`_id::text FROM `+table+` WHERE `+column+`=$1 AND deleted_at IS NULL ORDER BY 1`, command.ItemID)
			if err != nil {
				return nil, err
			}
			proposed := command.Changes[kind+"_id"]
			if kind == "module" {
				proposed = command.Changes["module_ids"]
			}
			if kind == "cycle" && proposed != nil {
				proposed = []any{proposed}
			}
			added, err := rawIDs(proposed)
			if err != nil {
				return nil, err
			}
			ids = append(ids, added...)
			slices.Sort(ids)
			ids = slices.Compact(ids)
			// Retain the before-image in the review fingerprint as names and dates can
			// affect preservation/activity and whether a planning selection is valid.
			containers, err := r.rows(ctx, `SELECT to_jsonb(c) FROM `+kind+`s c WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, ids)
			if err != nil {
				return nil, err
			}
			relations[kind+"_containers"] = containers
		}
		rows, err := r.rows(ctx, `SELECT to_jsonb(link) FROM `+table+` link WHERE `+column+`=$1 AND deleted_at IS NULL ORDER BY id FOR UPDATE`, command.ItemID)
		if err != nil {
			return nil, err
		}
		relations[kind] = rows
	}
	return relations, nil
}

func (r *Repository) applyItemEdit(ctx context.Context, q Request, command itemEditCommand, snapshot map[string]any) error {
	source := snapshot["item"].(map[string]any)
	if command.Restore != nil {
		value, err := r.resolvePropertyRestore(ctx, q, command, source)
		if err != nil {
			return err
		}
		command.Changes = map[string]any{"custom_properties": map[string]any{command.Restore.PropertyID: value}}
	}
	fields := command.Changes
	if command.Mode == "publish" || (command.Mode == "draft" && command.ProjectID != stringValue(source["project_id"])) {
		input, err := r.publicationInput(ctx, source, djangoinput.Input{Fields: maps.Clone(fields)})
		if err != nil {
			return err
		}
		fields = input.Fields
		if command.Mode == "draft" {
			// A destination change validates and rewrites all retained scoped
			// fields, while custom properties remain a presence-aware patch.
			changes := maps.Clone(command.Changes)
			for _, field := range []string{"state_id", "assignee_ids", "label_ids", "module_ids", "cycle_id"} {
				changes[field] = fields[field]
			}
			if _, present := changes["estimate_point"]; !present {
				changes["estimate_point"] = source["estimate_point_id"]
			}
			command.Changes = changes
		}
	}
	if err := r.lockItemEditSelections(ctx, q, fields); err != nil {
		return err
	}
	if err := r.validateItemEditSelections(ctx, q, fields); err != nil {
		return err
	}
	q.Method = "PATCH"
	if command.Mode == "edit" {
		if _, err := r.writeIssueTransaction(ctx, q, itemEditBody(command), "application/json"); err != nil {
			return err
		}
		return r.applyItemEditPlanning(ctx, q, command, snapshot["relations"].(map[string]any))
	}
	q.Operation = "draft_issue"
	if command.Mode == "publish" {
		q.Method = "POST"
		q.Operation = "_draft_to_issue"
	}
	_, err := r.writeDraft(ctx, q, itemEditBody(command), "application/json")
	return err
}

func (r *Repository) itemEditProjection(ctx context.Context, q Request, command itemEditCommand, workspace string) (Result, error) {
	if command.Mode == "publish" {
		row, sourceRequest, found, err := r.publishedDraftSource(ctx, q, workspace)
		if !found && err == nil {
			err = requiredObject()
		}
		if err != nil {
			return Result{}, err
		}
		projection, err := r.detail(ctx, row, sourceRequest, true, true)
		return Result{200, projection}, err
	}
	if command.Mode == "draft" {
		row, err := r.row(ctx, `SELECT to_jsonb(d) FROM draft_issues d WHERE id=$1 AND workspace_id=$2 AND created_by_id=$3 AND deleted_at IS NULL`, q.IssueID, workspace, q.ActorID)
		if err != nil {
			return Result{}, err
		}
		projection, err := r.draftProjection(ctx, row, q.ActorID)
		return Result{200, projection}, err
	}
	if err := r.allowProject(ctx, q, []int{15, 20}, true); err != nil {
		return Result{}, err
	}
	row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND i.workspace_id=$3 AND `+djangoapi.IssueVisibleSQL("i", "$4::uuid", "$3::uuid"), q.IssueID, q.ProjectID, workspace, q.ActorID)
	if err != nil {
		return Result{}, err
	}
	projection, err := r.detail(ctx, row, q, true, true)
	return Result{200, projection}, err
}

func itemEditRelationTable(mode, kind string) (string, string) {
	if mode != "edit" {
		return "draft_issue_" + kind + "s", "draft_issue_id"
	}
	if kind == "assignee" || kind == "label" {
		return "issue_" + kind + "s", "issue_id"
	}
	return kind + "_issues", "issue_id"
}
