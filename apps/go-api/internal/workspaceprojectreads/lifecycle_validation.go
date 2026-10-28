package workspaceprojectreads

import (
	"context"
	"regexp"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

var forbiddenProjectCharacters = regexp.MustCompile(`^.*[&+,:;$^}{*=?@#|'<>.()%!-].*$`)
var projectBooleanFields = []string{"module_view", "cycle_view", "issue_views_view", "page_view", "intake_view", "guest_view_all_features", "is_issue_type_enabled", "is_time_tracking_enabled"}

func (r *LifecycleRepository) validateProjectFields(ctx context.Context, workspace, project string, input djangoinput.Input, app, replace, create bool) (map[string]any, error) {
	v := &projectFieldValidation{input: input, values: map[string]any{}, failures: map[string]any{}}
	if err := v.validateScalars(app, replace, create); err != nil {
		return nil, err
	}
	if err := r.validateProjectReferences(ctx, v, app, create); err != nil {
		return nil, err
	}
	if app {
		if err := r.validateProjectCover(ctx, workspace, project, v, create); err != nil {
			return nil, err
		}
		if err := r.validateProjectNames(ctx, workspace, project, v); err != nil {
			return nil, err
		}
	}
	// Field errors precede object-level validation, including relationship scope.
	if len(v.failures) > 0 {
		return nil, projectFailure(400, v.failures)
	}
	if !app {
		if err := v.apiNames(); err != nil {
			return nil, err
		}
	}
	if err := r.validateProjectRelationScopes(ctx, workspace, project, v.values, app); err != nil {
		return nil, err
	}
	if app {
		if err := v.descriptionHTML(); err != nil {
			return nil, err
		}
	}
	return v.values, nil
}
