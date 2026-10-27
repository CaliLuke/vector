//go:build integration

package workitemtypes_test

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAPIPQLPropertyRejectsUnmappedNativeFamilies(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	formula, rich, cascade := uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: formula, Title: "Computed", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "1", Result: "Number"}},
		property.Definition{ID: rich, Title: "Rich notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: uuid.NewString(), Name: "Country"}, {ID: uuid.NewString(), Name: "Region"}}}},
	)
	for _, id := range []string{formula, rich, cascade} {
		requirePQLPropertyError(t, f, repo, f.owner, `title ~ "Discovery" OR `+pqlCF(id)+` IS NULL`, 400)
	}
}

func TestAPIPQLPropertyDistinguishesValueErrorsFromDefinitionFailures(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	require.NoError(t, err)
	text, number := uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: text, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Paragraph"}},
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
	)
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], map[string]any{text: "visible", number: "secret invalid value"})
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(text) + ` = "visible"`: {ids[0]}, pqlCF(number) + ` IS NULL`: {ids[1]},
		`NOT ` + pqlCF(number) + ` IS NULL`: {},
	})
	var original []byte
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT definition FROM work_item_properties WHERE id=$1`, number).Scan(&original))
	f.exec(`UPDATE work_item_properties SET definition='[]'::jsonb WHERE id=$1`, number)
	failure := requirePQLPropertyError(t, f, repo, f.owner, pqlCF(text)+` = "visible"`, 400)
	require.NotContains(t, fmt.Sprint(failure), "secret")
	f.exec(`UPDATE work_item_properties SET definition=$2::jsonb WHERE id=$1`, number, original)
	f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{contexts}',$2::jsonb) WHERE id=$1`, number, []property.Context{{ID: uuid.NewString(), Name: "secret context", Types: []string{f.task}, Overrides: []string{"secret override"}}})
	failure = requirePQLPropertyError(t, f, repo, f.owner, pqlCF(text)+` = "visible" OR title ~ "Discovery"`, 400)
	require.NotContains(t, fmt.Sprint(failure), "secret")
}
