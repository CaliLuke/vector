//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPropertyAPIComputedValuesUseCurrentScopeWithoutStoredRows(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, items := seedDiscoveryItems(t, f)
	number, calculation, fixed, broken := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: number, Name: "effort", Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Calculated effort", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: "Number"}},
		property.Definition{ID: broken, Title: "Broken formula", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "1 / 0", Result: "Number"}},
		property.Definition{ID: fixed, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Project guidance"}, Contexts: []property.Context{{ID: uuid.NewString(), Name: "Hidden in bug type", Types: []string{f.bug}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}},
	)
	q := types.PropertyValueAPIQuery{APIScope: types.APIScope{ActorID: uuid.MustParse(f.owner), WorkspaceID: uuid.MustParse(f.workspace), WorkspaceSlug: "types", ProjectID: uuid.MustParse(f.project)}, IssueID: uuid.MustParse(items[0]), IncludeRevision: true}
	read := func(id string) map[string]any {
		t.Helper()
		q.PropertyID = uuid.MustParse(id)
		raw, err := f.repo.ReadPropertyValueAPI(f.ctx, q)
		require.NoError(t, err)
		var record map[string]any
		require.NoError(t, json.Unmarshal(raw, &record))
		require.NotEmpty(t, record["id"])
		require.Nil(t, record["created_at"])
		require.Nil(t, record["updated_at"])
		return record
	}
	empty := read(calculation)
	require.Nil(t, empty["value"], "an empty formula is a value projection, not a missing property")
	require.Equal(t, "Project guidance", read(fixed)["value"])
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, items[0], map[string]any{number: 3})
	computed := read(calculation)
	require.Equal(t, float64(6), computed["value"])
	require.Equal(t, empty["id"], computed["id"])
	require.NotEqual(t, empty["revision"], computed["revision"])
	require.Equal(t, computed, read(calculation))
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, items[0], map[string]any{number: 0, fixed: "Stale content"})
	require.Equal(t, float64(0), read(calculation)["value"])
	require.Equal(t, "Project guidance", read(fixed)["value"])
	q.PropertyID = uuid.MustParse(broken)
	_, err := f.repo.ReadPropertyValueAPI(f.ctx, q)
	assertTypeError(t, err, 422)
	q.PropertyID, q.IssueID = uuid.MustParse(calculation), uuid.MustParse(items[2])
	_, err = f.repo.ReadPropertyValueAPI(f.ctx, q)
	require.ErrorIs(t, err, projectmutations.ErrNotFound, "another type cannot read this formula")
	q.IssueID, q.ActorID = uuid.MustParse(items[0]), uuid.MustParse(f.outsider)
	_, err = f.repo.ReadPropertyValueAPI(f.ctx, q)
	require.ErrorIs(t, err, projectmutations.ErrForbidden)
	q.ActorID = uuid.MustParse(f.owner)
	f.exec(`UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.owner)
	_, err = f.repo.ReadPropertyValueAPI(f.ctx, q)
	require.ErrorIs(t, err, projectmutations.ErrForbidden, "current membership is required on every read")
}
