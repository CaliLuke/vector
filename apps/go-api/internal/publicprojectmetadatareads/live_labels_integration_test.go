//go:build integration

package publicprojectmetadatareads

import (
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestPublicCurrentIssueLabelsRespectLiveTargetScope(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	board := PublicBoard{ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace)}
	for _, id := range []string{testsupport.LabelIssue, testsupport.LabelEmptyIssue} {
		raw, readErr := repository.IssueDetail(ctx, board, id)
		if readErr != nil {
			t.Fatal(readErr)
		}
		row := testsupport.LabelReadObject(t, raw)
		if row["id"] != id {
			t.Fatalf("public issue ID=%v", row["id"])
		}
		if id == testsupport.LabelIssue {
			testsupport.RequireLabelIDs(t, row["label_ids"], testsupport.LabelKept)
		} else {
			testsupport.RequireLabelIDs(t, row["label_ids"])
		}
	}
	board.ProjectID = uuid.MustParse("00000000-0000-4000-8000-000000000099")
	raw, err := repository.IssueDetail(ctx, board, testsupport.LabelIssue)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "null" {
		t.Fatalf("another board's issue=%s", raw)
	}
}
