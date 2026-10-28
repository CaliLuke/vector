package workspaceprojectreads

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
)

func TestProjectFieldFailuresPrecedeRelationshipScopeFailures(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		projectLifecycleBoolRow(true), // The state exists, but field failures prevent scope validation.
	}}}
	repo, err := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{
		"name":          "invalid!",
		"page_view":     nil,
		"default_state": "00000000-0000-4000-8000-000000000001",
	}}, true, false, false)
	want := map[string]any{
		"name":      []string{"PROJECT_NAME_CANNOT_CONTAIN_SPECIAL_CHARACTERS"},
		"page_view": []string{"This field may not be null."},
	}
	assertProjectValidationFailure(t, err, want)
	if len(db.queries) != 1 {
		t.Fatalf("field errors must prevent relationship scope reads: %v", db.queries)
	}
}

func TestProjectRelationshipFailuresPreserveSessionAggregationAndAPIPrecedence(t *testing.T) {
	for _, test := range []struct {
		name string
		app  bool
		want map[string]any
	}{
		{name: "session collects failures", app: true, want: map[string]any{
			"project_lead":  []string{"The provided user is not an active member of this workspace."},
			"default_state": []string{"The provided state does not belong to this project."},
		}},
		{name: "API returns first failure", want: map[string]any{
			"project_lead": []string{"The provided user is not a member of this workspace."},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
				projectLifecycleBoolRow(true), projectLifecycleBoolRow(true),
				projectLifecycleBoolRow(false), projectLifecycleBoolRow(false),
			}}}
			repo, err := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
			if err != nil {
				t.Fatal(err)
			}
			_, err = repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{
				"project_lead":  "00000000-0000-4000-8000-000000000001",
				"default_state": "00000000-0000-4000-8000-000000000002",
			}}, test.app, false, false)
			assertProjectValidationFailure(t, err, test.want)
		})
	}
}

func TestAPIProjectFieldFailuresPrecedeSpecialCharacterFailure(t *testing.T) {
	repo, err := NewLifecycleRepository(&projectLifecycleDBFake{}, &projectLifecycleEffectsFake{}, "", validProjectCover)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: map[string]any{
		"name": "invalid!", "page_view": nil,
	}}, false, false, false)
	assertProjectValidationFailure(t, err, map[string]any{"page_view": []string{"This field may not be null."}})
}

func assertProjectValidationFailure(t *testing.T, err error, want map[string]any) {
	t.Helper()
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || !reflect.DeepEqual(response.Body, want) {
		t.Fatalf("validation error = %v, want HTTP 400 with %#v", err, want)
	}
}
