package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) publishedDraft(ctx context.Context, q Request, workspace any) (Result, bool, error) {
	row, q, found, err := r.publishedDraftSource(ctx, q, workspace)
	if err != nil || !found {
		return Result{}, found, err
	}
	body, projectionErr := r.visibleAppWriteProjection(ctx, q, row, djangoinput.Input{Fields: map[string]any{"assignee_ids": row["assignees"], "label_ids": row["labels"]}})
	return Result{201, body}, true, projectionErr
}

func (r *Repository) publishedDraftSource(ctx context.Context, q Request, workspace any) (map[string]any, Request, bool, error) {
	var project, issue string
	err := r.db.QueryRow(ctx, `SELECT project_id::text,issue_id::text FROM work_item_draft_publications WHERE draft_id=$1 AND workspace_id=$2 AND actor_id=$3`, q.Params["draft_id"], workspace, q.ActorID).Scan(&project, &issue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, q, false, nil
	}
	if err != nil {
		return nil, q, false, err
	}
	q.ProjectID, q.IssueID = project, issue
	if err = r.allowProject(ctx, q, []int{15, 20}, false); err != nil {
		return nil, q, true, err
	}
	row, err := r.row(ctx, `SELECT `+djangoapi.IssueRowSQL("i")+` FROM issues i WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND deleted_at IS NULL`, issue, workspace, project)
	if err != nil {
		return nil, q, true, err
	}
	return row, q, true, nil
}

// Saved values and relationships form the publication input; the request only
// overrides fields the owner edited in the publication form. Read under its lock.
func (r *Repository) publicationInput(ctx context.Context, draft map[string]any, input djangoinput.Input) (djangoinput.Input, error) {
	input, err := normalizePublicationForm(input)
	if err != nil {
		return input, err
	}
	fields := projectFields(draft, []string{"name", "description_html", "description_json", "priority", "start_date", "target_date", "state_id", "parent_id", "type_id", "estimate_point", "external_id", "external_source", "custom_properties"})
	for _, relation := range []struct{ table, column, field string }{
		{"draft_issue_cycles", "cycle_id", "cycle_id"}, {"draft_issue_modules", "module_id", "module_ids"},
		{"draft_issue_assignees", "assignee_id", "assignee_ids"}, {"draft_issue_labels", "label_id", "label_ids"},
	} {
		ids, err := r.filteredIDs(ctx, `SELECT DISTINCT `+relation.column+`::text FROM `+relation.table+` WHERE draft_issue_id=$1 AND deleted_at IS NULL ORDER BY 1`, draft["id"])
		if err != nil {
			return input, err
		}
		list := make([]any, len(ids))
		for i, id := range ids {
			list[i] = id
		}
		fields[relation.field] = list
		if relation.field == "cycle_id" {
			fields[relation.field] = nil
			_, cycleOverridden := input.Fields["cycle_id"]
			if len(ids) > 1 && !cycleOverridden {
				return input, invalid(map[string]any{"cycle_id": []string{"Choose one cycle before publishing."}})
			}
			if len(ids) == 1 {
				fields[relation.field] = ids[0]
			}
		}
	}
	maps.Copy(fields, input.Fields)
	input.Fields, input.Raw = fields, nil
	return input, nil
}

func normalizePublicationForm(input djangoinput.Input) (djangoinput.Input, error) {
	if !input.Form {
		return input, nil
	}
	for _, field := range []string{"assignee_ids", "label_ids", "module_ids"} {
		if value, present := input.ListValue(field, false); present {
			input.Fields[field] = value
		}
	}
	if value, present := input.Fields["description_json"]; present {
		var parsed any
		decoder := json.NewDecoder(strings.NewReader(djangoinput.ScalarString(value)))
		decoder.UseNumber()
		if err := decoder.Decode(&parsed); err != nil {
			return input, invalid(map[string]any{"description_json": []string{"Value must be valid JSON."}})
		}
		input.Fields["description_json"] = parsed
	}
	for _, field := range []string{"start_date", "target_date", "archived_at", "cycle_id"} {
		if input.Fields[field] == "" {
			input.Fields[field] = nil
		}
	}
	input.Form = false
	return input, nil
}

func (r *Repository) validatePublicationRelations(ctx context.Context, q Request, fields map[string]any) error {
	if err := r.allowProject(ctx, q, []int{15, 20}, false); err != nil {
		return err
	}
	return r.validateItemEditSelections(ctx, q, fields)
}

func (r *Repository) validateItemEditSelections(ctx context.Context, q Request, fields map[string]any) error {
	for _, relation := range []struct{ field, query string }{
		{"assignee_ids", `SELECT count(*) FROM project_members m JOIN workspace_members w ON w.workspace_id=m.workspace_id AND w.member_id=m.member_id JOIN users u ON u.id=m.member_id WHERE m.project_id=NULLIF($1,'')::uuid AND m.member_id=ANY($2::uuid[]) AND m.role>=15 AND m.is_active AND m.deleted_at IS NULL AND w.is_active AND w.deleted_at IS NULL AND u.is_active`},
		{"label_ids", `SELECT count(*) FROM labels WHERE project_id=NULLIF($1,'')::uuid AND id=ANY($2::uuid[]) AND deleted_at IS NULL`},
		{"module_ids", `SELECT count(*) FROM modules WHERE project_id=NULLIF($1,'')::uuid AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND archived_at IS NULL`},
		{"cycle_id", `SELECT count(*) FROM cycles WHERE project_id=NULLIF($1,'')::uuid AND id=ANY($2::uuid[]) AND deleted_at IS NULL AND archived_at IS NULL`},
	} {
		value := fields[relation.field]
		if relation.field == "cycle_id" && value != nil {
			value = []any{value}
		}
		if value == nil {
			continue
		}
		ids, err := rawIDs(value)
		if err != nil {
			return err
		}
		var count int
		if err = r.db.QueryRow(ctx, relation.query, q.ProjectID, ids).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return invalid(map[string]any{relation.field: []string{"One or more selections are no longer available in this project."}})
		}
	}
	return nil
}

// Select and authorize the publication target without first mutating the private draft.
func (r *Repository) publicationDestination(ctx context.Context, q Request, draft map[string]any, input djangoinput.Input) (Request, error) {
	value := draft["project_id"]
	if explicit, ok := input.Fields["project_id"]; ok {
		value = explicit
	} else if explicit, ok := input.Fields["project"]; ok {
		value = explicit
	}
	if value == nil || value == "" {
		return q, invalid(map[string]any{"project_id": []string{"Select a project before publishing."}})
	}
	id, err := djangoinput.UUID(value)
	if err != nil {
		return q, invalid(map[string]any{"project_id": []string{"Enter a valid project ID."}})
	}
	if alias, present := input.Fields["project"]; present {
		aliasID, aliasErr := djangoinput.UUID(alias)
		if aliasErr != nil || aliasID != id {
			return q, invalid(map[string]any{"project_id": []string{"Conflicting project IDs."}})
		}
	}
	q.ProjectID = id.String()
	return q, r.allowProject(ctx, q, []int{15, 20}, false)
}
