//go:build integration

package issuequery_test

import (
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/issuequery"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// Compile public plans and evaluate their complete SQL truth tables. These rows
// are already authorized candidates; route visibility is tested by workitems.
func TestPQLPropertySQLPreservesUnknownAndRequestTimezone(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, `CREATE TABLE issues(id uuid,priority text);
INSERT INTO issues VALUES ('62000000-0000-4000-8000-000000000001','low'),('62000000-0000-4000-8000-000000000002','low'),('62000000-0000-4000-8000-000000000003','urgent');`)
	ids := []string{pqlPropertyItem, "62000000-0000-4000-8000-000000000002", "62000000-0000-4000-8000-000000000003"}
	request := issuequery.PQLContext{Timezone: "America/Los_Angeles", Now: time.Date(2026, 10, 28, 1, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		name   string
		kind   issuequery.PQLPropertyKind
		value  any
		suffix string
		want   []pgtype.Bool
	}{
		{"null", issuequery.PQLPropertyText, "value", `IS NULL`, []pgtype.Bool{{Bool: false, Valid: true}, {Bool: true, Valid: true}, {}}},
		{"negative", issuequery.PQLPropertyText, "value", `!= "other"`, []pgtype.Bool{{Bool: true, Valid: true}, {}, {}}},
		{"literal wildcards", issuequery.PQLPropertyText, "50%_\\", `~ "%_\\"`, []pgtype.Bool{{Bool: true, Valid: true}, {}, {}}},
		{"calendar date", issuequery.PQLPropertyDate, "2026-10-27", `= today()`, []pgtype.Bool{{Bool: true, Valid: true}, {}, {}}},
		{"zero", issuequery.PQLPropertyNumber, float64(0), `BETWEEN -1 AND 1`, []pgtype.Bool{{Bool: true, Valid: true}, {}, {}}},
		{"false", issuequery.PQLPropertyBoolean, false, `= false`, []pgtype.Bool{{Bool: true, Valid: true}, {}, {}}},
		{"selection negation", issuequery.PQLPropertySelection, []string{pqlPropertyItem}, `NOT IN ("` + pqlPropertyItem + `")`, []pgtype.Bool{{Bool: false, Valid: true}, {}, {}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := issuequery.PreparePQL(`cf["`+pqlPropertyID+`"] `+tc.suffix, request, propertyResolver(tc.kind))
			require.NoError(t, err)
			values := map[string][]issuequery.PQLPropertyValue{pqlPropertyID: {{ItemID: ids[0], Value: tc.value}, {ItemID: ids[1], Value: nil}}}
			for index, id := range ids {
				args := []any{id}
				sql, err := plan.SQL(values, &args)
				require.NoError(t, err)
				var actual pgtype.Bool
				require.NoError(t, db.QueryRow(ctx, `SELECT `+sql+` FROM issues issue WHERE issue.id=$1`, args...).Scan(&actual))
				require.Equal(t, tc.want[index], actual, id)
			}
		})
	}
}
