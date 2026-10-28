//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresIssueActivityWorkflowIsIdempotentPerEvent(t *testing.T) {
	executionURL, pool := openCatalogueActivityDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	ownerID, receiverID, mentionID, workspaceID, projectID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM email_notification_logs WHERE receiver_id=ANY($1::uuid[])`, []uuid.UUID{receiverID, mentionID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE receiver_id=$1`, receiverID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM notifications WHERE receiver_id=$1`, mentionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM email_notification_logs WHERE receiver_id=$1`, receiverID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM email_notification_logs WHERE receiver_id=$1`, mentionID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM issue_subscribers WHERE issue_id=$1`, issueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_members WHERE member_id=$1 AND project_id=$2`, receiverID, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_members WHERE member_id=$1 AND project_id=$2`, mentionID, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM issue_mentions WHERE issue_id=$1`, issueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM issue_activities WHERE issue_id=$1`, issueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM issues WHERE id=$1`, issueID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, ownerID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, receiverID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, mentionID)
	}()
	username := "issue-activity-" + ownerID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Activity Test','Activity','Test','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, ownerID, username, now); err != nil {
		t.Fatalf("insert activity owner fixture: %v", err)
	}
	receiverUsername := "issue-activity-receiver-" + receiverID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Activity Receiver','Activity','Receiver','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, receiverID, receiverUsername, now); err != nil {
		t.Fatalf("insert activity receiver fixture: %v", err)
	}
	mentionUsername := "issue-activity-mention-" + mentionID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Activity Mention','Activity','Mention','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, mentionID, mentionUsername, now); err != nil {
		t.Fatalf("insert activity mention fixture: %v", err)
	}
	receiverEmail := "activity-receiver-" + receiverID.String() + "@example.test"
	mentionEmail := "activity-mention-" + mentionID.String() + "@example.test"
	if _, err := pool.Exec(ctx, `UPDATE users SET email=CASE id WHEN $1 THEN $3 WHEN $2 THEN $4 END WHERE id=ANY($5::uuid[])`, receiverID, mentionID, receiverEmail, mentionEmail, []uuid.UUID{receiverID, mentionID}); err != nil {
		t.Fatalf("set notification receiver emails: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Activity Test',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, "ia-"+ownerID.String()[:8], now); err != nil {
		t.Fatalf("insert activity workspace fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Activity Test','',0,$2,$3,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$4,$4)`, projectID, workspaceID, "IA"+ownerID.String()[:3], now); err != nil {
		t.Fatalf("insert activity project fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,'Activity Test','{}'::jsonb,'<p>test</p>','medium',1,0,false,$5,$5)`, issueID, projectID, workspaceID, ownerID, now); err != nil {
		t.Fatalf("insert activity issue fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id,workspace_id,project_id,member_id,role,view_props,default_props,sort_order,preferences,is_active,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,15,'{}'::jsonb,'{}'::jsonb,0,'{}'::jsonb,true,$5,$5,NULL)`, uuid.New(), workspaceID, projectID, receiverID, now); err != nil {
		t.Fatalf("insert notification project member: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id,workspace_id,project_id,member_id,role,view_props,default_props,sort_order,preferences,is_active,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,15,'{}'::jsonb,'{}'::jsonb,0,'{}'::jsonb,true,$5,$5,NULL)`, uuid.New(), workspaceID, projectID, mentionID, now); err != nil {
		t.Fatalf("insert mention project member: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,issue_id,subscriber_id) VALUES ($1,$2,$2,NULL,NULL,NULL,$3,$4,$5,$6)`, uuid.New(), now, workspaceID, projectID, issueID, receiverID); err != nil {
		t.Fatalf("insert notification issue subscriber: %v", err)
	}
	store, err := NewPostgresIssueActivityStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	emailSender := &recordingIssueEmailSender{failRemaining: 1}
	emailStore, err := NewPostgresIssueEmailStore(pool, fixedIssueEmailConfig{}, emailSender)
	if err != nil {
		t.Fatal(err)
	}
	defer emailStore.Close()
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-issue-activity-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterIssueActivityWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	emailDispatcher, err := RegisterIssueEmailNotificationWorkflow(app, emailStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if startErr := app.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	appStopped := false
	defer func() {
		if appStopped {
			return
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown issue activity runtime: %v", shutdownErr)
		}
	}()

	input := IssueActivityInput{Type: "issue.activity.deleted", IssueID: stringPointer(issueID.String()), ActorID: stringPointer(ownerID.String()), ProjectID: projectID.String(), Epoch: float64(now.Unix())}
	eventID := uuid.NewString()
	firstID, err := dispatcher.Admit(ctx, eventID, input)
	if err != nil {
		t.Fatalf("admit first activity event: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, firstID)
	if _, admitErr := dispatcher.Admit(ctx, eventID, input); admitErr != nil {
		t.Fatalf("reconcile duplicate activity event: %v", admitErr)
	}
	// This simulates an activity retry after its PostgreSQL commit but before
	// the durable framework records completion.
	if _, recordIssueActivityErr := store.RecordIssueActivity(ctx, eventID, input); recordIssueActivityErr != nil {
		t.Fatalf("replay committed activity effect: %v", recordIssueActivityErr)
	}
	var count int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=$1 AND field='issue' AND verb='deleted' AND actor_id=$2`, issueID, ownerID).Scan(&count); scanErr != nil {
		t.Fatalf("count replayed activity rows: %v", scanErr)
	}
	if count != 1 {
		t.Fatalf("duplicate event created %d activity rows, want 1", count)
	}

	secondID, err := dispatcher.Admit(ctx, uuid.NewString(), input)
	if err != nil {
		t.Fatalf("admit second identical activity event: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, secondID)
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=$1 AND field='issue' AND verb='deleted' AND actor_id=$2`, issueID, ownerID).Scan(&count); scanErr != nil {
		t.Fatalf("count distinct activity rows: %v", scanErr)
	}
	if count != 2 {
		t.Fatalf("distinct identical event created %d activity rows, want 2", count)
	}
	assertCatalogueActivityWorkflows(t, ctx, app, dispatcher, store, pool, input)

	update := IssueActivityInput{
		Type: "issue.activity.updated", ProjectID: projectID.String(), IssueID: stringPointer(issueID.String()), ActorID: stringPointer(ownerID.String()), Notification: true, Origin: "https://plane.example.test",
		RequestedData:   json.RawMessage(`{"name":"Activity Updated","priority":"high","description_html":"<p><mention-component entity_name=\"user_mention\" entity_identifier=\"` + mentionID.String() + `\"></mention-component></p>"}`),
		CurrentInstance: json.RawMessage(`{"name":"Activity Test","priority":"medium","description_html":"<p>test</p>"}`),
	}
	updateID, err := dispatcher.Admit(ctx, uuid.NewString(), update)
	if err != nil {
		t.Fatalf("admit issue update activity: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, updateID)
	var updateCount int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_activities WHERE issue_id=$1 AND actor_id=$2 AND verb='updated' AND field IN ('name','priority')`, issueID, ownerID).Scan(&updateCount); scanErr != nil {
		t.Fatalf("count issue update activities: %v", scanErr)
	}
	if updateCount != 2 {
		t.Fatalf("issue update created %d activity rows, want 2", updateCount)
	}
	var notificationCount, emailCount int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE receiver_id=$1 AND entity_identifier=$2`, receiverID, issueID).Scan(&notificationCount); scanErr != nil {
		t.Fatalf("count native issue notifications: %v", scanErr)
	}
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_notification_logs WHERE receiver_id=$1 AND entity_identifier=$2`, receiverID, issueID).Scan(&emailCount); scanErr != nil {
		t.Fatalf("count native issue email logs: %v", scanErr)
	}
	if notificationCount != 3 || emailCount != 2 {
		t.Fatalf("notification fanout created notifications=%d email logs=%d, want one per activity and email only for name and priority changes", notificationCount, emailCount)
	}
	var mentionNotifications, mentionEmailLogs, mentionSubscribers, issueMentions int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE receiver_id=$1 AND entity_identifier=$2 AND sender='in_app:issue_activities:mentioned'`, mentionID, issueID).Scan(&mentionNotifications); scanErr != nil {
		t.Fatalf("count mention notifications: %v", scanErr)
	}
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_notification_logs WHERE receiver_id=$1 AND entity_identifier=$2`, mentionID, issueID).Scan(&mentionEmailLogs); scanErr != nil {
		t.Fatalf("count mention email logs: %v", scanErr)
	}
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2 AND deleted_at IS NULL`, issueID, mentionID).Scan(&mentionSubscribers); scanErr != nil {
		t.Fatalf("count mention subscriber: %v", scanErr)
	}
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_mentions WHERE issue_id=$1 AND mention_id=$2 AND deleted_at IS NULL`, issueID, mentionID).Scan(&issueMentions); scanErr != nil {
		t.Fatalf("count issue mention: %v", scanErr)
	}
	if mentionNotifications != 1 || mentionEmailLogs != 1 || mentionSubscribers != 1 || issueMentions != 1 {
		t.Fatalf("description mention effects notification=%d email=%d subscriber=%d issue_mention=%d, want one each", mentionNotifications, mentionEmailLogs, mentionSubscribers, issueMentions)
	}
	pendingBatches, err := emailStore.pendingBatches(ctx)
	if err != nil {
		t.Fatalf("collect email batches before durable admission: %v", err)
	}
	if len(pendingBatches) != 2 {
		t.Fatalf("collected %d pending receiver batches, want two", len(pendingBatches))
	}
	var stillPending int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_notification_logs WHERE receiver_id=ANY($1::uuid[]) AND entity_identifier=$2 AND processed_at IS NULL`, []uuid.UUID{receiverID, mentionID}, issueID).Scan(&stillPending); scanErr != nil {
		t.Fatalf("check pending rows after collector-only pass: %v", scanErr)
	}
	if stillPending != 3 {
		t.Fatalf("collector-only pass marked %d rows processed; want all three replayable until child admission and delivery", 3-stillPending)
	}
	emailWorkflowID, err := emailDispatcher.Admit(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("admit issue email delivery: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, emailWorkflowID)
	if got := emailSender.count(); got != 2 {
		t.Fatalf("email delivery sent %d messages, want one per receiver/issue batch", got)
	}
	if got := emailSender.attempts(); got != 3 {
		t.Fatalf("SMTP sender was called %d times, want one failed attempt and two successful batch deliveries", got)
	}
	if !emailSender.contains("@Activity Mention") {
		t.Fatal("mention activity email did not render the mentioned user's display name")
	}
	var sentCount, processedCount int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE sent_at IS NOT NULL),COUNT(*) FILTER (WHERE processed_at IS NOT NULL) FROM email_notification_logs WHERE receiver_id=ANY($1::uuid[]) AND entity_identifier=$2`, []uuid.UUID{receiverID, mentionID}, issueID).Scan(&sentCount, &processedCount); scanErr != nil {
		t.Fatalf("count delivered email notification logs: %v", scanErr)
	}
	if sentCount != 3 || processedCount != 3 {
		t.Fatalf("email delivery status sent=%d processed=%d, want three each", sentCount, processedCount)
	}
	// Re-admitting a new Beat firing after success must observe sent_at and not
	// send the same logs a second time.
	secondEmailWorkflowID, err := emailDispatcher.Admit(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("admit repeated issue email scan: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, secondEmailWorkflowID)
	if got := emailSender.count(); got != 2 {
		t.Fatalf("repeated scan sent %d messages, want no duplicate delivery", got)
	}
	scheduleID, err := emailDispatcher.StartBeatSchedule(ctx)
	if err != nil {
		t.Fatalf("start durable five-minute email schedule: %v", err)
	}
	waitForEmailBeatTimer(t, ctx, pool, scheduleID)
	duplicateScheduleID, err := emailDispatcher.StartBeatSchedule(ctx)
	if err != nil {
		t.Fatalf("reconcile durable email schedule start: %v", err)
	}
	if duplicateScheduleID != scheduleID {
		t.Fatalf("duplicate schedule start returned %q, want stable ID %q", duplicateScheduleID, scheduleID)
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if shutdownErr := app.Shutdown(shutdownCtx); shutdownErr != nil {
		shutdownCancel()
		t.Fatalf("stop email schedule worker for restart test: %v", shutdownErr)
	}
	shutdownCancel()
	appStopped = true

	restartedApp := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-issue-activity-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	restartedEmailDispatcher, err := RegisterIssueEmailNotificationWorkflow(restartedApp, emailStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatalf("register restarted email worker: %v", err)
	}
	if startErr := restartedApp.Start(ctx); startErr != nil {
		t.Fatalf("restart email worker: %v", startErr)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if shutdownErr := restartedApp.Shutdown(shutdownCtx); shutdownErr != nil {
			t.Errorf("shutdown restarted email worker: %v", shutdownErr)
		}
	}()
	restartedScheduleID, err := restartedEmailDispatcher.StartBeatSchedule(ctx)
	if err != nil {
		t.Fatalf("recover durable email Beat schedule after worker restart: %v", err)
	}
	if restartedScheduleID != scheduleID {
		t.Fatalf("restarted schedule ID = %q, want %q", restartedScheduleID, scheduleID)
	}
	waitForEmailBeatTimer(t, ctx, pool, scheduleID)
}

func waitForEmailBeatTimer(t *testing.T, ctx context.Context, pool *pgxpool.Pool, scheduleID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM plane_execution.workflow_timer_subscriptions WHERE instance_id=$1`, scheduleID).Scan(&count); err != nil {
			t.Fatalf("inspect durable email Beat timer: %v", err)
		}
		if count == 1 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("wait for durable email Beat timer: %v", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("durable email Beat schedule did not arm a PostgreSQL timer")
}

type fixedIssueEmailConfig struct{}

func (fixedIssueEmailConfig) SMTPConfiguration(context.Context) (instancemail.SMTPConfig, string, error) {
	return instancemail.SMTPConfig{Host: "smtp.example.test", Port: 587}, "Plane <noreply@example.test>", nil
}

type recordingIssueEmailSender struct {
	mu            sync.Mutex
	messages      []instancemail.Message
	attemptCount  int
	failRemaining int
}

func (sender *recordingIssueEmailSender) Send(_ context.Context, _ instancemail.SMTPConfig, message instancemail.Message) error {
	if message.To == "" || message.Subject == "" || message.Body == "" || message.HTMLBody == "" {
		return errors.New("email message is incomplete")
	}
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.attemptCount++
	if sender.failRemaining > 0 {
		sender.failRemaining--
		return errors.New("temporary SMTP failure")
	}
	sender.messages = append(sender.messages, message)
	return nil
}

func (sender *recordingIssueEmailSender) count() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return len(sender.messages)
}

func (sender *recordingIssueEmailSender) attempts() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return sender.attemptCount
}

func (sender *recordingIssueEmailSender) contains(value string) bool {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	for _, message := range sender.messages {
		if strings.Contains(message.Body, value) || strings.Contains(message.HTMLBody, value) {
			return true
		}
	}
	return false
}
