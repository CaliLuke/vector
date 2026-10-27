//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneWebhookRequiresExplicitSubscriptionAndProjectsPublicFields(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.create(t)
	legacy := f.hook(t, false) // All existing event flags are true.
	f.exec(t, `INSERT INTO webhooks(id,url,created_at,updated_at,workspace_id,is_active,secret_key,project,issue,module,cycle,issue_comment,is_internal,version)
	 SELECT gen_random_uuid(),url || '/legacy',now(),now(),workspace_id,is_active,secret_key,project,issue,module,cycle,issue_comment,is_internal,version FROM webhooks WHERE id=$1`, legacy)
	var defaultOptIn bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT milestone FROM webhooks WHERE url LIKE '%/legacy'`).Scan(&defaultOptIn))
	require.False(t, defaultOptIn)
	wanted := f.hook(t, true)
	inactive := f.hook(t, true)
	deleted := f.hook(t, true)
	f.exec(t, `UPDATE webhooks SET is_active=false WHERE id=$1`, inactive)
	f.exec(t, `UPDATE webhooks SET deleted_at=now() WHERE id=$1`, deleted)
	f.exec(t, `UPDATE milestones SET created_at='2026-01-02T03:04:05.123456Z',updated_at='2026-02-03T04:05:06Z' WHERE id=$1`, id)

	sends, err := f.store.Fanout(f.ctx, f.creation(t))
	require.NoError(t, err)
	require.Len(t, sends, 1)
	require.Equal(t, wanted, sends[0].WebhookID)
	require.Equal(t, testsupport.LabelWorkspace, sends[0].WorkspaceID)
	require.JSONEq(t, `{"id":"`+id.String()+`","title":"Release <one>","target_date":null,"external_source":null,"external_id":null,"created_at":"2026-01-02T03:04:05.123456Z","updated_at":"2026-02-03T04:05:06Z","project":"`+testsupport.LabelProject+`","workspace":"`+testsupport.LabelWorkspace+`"}`, string(sends[0].EventData))
}

func TestMilestoneWebhookStaysInOriginalWorkspaceAfterSlugReassignment(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	f.create(t)
	wanted := f.hook(t, true)
	activity := f.creation(t)
	f.exec(t, `UPDATE workspaces SET slug='renamed' WHERE id=$1`, testsupport.LabelWorkspace)
	foreign := uuid.NewString()
	f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Other',$2,'milestones','UTC','',now(),now())`, foreign, testsupport.LabelActor)
	f.hook(t, true) // This hook now belongs to the new owner of the old slug.

	sends, err := f.store.Fanout(f.ctx, activity)
	require.NoError(t, err)
	require.Len(t, sends, 1)
	require.Equal(t, wanted, sends[0].WebhookID)
	f.exec(t, `UPDATE workspaces SET slug='renamed-again' WHERE id=$1`, testsupport.LabelWorkspace)
	var delivered webhookPayload
	f.store.requestClient = func(_ context.Context, _ string, _ http.Header, body []byte) (*http.Response, error) {
		require.NoError(t, json.Unmarshal(body, &delivered))
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("accepted"))}, nil
	}
	require.NoError(t, f.store.Deliver(f.ctx, sends[0], 1))
	require.Equal(t, testsupport.LabelWorkspace, delivered.WorkspaceID)
	require.Equal(t, "renamed-again", delivered.WorkspaceSlug)
	activity.WorkspaceID = foreign
	other, err := f.store.Fanout(f.ctx, activity)
	require.NoError(t, err)
	require.Empty(t, other, "foreign workspace must not serialize the milestone")
}

func TestMilestoneWebhookQueuedDeliveryRechecksCurrentSubscription(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	f.create(t)
	hook := f.hook(t, true)
	sends, err := f.store.Fanout(f.ctx, f.creation(t))
	require.NoError(t, err)
	require.Len(t, sends, 1)
	repo, err := workspacewebhooks.NewRepository(f.pool)
	require.NoError(t, err)
	raw, err := repo.Update(f.ctx, workspacewebhooks.PatchParams{ActorID: stduuid.MustParse(testsupport.LabelActor), ID: stduuid.MustParse(hook), Slug: "milestones", MilestonePresent: true, Milestone: false})
	require.NoError(t, err)
	var updated map[string]any
	require.NoError(t, json.Unmarshal(raw, &updated))
	require.Equal(t, false, updated["milestone"])
	calls := 0
	f.store.requestClient = func(context.Context, string, http.Header, []byte) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader("accepted"))}, nil
	}
	require.NoError(t, f.store.Deliver(f.ctx, sends[0], 1))
	require.Zero(t, calls)
	f.exec(t, `UPDATE webhooks SET milestone=true,is_active=false WHERE id=$1`, hook)
	require.NoError(t, f.store.Deliver(f.ctx, sends[0], 2))
	require.Zero(t, calls)
	f.exec(t, `UPDATE webhooks SET is_active=true WHERE id=$1`, hook)
	require.NoError(t, f.store.Deliver(f.ctx, sends[0], 3))
	require.Equal(t, 1, calls)
}

