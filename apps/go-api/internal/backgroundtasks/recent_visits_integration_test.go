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

func TestPostgresRecentVisitAdmissionRefreshesAndEvicts(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	userID, workspaceID, projectID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	username := "recent-visit-" + userID.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Recent Visit','Recent','Visit','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, userID, username, now); err != nil {
		t.Fatalf("insert recent visit user fixture: %v", err)
	}
	slug := "rv-" + userID.String()[:8]
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Recent Visit',$2,$3,'UTC','',$4,$4)`, workspaceID, userID, slug, now); err != nil {
		t.Fatalf("insert recent visit workspace fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Recent Visit','',0,$2,$3,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$4,$4)`, projectID, workspaceID, "RV"+userID.String()[:3], now); err != nil {
		t.Fatalf("insert recent visit project fixture: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_recent_visits WHERE user_id=$1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
	}()

	store, err := NewPostgresRecentVisitStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-recent-visits-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterRecentVisitWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
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
			t.Errorf("shutdown recent visit runtime: %v", shutdownErr)
		}
	}()

	entityID := uuid.NewString()
	input := RecentVisitInput{Slug: slug, EntityName: "issue", EntityIdentifier: &entityID, UserID: userID.String(), ProjectID: projectID.String()}
	eventID := uuid.NewString()
	instanceID, err := dispatcher.Admit(ctx, eventID, input)
	if err != nil {
		t.Fatalf("admit initial recent visit: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, instanceID)
	var visitID uuid.UUID
	var createdAt, visitedAt time.Time
	if scanErr := pool.QueryRow(ctx, `SELECT id,created_at,visited_at FROM user_recent_visits WHERE user_id=$1 AND entity_name='issue' AND entity_identifier=$2 AND project_id=$3 AND deleted_at IS NULL`, userID, entityID, projectID).Scan(&visitID, &createdAt, &visitedAt); scanErr != nil {
		t.Fatalf("read inserted recent visit: %v", scanErr)
	}
	// Replaying one event ID must reconcile the same workflow. A later identical
	// request receives a different ID and refreshes the existing row instead.
	if _, admitErr := dispatcher.Admit(ctx, eventID, input); admitErr != nil {
		t.Fatalf("reconcile repeated recent visit admission: %v", admitErr)
	}
	time.Sleep(2 * time.Millisecond)
	laterEventID := uuid.NewString()
	laterInstanceID, err := dispatcher.Admit(ctx, laterEventID, input)
	if err != nil {
		t.Fatalf("admit later identical recent visit: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, laterInstanceID)
	var refreshedCreatedAt, refreshedVisitedAt time.Time
	var visitCount int
	if scanErr := pool.QueryRow(ctx, `SELECT COUNT(*),MIN(created_at),MAX(visited_at) FROM user_recent_visits WHERE user_id=$1 AND entity_name='issue' AND entity_identifier=$2 AND project_id=$3 AND deleted_at IS NULL`, userID, entityID, projectID).Scan(&visitCount, &refreshedCreatedAt, &refreshedVisitedAt); scanErr != nil {
		t.Fatalf("read refreshed recent visit: %v", scanErr)
	}
	var sameID bool
	if scanErr := pool.QueryRow(ctx, `SELECT id=$1 FROM user_recent_visits WHERE user_id=$2 AND entity_name='issue' AND entity_identifier=$3 AND project_id=$4 AND deleted_at IS NULL`, visitID, userID, entityID, projectID).Scan(&sameID); scanErr != nil {
		t.Fatalf("verify stable recent visit identity: %v", scanErr)
	}
	if visitCount != 1 || !sameID || !refreshedCreatedAt.Equal(createdAt) || !refreshedVisitedAt.After(visitedAt) {
		t.Fatalf("refresh result count=%d same-id=%t created=%s visited=%s; want one stable row with refreshed visited_at", visitCount, sameID, refreshedCreatedAt, refreshedVisitedAt)
	}

	for index := 0; index < recentVisitLimit-1; index++ {
		otherID := uuid.New()
		otherInput := input
		otherInput.EntityIdentifier = stringPointer(otherID.String())
		otherInstanceID, admitErr := dispatcher.Admit(ctx, uuid.NewString(), otherInput)
		if admitErr != nil {
			t.Fatalf("admit capped visit %d: %v", index, admitErr)
		}
		waitVersionWorkflow(t, ctx, app, otherInstanceID)
	}
	newInput := input
	newInput.EntityIdentifier = stringPointer(uuid.NewString())
	newInstanceID, err := dispatcher.Admit(ctx, uuid.NewString(), newInput)
	if err != nil {
		t.Fatalf("admit visit over cap: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, newInstanceID)
	var liveCount int
	var oldestLive int
	var originalDeleted bool
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE deleted_at IS NULL),COUNT(*) FILTER (WHERE entity_identifier=$2 AND deleted_at IS NULL),BOOL_OR(id=$3 AND deleted_at IS NOT NULL) FROM user_recent_visits WHERE user_id=$1`, userID, *input.EntityIdentifier, visitID).Scan(&liveCount, &oldestLive, &originalDeleted); err != nil {
		t.Fatalf("verify capped recent visits: %v", err)
	}
	if liveCount != recentVisitLimit || oldestLive != 0 || !originalDeleted {
		t.Fatalf("cap result live=%d original-live=%d original-deleted=%t, want %d, 0, true", liveCount, oldestLive, originalDeleted, recentVisitLimit)
	}
}

func stringPointer(value string) *string { return &value }
