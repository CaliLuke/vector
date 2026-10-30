//go:build integration

package intakeinboxreads

import (
	"slices"
	"testing"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestDraftListAppliesNativeRelationFilters(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	if _, err := connection.Exec(ctx, `
INSERT INTO cycles(id,workspace_id,project_id,name) VALUES('00000000-0000-4000-8000-000000000090','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Cycle');
INSERT INTO modules(id,workspace_id,project_id,name) VALUES('00000000-0000-4000-8000-000000000091','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','Module');
INSERT INTO draft_issue_assignees(draft_issue_id,assignee_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000003',now());
INSERT INTO draft_issue_cycles(draft_issue_id,cycle_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000090',now());
INSERT INTO draft_issue_modules(draft_issue_id,module_id,deleted_at) VALUES('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000091',now());
INSERT INTO draft_issue_assignees(draft_issue_id,assignee_id) VALUES('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000003');
INSERT INTO draft_issue_cycles(draft_issue_id,cycle_id) VALUES('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000090');
INSERT INTO draft_issue_modules(draft_issue_id,module_id) VALUES('00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000091');
INSERT INTO draft_issues(id,workspace_id,created_by_id,name,deleted_at) VALUES
 ('00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000003','Other workspace',NULL),
 ('00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000099','Other creator',NULL),
 ('00000000-0000-4000-8000-000000000022','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000003','Deleted',now());
INSERT INTO draft_issue_labels(draft_issue_id,label_id) SELECT id,'00000000-0000-4000-8000-000000000006' FROM draft_issues WHERE name IN ('Other workspace','Other creator','Deleted');
INSERT INTO draft_issue_assignees(draft_issue_id,assignee_id) SELECT id,'00000000-0000-4000-8000-000000000003' FROM draft_issues WHERE name IN ('Other workspace','Other creator','Deleted');
INSERT INTO draft_issue_cycles(draft_issue_id,cycle_id) SELECT id,'00000000-0000-4000-8000-000000000090' FROM draft_issues WHERE name IN ('Other workspace','Other creator','Deleted');
INSERT INTO draft_issue_modules(draft_issue_id,module_id) SELECT id,'00000000-0000-4000-8000-000000000091' FROM draft_issues WHERE name IN ('Other workspace','Other creator','Deleted');
`); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, value string
		want        []string
	}{
		{"labels", testsupport.LabelKept, []string{testsupport.LabelDraft}},
		{"labels", "None," + testsupport.LabelKept, []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft}},
		{"created_by", "None," + testsupport.LabelActor, []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"parent", "None,00000000-0000-4000-8000-000000000090", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"assignees", "None", []string{testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"assignees", "", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"assignees", "None,00000000-0000-4000-8000-000000000003", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"cycle", "None", []string{testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"cycle", "", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"cycle", "None,00000000-0000-4000-8000-000000000090", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"module", "None", []string{testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"module", "", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"module", "None,00000000-0000-4000-8000-000000000091", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"labels", testsupport.LabelDeleted, []string{}},
		{"labels", "None", []string{testsupport.LabelEmptyDraft}},
		{"labels", "", []string{testsupport.LabelDraft, testsupport.LabelEmptyDraft, testsupport.LabelProjectlessDraft}},
		{"assignees", testsupport.LabelActor, []string{testsupport.LabelDraft}},
		{"cycle", "00000000-0000-4000-8000-000000000090", []string{testsupport.LabelDraft}},
		{"module", "00000000-0000-4000-8000-000000000091", []string{testsupport.LabelDraft}},
	} {
		t.Run(test.name+"/"+test.value, func(t *testing.T) {
			raw, err := repository.DraftIssueList(ctx, uuid.MustParse(testsupport.LabelActor), "demo", ListOptions{Filters: map[string]string{test.name: test.value}, PresentFilters: map[string]bool{test.name: true}})
			if err != nil {
				t.Fatal(err)
			}
			result := testsupport.LabelReadObject(t, raw)
			ids := []string{}
			for _, row := range result["results"].([]any) {
				ids = append(ids, row.(map[string]any)["id"].(string))
			}
			slices.Sort(ids)
			if !slices.Equal(ids, test.want) {
				t.Fatalf("IDs=%v want=%v", ids, test.want)
			}
			if result["total_count"] != float64(len(test.want)) {
				t.Fatalf("total_count=%v", result["total_count"])
			}
		})
	}
}

func TestIntakePaginationSupportsOffsetsBeyondInt32(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	actor, project := uuid.MustParse(testsupport.LabelActor), uuid.MustParse(testsupport.LabelProject)
	options := ListOptions{PerPage: "1000", Cursor: "1000:2147484:0", Filters: map[string]string{"status": "0"}}
	draft, draftErr := repository.DraftIssueList(ctx, actor, "demo", options)
	intake, intakeErr := repository.IntakeIssueList(ctx, actor, "demo", project, options)
	for _, test := range []struct {
		name string
		raw  []byte
		err  error
	}{{"draft", draft, draftErr}, {"intake", intake, intakeErr}} {
		t.Run(test.name, func(t *testing.T) {
			if test.err != nil {
				t.Fatal(test.err)
			}
			result := testsupport.LabelReadObject(t, test.raw)
			if len(result["results"].([]any)) != 0 || result["next_cursor"] != "1000:2147485:0" || result["prev_cursor"] != "1000:2147483:1" {
				t.Fatalf("page=%v", result)
			}
		})
	}
}