func TestMilestoneWebhookUpdateAndDeletionUseCommittedNativeEvents(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.create(t)
	f.hook(t, true)
	title := "Release two"
	_, err := f.repo.Update(f.ctx, f.opts, id, milestones.Fields{Title: milestones.StringField{Present: true, Value: &title}})
	require.NoError(t, err)
	update := f.creation(t)
	require.Equal(t, "updated", update.Verb)
	require.JSONEq(t, `"title"`, string(update.Field))
	require.JSONEq(t, `"Release <one>"`, string(update.OldValue))
	require.JSONEq(t, `"Release two"`, string(update.NewValue))
	_, err = f.repo.Delete(f.ctx, f.opts, id)
	require.NoError(t, err)
	deletion, err := webhookActivityInputFromTask(f.effect(t, webhookActivityCeleryTask))
	require.NoError(t, err)
	sends, err := f.store.Fanout(f.ctx, deletion)
	require.NoError(t, err)
	require.Len(t, sends, 1)
	require.Equal(t, "deleted", sends[0].Action)
	require.JSONEq(t, `{"id":"`+id.String()+`"}`, string(sends[0].EventData))
	foreign := uuid.NewString()
	f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, foreign, testsupport.LabelActor)
	deletion.WorkspaceID = foreign
	sends, err = f.store.Fanout(f.ctx, deletion)
	require.NoError(t, err)
	require.Empty(t, sends)
}

func TestMilestoneWebhookAssociationKeepsTitleSnapshotAndItemWorkspace(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	id := f.create(t)
	hook := f.hook(t, false) // Issue subscription alone receives association events.
	item := uuid.New()
	f.exec(t, `INSERT INTO issues(id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Item','{}','','medium',1,0,false,now(),now())`, item, testsupport.LabelProject, testsupport.LabelWorkspace, testsupport.LabelActor)
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	activity, err := webhookActivityInputFromTask(f.effect(t, webhookActivityCeleryTask))
	require.NoError(t, err)
	_, err = f.repo.Delete(f.ctx, f.opts, id)
	require.NoError(t, err)
	sends, err := f.store.Fanout(f.ctx, activity)
	require.NoError(t, err)
	require.Len(t, sends, 1)
	require.Equal(t, hook, sends[0].WebhookID)
	require.Equal(t, "issue", sends[0].Event)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(sends[0].Activity, &fields))
	require.Equal(t, "milestone", fields["field"])
	require.Equal(t, "Release <one>", fields["new_value"])
	require.Equal(t, id.String(), fields["new_identifier"])
	require.Nil(t, fields["old_identifier"])

	foreignWorkspace, foreignProject := uuid.NewString(), uuid.NewString()
	f.exec(t, `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, foreignWorkspace, testsupport.LabelActor)
	f.exec(t, `INSERT INTO projects(id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at)
 SELECT $1,'Foreign',description,network,$2,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at FROM projects WHERE id=$3`, foreignProject, foreignWorkspace, testsupport.LabelProject)
	_, err = f.pool.Exec(f.ctx, `UPDATE issues SET project_id=$2,workspace_id=$3 WHERE id=$1`, item, foreignProject, foreignWorkspace)
	require.ErrorContains(t, err, "cannot move between workspaces")
	foreignItem := uuid.NewString()
	f.exec(t, `INSERT INTO issues(id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at)
 VALUES($1,$2,$3,$4,'Foreign item','{}','','medium',1,0,false,now(),now())`, foreignItem, foreignProject, foreignWorkspace, testsupport.LabelActor)
	activity.EventID = foreignItem
	sends, err = f.store.Fanout(f.ctx, activity)
	require.NoError(t, err)
	require.Empty(t, sends, "a mismatched event cannot serialize an item from another workspace")
}
