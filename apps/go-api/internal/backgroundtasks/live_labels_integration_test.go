//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCurrentIssueExportUsesLiveLabelsAndRetainsVisibility(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	if _, err := connection.Exec(ctx, `INSERT INTO exporters(token,type,initiated_by_id) VALUES('label-contract','issue_exports',$1)`, testsupport.LabelActor); err != nil {
		t.Fatal(err)
	}
	rows, err := connection.Query(ctx, issueExportQuery, testsupport.LabelWorkspace, []string{testsupport.LabelProject}, "label-contract")
	if err != nil {
		t.Fatal(err)
	}
	records := map[string]map[string]any{}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		record := testsupport.LabelReadObject(t, raw)
		records[record["name"].(string)] = record
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("current export rows=%v", records)
	}
	testsupport.RequireLabelIDs(t, records["Mixed labels"]["labels"], "Kept")
	testsupport.RequireLabelIDs(t, records["No live labels"]["labels"])
	if records["Mixed labels"]["identifier"] != "CUR-1" {
		t.Fatalf("export shape=%v", records["Mixed labels"])
	}
	if _, err = connection.Exec(ctx, `UPDATE project_members SET is_active=false`); err != nil {
		t.Fatal(err)
	}
	rows, err = connection.Query(ctx, issueExportQuery, testsupport.LabelWorkspace, []string{testsupport.LabelProject}, "label-contract")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("export exposed issue after project membership revocation")
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentAnalyticsExportUsesLiveLabelAxisAndSegments(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	pool, err := pgxpool.New(ctx, connection.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := analyticsExportStore{pool: pool}
	for _, tc := range []struct {
		name  string
		input string
		want  [][]string
	}{
		{"label-axis", `{"x_axis":"labels__id","y_axis":"issue_count"}`, [][]string{{"Label", "Issue Count"}, {"Kept", "1"}}},
		{"label-segment", `{"x_axis":"priority","y_axis":"issue_count","segment":"labels__id"}`, [][]string{{"Priority", "Issue Count", "Kept"}, {"low", "2", "1"}}},
		{"selected-deleted-label", `{"x_axis":"labels__id","y_axis":"issue_count","labels":"00000000-0000-4000-8000-000000000007"}`, [][]string{{"Label", "Issue Count"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, readErr := store.rows(ctx, "demo", json.RawMessage(tc.input))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !reflect.DeepEqual(actual, tc.want) {
				t.Fatalf("analytics export=%v want=%v", actual, tc.want)
			}
		})
	}
}
