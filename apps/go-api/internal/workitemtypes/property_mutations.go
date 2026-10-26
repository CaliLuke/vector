package workitemtypes

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) MutateProperty(ctx context.Context, actor, slug string, input PropertyInput) (PropertyCatalog, error) {
	if input.ReviewID != "" {
		return r.commitPropertyRemoval(ctx, actor, slug, input)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PropertyCatalog{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	workspace, admin, err := lockWorkspaceSchema(ctx, tx, actor, slug)
	if err != nil {
		return PropertyCatalog{}, err
	}
	types, err := loadCatalog(ctx, tx, workspace, admin)
	if err != nil {
		return PropertyCatalog{}, err
	}
	if !types.Enabled {
		return PropertyCatalog{}, reject(409, "Enable work item types first.")
	}
	if input.Revision != types.Revision {
		return PropertyCatalog{}, reject(409, "Work item configuration changed. Reload and try again.")
	}
	before, err := loadProperties(ctx, tx, workspace, actor, admin)
	if err != nil {
		return before, err
	}
	switch input.Action {
	case "create", "update":
		err = saveProperty(ctx, tx, workspace, actor, input, before, false)
	case "attach":
		err = attachProperties(ctx, tx, input, before, types)
	case "delete", "detach":
		err = removeProperty(ctx, tx, workspace, input, before, false)
	default:
		err = reject(422, "Unknown property action.")
	}
	if err != nil {
		return PropertyCatalog{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE work_item_type_settings SET revision=revision+1 WHERE workspace_id=$1`, workspace); err != nil {
		return PropertyCatalog{}, err
	}
	out, err := loadProperties(ctx, tx, workspace, actor, admin)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func saveProperty(ctx context.Context, tx pgx.Tx, workspace, actor string, input PropertyInput, before PropertyCatalog, reviewed bool) error {
	def := canonicalProperty(input.Definition)
	var previous *property.Definition
	used := false
	for _, record := range before.Properties {
		if record.Definition.ID == def.ID {
			old := record.Definition
			previous = &old
			used = record.Used
		}
	}
	if input.Action == "create" && previous != nil {
		return reject(409, "This property already exists.")
	}
	if input.Action == "update" && previous == nil {
		return reject(404, "Property not found.")
	}
	if previous == nil && def.Name == "" {
		def.Name = formula.ReferenceKey(def.Title)
	}
	if err := property.ValidateDefinition(def, previous, used); err != nil {
		return reject(422, err.Error())
	}
	if err := validatePropertyScopes(ctx, tx, workspace, def); err != nil {
		return err
	}
	after := before
	after.Properties = slices.Clone(before.Properties)
	if previous == nil {
		after.Properties = append(after.Properties, PropertyRecord{Definition: def})
	} else {
		for i := range after.Properties {
			if after.Properties[i].Definition.ID == def.ID {
				after.Properties[i].Definition = def
			}
		}
	}
	if err := validatePropertyChange(def, after); err != nil {
		return err
	}
	// Value-removing updates require the impact/preservation workflow. Ordinary
	// required/active changes are safe: stored values remain untouched.
	if previous != nil && !reviewed {
		if err := rejectDestructivePropertyEdit(ctx, tx, workspace, *previous, def); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(def)
	if err != nil {
		return err
	}
	if previous == nil {
		_, err = tx.Exec(ctx, `INSERT INTO work_item_properties(id,workspace_id,definition,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$4)`, def.ID, workspace, raw, actor)
	} else {
		_, err = tx.Exec(ctx, `UPDATE work_item_properties SET definition=$3,updated_at=clock_timestamp(),updated_by_id=$4 WHERE id=$1 AND workspace_id=$2`, def.ID, workspace, raw, actor)
	}
	return err
}
func attachProperties(ctx context.Context, tx pgx.Tx, input PropertyInput, before PropertyCatalog, types Catalog) error {
	if !slices.ContainsFunc(types.Types, func(t Type) bool { return t.ID == input.TypeID }) {
		return reject(404, "Work item type not found.")
	}
	if len(input.PropertyIDs) > 100 {
		return reject(422, "A type supports at most 100 custom properties.")
	}
	seen := map[string]bool{}
	for _, id := range input.PropertyIDs {
		if seen[id] || !slices.ContainsFunc(before.Properties, func(p PropertyRecord) bool { return p.Definition.ID == id }) {
			return reject(422, "Choose distinct properties from this workspace.")
		}
		seen[id] = true
	}
	for i, link := range before.Attachments {
		if link.TypeID == input.TypeID {
			for _, id := range link.PropertyIDs {
				if !seen[id] {
					return reject(409, "Detach properties through the preservation review.")
				}
			}
			before.Attachments[i].PropertyIDs = input.PropertyIDs
		}
	}
	if err := validatePropertyAttachments(before); err != nil {
		return err
	}
	for index, id := range input.PropertyIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO work_item_type_properties(type_id,property_id,sort_order) VALUES($1,$2,$3) ON CONFLICT(type_id,property_id) DO UPDATE SET sort_order=excluded.sort_order`, input.TypeID, id, index); err != nil {
			return err
		}
	}
	return nil
}
func validatePropertyScopes(ctx context.Context, tx pgx.Tx, workspace string, def property.Definition) error {
	for _, c := range def.Contexts {
		for _, scope := range []struct {
			table string
			ids   []string
		}{{"projects", c.Projects}, {"issue_types", c.Types}} {
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+scope.table+` WHERE workspace_id=$1 AND deleted_at IS NULL AND id=ANY($2::uuid[])`, workspace, scope.ids).Scan(&count); err != nil {
				return err
			}
			if count != len(scope.ids) {
				return reject(422, "Context scopes must belong to this workspace.")
			}
		}
	}
	configs := []property.Configuration{def.Configuration}
	for _, c := range def.Contexts {
		if slices.Contains(c.Overrides, "default_value") {
			configs = append(configs, c.Configuration)
		}
	}
	if def.Kind != property.Member {
		return nil
	}
	for _, c := range configs {
		ids := []string{}
		switch value := c.Default.(type) {
		case string:
			ids = append(ids, value)
		case []any:
			for _, id := range value {
				if s, ok := id.(string); ok {
					ids = append(ids, s)
				}
			}
		case []string:
			ids = value
		}
		if len(ids) == 0 {
			continue
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(DISTINCT m.member_id) FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.workspace_id=$1 AND m.is_active AND m.deleted_at IS NULL AND u.is_active AND m.member_id=ANY($2::uuid[])`, workspace, ids).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return reject(422, "Default members must be active members of this workspace.")
		}
	}
	return nil
}
func validatePropertyAttachments(catalog PropertyCatalog) error {
	records := map[string]property.Definition{}
	for _, record := range catalog.Properties {
		records[record.Definition.ID] = record.Definition
	}
	for _, link := range catalog.Attachments {
		names := map[string]bool{}
		attached := []PropertyRecord{}
		for _, id := range link.PropertyIDs {
			def, exists := records[id]
			if !exists {
				return reject(422, "An attached property no longer exists.")
			}
			title := strings.ToLower(strings.TrimSpace(def.Title))
			if names[title] {
				return reject(409, "A type cannot have two properties with the same title.")
			}
			names[title] = true
			attached = append(attached, PropertyRecord{Definition: def})
		}
		for _, record := range attached {
			if err := validateStandaloneFormula(record.Definition, attached); err != nil {
				return err
			}
		}
	}
	return nil
}
func formulaFields(records []PropertyRecord) map[string]formula.Field {
	fields := map[string]formula.Field{}
	for _, name := range []string{"name", "priority", "estimate_point", "start_date", "due_date"} {
		kind := formula.Text
		switch name {
		case "estimate_point":
			kind = formula.Number
		case "start_date", "due_date":
			kind = formula.Date
		}
		fields[name] = formula.Field{ID: name, Kind: kind}
	}
	for _, record := range records {
		def := record.Definition
		if slices.Contains([]property.Kind{property.Text, property.Number, property.Boolean, property.Date}, def.Kind) {
			fields[def.Name] = formula.Field{ID: def.ID, Kind: formula.Kind(def.Kind)}
		}
	}
	return fields
}
func validateStandaloneFormula(def property.Definition, records []PropertyRecord) error {
	if def.Kind != property.Formula {
		return nil
	}
	if _, err := formula.Compile(def.Configuration.Expression, def.Configuration.Result, formulaFields(records)); err != nil {
		return reject(422, def.Title+": "+err.Error())
	}
	return nil
}

func removeProperty(ctx context.Context, tx pgx.Tx, workspace string, input PropertyInput, before PropertyCatalog, reviewed bool) error {
	// The preview reuses its original catalog after the SQL savepoint rolls back.
	// Copy the slice owners as well: rollback cannot undo in-memory mutations.
	before.Properties = slices.Clone(before.Properties)
	before.Attachments = slices.Clone(before.Attachments)
	index := slices.IndexFunc(before.Properties, func(record PropertyRecord) bool { return record.Definition.ID == input.ID })
	if index < 0 {
		return reject(404, "Property not found.")
	}
	if !reviewed && input.Action == "delete" && before.Properties[index].Used {
		return reject(409, "This property has stored values or history. Review preservation before deleting it.")
	}
	var used bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE workspace_id=$1 AND type_id=NULLIF($2,'')::uuid AND custom_properties ? $3 UNION ALL SELECT 1 FROM draft_issues WHERE workspace_id=$1 AND type_id=NULLIF($2,'')::uuid AND custom_properties ? $3)`, workspace, input.TypeID, input.ID).Scan(&used); err != nil {
		return err
	}
	if used && !reviewed {
		return reject(409, "This attachment has stored values. Review preservation before detaching it.")
	}
	if input.Action == "delete" {
		before.Properties = slices.Delete(before.Properties, index, index+1)
	}
	found := false
	for i, link := range before.Attachments {
		if input.Action == "delete" || link.TypeID == input.TypeID {
			found = found || slices.Contains(link.PropertyIDs, input.ID)
			before.Attachments[i].PropertyIDs = slices.DeleteFunc(slices.Clone(link.PropertyIDs), func(id string) bool { return id == input.ID })
		}
	}
	if input.Action == "detach" && !found {
		return reject(404, "Property attachment not found.")
	}
	if err := validatePropertyAttachments(before); err != nil {
		return err
	}
	for _, record := range before.Properties {
		if err := validateStandaloneFormula(record.Definition, before.Properties); err != nil {
			return err
		}
	}
	if input.Action == "delete" {
		_, err := tx.Exec(ctx, `DELETE FROM work_item_properties WHERE workspace_id=$1 AND id=$2`, workspace, input.ID)
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM work_item_type_properties WHERE type_id=$1 AND property_id=$2`, input.TypeID, input.ID)
	return err
}

func validatePropertyChange(def property.Definition, after PropertyCatalog) error {
	for _, other := range after.Properties {
		if other.Definition.ID != def.ID && other.Definition.Name == def.Name {
			return reject(409, "A property with this internal name already exists.")
		}
	}
	if err := validatePropertyAttachments(after); err != nil {
		return err
	}
	for _, record := range after.Properties {
		if err := validateStandaloneFormula(record.Definition, after.Properties); err != nil {
			return err
		}
	}
	return nil
}
