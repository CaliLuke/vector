package issuequery_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/stretchr/testify/require"
)

func TestPQLIdentityExpressionsBindAllCallerValues(t *testing.T) {
	args := []any{"scope"}
	sql, err := issuequery.BuildPQL(`childOf("PROJ-00012") AND assignee = currentUser() AND createdBy IN membersOf("project:00000000-0000-4000-8000-000000000002")`, issuequery.PQLContext{ActorID: pqlActor}, &args)
	require.NoError(t, err)
	require.NotContains(t, sql, "PROJ")
	require.NotContains(t, sql, pqlActor)
	require.Contains(t, args, "PROJ")
	require.Contains(t, args, "00012")
	require.Contains(t, args, pqlActor)
	require.Contains(t, args, "00000000-0000-4000-8000-000000000002")
}

func TestPQLIdentityExpressionsRejectInvalidCallsWithoutChangingArguments(t *testing.T) {
	for _, query := range []string{
		`childOf()`, `parentOf("A-1","B-2")`, `linkedTo(1)`, `blockedBy(currentUser())`,
		`blocks("secret")`, `duplicateOf("secret-")`, `childOf("-1")`, `childOf("A-+1")`,
		`childOf("A-١")`, `parentOf("A B-1")`, `childOf("A\u0000-1")`, `childOf("A\u00a0B-1")`,
		`isTopLevel(1)`, `isSubWorkItem(1)`, `hasChildren(1)`, `hasNoAssignee(1)`,
		`assignee > currentUser()`, `mention ~ "secret"`, `subscriber BETWEEN "secret" AND "secret"`,
		`assignee = currentUser(1)`, `subscriber = "secret"`, `mention IN activeCycle()`,
		`createdBy IN workspaceMembers(1)`, `assignee IN membersOf()`, `assignee IN membersOf(1)`,
		`assignee IN membersOf("project:secret")`, `subscriber IN membersOf("teamspace:00000000-0000-4000-8000-000000000002")`,
		`createdBy IN membersOf("Project:00000000-0000-4000-8000-000000000002")`,
		`label IN workspaceMembers()`, `assignee IN (workspaceMembers())`, `createdBy = workspaceMembers()`,
		`assignee IN membersOf("project:00000000-0000-4000-8000-000000000002",1)`,
		`title = childOf("A-1")`, `text > "secret"`,
	} {
		t.Run(query, func(t *testing.T) {
			args := []any{"scope"}
			_, err := issuequery.BuildPQL(`title="bound" AND (`+query+`)`, issuequery.PQLContext{ActorID: pqlActor}, &args)
			var invalid *issuequery.FilterError
			require.ErrorAs(t, err, &invalid)
			require.NotContains(t, err.Error(), "secret")
			require.Equal(t, []any{"scope"}, args)
		})
	}
}

func TestPQLActorDependentBranchesRequireActorEvenBehindOr(t *testing.T) {
	for _, query := range []string{`isTopLevel()`, `isSubWorkItem()`, `hasChildren()`, `childOf("A-1")`, `assignee IS NULL`, `mention IS EMPTY`, `subscriber IS NOT NULL`, `createdBy IN workspaceMembers()`} {
		t.Run(query, func(t *testing.T) {
			args := []any{}
			_, err := issuequery.BuildPQL(`title="anything" OR `+query, issuequery.PQLContext{}, &args)
			require.Error(t, err)
			require.Empty(t, args)
		})
	}
}

func TestPQLItemIdentifierAllowsLargeDecimalSequenceAsBoundData(t *testing.T) {
	args := []any{}
	_, err := issuequery.BuildPQL(`childOf("A-`+strings.Repeat("9", 100)+`")`, issuequery.PQLContext{ActorID: pqlActor}, &args)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("9", 100), args[1])
}
