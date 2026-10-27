//go:build integration

package milestones_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/notifications"
	"github.com/CaliLuke/go-plane/internal/personaldashboard"
	"github.com/CaliLuke/go-plane/internal/workitemhistory"
	"github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type activityReadFixture struct {
	writeFixture
	itemID, activityID, notificationID uuid.UUID
}

func newActivityReadFixture(t *testing.T) activityReadFixture {
	t.Helper()
	f := newWriteFixture(t)
	item := f.item(t)
	milestone := f.milestone(t, "Retained secret milestone")
	_, err := f.repo.Delete(f.ctx, f.opts, uuid.MustParse(milestone.MilestoneID))
	require.NoError(t, err)
	activity, notification := uuid.New(), uuid.New()
	_, err = f.db.Exec(f.ctx, `INSERT INTO issue_activities(id,created_at,updated_at,workspace_id,project_id,issue_id,actor_id,verb,field,old_value,old_identifier,comment,attachments,epoch)
 VALUES($1,'2026-01-01','2026-01-01',$2,$3,$4,$5,'updated','milestone','Retained secret milestone',$6,'removed milestone','{}',1767225600),
 (gen_random_uuid(),'2026-01-02','2026-01-02',$2,$3,$4,$5,'updated','priority','low',NULL,'changed priority','{}',1767312000)`, activity, f.opts.WorkspaceID, f.opts.ProjectID, item, f.opts.ActorID, milestone.MilestoneID)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `INSERT INTO notifications(id,created_at,updated_at,workspace_id,project_id,receiver_id,entity_identifier,entity_name,title,message_html,sender,data)
 VALUES($1,'2026-01-01','2026-01-01',$2,$3,$4,$5,'issue','Retained secret milestone','','Issue changed','{"issue_activity":{"field":"milestone","old_value":"Retained secret milestone"}}'),
 (gen_random_uuid(),'2026-01-02','2026-01-02',$2,$3,$4,$5,'issue','Priority changed','','Issue changed','{"issue_activity":{"field":"priority"}}')`, notification, f.opts.WorkspaceID, f.opts.ProjectID, f.opts.ActorID, item)
	require.NoError(t, err)
	return activityReadFixture{f, item, activity, notification}
}

func (f activityReadFixture) activityRequest() activityreads.Request {
	return activityreads.Request{Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities", Params: map[string]string{"slug": "milestones", "project_id": f.opts.ProjectID.String(), "issue_id": f.itemID.String(), "per_page": "1"}, Present: map[string]bool{"per_page": true}}
}

func (f activityReadFixture) checkFeeds(t *testing.T, db pgx.Tx, want int) {
	t.Helper()
	repo, err := activityreads.NewRepository(db)
	require.NoError(t, err)
	principal := activityreads.Principal{UserID: f.opts.ActorID, Authenticated: true, Timezone: "UTC"}
	for _, bound := range []bool{false, true} {
		q := f.activityRequest()
		if bound {
			q.ExpectedWorkspaceID = &f.opts.WorkspaceID
		}
		page, readErr := repo.Read(f.ctx, principal, q)
		require.NoError(t, readErr)
		require.Equal(t, want, page.Body.(map[string]any)["total_count"])
		require.Len(t, page.Body.(map[string]any)["results"], 1)
		require.Equal(t, want == 2, page.Body.(map[string]any)["next_page_results"])
		q.Operation += "_by_pk"
		q.Params["pk"] = f.activityID.String()
		detail, readErr := repo.Read(f.ctx, principal, q)
		if want == 2 {
			require.NoError(t, readErr)
			require.Equal(t, "Retained secret milestone", detail.Body.(map[string]any)["old_value"])
		} else {
			if bound {
				require.ErrorIs(t, readErr, activityreads.ErrNotFound)
			} else {
				var failure *activityreads.HTTPError
				require.ErrorAs(t, readErr, &failure)
				require.Equal(t, 404, failure.Status)
			}
			require.Nil(t, detail.Body)
		}
	}
	profile, err := repo.Read(f.ctx, principal, activityreads.Request{Operation: "get_api_users_me_activities", Params: map[string]string{}})
	require.NoError(t, err)
	require.Equal(t, want, profile.Body.(map[string]any)["total_count"])
	history, err := workitemhistory.NewRepository(db)
	require.NoError(t, err)
	for _, kind := range []string{"", "issue-property"} {
		raw, readErr := history.Read(f.ctx, workitemhistory.Request{Operation: "history", UserID: f.opts.ActorID, Slug: "milestones", ProjectID: f.opts.ProjectID, ResourceID: f.itemID, Timezone: "UTC", ActivityType: &kind})
		require.NoError(t, readErr)
		var rows []json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &rows))
		require.Len(t, rows, want)
	}
	insights, err := workspaceuserinsights.NewRepository(db)
	require.NoError(t, err)
	raw, err := insights.Activity(f.ctx, f.opts.ActorID.String(), "milestones", f.opts.ActorID.String(), nil, djangoapi.Query{Params: map[string]string{"per_page": "1"}, Present: map[string]bool{"per_page": true}})
	require.NoError(t, err)
	var page struct {
		Total   int `json:"total_count"`
		Results []json.RawMessage
		Next    bool `json:"next_page_results"`
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	require.Equal(t, want, page.Total)
	require.Len(t, page.Results, 1)
	require.Equal(t, want == 2, page.Next)
	rows, err := insights.ExportActivity(f.ctx, f.opts.ActorID.String(), "milestones", f.opts.ActorID.String(), "2026-01-01")
	require.NoError(t, err)
	require.Len(t, rows, want-1)
}

