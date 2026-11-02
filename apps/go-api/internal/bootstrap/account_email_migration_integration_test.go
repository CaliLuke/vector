//go:build integration

package bootstrap_test

import (
	_ "embed"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:embed schema_migrations/000048_account_email_changes.up.sql
var accountEmailUp string

//go:embed schema_migrations/000048_account_email_changes.down.sql
var accountEmailDown string

func TestAccountEmailUpgradePreservesQueuedNotificationIdentityAndRetry(t *testing.T) {
	f := newMilestoneSchemaFixture(t)
	f.exec(t, accountEmailDown)
	id, control := uuid.NewString(), uuid.NewString()
	f.exec(t, `INSERT INTO work_item_effects(id,kind,payload,attempts,next_attempt_at,last_error) VALUES($1,'webhook','{"name":"plane.bgtasks.webhook_task.send_webhook_deactivation_email","args":["hook","creator","https://example.test","failed"]}',2,clock_timestamp()+interval '1 minute','temporary outage'),($2,'webhook','{"name":"plane.bgtasks.webhook_task.send_webhook"}',0,clock_timestamp(),'')`, id, control)
	var before string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (to_jsonb(e)-'kind')::text FROM work_item_effects e WHERE id=$1`, id).Scan(&before))
	f.exec(t, accountEmailUp)
	var kind, after string
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT kind,(to_jsonb(e)-'kind')::text FROM work_item_effects e WHERE id=$1`, id).Scan(&kind, &after))
	require.Equal(t, "email_task", kind)
	require.Equal(t, before, after, "payload, identity, due time and retry state must survive upgrade")
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT kind FROM work_item_effects WHERE id=$1`, control).Scan(&kind))
	require.Equal(t, "webhook", kind)
	f.exec(t, accountEmailDown)
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT kind,(to_jsonb(e)-'kind')::text FROM work_item_effects e WHERE id=$1`, id).Scan(&kind, &after))
	require.Equal(t, "webhook", kind)
	require.Equal(t, before, after)
	f.exec(t, accountEmailUp)
}
