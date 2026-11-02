package workitems

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func TestFormActivityPreservesFieldOrderAndLastRepeatedValue(t *testing.T) {
	input, err := requestinput.Parse(strings.NewReader("name=Example&label_ids=first&assignee_ids=member&label_ids=last"), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	got, err := dumpRequest(input)
	if err != nil || got != `{"name": "Example", "label_ids": "last", "assignee_ids": "member"}` {
		t.Fatalf("activity=%s err=%v", got, err)
	}
}

func TestIssueModuleAnnotationsOmitDeletedRelationsAndModules(t *testing.T) {
	ordinary := issueAnnotationsSQL("$1", false)
	collection := issueCollectionAnnotationsSQL("$1")
	for name, sql := range map[string]string{"ordinary": ordinary, "collection": collection} {
		if !strings.Contains(sql, "WHERE m.issue_id=i.id") {
			t.Errorf("%s module annotation lost its issue relation: %s", name, sql)
		}
		if !strings.Contains(sql, "m.deleted_at IS NULL AND md.deleted_at IS NULL") {
			t.Errorf("%s module annotation must omit deleted bridges and modules: %s", name, sql)
		}
	}
	if !strings.Contains(ordinary, "md.deleted_at IS NULL AND md.archived_at IS NULL") {
		t.Fatalf("ordinary issue annotation must keep its existing live, unarchived module policy: %s", ordinary)
	}
	if strings.Contains(collection, "md.archived_at IS NULL") {
		t.Fatalf("public collection annotation must continue including archived modules: %s", collection)
	}
	if strings.Contains(collection, "pm.is_active") {
		t.Fatalf("public collection annotation unexpectedly added active-assignee filtering: %s", collection)
	}
}
