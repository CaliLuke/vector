//go:build integration

package issuequery_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

// Exercise public compiler output as booleans, independently of root-item
// authorization: earlier admission must not substitute for roster guards.
func TestPQLRosterPredicatesRecheckActorAdmissionInTheDataStatement(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+`
ALTER TABLE issue_assignees ADD COLUMN project_id uuid, ADD COLUMN workspace_id uuid;
INSERT INTO issue_assignees(issue_id,assignee_id,project_id,workspace_id)
SELECT id,created_by_id,project_id,workspace_id FROM issues WHERE sequence_id=1;
`)
	for _, tc := range []struct {
		name, mutation string
		want           []bool
	}{
		{"active", `SELECT 1`, []bool{true, true, true}},
		{"inactive actor", `UPDATE users SET is_active=false WHERE id='00000000-0000-4000-8000-000000000003'`, []bool{false, false, false}},
		{"inactive workspace membership", `UPDATE workspace_members SET is_active=false WHERE member_id='00000000-0000-4000-8000-000000000003'`, []bool{false, false, false}},
		{"deleted workspace membership", `UPDATE workspace_members SET deleted_at=now() WHERE member_id='00000000-0000-4000-8000-000000000003'`, []bool{false, false, false}},
		{"unknown workspace role", `UPDATE workspace_members SET role=0 WHERE member_id='00000000-0000-4000-8000-000000000003'`, []bool{false, false, false}},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=now() WHERE slug='demo'`, []bool{false, false, false}},
		{"guest project roster", `UPDATE workspace_members SET role=5 WHERE member_id='00000000-0000-4000-8000-000000000003'`, []bool{true, false, true}},
		{"revoked project with directory fallback", `UPDATE project_members SET deleted_at=now() WHERE project_id='00000000-0000-4000-8000-000000000002' AND member_id='00000000-0000-4000-8000-000000000003'`, []bool{true, true, false}},
		{"revoked project without directory", `UPDATE workspace_members SET role=5 WHERE member_id='00000000-0000-4000-8000-000000000003'; UPDATE project_members SET is_active=false WHERE project_id='00000000-0000-4000-8000-000000000002' AND member_id='00000000-0000-4000-8000-000000000003'`, []bool{false, false, false}},
		{"deleted project without directory", `UPDATE workspace_members SET role=5 WHERE member_id='00000000-0000-4000-8000-000000000003'; UPDATE projects SET deleted_at=now() WHERE id='00000000-0000-4000-8000-000000000002'`, []bool{false, false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(ctx)) }()
			_, err = tx.Exec(ctx, tc.mutation)
			require.NoError(t, err)
			args := []any{testsupport.LabelIssue}
			predicates := []string{}
			for _, query := range []string{`assignee IS NOT NULL`, `createdBy IN workspaceMembers()`, `createdBy IN membersOf("project:00000000-0000-4000-8000-000000000002")`} {
				predicate, buildErr := issuequery.BuildPQL(query, issuequery.PQLContext{ActorID: testsupport.LabelActor}, &args)
				require.NoError(t, buildErr)
				predicates = append(predicates, predicate)
			}
			got := make([]bool, 3)
			err = tx.QueryRow(ctx, `SELECT `+strings.Join(predicates, ",")+` FROM issues issue WHERE issue.id=$1`, args...).Scan(&got[0], &got[1], &got[2])
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
