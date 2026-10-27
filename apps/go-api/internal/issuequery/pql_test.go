package issuequery_test

import (
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/stretchr/testify/require"
)

const pqlActor = "00000000-0000-4000-8000-000000000003"

func TestPQLBindsTypedValuesAndEscapesWildcards(t *testing.T) {
	args := []any{"scope"}
	sql, err := issuequery.BuildPQL(`priority IN ("urgent", "high") AND title ~ "50%_\\'; DROP TABLE issues;--" AND createdBy = currentUser()`, issuequery.PQLContext{ActorID: pqlActor}, &args)
	require.NoError(t, err)
	require.NotContains(t, sql, "DROP")
	require.NotContains(t, sql, pqlActor)
	require.Contains(t, sql, "$2::text")
	require.Equal(t, []any{"scope", "urgent", "high", `%50\%\_\\'; DROP TABLE issues;--%`, pqlActor}, args)
}

func TestPQLSupportedScalarForms(t *testing.T) {
	for _, query := range []string{
		`title = "雪"`, `id ~ "WEB"`, `title != "x"`, `title IS EMPTY`, `title IS NOT EMPTY`,
		`priority NOT IN ("none", "low")`, `createdBy IS NULL`, `project IS NOT NULL`,
		`isDraft() OR isArchived()`, `isDraft = false AND NOT isArchived = true`,
		`dueDate BETWEEN "2026-01-01" AND "2026-02-01"`, `dueDate BETWEEN ("2026-01-01", "2026-02-01")`,
		`(title = "a" or title = "b") aNd nOt (title = "c")`,
		`title="a" OR title="b" OR title="c" OR title="d" OR title="e"`,
	} {
		t.Run(query, func(t *testing.T) {
			args := []any{}
			sql, err := issuequery.BuildPQL(query, issuequery.PQLContext{}, &args)
			require.NoError(t, err)
			require.NotEmpty(t, sql)
		})
	}
	args := []any{}
	_, err := issuequery.BuildPQL(`id ~ "WEB"`, issuequery.PQLContext{}, &args)
	require.NoError(t, err)
	require.Equal(t, []any{"WEB%"}, args)
}

func TestPQLRejectsInvalidOrUnavailableExpressionsAtomically(t *testing.T) {
	for _, query := range []string{
		`title = "secret" OR milestone = "secret"`, `text ~ 42`, `state IN openStates()`,
		`cf["secret"] = 42`, `cf[42] = "secret"`, `cf["secret" = 42`,
		`title =`, `title = "secret`, `title = "\q"`, `title='secret'`, `title = "secret";`,
		`title = "secret" trailing`, `title = "secret" -- comment`, `title = "secret" AND`,
		`(title="secret"`, `title IN ()`, `title IN ("secret",)`, `title IN ("secret" "other")`,
		`title IN "secret"`, `title IN (("secret"))`, `title ! "secret"`, `title HAS "secret"`,
		`title IS SOMETHING`, `priority ~ "high"`, `priority = "secret"`, `priority = 4`,
		`isDraft = "true"`, `isDraft IN (true)`, `isDraft > true`, `isDraft(1)`, `title()`,
		`project = "secret"`, `project = currentUser()`, `createdBy = currentUser(1)`,
		`createdBy = unknown()`, `createdBy = true`, `createdBy = NULL`, `Title = "secret"`,
		`dueDate = "2026-02-30"`, `dueDate = "2026-01-01T12:00:00Z"`, `dueDate ~ "2026"`,
		`dueDate = "0000-01-01"`, `createdAt = "0000-01-01T00:00:00Z"`,
		`dueDate IN ("2026-01-01")`, `dueDate = 42`, `dueDate = today()-7`,
		`dueDate BETWEEN "2026-01-01"`, `dueDate BETWEEN ("2026-01-01" AND "2026-02-01")`,
		`dueDate BETWEEN ("2026-01-01",)`, `dueDate BETWEEN ("2026-01-01", "2026-02-01"`,
		`createdAt = unknown()`, `createdAt = today(1)`, `createdAt = daysAgo()`,
		`createdAt = daysAgo(-1)`, `createdAt = daysAgo(1.5)`, `createdAt = daysAgo(10001)`,
		`createdAt = daysAgo("1")`, `createdAt = daysAgo(1,2)`, `createdAt = now(1)`,
		`title = "\u0000"`, `wasEver()`, `true`, `= "secret"`,
		strings.Repeat(`title="secret" OR `, 5) + `title="secret"`,
		strings.Repeat("NOT ", 64) + `title="secret"`,
		strings.Repeat("(", 64) + `title="secret"` + strings.Repeat(")", 64),
		`title = ` + strings.Repeat("f(", 64) + `"secret"` + strings.Repeat(")", 64),
		`title IN (` + strings.Repeat(`"secret",`, 256) + `"secret")`,
		strings.Repeat(" ", 16385), string([]byte{0xff}),
	} {
		t.Run(query[:min(len(query), 90)], func(t *testing.T) {
			backing := []any{"scope", "sentinel", "sentinel"}
			args := backing[:1]
			_, err := issuequery.BuildPQL(query, issuequery.PQLContext{Now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, &args)
			var invalid *issuequery.FilterError
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, "invalid_pql", invalid.Code)
			require.NotContains(t, err.Error(), "secret")
			require.Equal(t, []any{"scope"}, args)
			require.Equal(t, []any{"scope", "sentinel", "sentinel"}, backing)
		})
	}
}

func TestPQLBlankQueryPreservesLegacyArgumentsAndTimezoneBehavior(t *testing.T) {
	args := []any{"scope"}
	sql, err := issuequery.BuildPQL(" \t\n", issuequery.PQLContext{Timezone: "invalid"}, &args)
	require.NoError(t, err)
	require.Empty(t, sql)
	require.Equal(t, []any{"scope"}, args)
	_, err = issuequery.BuildPQL(`title = "x"`, issuequery.PQLContext{Timezone: "invalid"}, &args)
	require.Error(t, err)
	_, err = issuequery.BuildPQL(`createdAt = now()`, issuequery.PQLContext{}, &args)
	require.Error(t, err)
	_, err = issuequery.BuildPQL(`createdBy = currentUser()`, issuequery.PQLContext{}, &args)
	require.Error(t, err)
}
