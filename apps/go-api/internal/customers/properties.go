package customers

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func prepareProperty(in *Command, before map[string]any) error {
	kind, _ := in.Fields["property_type"].(string)
	if before != nil {
		kind, _ = before["property_type"].(string)
	}
	if !slices.Contains([]string{"TEXT", "DATETIME", "DECIMAL", "BOOLEAN", "OPTION", "RELATION", "URL", "EMAIL", "FILE"}, kind) {
		return projectmutations.ErrInvalid
	}
	if display, ok := in.Fields["display_name"].(string); ok {
		name := propertyName(display)
		if name == "" {
			return projectmutations.ErrInvalid
		}
		in.Fields["name"] = name
	}
	relation := in.Fields["relation_type"]
	if relation == nil && before != nil {
		relation = before["relation_type"]
	}
	if kind == "RELATION" {
		if relation != "ISSUE" && relation != "USER" {
			return projectmutations.ErrInvalid
		}
	} else if relation != nil {
		return projectmutations.ErrInvalid
	}
	if in.Options != nil && kind != "OPTION" {
		return projectmutations.ErrInvalid
	}
	if in.Action == "create" {
		format, _ := in.Fields["display_format"].(string)
		delete(in.Fields, "display_format")
		if kind == "TEXT" {
			if format == "" {
				format = "single-line"
			}
			if !slices.Contains([]string{"single-line", "multi-line", "readonly"}, format) {
				return projectmutations.ErrInvalid
			}
		} else if kind == "DATETIME" {
			if format == "" {
				format = "MMM dd, yyyy"
			}
			if !slices.Contains([]string{"MMM dd, yyyy", "dd/MM/yyyy", "MM/dd/yyyy", "yyyy/MM/dd"}, format) {
				return projectmutations.ErrInvalid
			}
		} else if format != "" {
			return projectmutations.ErrInvalid
		}
		if format != "" {
			in.Fields["settings"] = map[string]any{"display_format": format}
		}
	}
	return nil
}
func writeOptions(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID) error {
	for index, option := range in.Options {
		var optionID uuid.UUID
		for key, value := range option {
			switch key {
			case "id":
				text, ok := value.(string)
				if !ok || in.Action == "create" {
					return projectmutations.ErrInvalid
				}
				var err error
				optionID, err = uuid.Parse(text)
				if err != nil || optionID == uuid.Nil {
					return projectmutations.ErrInvalid
				}
			case "name", "description":
				text, ok := value.(string)
				if !ok || strings.ContainsRune(text, 0) || key == "name" && (strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > 255) {
					return projectmutations.ErrInvalid
				}
			case "is_default":
				if _, ok := value.(bool); !ok {
					return projectmutations.ErrInvalid
				}
			default:
				return projectmutations.ErrInvalid
			}
		}
		if optionID == uuid.Nil {
			if option["name"] == nil {
				return projectmutations.ErrInvalid
			}
			isDefault := option["is_default"] == true
			if _, err := tx.Exec(ctx, `INSERT INTO customer_property_options(workspace_id,property_id,name,description,is_default,sort_order,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, in.WorkspaceID, id, option["name"], option["description"], isDefault, 65535+index, in.ActorID); err != nil {
				return err
			}
		} else {
			tag, err := tx.Exec(ctx, `UPDATE customer_property_options SET name=COALESCE($4,name),description=COALESCE($5,description),is_default=COALESCE($6,is_default),updated_at=now(),updated_by_id=$7 WHERE id=$1 AND property_id=$2 AND workspace_id=$3 AND deleted_at IS NULL`, optionID, id, in.WorkspaceID, option["name"], option["description"], option["is_default"], in.ActorID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return projectmutations.ErrNotFound
			}
		}
	}
	return nil
}
func checkProperty(ctx context.Context, tx pgx.Tx, in Command, id uuid.UUID) error {
	p, err := record(ctx, tx, in.Query, id)
	if err != nil {
		return err
	}
	if err = validateValues(ctx, tx, in.Query, p, stringValues(p["default_value"]), true); err != nil {
		return err
	}
	if p["is_multi"] != true {
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM customer_property_options WHERE property_id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND is_default AND is_active`, id, in.WorkspaceID).Scan(&count); err != nil {
			return err
		}
		if count > 1 {
			return projectmutations.ErrInvalid
		}
	}
	existing, err := rows(ctx, tx, `SELECT jsonb_build_object('values',value_strings) FROM customer_property_values WHERE property_id=$1 AND workspace_id=$2`, id, in.WorkspaceID)
	if err != nil {
		return err
	}
	for _, row := range existing {
		values := stringValues(row["values"])
		if len(values) == 0 && p["is_required"] == true {
			return projectmutations.ErrInvalid
		}
		if err = validateValues(ctx, tx, in.Query, p, values, true); err != nil {
			return err
		}
	}
	return nil
}
