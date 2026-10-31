package customers

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	nativeproperty "github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var decimalValue = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func stringValues(value any) []string {
	if values, ok := value.([]string); ok {
		return slices.Clone(values)
	}
	out := []string{}
	if values, ok := value.([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
	}
	return out
}
func allowedReference(ctx context.Context, tx pgx.Tx, q Query, property map[string]any, value string, lock bool) (bool, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return false, projectmutations.ErrInvalid
	}
	if property["property_type"] == "RELATION" && property["relation_type"] == "ISSUE" {
		_, err = visibleItem(ctx, tx, q, id, lock)
		if errors.Is(err, projectmutations.ErrNotFound) {
			return false, nil
		}
		return err == nil, err
	}
	suffix := ""
	var found uuid.UUID
	query := ""
	args := []any{id, q.WorkspaceID}
	if property["property_type"] == "OPTION" {
		query = `SELECT id FROM customer_property_options WHERE id=$1 AND workspace_id=$2 AND property_id=$3 AND deleted_at IS NULL AND is_active`
		args = append(args, property["id"])
		if lock {
			suffix = " FOR SHARE"
		}
	} else {
		query = `SELECT u.id FROM users u JOIN workspace_members wm ON wm.member_id=u.id WHERE u.id=$1 AND u.is_active AND wm.workspace_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)`
		if lock {
			suffix = " FOR SHARE OF u,wm"
		}
	}
	err = tx.QueryRow(ctx, query+suffix, args...).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