func (f activityReadFixture) checkNotifications(t *testing.T, db pgx.Tx, want int) {
	t.Helper()
	repo, err := notifications.NewRepository(db)
	require.NoError(t, err)
	principal := notifications.Principal{UserID: f.opts.ActorID.String(), Timezone: "UTC"}
	result, err := repo.List(f.ctx, principal, "milestones", url.Values{"per_page": {"1"}, "cursor": {"1:0:0"}})
	require.NoError(t, err)
	page := result.(map[string]any)
	require.Equal(t, int64(want), page["total_count"])
	require.Equal(t, want == 2, page["next_page_results"])
	require.Len(t, page["results"], 1)
	counts, err := repo.Counts(f.ctx, principal, "milestones")
	require.NoError(t, err)
	require.Equal(t, int64(want), counts["total_unread_notifications_count"])
	detail, err := repo.Detail(f.ctx, principal, "milestones", f.notificationID.String(), false)
	if want == 2 {
		require.NoError(t, err)
		require.Equal(t, "Retained secret milestone", detail["title"])
	} else {
		var failure *notifications.HTTPError
		require.ErrorAs(t, err, &failure)
		require.Equal(t, 404, failure.Status)
		require.Nil(t, detail)
		encoded, encodeErr := json.Marshal(result)
		require.NoError(t, encodeErr)
		require.NotContains(t, string(encoded), "Retained secret milestone")
		require.NotContains(t, string(encoded), f.notificationID.String())
	}
}

func TestMilestoneHistoryReadersPreserveDeletedSnapshotsAndRestrictGuests(t *testing.T) {
	f := newActivityReadFixture(t)
	for _, tc := range []struct {
		name, change string
		want         int
	}{
		{"member sees deleted milestone", "SELECT 1", 2},
		{"restricted guest owns item", "UPDATE project_members SET role=5", 1},
		{"full guest", "UPDATE project_members SET role=5; UPDATE projects SET guest_view_all_features=true", 2},
		{"inactive actor", "UPDATE users SET is_active=false", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := f.db.Begin(f.ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(context.WithoutCancel(f.ctx))) }()
			_, err = tx.Exec(f.ctx, tc.change)
			require.NoError(t, err)
			f.checkNotifications(t, tx, tc.want)
			// Bound MCP scope independently rejects inactive actors before row filtering.
			if tc.name != "inactive actor" {
				f.checkFeeds(t, tx, tc.want)
			}
		})
	}
}

