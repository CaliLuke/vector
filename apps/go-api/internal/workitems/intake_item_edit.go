package workitems

import (
	"context"
	"maps"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

// Intake edits share the ordinary item's value and preservation contract. The
// surrounding project transaction owns the schema lock and durable effects.
func (r *Repository) saveIntakeIssueEdit(ctx context.Context, actor string, old map[string]any, changes issueChanges, input djangoinput.Input) (map[string]any, error) {
	submitted, err := submittedCustomValues(input)
	if err != nil {
		return nil, err
	}
	updated := maps.Clone(old)
	maps.Copy(updated, changes.values)
	members, err := r.validateCustomValues(ctx, actor, updated, old, submitted, false, false, hasSubmittedType(input.Fields))
	if err != nil {
		return nil, err
	}
	if err = r.replaceIssueRelations(ctx, old, changes, false); err != nil {
		return nil, err
	}
	saved, err := r.saveIssue(ctx, actor, updated, old)
	if err != nil {
		return nil, err
	}
	if err = r.subscribePropertyMembers(ctx, actor, saved, members); err != nil {
		return nil, err
	}
	return saved, nil
}
