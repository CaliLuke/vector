//go:build integration

package activityreads

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestActivityExpansionUsesCurrentLiveLabels(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	for _, id := range []string{testsupport.LabelIssue, testsupport.LabelEmptyIssue} {
		var raw []byte
		if err := connection.QueryRow(ctx, "SELECT "+activityIssueExpansion+" FROM (VALUES($1::uuid)) a(issue_id)", id).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		row := testsupport.LabelReadObject(t, raw)
		if row["id"] != id {
			t.Fatalf("current issue ID=%v", row["id"])
		}
		if id == testsupport.LabelIssue {
			testsupport.RequireLabelIDs(t, row["labels"], testsupport.LabelKept)
		} else {
			testsupport.RequireLabelIDs(t, row["labels"])
		}
	}
}
