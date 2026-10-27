package issuequery_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/stretchr/testify/require"
)

const pqlPropertyID = "61000000-0000-4000-8000-000000000001"
const pqlPropertyItem = "62000000-0000-4000-8000-000000000001"

func propertyResolver(kind issuequery.PQLPropertyKind) issuequery.PQLPropertyResolver {
	return func(id string) (issuequery.PQLPropertyDefinition, bool, error) {
		return issuequery.PQLPropertyDefinition{Kind: kind, ValidateSelections: func([]string) (bool, error) { return true, nil }}, id == pqlPropertyID, nil
	}
}

func TestPQLPropertyPreparationChecksEveryBranchAndTypedOperand(t *testing.T) {
	request := issuequery.PQLContext{Now: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		kind           issuequery.PQLPropertyKind
		valid, invalid []string
	}{
		{issuequery.PQLPropertyText, []string{`= "x"`, `!= "x"`, `~ "%_"`, `IS NULL`, `IS NOT EMPTY`}, []string{`IN ("x")`, `> "x"`, `= currentUser()`}},
		{issuequery.PQLPropertyNumber, []string{`= 0`, `!= -1.25`, `BETWEEN -3 AND 4.5`, `IS NULL`}, []string{`= "1"`, `= true`, `= 1..2`, `IN (1,2)`, `~ "1"`, `IS EMPTY`}},
		{issuequery.PQLPropertyDate, []string{`>= today()`, `BETWEEN daysAgo(3) AND now()`, `IS NOT NULL`}, []string{`= 2026`, `IN ("2026-10-27")`, `IS EMPTY`}},
		{issuequery.PQLPropertyBoolean, []string{`= false`, `= true`, `IS NULL`}, []string{`!= false`, `= "true"`, `IN (true)`, `IS EMPTY`}},
		{issuequery.PQLPropertySelection, []string{`= "` + pqlPropertyItem + `"`, `NOT IN ("` + pqlPropertyItem + `")`, `IS NULL`}, []string{`= "secret"`, `= currentUser()`, `IN workspaceMembers()`, `IS EMPTY`}},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			for _, suffix := range tc.valid {
				_, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] `+suffix, request, propertyResolver(tc.kind))
				require.NoError(t, err, suffix)
			}
			for _, suffix := range tc.invalid {
				_, err := issuequery.PreparePQL(`priority = "high" OR cf["`+pqlPropertyID+`"] `+suffix, request, propertyResolver(tc.kind))
				require.Error(t, err, suffix)
				require.NotContains(t, err.Error(), "secret")
			}
		})
	}
	_, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] IS NULL OR unknown = 1`, request, propertyResolver(issuequery.PQLPropertyText))
	require.Error(t, err)
	_, err = issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] IS NULL`, request, propertyResolver("Formula"))
	require.Error(t, err)
}

func TestPQLPropertySQLCopiesMaterializationAndBindsValues(t *testing.T) {
	const literal = "secret'; DROP TABLE issues;--"
	resolved := 0
	resolve := func(id string) (issuequery.PQLPropertyDefinition, bool, error) {
		resolved++
		return propertyResolver(issuequery.PQLPropertyText)(id)
	}
	plan, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] = "`+literal+`" OR cf["`+pqlPropertyID+`"] IS NULL`, issuequery.PQLContext{}, resolve)
	require.NoError(t, err)
	require.Equal(t, 1, resolved)
	ids := plan.PropertyIDs()
	require.Equal(t, []string{pqlPropertyID}, ids)
	ids[0] = "changed"
	values := map[string][]issuequery.PQLPropertyValue{pqlPropertyID: {{ItemID: pqlPropertyItem, Value: literal}}}
	args := []any{"existing"}
	sql, err := plan.SQL(values, &args)
	require.NoError(t, err)
	require.Equal(t, 1, resolved, "SQL must not retain or reexecute a snapshot resolver")
	require.NotContains(t, sql, literal)
	require.NotContains(t, sql, pqlPropertyID)
	require.NotContains(t, sql, pqlPropertyItem)
	require.Equal(t, "existing", args[0])
	require.Len(t, args, 3, "one projection binding is shared by both property predicates")
	values[pqlPropertyID][0].Value = "changed"
	require.Contains(t, args[1], literal)
	require.Equal(t, literal, args[2])
}

