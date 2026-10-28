//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestIssueActivityFieldOrderWhenDescriptionIsCoalesced(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	item := f.activityItem(t).String()
	actor := testsupport.LabelActor
	store := f.activityStore(t)
	input := IssueActivityInput{
		Type: "issue.activity.updated", ProjectID: testsupport.LabelProject,
		IssueID: &item, ActorID: &actor,
		RequestedData:   json.RawMessage(`{"description_html":"<p>first</p>"}`),
		CurrentInstance: json.RawMessage(`{"description_html":""}`),
	}
	initial, err := store.RecordIssueActivity(f.ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.Len(t, initial, 1)
	f.exec(t, `UPDATE issue_activities SET created_at='2000-01-01' WHERE id=$1`, initial[0])

	input.CurrentInstance = json.RawMessage(`{"name":"Before","priority":"medium","description_html":"<p>first</p>","archived_at":null}`)
	input.RequestedData = json.RawMessage(`{"name":"After","priority":"high","description_html":"<p>second</p>","archived_at":"2025-01-01T00:00:00Z"}`)
	ids, err := store.RecordIssueActivity(f.ctx, uuid.NewString(), input)
	require.NoError(t, err)
	rows, err := f.pool.Query(f.ctx, `SELECT a.field FROM unnest($1::uuid[]) WITH ORDINALITY emitted(id,position)
 JOIN issue_activities a ON a.id=emitted.id ORDER BY emitted.position`, ids)
	require.NoError(t, err)
	fields, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	require.Equal(t, []string{"name", "priority", "archived_at"}, fields)

	var count int
	var refreshed bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*),bool_and(created_at>'2000-01-01')
 FROM issue_activities WHERE issue_id=$1 AND field='description'`, item).Scan(&count, &refreshed))
	require.Equal(t, 1, count)
	require.True(t, refreshed, "the existing description activity must be refreshed without inserting a new row")
}
