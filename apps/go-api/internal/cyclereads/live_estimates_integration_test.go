//go:build integration

package cyclereads

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestCycleEstimatesExcludeDeletedAndForeignCatalogueRows(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, testsupport.EstimateReadSchema)
	progress := "SELECT (stats->>'total_estimate_points')::float8 FROM (" + cycleProgressSQL + ") result(stats)"
	from := " FROM cycles cycle JOIN projects project ON project.id=cycle.project_id JOIN workspaces workspace ON workspace.id=cycle.workspace_id WHERE $1::uuid IS NOT NULL"
	distribution := "(" + estimateDistributionSQL + ")"
	reads := []testsupport.EstimateReadCase{
		{Name: "progress", SQL: progress, Args: []any{"estimates", "00000000-0000-4000-8000-000000000002", "00000000-0000-4000-8000-000000000003"}, Want: [3]float64{11, 3, 0}},
		{Name: "assignee totals", SQL: "SELECT COALESCE((" + distribution + "->'assignees'->0->>'total_estimates')::float8,0)" + from, Want: [3]float64{11, 3, 0}},
		{Name: "completed totals", SQL: "SELECT COALESCE((" + distribution + "->'assignees'->0->>'completed_estimates')::float8,0)" + from, Want: [3]float64{3, 3, 0}},
		{Name: "label totals", SQL: "SELECT COALESCE((" + distribution + "->'labels'->0->>'total_estimates')::float8,0)" + from, Want: [3]float64{11, 3, 0}},
		{Name: "burndown", SQL: "SELECT (" + estimateBurndownSQL + "->>CURRENT_DATE::text)::float8" + from, Want: [3]float64{8, 0, 0}},
	}
	for i := range reads {
		if reads[i].Args == nil {
			reads[i].Args = []any{"00000000-0000-4000-8000-000000000009"}
		}
	}
	testsupport.CheckEstimateDeletionReads(t, ctx, connection, reads)
}
