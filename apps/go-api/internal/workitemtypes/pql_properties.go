package workitemtypes

import (
	"context"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// PQLProperty resolves only comparison metadata in this authorized snapshot.
// Full definitions, contexts and reference catalogs never become query inputs.
func (snapshot *PropertyDiscoverySnapshot) PQLProperty(ctx context.Context, id string) (issuequery.PQLPropertyDefinition, bool, error) {
	definition, options, available, err := snapshot.propertyFilterScope(id, "", "")
	if err != nil {
		return issuequery.PQLPropertyDefinition{}, false, &PropertyDefinitionError{cause: err}
	}
	if definition == nil || !available {
		return issuequery.PQLPropertyDefinition{}, false, nil
	}
	resolved := issuequery.PQLPropertyDefinition{}
	switch definition.Kind {
	case property.Text, property.URL:
		resolved.Kind = issuequery.PQLPropertyText
	case property.Number:
		resolved.Kind = issuequery.PQLPropertyNumber
	case property.Date:
		resolved.Kind = issuequery.PQLPropertyDate
	case property.Boolean:
		resolved.Kind = issuequery.PQLPropertyBoolean
	case property.Dropdown, property.Member, property.Release:
		resolved.Kind = issuequery.PQLPropertySelection
		resolved.ValidateSelections = func(ids []string) (bool, error) {
			catalog := options
			if definition.Kind == property.Member || definition.Kind == property.Release {
				var readErr error
				catalog, readErr = snapshot.predicateReferences(ctx, snapshot.tx, definition.Kind)
				if readErr != nil {
					return false, readErr
				}
			}
			for _, id := range ids {
				if !catalog[id] {
					return false, nil
				}
			}
			return true, nil
		}
	}
	return resolved, true, nil
}

// PropertyDefinitionError distinguishes invalid stored configuration from
// database/cancellation failures without exposing its contents to callers.
type PropertyDefinitionError struct{ cause error }

func (e *PropertyDefinitionError) Error() string {
	return "Work item property definition cannot be evaluated."
}
func (e *PropertyDefinitionError) Unwrap() error { return e.cause }