func TestPQLPropertySQLRejectsInvalidMaterializationAtomically(t *testing.T) {
	plan, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] >= 2`, issuequery.PQLContext{}, propertyResolver(issuequery.PQLPropertyNumber))
	require.NoError(t, err)
	for _, rows := range [][]issuequery.PQLPropertyValue{
		{{ItemID: "not-an-id", Value: float64(2)}},
		{{ItemID: pqlPropertyItem, Value: float64(2)}, {ItemID: strings.ToUpper(pqlPropertyItem), Value: nil}},
		{{ItemID: pqlPropertyItem, Value: "2"}},
		{{ItemID: pqlPropertyItem, Value: math.Inf(1)}},
		{{ItemID: pqlPropertyItem, Value: math.NaN()}},
	} {
		backing := []any{"existing", "untouched"}
		args := backing[:1]
		_, err = plan.SQL(map[string][]issuequery.PQLPropertyValue{pqlPropertyID: rows}, &args)
		require.Error(t, err)
		require.Equal(t, []any{"existing"}, args)
		require.Equal(t, "untouched", backing[1])
	}
	args := []any{}
	_, err = plan.SQL(nil, &args)
	require.Error(t, err)
	_, err = plan.SQL(map[string][]issuequery.PQLPropertyValue{pqlPropertyID: nil}, &args)
	require.NoError(t, err, "an empty authorized candidate set must still compile")
}

func TestPQLPropertyResolutionPreservesOperationalErrors(t *testing.T) {
	broken := errors.New("database unavailable")
	query := `cf["` + pqlPropertyID + `"] IS NULL`
	_, err := issuequery.PreparePQL(query, issuequery.PQLContext{}, func(string) (issuequery.PQLPropertyDefinition, bool, error) {
		return issuequery.PQLPropertyDefinition{}, false, broken
	})
	require.ErrorIs(t, err, broken)
	args := []any{}
	_, err = issuequery.BuildPQL(query, issuequery.PQLContext{}, &args)
	require.Error(t, err, "a custom property cannot execute without a resolver")
	_, err = issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] = "`+pqlPropertyItem+`"`, issuequery.PQLContext{}, func(string) (issuequery.PQLPropertyDefinition, bool, error) {
		return issuequery.PQLPropertyDefinition{Kind: issuequery.PQLPropertySelection, ValidateSelections: func([]string) (bool, error) { return false, broken }}, true, nil
	})
	require.ErrorIs(t, err, broken)
}

func TestPQLPropertySQLRequiresNormalizedValueTypes(t *testing.T) {
	for _, tc := range []struct {
		kind  issuequery.PQLPropertyKind
		value any
	}{
		{issuequery.PQLPropertyText, 12},
		{issuequery.PQLPropertyText, "invalid\x00text"},
		{issuequery.PQLPropertyText, string([]byte{0xff})},
		{issuequery.PQLPropertyDate, "2026-02-30"},
		{issuequery.PQLPropertyDate, "2026-10-27T00:00:00Z"},
		{issuequery.PQLPropertyBoolean, "false"},
		{issuequery.PQLPropertySelection, []string{"not-an-id"}},
		{issuequery.PQLPropertySelection, []any{pqlPropertyItem}},
	} {
		plan, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] IS NULL`, issuequery.PQLContext{}, propertyResolver(tc.kind))
		require.NoError(t, err)
		args := []any{"existing"}
		_, err = plan.SQL(map[string][]issuequery.PQLPropertyValue{pqlPropertyID: {{ItemID: pqlPropertyItem, Value: tc.value}}}, &args)
		require.Error(t, err, "%s %v", tc.kind, tc.value)
		require.Equal(t, []any{"existing"}, args)
	}
}

func TestPQLPropertyDetectionUsesSyntax(t *testing.T) {
	uses, err := issuequery.PQLUsesProperties(`title = "cf[\"x\"]"`)
	require.NoError(t, err)
	require.False(t, uses)
	uses, err = issuequery.PQLUsesProperties(`NOT cf["` + pqlPropertyID + `"] IS NULL`)
	require.NoError(t, err)
	require.True(t, uses)
	_, err = issuequery.PQLUsesProperties(`cf["x"`)
	require.Error(t, err)
}
