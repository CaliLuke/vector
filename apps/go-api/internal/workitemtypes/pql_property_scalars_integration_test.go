//go:build integration

package workitemtypes_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const pqlPropertyOperation = "api_v1_workspaces_by_slug_projects_by_project_id_issues"

func pqlCF(id string) string { return fmt.Sprintf(`cf[%q]`, id) }

func assertPQLPropertyResults(t *testing.T, f projectTypeFixture, repo *workitems.Repository, examples map[string][]string) {
	t.Helper()
	for query, expected := range examples {
		t.Run(query, func(t *testing.T) {
			page := readPropertyPage(t, f, repo, pqlPropertyOperation, f.owner, map[string]string{"pql": query, "per_page": "100", "include_properties": "true"}, true)
			actual := []string{}
			for _, row := range page.Results {
				actual = append(actual, row.ID)
				require.Nil(t, row.Values, "internal predicate evaluation must not enable API value projection")
				require.True(t, row.EvaluatedAt.IsZero())
			}
			slices.Sort(actual)
			want := append([]string{}, expected...)
			slices.Sort(want)
			require.Equal(t, want, actual)
			require.Equal(t, len(expected), page.Total)
		})
	}
}

func TestAPIPQLPropertyScalarsUseEffectiveValuesAndSQLNulls(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	text, number, date, boolean, link, fixed := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: text, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Paragraph"}},
		property.Definition{ID: number, Title: "Score", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: date, Title: "Review date", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "Jan 15, 2025"}},
		property.Definition{ID: boolean, Title: "Approved", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: link, Title: "Reference", Kind: property.URL, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: fixed, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Effective guidance"}},
	)
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], map[string]any{text: "Alpha 50%_\\", number: 0, date: "2026-10-27", link: "https://example.test/one", fixed: "stale text"})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[1], map[string]any{text: "Beta", number: 9, date: "2026-11-01", boolean: true})
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(text) + ` ~ "50%_"`: {ids[0]}, pqlCF(text) + ` ~ "\\"`: {ids[0]},
		pqlCF(text) + ` ~ "alpha"`: {ids[0]}, pqlCF(text) + ` = "alpha"`: {},
		pqlCF(text) + ` != "Beta"`: {ids[0]}, `NOT ` + pqlCF(text) + ` ~ "alpha"`: {ids[1]},
		pqlCF(text) + ` IS NULL`: {}, pqlCF(text) + ` IS NOT EMPTY`: {ids[0], ids[1]},
		pqlCF(number) + ` = 0`: {ids[0]}, pqlCF(number) + ` != 0`: {ids[1]},
		pqlCF(number) + ` > 0`: {ids[1]}, pqlCF(number) + ` <= 0`: {ids[0]},
		pqlCF(number) + ` BETWEEN 0 AND 9`:          {ids[0], ids[1]},
		pqlCF(number) + ` = 9.00000000000000000001`: {},
		pqlCF(date) + ` = "2026-10-27"`:             {ids[0]}, pqlCF(date) + ` > "2026-10-27"`: {ids[1]},
		pqlCF(date) + ` BETWEEN "2026-10-27" AND "2026-11-01"`: {ids[0], ids[1]},
		pqlCF(boolean) + ` = false`:                            {ids[0]}, pqlCF(boolean) + ` = true`: {ids[1]},
		pqlCF(boolean) + ` IS NULL`: {}, `NOT ` + pqlCF(boolean) + ` = true`: {ids[0]},
		pqlCF(link) + ` ~ "EXAMPLE"`: {ids[0]}, pqlCF(link) + ` IS NULL`: {ids[1]},
		pqlCF(fixed) + ` = "Effective guidance"`: {ids[0], ids[1]}, pqlCF(fixed) + ` ~ "stale"`: {},
	})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[1], map[string]any{text: "  ", number: nil, date: nil})
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(text) + ` IS NULL`: {ids[1]}, pqlCF(text) + ` IS EMPTY`: {ids[1]},
		`NOT ` + pqlCF(text) + ` IS EMPTY`: {ids[0]}, pqlCF(text) + ` != "Beta"`: {ids[0]},
		pqlCF(number) + ` IS NULL`: {ids[1]}, `NOT ` + pqlCF(number) + ` >= 0`: {},
		pqlCF(date) + ` IS NULL`: {ids[1]}, pqlCF(boolean) + ` = false`: {ids[0], ids[1]},
	})
}

func TestAPIPQLPropertyMixedLogicFiltersBeforeCountAndPagination(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	number := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,CASE WHEN id=$2 THEN 2 ELSE 9 END)`, number, ids[0])
	f.exec(`UPDATE issues SET priority='urgent' WHERE id=$1`, ids[2])
	query := pqlCF(number) + ` >= 0 OR priority = "urgent"`
	seen := []string{}
	for pageIndex := 0; pageIndex < 3; pageIndex++ {
		page := readPropertyPage(t, f, repo, pqlPropertyOperation, f.owner, map[string]string{"pql": query, "per_page": "1", "cursor": fmt.Sprintf("1:%d:0", pageIndex)}, true)
		require.Equal(t, 3, page.Total)
		require.Len(t, page.Results, 1)
		seen = append(seen, page.Results[0].ID)
	}
	require.ElementsMatch(t, ids, seen)
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(number) + ` >= 8`: {ids[1]}, `NOT ` + pqlCF(number) + ` >= 8`: {ids[0]},
		`NOT (` + pqlCF(number) + ` >= 8 OR priority = "urgent")`:                     {ids[0]},
		`(` + pqlCF(number) + ` >= 8 OR priority = "urgent") AND title ~ "Discovery"`: {ids[1], ids[2]},
	})
}
