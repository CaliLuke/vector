package workitemtypes

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func changeAPIProperty(ctx context.Context, tx pgx.Tx, in *PropertyAPIMutation, before PropertyCatalog) (uuid.UUID, error) {
	id := in.PropertyID
	def := property.Definition{Configuration: property.Configuration{Active: true}, Contexts: []property.Context{}}
	if in.Action == "create" {
		id = uuid.New()
		def.ID = id.String()
	} else {
		found := false
		for _, row := range before.Properties {
			if row.Definition.ID == id.String() {
				def = row.Definition
				found = true
				break
			}
		}
		if !found {
			return id, projectmutations.ErrNotFound
		}
	}
	c := in.Changes
	if c.DisplayName != nil {
		def.Title = *c.DisplayName
	}
	if c.Description != nil {
		def.Description = *c.Description
	}
	if c.Kind != nil {
		kind, ok := apiPropertyKinds[*c.Kind]
		if *c.Kind == "RELATION" && c.RelationType != nil {
			kind, ok = apiRelationKinds[*c.RelationType]
		}
		if !ok {
			return id, reject(422, "This property kind requires native configuration support.")
		}
		def.Kind = kind
		def.Configuration.Format = ""
		if kind == property.Text {
			def.Configuration.Format = "Single line"
		}
		if kind == property.Date {
			def.Configuration.Format = "Jan 15, 2025"
		}
	}
	if c.RelationType != nil && (c.Kind == nil || *c.Kind != "RELATION") {
		kind, ok := apiRelationKinds[*c.RelationType]
		if !ok {
			return id, projectmutations.ErrInvalid
		}
		current, _ := apiKind(def.Kind)
		if current != "RELATION" {
			return id, projectmutations.ErrInvalid
		}
		def.Kind = kind
	}
	if c.DisplayFormat != nil {
		formats := apiTextFormats
		if def.Kind == property.Date {
			formats = apiDateFormats
		}
		if def.Kind == property.Text || def.Kind == property.Date {
			format, ok := formats[*c.DisplayFormat]
			if !ok {
				return id, projectmutations.ErrInvalid
			}
			def.Configuration.Format = format
		}
	}
	if c.Required != nil {
		def.Configuration.Required = *c.Required
	}
	if c.Multiple != nil {
		def.Configuration.Multiple = *c.Multiple
	}
	if c.Active != nil {
		def.Configuration.Active = *c.Active
	}
	if err := applyAPIPropertyConfiguration(&def, c); err != nil {
		return id, err
	}
	if c.Defaults != nil {
		var err error
		def.Configuration.Default, err = propertyAPIDefault(def, c.Defaults)
		if err != nil {
			return id, err
		}
	}
	optionMetadata := map[string]any{}
	if c.Options != nil {
		def.Configuration.Options = []property.Option{}
		for _, option := range c.Options {
			oid := uuid.NewString()
			def.Configuration.Options = append(def.Configuration.Options, property.Option{ID: oid, Name: option.Name})
			optionMetadata[oid] = newOptionMetadata(option, in.ActorID)
			if option.Default != nil && *option.Default {
				if err := setOptionDefault(&def, oid, true); err != nil {
					return id, err
				}
			}
		}
	}
	if err := saveProperty(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), PropertyInput{Action: in.Action, Definition: def}, before, false); err != nil {
		return id, err
	}
	metadata := map[string]any{}
	if c.ExternalID != nil {
		metadata["external_id"] = *c.ExternalID
	}
	if c.ExternalSource != nil {
		metadata["external_source"] = *c.ExternalSource
	}
	if c.Options != nil {
		metadata["options"] = optionMetadata
	}
	if len(metadata) > 0 {
		raw, err := json.Marshal(metadata)
		if err != nil {
			return id, err
		}
		if _, err = tx.Exec(ctx, `UPDATE work_item_properties SET api_metadata=api_metadata||$2::jsonb WHERE id=$1`, id, raw); err != nil {
			return id, err
		}
	}
	if in.Action == "create" && in.TypeID != uuid.Nil {
		current, err := loadProperties(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), true)
		if err != nil {
			return id, err
		}
		link := *in
		link.Attach = []uuid.UUID{id}
		link.Detach = nil
		if err = manageAPIProperties(ctx, tx, link, current); err != nil {
			return id, err
		}
	}
	in.PropertyID = id
	return id, nil
}
func propertyAPIDefault(def property.Definition, values []string) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if def.Kind == property.Number {
		if len(values) != 1 {
			return nil, projectmutations.ErrInvalid
		}
		number, err := strconv.ParseFloat(values[0], 64)
		if err != nil {
			return nil, projectmutations.ErrInvalid
		}
		return number, nil
	}
	if def.Configuration.Multiple {
		return values, nil
	}
	if len(values) != 1 {
		return nil, projectmutations.ErrInvalid
	}
	return values[0], nil
}
func newOptionMetadata(option PropertyAPIOption, actor uuid.UUID) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	out := map[string]any{"created_at": now, "updated_at": now, "created_by": actor.String(), "updated_by": actor.String()}
	for key, value := range map[string]*string{"description": option.Description, "color": option.Color, "external_source": option.ExternalSource, "external_id": option.ExternalID} {
		if value != nil {
			out[key] = *value
		}
	}
	return out
}
func setOptionDefault(def *property.Definition, id string, enabled bool) error {
	ids := slices.Clone(memberValueIDs(def.Configuration.Default))
	ids = slices.DeleteFunc(ids, func(current string) bool { return current == id })
	if enabled {
		if !def.Configuration.Multiple {
			ids = nil
		}
		ids = append(ids, id)
	}
	var err error
	def.Configuration.Default, err = propertyAPIDefault(*def, ids)
	return err
}
func changeAPIOption(ctx context.Context, tx pgx.Tx, in *PropertyAPIMutation, before PropertyCatalog) (uuid.UUID, error) {
	var def property.Definition
	for _, row := range before.Properties {
		if row.Definition.ID == in.PropertyID.String() {
			def = row.Definition
		}
	}
	if def.ID == "" {
		return in.OptionID, projectmutations.ErrNotFound
	}
	if def.Kind != property.Dropdown {
		return in.OptionID, reject(422, "Option actions require a dropdown property.")
	}
	def.Configuration.Options = slices.Clone(def.Configuration.Options)
	id := in.OptionID
	index := slices.IndexFunc(def.Configuration.Options, func(option property.Option) bool { return option.ID == id.String() })
	if in.Action == "create_option" {
		id = uuid.New()
		index = len(def.Configuration.Options)
		def.Configuration.Options = append(def.Configuration.Options, property.Option{ID: id.String(), Name: in.Option.Name})
	} else if index < 0 {
		return id, projectmutations.ErrNotFound
	}
	if in.Action == "delete_option" {
		def.Configuration.Options = slices.Delete(def.Configuration.Options, index, index+1)
		if err := setOptionDefault(&def, id.String(), false); err != nil {
			return id, err
		}
	} else {
		if in.Option.Name != "" {
			def.Configuration.Options[index].Name = in.Option.Name
		}
		if in.Option.Default != nil {
			if err := setOptionDefault(&def, id.String(), *in.Option.Default); err != nil {
				return id, err
			}
		}
	}
	if err := saveProperty(ctx, tx, in.WorkspaceID.String(), in.ActorID.String(), PropertyInput{Action: "update", Definition: def}, before, false); err != nil {
		return id, err
	}
	var metadata map[string]any
	if err := tx.QueryRow(ctx, `SELECT api_metadata FROM work_item_properties WHERE id=$1`, in.PropertyID).Scan(&metadata); err != nil {
		return id, err
	}
	options, _ := metadata["options"].(map[string]any)
	if options == nil {
		options = map[string]any{}
	}
	if in.Action == "delete_option" {
		delete(options, id.String())
	} else {
		current, _ := options[id.String()].(map[string]any)
		if current == nil {
			current = newOptionMetadata(in.Option, in.ActorID)
		} else {
			for key, value := range newOptionMetadata(in.Option, in.ActorID) {
				if strings.HasPrefix(key, "created_") {
					continue
				}
				current[key] = value
			}
		}
		options[id.String()] = current
	}
	metadata["options"] = options
	raw, err := json.Marshal(metadata)
	if err != nil {
		return id, err
	}
	_, err = tx.Exec(ctx, `UPDATE work_item_properties SET api_metadata=$2 WHERE id=$1`, in.PropertyID, raw)
	return id, err
}
