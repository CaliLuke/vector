//go:build integration

package workitems

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestDraftProjectionUsesNativeDraftLabelOwnership(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed)
	repository := &draftOperation{db: connection}
	cases := []struct {
		id     string
		labels []string
	}{{testsupport.LabelDraft, []string{testsupport.LabelKept}}, {testsupport.LabelEmptyDraft, []string{}}, {testsupport.LabelProjectlessDraft, []string{testsupport.LabelProjectless}}}
	for _, testCase := range cases {
		value, readErr := djangoapi.JSON(ctx, connection, "SELECT to_jsonb(draft) FROM draft_issues draft WHERE draft.id=$1", testCase.id)
		if readErr != nil {
			t.Fatal(readErr)
		}
		row, projectionErr := repository.draftProjection(ctx, value.(map[string]any), testsupport.LabelActor)
		if projectionErr != nil {
			t.Fatal(projectionErr)
		}
		testsupport.RequireLabelIDs(t, row["label_ids"], testCase.labels...)
	}
}
