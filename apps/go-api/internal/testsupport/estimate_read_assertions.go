//go:build integration

package testsupport

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

type EstimateReadCase struct {
	Name string
	SQL  string
	Args []any
	Want [3]float64
}

// CheckEstimateDeletionReads executes the real read projections while the
// issue-to-point references remain intact throughout both soft deletions.
func CheckEstimateDeletionReads(t *testing.T, ctx context.Context, connection *pgx.Conn, reads []EstimateReadCase) {
	t.Helper()
	for step, name := range []string{"live", "inactive system deleted", "active system deleted"} {
		if step > 0 {
			id := "00000000-0000-4000-8000-000000000102"
			if step == 2 {
				id = "00000000-0000-4000-8000-000000000101"
			}
			if _, err := connection.Exec(ctx, "UPDATE estimates SET deleted_at=now() WHERE id=$1", id); err != nil {
				t.Fatal(err)
			}
		}
		t.Run(name, func(t *testing.T) {
			for _, read := range reads {
				t.Run(read.Name, func(t *testing.T) {
					var got float64
					if err := connection.QueryRow(ctx, read.SQL, read.Args...).Scan(&got); err != nil {
						t.Fatal(err)
					}
					if got != read.Want[step] {
						t.Errorf("estimate projection = %v; want %v", got, read.Want[step])
					}
				})
			}
		})
	}
}
