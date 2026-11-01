//go:build integration

package workitems

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestBulkDatesValidateWholeBatchBeforeEffectsOrWrites(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+`
CREATE TABLE work_item_effects(id uuid PRIMARY KEY,kind text,payload jsonb);
UPDATE issues SET start_date='2026-01-01',target_date='2026-01-10';
`)
	legacy := &versionTaskRecorder{}
	repository := &Repository{db: db, publisher: legacy, transactionalEffects: true, now: time.Now}
	q := Request{Slug: "demo", ProjectID: testsupport.LabelProject, ActorID: testsupport.LabelActor}
	const first = "00000000-0000-4000-8000-000000000004"
	const second = "00000000-0000-4000-8000-000000000005"
	for _, invalid := range []any{"2026-1-2", "2026-02-30", true, "2026-01-11"} {
		t.Run("reject/"+requestLabel(invalid), func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"updates": []any{
				map[string]any{"id": first, "start_date": "2026-01-02"},
				map[string]any{"id": second, "start_date": invalid},
			}})
			require.NoError(t, err)
			_, err = repository.issueDates(ctx, q, strings.NewReader(string(body)), "application/json")
			var response *apicontract.HTTPError
			require.True(t, errors.As(err, &response), "error=%v", err)
			require.Equal(t, 400, response.Status)
			var changed, effects int
			require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM issues WHERE start_date<>'2026-01-01' OR target_date<>'2026-01-10'`).Scan(&changed))
			require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects))
			require.Zero(t, changed)
			require.Zero(t, effects, "invalid later item must prevent earlier activity admission")
			require.Empty(t, legacy.names)
		})
	}
	body := `{"updates":[{"id":"` + first + `","start_date":"2024-02-29"},{"id":"` + second + `","target_date":null}]}`
	result, err := repository.issueDates(ctx, q, strings.NewReader(body), "application/json")
	require.NoError(t, err)
	require.Equal(t, 200, result.Status)
	var start string
	var target *string
	var effects int
	require.NoError(t, db.QueryRow(ctx, `SELECT start_date::text FROM issues WHERE id=$1`, first).Scan(&start))
	require.NoError(t, db.QueryRow(ctx, `SELECT target_date::text FROM issues WHERE id=$1`, second).Scan(&target))
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects))
	require.Equal(t, "2024-02-29", start)
	require.Nil(t, target)
	require.Equal(t, 2, effects)
	require.Empty(t, legacy.names)
}

func requestLabel(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return "non-string"
}
