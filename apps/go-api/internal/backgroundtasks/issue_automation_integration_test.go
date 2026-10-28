//go:build integration

package backgroundtasks

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

func TestPostgresIssueAutomationArchivesAndClosesWithRecoverableEffects(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ownerID, workspaceID, projectID := uuid.New(), uuid.New(), uuid.New()
	completedStateID, backlogStateID, cancelledStateID := uuid.New(), uuid.New(), uuid.New()
	archivedIssueID, closedIssueID, youngIssueID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	old := now.Add(-90 * 24 * time.Hour)
	username := "issue-automation-" + ownerID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Automation Test','Automation','Test','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, ownerID, username, now); err != nil {
		t.Fatalf("insert automation owner: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Issue Automation',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, "ia-"+ownerID.String()[:8], now); err != nil {
		t.Fatalf("insert automation workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,created_by_id,module_view,cycle_view,issue_views_view,page_view,intake_view,archive_in,close_in,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,logo_props,timezone,created_at,updated_at) VALUES ($1,'Issue Automation','',0,$2,$3,$4,false,false,false,true,false,1,1,false,false,false,'{}'::jsonb,'UTC',$5,$5)`, projectID, workspaceID, "IA"+ownerID.String()[:3], ownerID, now); err != nil {
		t.Fatalf("insert automation project: %v", err)
	}
	for _, state := range []struct {
		id    uuid.UUID
		name  string
		group string
		seq   float64
	}{
		{completedStateID, "Completed", "completed", 1},
		{backlogStateID, "Backlog", "backlog", 2},
		{cancelledStateID, "Cancelled", "cancelled", 3},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO states (created_at,updated_at,id,name,description,color,slug,project_id,workspace_id,sequence,"group","default",is_triage) VALUES ($1,$1,$2,$3,'','','',$4,$5,$6,$7,false,false)`, now, state.id, state.name, projectID, workspaceID, state.seq, state.group); err != nil {
			t.Fatalf("insert automation state %s: %v", state.name, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE projects SET default_state_id=$2 WHERE id=$1`, projectID, cancelledStateID); err != nil {
		t.Fatalf("set automation default state: %v", err)
	}
	for _, issue := range []struct {
		id      uuid.UUID
		name    string
		stateID uuid.UUID
		updated time.Time
		seq     int
	}{
		{archivedIssueID, "Archive candidate", completedStateID, old, 1},
		{closedIssueID, "Close candidate", backlogStateID, old, 2},
		{youngIssueID, "Young candidate", completedStateID, now.Add(-24 * time.Hour), 3},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,state_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,'{}'::jsonb,'','medium',$7,0,false,$8,$9)`, issue.id, projectID, workspaceID, ownerID, issue.stateID, issue.name, issue.seq, issue.updated, issue.updated); err != nil {
			t.Fatalf("insert automation issue %s: %v", issue.name, err)
		}
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE entity_identifier=ANY($1::uuid[])`, []uuid.UUID{archivedIssueID, closedIssueID, youngIssueID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM issue_activities WHERE issue_id=ANY($1::uuid[])`, []uuid.UUID{archivedIssueID, closedIssueID, youngIssueID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM issues WHERE id=ANY($1::uuid[])`, []uuid.UUID{archivedIssueID, closedIssueID, youngIssueID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID)
	}()

	activityStore, err := NewPostgresIssueActivityStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	defer activityStore.Close()
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-issue-automation-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	issueActivities, err := RegisterIssueActivityWorkflow(app, activityStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewPostgresIssueAutomationStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := RegisterIssueAutomationWorkflow(app, store, issueActivities, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown issue automation worker: %v", shutdownErr)
		}
	}()

	instanceID, err := dispatcher.Admit(ctx, now)
	if err != nil {
		t.Fatalf("admit issue automation: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, instanceID)

	var archivedAt *time.Time
	var actualStateID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT archived_at::timestamptz FROM issues WHERE id=$1`, archivedIssueID).Scan(&archivedAt); err != nil {
		t.Fatalf("read archived issue: %v", err)
	}
	if archivedAt == nil || !archivedAt.UTC().Equal(now.Truncate(24*time.Hour)) {
		t.Fatalf("archive date=%v, want %s", archivedAt, now.Format("2006-01-02"))
	}
	if err := pool.QueryRow(ctx, `SELECT state_id FROM issues WHERE id=$1`, closedIssueID).Scan(&actualStateID); err != nil {
		t.Fatalf("read closed issue state: %v", err)
	}
	if actualStateID != cancelledStateID {
		t.Fatalf("closed issue state=%s, want %s", actualStateID, cancelledStateID)
	}
	var youngArchivedAt *time.Time
	var youngState uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT archived_at::timestamptz,state_id FROM issues WHERE id=$1`, youngIssueID).Scan(&youngArchivedAt, &youngState); err != nil {
		t.Fatalf("read untouched issue: %v", err)
	}
	if youngArchivedAt != nil || youngState != completedStateID {
		t.Fatalf("young issue changed: archived_at=%v state=%s", youngArchivedAt, youngState)
	}
	var archiveActivity, closeActivity int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=$1 AND field='archived_at' AND comment='Plane has archived the issue'`, archivedIssueID).Scan(&archiveActivity); err != nil {
		t.Fatalf("count archive activity: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=$1 AND field='state' AND comment='Plane updated the state to '`, closedIssueID).Scan(&closeActivity); err != nil {
		t.Fatalf("count close activity: %v", err)
	}
	if archiveActivity != 1 || closeActivity != 1 {
		t.Fatalf("automation activity rows archive=%d close=%d, want one each", archiveActivity, closeActivity)
	}

	// A retry after the PostgreSQL effect commits must replay both durable
	// child events, while the deterministic activity IDs prevent duplicates.
	for _, operation := range []issueAutomationOperation{archiveIssuesOperation, closeIssuesOperation} {
		batch, err := store.ApplyIssueAutomationBatch(ctx, issueAutomationBatchInput{ScheduledAt: now, Operation: operation})
		if err != nil {
			t.Fatalf("replay %s batch after commit: %v", operation, err)
		}
		if len(batch.Events) != 1 {
			t.Fatalf("replayed %s batch returned %d child events, want 1", operation, len(batch.Events))
		}
		childID, err := issueActivities.Admit(ctx, batch.Events[0].EventID, batch.Events[0].Activity)
		if err != nil {
			t.Fatalf("reconcile %s child activity: %v", operation, err)
		}
		waitVersionWorkflow(t, ctx, app, childID)
	}
	var totalActivities int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[])`, []uuid.UUID{archivedIssueID, closedIssueID}).Scan(&totalActivities); err != nil {
		t.Fatalf("count activities after workflow retry: %v", err)
	}
	if totalActivities != 2 {
		t.Fatalf("workflow retry created %d activities, want 2", totalActivities)
	}
}
