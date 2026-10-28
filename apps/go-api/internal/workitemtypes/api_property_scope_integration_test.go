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

func TestPropertyAPIUsesProjectContextAndDistinguishesMissingOptions(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Context date", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "Jan 15, 2025"}, Contexts: []property.Context{{ID: uuid.NewString(), Name: "Project format", Projects: []string{f.project}, Overrides: []string{"format"}, Configuration: property.Configuration{Format: "2025/01/15"}}}})
	q := types.PropertyAPIQuery{APIScope: types.APIScope{ActorID: uuid.MustParse(f.owner), WorkspaceID: uuid.MustParse(f.workspace), WorkspaceSlug: "types", ProjectID: uuid.MustParse(f.project)}, Action: "retrieve", PropertyID: uuid.MustParse(id)}
	raw, err := f.repo.ReadPropertyAPI(f.ctx, q)
	require.NoError(t, err)
	var record map[string]any
	require.NoError(t, json.Unmarshal(raw, &record))
	require.Equal(t, map[string]any{"display_format": "yyyy/MM/dd"}, record["settings"])
	q.Action = "list_options"
	empty, err := f.repo.ReadPropertyAPI(f.ctx, q)
	require.NoError(t, err)
	require.JSONEq(t, `[]`, string(empty))
	q.PropertyID = uuid.New()
	_, err = f.repo.ReadPropertyAPI(f.ctx, q)
	require.ErrorIs(t, err, projectmutations.ErrNotFound)
}

func TestPropertyAPIUpdatesFormatWithoutRepeatingKind(t *testing.T) {
	f := newProjectTypeFixture(t)
	date, text := uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: date, Title: "Date", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "Jan 15, 2025"}},
		property.Definition{ID: text, Title: "Text", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
	)
	for _, example := range []struct{ id, format string }{{date, "yyyy/MM/dd"}, {text, "multi-line"}} {
		q := types.PropertyAPIQuery{APIScope: types.APIScope{ActorID: uuid.MustParse(f.owner), WorkspaceID: uuid.MustParse(f.workspace), WorkspaceSlug: "types"}, Action: "update", PropertyID: uuid.MustParse(example.id)}
		_, err := f.repo.MutatePropertyAPI(f.ctx, types.PropertyAPIMutation{PropertyAPIQuery: q, Changes: types.PropertyAPIChanges{DisplayFormat: &example.format}})
		if example.id == text {
			assertTypeError(t, err, 422)
			example.format = "single-line"
		} else {
			require.NoError(t, err)
		}
		q.Action = "retrieve"
		raw, err := f.repo.ReadPropertyAPI(f.ctx, q)
		require.NoError(t, err)
		var record map[string]any
		require.NoError(t, json.Unmarshal(raw, &record))
		require.Equal(t, map[string]any{"display_format": example.format}, record["settings"])
	}
}
