//go:build integration

package cyclereads

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const patchDateCycle = "10000000-0000-4000-8000-000000000004"

func cyclePatchDatesFixture(t *testing.T) (context.Context, *pgx.Conn, *LifecycleRepository, *cycleCreationEffects) {
	t.Helper()
	ctx, connection, repository, effects := cycleLifecycleFixture(t)
	_, err := connection.Exec(ctx, `UPDATE users SET user_timezone='America/Los_Angeles'; UPDATE projects SET timezone='Asia/Tokyo'`)
	require.NoError(t, err)
	_, err = connection.Exec(ctx, `INSERT INTO cycles(id,workspace_id,project_id,name,start_date,end_date) VALUES($1,$2,$3,'Dates','2026-10-10','2026-10-20')`, patchDateCycle, lifecycleWorkspace, lifecycleProject)
	require.NoError(t, err)
	repository.now = func() time.Time { return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) }
	return ctx, connection, repository, effects
}

func TestCycleAPIPatchPersistsDatePresenceAndTimezoneRules(t *testing.T) {
	ctx, connection, repository, effects := cyclePatchDatesFixture(t)
	for _, test := range []struct {
		name       string
		request    map[string]any
		start, end string
	}{
		{"omitted dates", map[string]any{"name": "Renamed"}, "2026-10-10T00:00:00Z", "2026-10-20T00:00:00Z"},
		{"single start keeps instant", map[string]any{"start_date": "2026-10-05"}, "2026-10-05T07:00:00Z", "2026-10-20T00:00:00Z"},
		{"single end may precede stored start", map[string]any{"end_date": "2026-10-01"}, "2026-10-10T00:00:00Z", "2026-10-01T07:00:00Z"},
		{"paired dates use actor days and project boundaries", map[string]any{"start_date": "2026-10-05T01:00:00Z", "end_date": "2026-10-06T03:00:00Z"}, "2026-10-03T15:00:01Z", "2026-10-05T14:59:00Z"},
		{"null start", map[string]any{"start_date": nil}, "", "2026-10-20T00:00:00Z"},
		{"null end", map[string]any{"end_date": nil}, "2026-10-10T00:00:00Z", ""},
		{"null start with single end", map[string]any{"start_date": nil, "end_date": "2026-10-01"}, "", "2026-10-01T07:00:00Z"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := connection.Exec(ctx, `UPDATE cycles SET start_date='2026-10-10',end_date='2026-10-20',updated_at=NULL WHERE id=$1`, patchDateCycle)
			require.NoError(t, err)
			effects.task, effects.data = "", nil
			original := maps.Clone(test.request)
			_, err = repository.Patch(ctx, "lifecycle", lifecycleProject, patchDateCycle, lifecycleActor, true, test.request, test.request)
			require.NoError(t, err)
			var start, end string
			require.NoError(t, connection.QueryRow(ctx, `SELECT COALESCE(to_char(start_date AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),''),COALESCE(to_char(end_date AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),'') FROM cycles WHERE id=$1`, patchDateCycle).Scan(&start, &end))
			require.Equal(t, test.start, start)
			require.Equal(t, test.end, end)
			require.Equal(t, original, test.request, "caller input must retain raw dates")
			require.Equal(t, "plane.bgtasks.webhook_task.model_activity", effects.task)
			require.Equal(t, original, effects.data["requested_data"], "webhook retains raw requested dates")
		})
	}
}

func TestCycleAPIPatchRejectsInvalidDatesBeforeWriteOrWebhook(t *testing.T) {
	ctx, connection, repository, effects := cyclePatchDatesFixture(t)
	for _, test := range []struct {
		name           string
		request        map[string]any
		field, message string
	}{
		{"malformed start", map[string]any{"start_date": "invalid"}, "start_date", "Datetime has wrong format."},
		{"non-string end", map[string]any{"end_date": 42}, "end_date", "Datetime has wrong format."},
		{"reversed paired instants", map[string]any{"start_date": "2026-10-05T04:00:00Z", "end_date": "2026-10-05T03:00:00Z"}, "non_field_errors", "Start date cannot exceed end date"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := maps.Clone(test.request)
			_, err := repository.Patch(ctx, "lifecycle", lifecycleProject, patchDateCycle, lifecycleActor, true, test.request, test.request)
			var invalid *djangoapi.HTTPError
			require.ErrorAs(t, err, &invalid)
			require.Equal(t, 400, invalid.Status)
			require.Equal(t, []string{test.message}, invalid.Body[test.field])
			var unchanged bool
			require.NoError(t, connection.QueryRow(ctx, `SELECT start_date='2026-10-10' AND end_date='2026-10-20' AND updated_at IS NULL FROM cycles WHERE id=$1`, patchDateCycle).Scan(&unchanged))
			require.True(t, unchanged, "invalid date patch changed the persisted cycle")
			require.Empty(t, effects.task)
			require.Equal(t, original, test.request)
		})
	}
}
