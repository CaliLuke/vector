package workitems

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func TestAPIIssueActivitySnapshotPreservesSourceSerializerOrder(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(issueSourceFixtureDir, "issue-core-v1-issues-detail-patch.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Effects []struct {
			Name    string `json:"name"`
			Payload struct {
				Kwargs struct {
					Current string `json:"current_instance"`
				} `json:"kwargs"`
			} `json:"payload"`
		} `json:"effects"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, effect := range fixture.Effects {
		if effect.Name != "plane.bgtasks.issue_activities_task.issue_activity" {
			continue
		}
		var projection map[string]any
		decoder := json.NewDecoder(strings.NewReader(effect.Payload.Kwargs.Current))
		decoder.UseNumber()
		if err := decoder.Decode(&projection); err != nil {
			t.Fatal(err)
		}
		got, err := orderedDump(projection, apiIssueOrder)
		if err != nil || got != effect.Payload.Kwargs.Current {
			t.Fatalf("activity snapshot=%s\nsource=%s\nerr=%v", got, effect.Payload.Kwargs.Current, err)
		}
		return
	}
	t.Fatal("source fixture has no issue activity snapshot")
}

func TestFormActivityPreservesFieldOrderAndLastRepeatedValue(t *testing.T) {
	input, err := djangoinput.Parse(strings.NewReader("name=Example&label_ids=first&assignee_ids=member&label_ids=last"), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	got, err := dumpRequest(input)
	if err != nil || got != `{"name": "Example", "label_ids": "last", "assignee_ids": "member"}` {
		t.Fatalf("activity=%s err=%v", got, err)
	}
}

func TestIntakeActivitySnapshotsPreserveNestedSerializerOrder(t *testing.T) {
	for _, name := range []string{"lifecycle-intake-patch-accept", "lifecycle-closure-api-intake-nullable-metadata"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(issueSourceFixtureDir, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Effects []struct {
					Payload struct {
						Kwargs struct {
							Type    string `json:"type"`
							Current string `json:"current_instance"`
						} `json:"kwargs"`
					} `json:"payload"`
				} `json:"effects"`
			}
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			for _, effect := range fixture.Effects {
				if effect.Payload.Kwargs.Type != "intake.activity.created" {
					continue
				}
				var projection map[string]any
				d := json.NewDecoder(strings.NewReader(effect.Payload.Kwargs.Current))
				d.UseNumber()
				if err := d.Decode(&projection); err != nil {
					t.Fatal(err)
				}
				got, err := dumpIntakeSnapshot(projection, strings.Contains(name, "api-intake"))
				if err != nil || got != effect.Payload.Kwargs.Current {
					t.Fatalf("got=%s\nsource=%s\nerr=%v", got, effect.Payload.Kwargs.Current, err)
				}
				return
			}
			t.Fatal("fixture has no intake activity")
		})
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
