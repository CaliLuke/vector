//go:build integration

package modulereads

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestModuleEstimatesExcludeDeletedAndForeignCatalogueRows(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.EstimateReadSchema)
	from := " FROM modules module WHERE $1::uuid IS NOT NULL"
	reads := []testsupport.EstimateReadCase{
		{Name: "started total", SQL: "SELECT " + estimateCount("module", "started") + from, Want: [3]float64{11, 3, 0}},
		{Name: "assignee totals", SQL: "SELECT COALESCE((" + issueAssigneeDistribution("module", true) + "->0->>'total_estimates')::float8,0)" + from, Want: [3]float64{11, 3, 0}},
		{Name: "completed totals", SQL: "SELECT COALESCE((" + issueAssigneeDistribution("module", true) + "->0->>'completed_estimates')::float8,0)" + from, Want: [3]float64{3, 3, 0}},
		{Name: "label totals", SQL: "SELECT COALESCE((" + issueLabelDistribution("module", true) + "->0->>'total_estimates')::float8,0)" + from, Want: [3]float64{11, 3, 0}},
		{Name: "burndown", SQL: "SELECT (" + moduleBurndown("module", true, false) + "->>CURRENT_DATE::text)::float8" + from, Want: [3]float64{8, 0, 0}},
	}
	for i := range reads {
		reads[i].Args = []any{"00000000-0000-4000-8000-000000000009"}
	}
	testsupport.CheckEstimateDeletionReads(t, ctx, connection, reads)
}
