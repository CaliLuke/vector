package workitems

import "context"

// Strict validation precedes legacy serializer filtering. Hold its selected
// records through the write so a concurrent deactivation cannot silently remove
// an explicitly reviewed assignee, label, state or estimate.
func (r *Repository) lockItemEditSelections(ctx context.Context, q Request, fields map[string]any) error {
	if value, present := fields["assignee_ids"]; present {
		ids, err := rawIDs(value)
		if err != nil {
			return err
		}
		if _, err = r.db.Exec(ctx, `SELECT m.id FROM project_members m JOIN workspace_members w ON w.workspace_id=m.workspace_id AND w.member_id=m.member_id JOIN users u ON u.id=m.member_id WHERE m.member_id=ANY($1::uuid[]) AND m.project_id=NULLIF($2,'')::uuid ORDER BY m.id,w.id,u.id FOR SHARE OF m,w,u`, ids, q.ProjectID); err != nil {
			return err
		}
	}
	for _, selection := range []struct{ field, table string }{{"label_ids", "labels"}, {"state_id", "states"}, {"estimate_point", "estimate_points"}} {
		value, present := fields[selection.field]
		if !present || value == nil {
			continue
		}
		if selection.field != "label_ids" {
			value = []any{value}
		}
		ids, err := rawIDs(value)
		if err != nil {
			return err
		}
		if _, err = r.db.Exec(ctx, `SELECT id FROM `+selection.table+` WHERE id=ANY($1::uuid[]) AND project_id=NULLIF($2,'')::uuid ORDER BY id FOR SHARE`, ids, q.ProjectID); err != nil {
			return err
		}
	}
	return nil
}

func validateItemEditSelections(ctx context.Context, db Database, q Request, fields map[string]any) error {
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
		if err = db.QueryRow(ctx, relation.query, q.ProjectID, ids).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return invalid(map[string]any{relation.field: []string{"One or more selections are no longer available in this project."}})
		}
	}
	return nil
}
