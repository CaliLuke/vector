package workitemtypes

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// A reviewed type switch may include unsaved edits to properties that are about
// to disappear. Validate these against their source configuration, then preserve
// the edited value. Destination validation still owns shared and new fields.
func submittedFormerValues(ctx context.Context, tx pgx.Tx, in ItemValueInput, previous, next []property.Definition, sourceVisible bool) (map[string]any, map[string]any, error) {
	stored, submitted := maps.Clone(in.Stored), maps.Clone(in.Submitted)
	if stored == nil {
		stored = map[string]any{}
	}
	source := in
	source.ProjectID, source.TypeID = in.PreviousProjectID, in.PreviousTypeID
	fields := map[string]string{}
	for _, def := range previous {
		value, edited := submitted[def.ID]
		if !edited || slices.ContainsFunc(next, func(candidate property.Definition) bool { return candidate.ID == def.ID }) {
			continue
		}
		config, _, err := property.Effective(def, source.ProjectID, source.TypeID)
		if err != nil {
			return nil, nil, err
		}
		if !config.Active || def.Kind == property.Formula || def.Kind == property.Text && config.Format == "Read-only" || in.Draft && def.Kind == property.Cascading {
			fields[def.ID] = "This former property is inactive or read-only."
			continue
		}
		if !sourceVisible && (def.Kind == property.Member || def.Kind == property.Release || def.Kind == property.Issue) && nonemptyPropertyValue(value) {
			fields[def.ID] = "This selection cannot be edited without access to its former project."
			continue
		}
		allowed, err := itemValueReferences(ctx, tx, source, []property.Definition{def}, true)
		if err != nil {
			return nil, nil, err
		}
		// The property is being removed; an explicit clear need not satisfy its old
		// required policy. All format, option and reference validation still applies.
		config.Required = false
		config.Levels = slices.Clone(config.Levels)
		for index := range config.Levels {
			config.Levels[index].Required = false
		}
		canonical, err := property.ValidateValue(def.Kind, config, value, in.Draft, allowed)
		if err != nil {
			fields[def.ID] = err.Error()
			continue
		}
		stored[def.ID] = canonical
		delete(submitted, def.ID)
	}
	if len(fields) > 0 {
		return nil, nil, &ItemValueError{Fields: fields}
	}
	return stored, submitted, nil
}

func transitionSourceVisible(ctx context.Context, tx pgx.Tx, in ItemValueInput) (bool, error) {
	if in.PreviousProjectID == "" {
		return true, nil
	}
	if in.ActorID == "" {
		return false, nil
	}
	var slug string
	if err := tx.QueryRow(ctx, `SELECT slug FROM workspaces WHERE id=$1 AND deleted_at IS NULL`, in.WorkspaceID).Scan(&slug); err != nil {
		return false, err
	}
	_, _, err := authorizeProject(ctx, tx, in.ActorID, slug, in.PreviousProjectID, false, false)
	var denied *Error
	if errors.As(err, &denied) && (denied.Status == 403 || denied.Status == 404) {
		return false, nil
	}
	return err == nil, err
}
