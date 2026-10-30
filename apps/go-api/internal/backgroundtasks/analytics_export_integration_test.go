//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
)

func TestPostgresAnalyticsExportBuildsAndEmailsCSV(t *testing.T) {
	databaseURL, executionURL, pool := openBackgroundTaskDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID, workspaceID, projectID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	slug := "analytics-export-" + userID.String()[:8]
	username := "analytics-export-" + userID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!',$2,'Analytics Actor','Analytics','Actor','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, userID, username, now); err != nil {
		t.Fatalf("insert analytics actor: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Analytics Export',$2,$3,'UTC','',$4,$4)`, workspaceID, userID, slug, now); err != nil {
		t.Fatalf("insert analytics workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Analytics Project','',0,$2,'ANL',false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$3,$3)`, projectID, workspaceID, now); err != nil {
		t.Fatalf("insert analytics project: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,'=Injected label','{}'::jsonb,'<p>report</p>','high',27,0,false,$5,$5)`, issueID, projectID, workspaceID, userID, now); err != nil {
		t.Fatalf("insert analytics issue: %v", err)
	}

	config := fixedAnalyticsSMTPConfig{}
	sender := &analyticsMessageRecorder{}
	store, err := NewPostgresAnalyticsExportStore(pool, config, sender)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-analytics-export-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterAnalyticsExportWorkflow(app, store, nil)
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
			t.Errorf("shutdown analytics export test runtime for %s: %v", databaseURL, shutdownErr)
		}
	}()
	data, _ := json.Marshal(map[string]any{"x_axis": "priority", "y_axis": "issue_count"})
	workflowID, err := dispatcher.Admit(ctx, AnalyticsExportInput{EventID: uuid.NewString(), Email: "actor@example.test", Slug: slug, Data: data})
	if err != nil {
		t.Fatalf("admit analytics export workflow: %v", err)
	}
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, workflowID)
	if err != nil {
		t.Fatalf("wait for analytics export workflow: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("analytics export workflow status=%q, error=%v", result.Status, result.Error)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent %d messages, want one", len(sender.messages))
	}
	message := sender.messages[0]
	if message.To != "actor@example.test" || message.Subject != "Your Export is ready" || len(message.Attachments) != 1 {
		t.Fatalf("sent analytics email = %#v", message)
	}
	attachment := message.Attachments[0]
	if attachment.Name != slug+"-analytics.csv" || attachment.ContentType != "text/csv; charset=utf-8" {
		t.Fatalf("analytics CSV attachment = %#v", attachment)
	}
	csv := string(attachment.Data)
	if !strings.Contains(csv, `"Priority","Issue Count"`) || !strings.Contains(csv, `"high","1"`) {
		t.Fatalf("analytics CSV missing expected grouped row: %q", csv)
	}
}

type fixedAnalyticsSMTPConfig struct{}

func (fixedAnalyticsSMTPConfig) SMTPConfiguration(context.Context) (instancemail.SMTPConfig, string, error) {
	return instancemail.SMTPConfig{Host: "smtp.example.test", Port: 25}, "sender@example.test", nil
}

type analyticsMessageRecorder struct{ messages []instancemail.Message }

func (recorder *analyticsMessageRecorder) Send(_ context.Context, _ instancemail.SMTPConfig, message instancemail.Message) error {
	recorder.messages = append(recorder.messages, message)
	return nil
}