func validateValues(ctx context.Context, tx pgx.Tx, q Query, p map[string]any, values []string, definition bool) error {
	if !definition && p["is_active"] != true {
		return projectmutations.ErrInvalid
	}
	if len(values) > 1 && p["is_multi"] != true || len(values) == 0 && p["is_required"] == true && !definition {
		return projectmutations.ErrInvalid
	}
	settings, _ := p["settings"].(map[string]any)
	if !definition && p["property_type"] == "TEXT" && settings["display_format"] == "readonly" {
		return projectmutations.ErrInvalid
	}
	for _, value := range values {
		if strings.ContainsRune(value, 0) || len(value) > 65536 {
			return projectmutations.ErrInvalid
		}
		kind := nativeproperty.Text
		switch p["property_type"] {
		case "OPTION", "RELATION":
			allowed, err := allowedReference(ctx, tx, q, p, value, true)
			if err != nil {
				return err
			}
			if !allowed {
				return projectmutations.ErrInvalid
			}
			continue
		case "DECIMAL":
			if len(value) > 4096 || !decimalValue.MatchString(value) {
				return projectmutations.ErrInvalid
			}
			continue
		case "BOOLEAN":
			if value != "True" && value != "False" {
				return projectmutations.ErrInvalid
			}
			continue
		case "DATETIME":
			kind = nativeproperty.Date
		case "URL":
			kind = nativeproperty.URL
		case "EMAIL":
			kind = nativeproperty.Email
		case "FILE":
			kind = nativeproperty.File
		case "TEXT":
		default:
			return projectmutations.ErrInvalid
		}
		config := nativeproperty.Configuration{}
		if p["property_type"] == "TEXT" && settings["display_format"] == "single-line" {
			config.Format = "Single line"
		}
		normalized, err := nativeproperty.ValidateValue(kind, config, value, false, nil)
		if err != nil || normalized == nil {
			return projectmutations.ErrInvalid
		}
	}
	return nil
}
func visibleValues(ctx context.Context, tx pgx.Tx, q Query, p map[string]any, values []string) ([]string, error) {
	out := []string{}
	for _, value := range values {
		if p["property_type"] == "RELATION" || p["property_type"] == "OPTION" {
			allowed, err := allowedReference(ctx, tx, q, p, value, false)
			if errors.Is(err, projectmutations.ErrInvalid) {
				continue // Imported malformed identities are not visible references.
			}
			if err != nil {
				return nil, err
			}
			if !allowed {
				continue
			}
		}
		out = append(out, value)
	}
	return out, nil
}
func readValues(ctx context.Context, tx pgx.Tx, q Query) (map[string][]string, error) {
	if q.ID != uuid.Nil {
		propertyQuery := q
		propertyQuery.Tool = "customer_property"
		if _, err := record(ctx, tx, propertyQuery, q.ID); err != nil {
			return nil, err
		}
	}
	list, err := rows(ctx, tx, `SELECT to_jsonb(p)||jsonb_build_object('saved_values',v.value_strings) FROM customer_property_values v JOIN customer_properties p ON p.id=v.property_id AND p.workspace_id=v.workspace_id WHERE v.customer_id=$1 AND v.workspace_id=$2 AND p.deleted_at IS NULL AND p.is_active AND ($3::uuid IS NULL OR p.id=$3) ORDER BY p.id`, q.CustomerID, q.WorkspaceID, optionalID(q.ID))
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	if q.ID != uuid.Nil {
		out[q.ID.String()] = []string{}
	}
	for _, p := range list {
		values, err := visibleValues(ctx, tx, q, p, stringValues(p["saved_values"]))
		if err != nil {
			return nil, err
		}
		out[p["id"].(string)] = values
	}
	return out, nil
}
func setValues(ctx context.Context, tx pgx.Tx, in Command) error {
	for id, values := range in.Values {
		q := in.Query
		q.Tool = "customer_property"
		p, err := record(ctx, tx, q, uuid.MustParse(id))
		if err != nil {
			return err
		}
		if err = validateValues(ctx, tx, q, p, values, false); err != nil {
			return err
		}
		raw, _ := json.Marshal(values)
		if _, err = tx.Exec(ctx, `INSERT INTO customer_property_values(customer_id,property_id,workspace_id,value_strings,updated_by_id) VALUES($1,$2,$3,$4::jsonb,$5) ON CONFLICT(customer_id,property_id) DO UPDATE SET value_strings=EXCLUDED.value_strings,updated_at=now(),updated_by_id=EXCLUDED.updated_by_id`, in.CustomerID, id, in.WorkspaceID, raw, in.ActorID); err != nil {
			return err
		}
	}
	return nil
}
func applyDefaults(ctx context.Context, tx pgx.Tx, in Command) error {
	properties, err := rows(ctx, tx, `SELECT to_jsonb(p) FROM customer_properties p WHERE workspace_id=$1 AND deleted_at IS NULL AND is_active ORDER BY id`, in.WorkspaceID)
	if err != nil {
		return err
	}
	for _, p := range properties {
		values := stringValues(p["default_value"])
		if len(values) == 0 && p["property_type"] == "OPTION" {
			options, rowsErr := rows(ctx, tx, `SELECT jsonb_build_object('id',id) FROM customer_property_options WHERE property_id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND is_active AND is_default ORDER BY sort_order,id`, p["id"], in.WorkspaceID)
			if rowsErr != nil {
				return rowsErr
			}
			for _, option := range options {
				values = append(values, option["id"].(string))
			}
		}
		if len(values) == 0 {
			continue
		}
		if err = validateValues(ctx, tx, in.Query, p, values, true); err != nil {
			return err
		}
		raw, _ := json.Marshal(values)
		if _, err = tx.Exec(ctx, `INSERT INTO customer_property_values(customer_id,property_id,workspace_id,value_strings,updated_by_id) VALUES($1,$2,$3,$4::jsonb,$5)`, in.CustomerID, p["id"], in.WorkspaceID, raw, in.ActorID); err != nil {
			return err
		}
	}
	return nil
}

// authorizeDefaultReferences checks selected defaults before receipt lookup.
// Retained definitions bind authorization after a property has been soft-deleted.
func authorizeDefaultReferences(ctx context.Context, tx pgx.Tx, in Command) error {
	defaults, _ := in.Fields["default_value"].([]string)
	if in.Tool != "customer_property" || len(defaults) == 0 {
		return nil
	}
	property := maps.Clone(in.Fields)
	property["id"] = in.ID.String()
	if in.Action == "update" {
		var kind, relation string
		err := tx.QueryRow(ctx, `SELECT property_type,COALESCE(relation_type,'') FROM customer_properties WHERE id=$1 AND workspace_id=$2 FOR SHARE`, in.ID, in.WorkspaceID).Scan(&kind, &relation)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		if err != nil {
			return err
		}
		property["property_type"] = kind
		if property["relation_type"] == nil {
			property["relation_type"] = relation
		}
	}
	if property["property_type"] != "RELATION" && property["property_type"] != "OPTION" {
		return nil
	}
	for _, value := range defaults {
		allowed, err := allowedReference(ctx, tx, in.Query, property, value, true)
		if err != nil {
			return err
		}
		if !allowed {
			return projectmutations.ErrInvalid
		}
	}
	return nil
}
