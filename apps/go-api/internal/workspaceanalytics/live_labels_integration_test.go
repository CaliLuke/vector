//go:build integration

package workspaceanalytics

import (
	"encoding/json"
	"testing"
	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestChartLabelsUseCanonicalLiveRelations(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz);
CREATE TABLE project_members(project_id uuid,member_id uuid,is_active boolean,deleted_at timestamptz);
CREATE TABLE states(id uuid PRIMARY KEY,"group" text);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,state_id uuid,priority text,created_at timestamptz,is_draft bool,archived_at timestamptz,deleted_at timestamptz);
CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,name text,deleted_at timestamptz);
CREATE TABLE issue_labels(issue_id uuid,label_id uuid,deleted_at timestamptz);
CREATE TABLE cycle_issues(issue_id uuid,cycle_id uuid,deleted_at timestamptz);
CREATE TABLE module_issues(issue_id uuid,module_id uuid,deleted_at timestamptz);
INSERT INTO workspaces VALUES('00000000-0000-4000-8000-000000000001','demo',NULL);
INSERT INTO projects VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',NULL,NULL);
INSERT INTO project_members VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003',true,NULL);
INSERT INTO issues VALUES('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',NULL,'low',now(),false,NULL,NULL);
INSERT INTO labels VALUES
 ('00000000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Kept',NULL),
 ('00000000-0000-4000-8000-000000000007','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Deleted',now()),
 ('00000000-0000-4000-8000-000000000008','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Removed',NULL),
 ('00000000-0000-4000-8000-000000000009','00000000-0000-4000-8000-000000000099','00000000-0000-4000-8000-000000000001','Foreign project',NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000099','Foreign workspace',NULL);
INSERT INTO issue_labels SELECT '00000000-0000-4000-8000-000000000004',id,CASE WHEN name='Removed' THEN now() END FROM labels;
`)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	for _, axis := range [][2]string{{"LABELS", ""}, {"PRIORITY", "LABELS"}} {
		t.Run(axis[0]+"/"+axis[1], func(t *testing.T) {
			raw, readErr := repository.Chart(ctx, actor, "demo", "00000000-0000-4000-8000-000000000002", "", "", axis[0], axis[1], "", "")
			if readErr != nil {
				t.Fatal(readErr)
			}
			var envelope struct {
				Data   []map[string]any `json:"data"`
				Schema map[string]any   `json:"schema"`
			}
			if decodeErr := json.Unmarshal(raw, &envelope); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if len(envelope.Data) != 1 {
				t.Fatalf("chart=%s", raw)
			}
			if axis[0] == "LABELS" {
				if envelope.Data[0]["key"] != "00000000-0000-4000-8000-000000000006" || envelope.Data[0]["name"] != "Kept" || envelope.Data[0]["count"] != float64(1) {
					t.Fatalf("chart=%s", raw)
				}
			} else if len(envelope.Schema) != 1 || envelope.Schema["00000000-0000-4000-8000-000000000006"] != "Kept" || envelope.Data[0]["00000000-0000-4000-8000-000000000006"] != float64(1) {
				t.Fatalf("chart=%s", raw)
			}
		})
	}
	if _, err = connection.Exec(ctx, "UPDATE project_members SET is_active=false"); err != nil {
		t.Fatal(err)
	}
	raw, err := repository.Chart(ctx, actor, "demo", "00000000-0000-4000-8000-000000000002", "", "", "LABELS", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	var denied struct {
		Data []any `json:"data"`
	}
	if err = json.Unmarshal(raw, &denied); err != nil {
		t.Fatal(err)
	}
	if len(denied.Data) != 0 {
		t.Fatalf("revoked chart=%s", raw)
	}
}
