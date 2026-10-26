//go:build integration

package djangoapi

import (
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestCurrentIssueSerializationAndOrderingUseLiveLabels(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	for _, id := range []string{testsupport.LabelIssue, testsupport.LabelEmptyIssue} {
		var raw []byte
		if err := connection.QueryRow(ctx, "SELECT "+IssueRowSQL("i")+" FROM issues i WHERE i.id=$1", id).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		row := testsupport.LabelReadObject(t, raw)
		if id == testsupport.LabelIssue {
			testsupport.RequireLabelIDs(t, row["labels"], testsupport.LabelKept)
		} else {
			testsupport.RequireLabelIDs(t, row["labels"])
		}
	}
	details, err := apiIssueRelations(ctx, connection, testsupport.LabelIssue, "labels")
	if err != nil {
		t.Fatal(err)
	}
	labels := details.([]any)
	if len(labels) != 1 || labels[0].(map[string]any)["id"] != testsupport.LabelKept {
		t.Fatalf("label details=%v", details)
	}
	if _, err = connection.Exec(ctx, `INSERT INTO labels(id,project_id,workspace_id,name) VALUES('00000000-0000-4000-8000-000000000015',$1,$2,'More')`, testsupport.LabelProject, testsupport.LabelWorkspace); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, `INSERT INTO issue_labels(issue_id,label_id) VALUES($1,'00000000-0000-4000-8000-000000000015')`, testsupport.LabelIssue); err != nil {
		t.Fatal(err)
	}
	order, joins := IssueOrderSQL("labels__name")
	rows, err := connection.Query(ctx, "SELECT i.id::text FROM issues i"+joins+" ORDER BY "+order+",i.id")
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
	if want := []string{testsupport.LabelIssue, testsupport.LabelIssue, testsupport.LabelEmptyIssue}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("live label order/multiplicity=%v want=%v", ids, want)
	}
}
