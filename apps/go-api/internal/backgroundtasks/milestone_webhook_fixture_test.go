//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type milestoneWebhookFixture struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	store *PostgresWebhookTaskStore
	repo  *milestones.Repository
	opts  milestones.WriteOptions
}

func newMilestoneWebhookFixture(t *testing.T) milestoneWebhookFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	_, pool := openBackgroundTaskClone(t, ctx)
	_, err := pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	store, err := NewPostgresWebhookTaskStore(pool, nil)
	require.NoError(t, err)
	repo, err := milestones.NewRepository(pool)
	require.NoError(t, err)
	return milestoneWebhookFixture{ctx, pool, store, repo, milestones.WriteOptions{Scope: milestones.Scope{
		ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace),
		ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "milestones", Timezone: "UTC",
	}, Origin: "https://example.test"}}
}

func (f milestoneWebhookFixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	_, err := f.pool.Exec(f.ctx, sql, args...)
	require.NoError(t, err)
}

func (f milestoneWebhookFixture) create(t *testing.T) uuid.UUID {
	t.Helper()
	title := "Release <one>"
	result, err := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: milestones.StringField{Present: true, Value: &title}})
	require.NoError(t, err)
	return uuid.MustParse(result.MilestoneID)
}

func (f milestoneWebhookFixture) hook(t *testing.T, subscribed bool) string {
	t.Helper()
	repo, err := workspacewebhooks.NewRepository(f.pool)
	require.NoError(t, err)
	raw, err := repo.Create(f.ctx, workspacewebhooks.CreateParams{
		ActorID: stduuid.MustParse(testsupport.LabelActor), Slug: "milestones", URL: "https://example.test/" + uuid.NewString(),
		IsActive: true, Milestone: subscribed, Project: true, Issue: true, Module: true, Cycle: true, IssueComment: true, Version: "v1",
	})
	require.NoError(t, err)
	var record struct {
		ID        string
		Milestone bool
	}
	require.NoError(t, json.Unmarshal(raw, &record))
	require.Equal(t, subscribed, record.Milestone)
	return record.ID
}

func (f milestoneWebhookFixture) effect(t *testing.T, name string) WebhookTaskInput {
	t.Helper()
	var id string
	var raw []byte
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT id::text,payload FROM work_item_effects WHERE payload->>'name'=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, name).Scan(&id, &raw))
	var payload struct {
		Name   string
		Kwargs map[string]json.RawMessage
	}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return WebhookTaskInput{EventID: id, TaskName: payload.Name, Kwargs: payload.Kwargs}
}

func (f milestoneWebhookFixture) creation(t *testing.T) webhookActivityInput {
	t.Helper()
	activities, err := f.store.ModelActivities(f.ctx, f.effect(t, webhookModelActivityTask))
	require.NoError(t, err)
	require.Len(t, activities, 1)
	return activities[0]
}