func TestMilestoneSavedNotificationsAndProfileRecheckAccessAndItemOwnership(t *testing.T) {
	f := newActivityReadFixture(t)
	for _, change := range []string{
		"UPDATE workspace_members SET is_active=false",
		"UPDATE workspace_members SET deleted_at=now()",
		"UPDATE project_members SET is_active=false",
		"UPDATE project_members SET deleted_at=now()",
		"UPDATE project_members SET role=99",
		"UPDATE projects SET archived_at=now()",
		"UPDATE projects SET deleted_at=now()",
		"UPDATE workspaces SET deleted_at=now()",
		"UPDATE issues SET is_draft=true",
		"UPDATE issues SET deleted_at=now()",
		`INSERT INTO projects SELECT (jsonb_populate_record(NULL::projects,to_jsonb(source)||jsonb_build_object('id','00000000-0000-4000-8000-000000000099','identifier','FOREIGN','name','Foreign project'))).* FROM projects source;
         INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(source)||jsonb_build_object('id',gen_random_uuid(),'project_id','00000000-0000-4000-8000-000000000099'))).* FROM project_members source;
         UPDATE issue_activities SET project_id='00000000-0000-4000-8000-000000000099' WHERE field='milestone';
         UPDATE notifications SET project_id='00000000-0000-4000-8000-000000000099' WHERE data#>>'{issue_activity,field}'='milestone'`,
		"UPDATE issue_activities SET issue_id=NULL WHERE field='milestone'; UPDATE notifications SET entity_identifier=NULL WHERE data#>>'{issue_activity,field}'='milestone'",
	} {
		t.Run(change, func(t *testing.T) {
			tx, err := f.db.Begin(f.ctx)
			require.NoError(t, err)
			defer func() { require.NoError(t, tx.Rollback(context.WithoutCancel(f.ctx))) }()
			_, err = tx.Exec(f.ctx, change)
			require.NoError(t, err)
			f.checkNotifications(t, tx, 1)
			repo, err := activityreads.NewRepository(tx)
			require.NoError(t, err)
			result, err := repo.Read(f.ctx, activityreads.Principal{UserID: f.opts.ActorID, Authenticated: true, Timezone: "UTC"}, activityreads.Request{Operation: "get_api_users_me_activities", Params: map[string]string{}})
			require.NoError(t, err)
			require.Equal(t, 1, result.Body.(map[string]any)["total_count"])
			encoded, err := json.Marshal(result.Body)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "Retained secret milestone")
			require.NotContains(t, string(encoded), f.activityID.String())
		})
	}
}

func TestMilestoneDashboardCountsExcludeHiddenHistory(t *testing.T) {
	f := newActivityReadFixture(t)
	_, err := f.db.Exec(f.ctx, `UPDATE issue_activities SET created_at=CURRENT_TIMESTAMP`)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, change string
		want         int
	}{
		{"member", "SELECT 1", 2},
		{"restricted guest", "UPDATE project_members SET role=5", 1},
		{"revoked membership", "UPDATE project_members SET is_active=false", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, txErr := f.db.Begin(f.ctx)
			require.NoError(t, txErr)
			defer func() { require.NoError(t, tx.Rollback(context.WithoutCancel(f.ctx))) }()
			_, txErr = tx.Exec(f.ctx, tc.change)
			require.NoError(t, txErr)
			repo, repoErr := personaldashboard.NewRepository(tx)
			require.NoError(t, repoErr)
			graph, readErr := repo.ActivityGraph(f.ctx, f.opts.ActorID.String(), "milestones")
			require.NoError(t, readErr)
			var rows []struct {
				Count int `json:"activity_count"`
			}
			require.NoError(t, json.Unmarshal(graph, &rows))
			require.Len(t, rows, 1)
			require.Equal(t, tc.want, rows[0].Count)
			dashboard, readErr := repo.Dashboard(f.ctx, f.opts.ActorID.String(), "milestones", "1")
			require.NoError(t, readErr)
			var result struct {
				Activities json.RawMessage `json:"issue_activities"`
			}
			require.NoError(t, json.Unmarshal(dashboard, &result))
			require.JSONEq(t, string(graph), string(result.Activities))
		})
	}
}
