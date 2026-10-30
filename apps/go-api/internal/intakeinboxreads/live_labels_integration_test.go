//go:build integration

package intakeinboxreads

import (
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestIntakeAndDraftLabelReadsUseNativeOwnerScope(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	const removedIssue = "00000000-0000-4000-8000-000000000016"
	const removedDraft = "00000000-0000-4000-8000-000000000017"
	if _, err := connection.Exec(ctx, `
INSERT INTO issues(id,workspace_id,project_id,name) SELECT '00000000-0000-4000-8000-000000000016',workspace_id,project_id,'Removed link only' FROM issues WHERE id='00000000-0000-4000-8000-000000000004';
INSERT INTO draft_issues(id,workspace_id,project_id,name) SELECT '00000000-0000-4000-8000-000000000017',workspace_id,project_id,'Removed draft link only' FROM draft_issues WHERE id='00000000-0000-4000-8000-000000000011';
INSERT INTO issue_labels(issue_id,label_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000016','00000000-0000-4000-8000-000000000008',now());
INSERT INTO draft_issue_labels(draft_issue_id,label_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000017','00000000-0000-4000-8000-000000000008',now());
`); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, query, id string
		args            []any
		labels          []string
	}{
		{"issue lite", "SELECT " + issueLiteProjection + " FROM issues issue WHERE issue.id=$1", testsupport.LabelIssue, []any{testsupport.LabelIssue}, []string{testsupport.LabelKept}},
		{"empty issue lite", "SELECT " + issueLiteProjection + " FROM issues issue WHERE issue.id=$1", testsupport.LabelEmptyIssue, []any{testsupport.LabelEmptyIssue}, []string{}},
		{"draft", "SELECT " + draftIssueProjection + " FROM draft_issues draft WHERE draft.id=$2", testsupport.LabelDraft, []any{testsupport.LabelActor, testsupport.LabelDraft}, []string{testsupport.LabelKept}},
		{"empty draft", "SELECT " + draftIssueProjection + " FROM draft_issues draft WHERE draft.id=$2", testsupport.LabelEmptyDraft, []any{testsupport.LabelActor, testsupport.LabelEmptyDraft}, []string{}},
		{"projectless draft", "SELECT " + draftIssueProjection + " FROM draft_issues draft WHERE draft.id=$2", testsupport.LabelProjectlessDraft, []any{testsupport.LabelActor, testsupport.LabelProjectlessDraft}, []string{testsupport.LabelProjectless}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var raw []byte
			if err := connection.QueryRow(ctx, testCase.query, testCase.args...).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			row := testsupport.LabelReadObject(t, raw)
			if row["id"] != testCase.id {
				t.Fatalf("projection ID=%v", row["id"])
			}
			testsupport.RequireLabelIDs(t, row["label_ids"], testCase.labels...)
		})
	}
	t.Run("intake detail", func(t *testing.T) {
		var raw []byte
		if err := connection.QueryRow(ctx, intakeIssueDetailSQL, testsupport.LabelIntake, testsupport.LabelProject, testsupport.LabelIssue, testsupport.LabelActor).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		row := testsupport.LabelReadObject(t, raw)
		issue := row["issue"].(map[string]any)
		if issue["id"] != testsupport.LabelIssue {
			t.Fatalf("intake issue=%v", issue)
		}
		testsupport.RequireLabelIDs(t, issue["label_ids"], testsupport.LabelKept)
	})
	for _, draft := range []bool{false, true} {
		table, alias, bridge, key := "issues", "issue", "issue_labels", "issue_id"
		none, mixed := []string{testsupport.LabelEmptyIssue, removedIssue}, testsupport.LabelIssue
		if draft {
			table, alias, bridge, key = "draft_issues", "draft", "draft_issue_labels", "draft_issue_id"
			none, mixed = []string{testsupport.LabelEmptyDraft, removedDraft}, testsupport.LabelDraft
		}
		all := []string{testsupport.LabelIssue, testsupport.LabelEmptyIssue, removedIssue}
		if draft {
			all = []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft, removedDraft}
		}
		filters := []struct {
			value string
			ids   []string
		}{{"", all}, {"None", none}, {"None," + testsupport.LabelKept, slices.Concat([]string{mixed}, none)}, {testsupport.LabelKept, []string{mixed}}, {testsupport.LabelDeleted, []string{}}, {"00000000-0000-4000-8000-000000000009", []string{}}, {"00000000-0000-4000-8000-000000000010", []string{}}}
		for _, filter := range filters {
			t.Run(table+" filter "+filter.value, func(t *testing.T) {
				builder := newWhereBuilder()
				builder.conditions = []string{"true"}
				addRelationUUIDFilter(builder, alias, bridge, key, "label_id", filter.value, true, true, true)
				rows, err := connection.Query(ctx, "SELECT "+alias+".id::text FROM "+table+" "+alias+" WHERE "+strings.Join(builder.conditions, " AND ")+" ORDER BY 1", builder.args...)
				if err != nil {
					t.Fatal(err)
				}
				defer rows.Close()
				ids := []string{}
				for rows.Next() {
					var id string
					if err = rows.Scan(&id); err != nil {
						t.Fatal(err)
					}
					ids = append(ids, id)
				}
				if err = rows.Err(); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(ids, filter.ids) {
					t.Fatalf("%s %s IDs=%v want=%v", table, filter.value, ids, filter.ids)
				}
			})
		}
	}
}
