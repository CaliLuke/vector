//go:build integration

package workitems

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIPQLMemberCollectionsUseReadableRosterIdentities(t *testing.T) {
	f := newPQLIdentityFixture(t)
	f.examples(t, map[string][]int{
		`assignee = "40000000-0000-4000-8000-000000000001"`:                                           {1},
		`assignee = "40000000-0000-4000-8000-000000000006"`:                                           {1},
		`assignee = "40000000-0000-4000-8000-000000000002"`:                                           {2},
		`assignee != "40000000-0000-4000-8000-000000000001"`:                                          {2, 3, 4},
		`assignee IN ("40000000-0000-4000-8000-000000000001","40000000-0000-4000-8000-000000000002")`: {1, 2},
		`assignee IS NULL`: {3, 4}, `assignee IS EMPTY`: {3, 4}, `hasNoAssignee()`: {3, 4},
		`assignee IS NOT NULL`: {1, 2}, `assignee IS NOT EMPTY`: {1, 2}, `NOT hasNoAssignee()`: {1, 2},
		`assignee IN workspaceMembers()`: {1, 2}, `assignee NOT IN workspaceMembers()`: {3, 4},
		`assignee IN membersOf("project:00000000-0000-4000-8000-000000000002")`:     {1},
		`assignee IN membersOf("project:00000000-0000-4000-8000-000000000099")`:     {2},
		`assignee NOT IN membersOf("project:00000000-0000-4000-8000-000000000099")`: {1, 3, 4},
		`assignee IN membersOf("project:00000000-0000-4000-8000-000000000088")`:     {},
		`assignee NOT IN membersOf("project:00000000-0000-4000-8000-000000000088")`: {1, 2, 3, 4},
		`assignee = currentUser()`: {}, `subscriber = currentUser()`: {1},
		`subscriber IN (currentUser(),"40000000-0000-4000-8000-000000000002")`: {1, 2},
		`subscriber NOT IN (currentUser())`:                                    {2, 3, 4}, `subscriber IS NULL`: {3, 4},
		`mention = "40000000-0000-4000-8000-000000000001"`: {2},
		`mention IN workspaceMembers()`:                    {1, 2}, `mention IS EMPTY`: {3, 4},
		`mention = currentUser()`: {}, `mention NOT IN membersOf("project:00000000-0000-4000-8000-000000000002")`: {1, 3, 4},
		`createdBy IN workspaceMembers()`: {1, 2, 3}, `createdBy NOT IN workspaceMembers()`: {},
		`createdBy IN membersOf("project:00000000-0000-4000-8000-000000000099")`:     {1, 2, 3},
		`createdBy IN membersOf("project:ffffffff-ffff-4fff-8fff-ffffffffffff")`:     {},
		`createdBy NOT IN membersOf("project:ffffffff-ffff-4fff-8fff-ffffffffffff")`: {1, 2, 3},
		`NOT createdBy IN membersOf("project:ffffffff-ffff-4fff-8fff-ffffffffffff")`: {1, 2, 3},
	})
}

func TestAPIPQLMemberListsRecheckDirectoryAndProjectPermissions(t *testing.T) {
	f := newPQLIdentityFixture(t)
	_, err := f.db.Exec(f.ctx, `UPDATE workspace_members SET role=5 WHERE member_id='00000000-0000-4000-8000-000000000003'; UPDATE project_members SET role=5 WHERE project_id='00000000-0000-4000-8000-000000000002' AND member_id='00000000-0000-4000-8000-000000000003'`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{
		`assignee IS NOT NULL`: {1}, `hasNoAssignee()`: {2, 3},
		`assignee IN workspaceMembers()`: {}, `assignee NOT IN workspaceMembers()`: {1, 2, 3},
		`createdBy NOT IN workspaceMembers()`:                                   {1, 2, 3},
		`assignee IN membersOf("project:00000000-0000-4000-8000-000000000002")`: {1},
		`mention IS NOT NULL`:                                                   {2}, `subscriber = currentUser()`: {1},
	})
	_, err = f.db.Exec(f.ctx, `UPDATE project_members SET is_active=false WHERE project_id='00000000-0000-4000-8000-000000000099' AND member_id='00000000-0000-4000-8000-000000000003'`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{
		`createdBy IN membersOf("project:00000000-0000-4000-8000-000000000099")`:     {},
		`createdBy NOT IN membersOf("project:00000000-0000-4000-8000-000000000099")`: {1, 2, 3},
	})
}

func TestAPIPQLMemberCollectionsDropDeletedIdentities(t *testing.T) {
	f := newPQLIdentityFixture(t)
	_, err := f.db.Exec(f.ctx, `UPDATE project_members SET deleted_at=now() WHERE member_id IN('40000000-0000-4000-8000-000000000001','40000000-0000-4000-8000-000000000006'); UPDATE workspace_members SET deleted_at=now() WHERE member_id='40000000-0000-4000-8000-000000000001'`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`assignee IS NOT NULL`: {2}, `hasNoAssignee()`: {1, 3, 4}, `mention IS NOT NULL`: {1}, `NOT assignee = "40000000-0000-4000-8000-000000000001"`: {1, 2, 3, 4}})
}
