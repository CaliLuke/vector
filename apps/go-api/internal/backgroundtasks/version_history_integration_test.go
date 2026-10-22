//go:build integration

package backgroundtasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresVersionHistoryWorkflowsTrackAndCoalesce(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	pageID, pageOwner := uuid.New(), uuid.New()
	issueTaskOwner := uuid.New()
	pageMarker := "durable-page-version-" + pageID.String()
	issueID, issueOwner := uuid.New(), pageOwner
	issueBatchID := uuid.New()
	issueMarker := "durable-issue-version-" + issueID.String()
	workspaceID, projectID := uuid.New(), uuid.New()
	fixtureNow := time.Now().UTC()
	username := "version-history-" + pageOwner.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Version Test','Version','Test','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, pageOwner, username, fixtureNow); err != nil {
		t.Fatalf("insert version owner fixture: %v", err)
	}
	issueTaskUsername := "issue-version-task-" + issueTaskOwner.String()
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,'Version Task','Version','Task','',$3,$3,$3,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, issueTaskOwner, issueTaskUsername, fixtureNow); err != nil {
		t.Fatalf("insert issue version task owner fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Version Test',$2,$3,'UTC','',$4,$4)`, workspaceID, pageOwner, "vh-"+pageID.String()[:8], fixtureNow); err != nil {
		t.Fatalf("insert version workspace fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Version Test','',0,$2,$3,false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$4,$4)`, projectID, workspaceID, "VH"+pageID.String()[:3], fixtureNow); err != nil {
		t.Fatalf("insert version project fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO pages (id,workspace_id,name,description_json,description_html,owned_by_id,access,color,is_locked,view_props,logo_props,is_global,sort_order,created_at,updated_at) VALUES ($1,$2,'Version Test','{}'::jsonb,'<p>before</p>',$3,0,'',false,'{}'::jsonb,'{}'::jsonb,false,0,$4,$4)`, pageID, workspaceID, pageOwner, fixtureNow); err != nil {
		t.Fatalf("insert version page fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,'Version Test','{}'::jsonb,'<p>before</p>','medium',1,0,false,$5,$5)`, issueID, projectID, workspaceID, pageOwner, fixtureNow); err != nil {
		t.Fatalf("insert version issue fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO issues (id,project_id,workspace_id,created_by_id,name,description_json,description_html,priority,sequence_id,sort_order,is_draft,created_at,updated_at) VALUES ($1,$2,$3,$4,'Version Sync Batch','{}'::jsonb,'<p>sync</p>','medium',2,0,false,$5,$5)`, issueBatchID, projectID, workspaceID, pageOwner, fixtureNow.Add(time.Second)); err != nil {
		t.Fatalf("insert second issue version fixture: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM issue_description_versions WHERE issue_id=ANY($1::uuid[])`, []uuid.UUID{issueID, issueBatchID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM page_versions WHERE page_id=$1`, pageID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM issues WHERE id=ANY($1::uuid[])`, []uuid.UUID{issueID, issueBatchID})
		_, _ = pool.Exec(context.Background(), `DELETE FROM pages WHERE id=$1`, pageID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, issueTaskOwner)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, pageOwner)
	}()
	priorHTML := "<p>before</p>"
	if _, err := pool.Exec(ctx, `UPDATE pages SET description_html=$2,description_stripped=$3,updated_at=clock_timestamp() WHERE id=$1`, pageID, "<p>"+pageMarker+"</p>", pageMarker); err != nil {
		t.Fatalf("set page version fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE issues SET description_html=$2,description_stripped=$3,updated_at=clock_timestamp() WHERE id=$1`, issueID, "<p>"+issueMarker+"</p>", issueMarker); err != nil {
		t.Fatalf("set issue version fixture: %v", err)
	}

	store, err := NewPostgresVersionHistoryStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-version-history-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterVersionHistoryWorkflows(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	pageTransactionStore, err := NewPostgresPageTransactionStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	pageTransactionDispatcher, err := RegisterPageTransactionWorkflow(app, pageTransactionStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	syncStore, err := NewPostgresIssueDescriptionSyncStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	syncDispatcher, err := RegisterIssueDescriptionSyncWorkflow(app, syncStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	issueVersionStore, err := NewPostgresIssueVersionSyncStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	issueVersionDispatcher, err := RegisterIssueVersionSyncWorkflow(app, issueVersionStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	issueVersionTaskStore, err := NewPostgresIssueVersionTaskStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	issueVersionTaskDispatcher, err := RegisterIssueVersionTaskWorkflow(app, issueVersionTaskStore, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := app.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown version history runtime: %v", err)
		}
	}()

	pageSnapshot, _ := json.Marshal(map[string]string{"description_html": priorHTML})
	pageInput := PageVersionInput{PageID: pageID.String(), ExistingInstance: string(pageSnapshot), UserID: pageOwner.String()}
	pageInstance, err := dispatcher.AdmitPage(ctx, uuid.NewString(), pageInput)
	if err != nil {
		t.Fatalf("admit page version: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, pageInstance)
	pageCount, gotPageHTML, gotPageText := queryPageVersion(t, ctx, pool, pageID, pageOwner)
	if pageCount != 1 || gotPageHTML != "<p>"+pageMarker+"</p>" || gotPageText != pageMarker {
		t.Fatalf("page versions count=%d html=%q text=%q", pageCount, gotPageHTML, gotPageText)
	}
	firstMentionID, imageTransactionID, secondMentionID := uuid.New(), uuid.New(), uuid.New()
	firstHTML := `<mention-component id="` + firstMentionID.String() + `" entity_name="issue" entity_identifier="` + issueID.String() + `"></mention-component><image-component id="` + imageTransactionID.String() + `" src="` + projectID.String() + `"></image-component>`
	pageTransactionInput := PageTransactionInput{PageID: pageID.String(), NewDescriptionHTML: firstHTML, OldDescriptionHTML: nil}
	pageTransactionEventID := uuid.NewString()
	pageTransactionInstance, err := pageTransactionDispatcher.Admit(ctx, pageTransactionEventID, pageTransactionInput)
	if err != nil {
		t.Fatalf("admit page transaction logs: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, pageTransactionInstance)
	var pageLogCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM page_logs WHERE page_id=$1 AND "transaction"=ANY($2::uuid[])`, pageID, []uuid.UUID{firstMentionID, imageTransactionID}).Scan(&pageLogCount); err != nil {
		t.Fatal(err)
	}
	if pageLogCount != 2 {
		t.Fatalf("page component logs=%d, want mention and image", pageLogCount)
	}
	if _, err := pageTransactionDispatcher.Admit(ctx, pageTransactionEventID, pageTransactionInput); err != nil {
		t.Fatalf("reconcile repeated page transaction admission: %v", err)
	}
	secondHTML := `<mention-component id="` + secondMentionID.String() + `" entity_name="page" entity_identifier="` + pageID.String() + `"></mention-component>`
	secondTransactionID := uuid.NewString()
	secondInput := PageTransactionInput{PageID: pageID.String(), NewDescriptionHTML: secondHTML, OldDescriptionHTML: firstHTML}
	secondInstance, err := pageTransactionDispatcher.Admit(ctx, secondTransactionID, secondInput)
	if err != nil {
		t.Fatalf("admit page transaction update: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, secondInstance)
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM page_logs WHERE page_id=$1`, pageID).Scan(&pageLogCount); err != nil {
		t.Fatal(err)
	}
	if pageLogCount != 1 {
		t.Fatalf("page component logs after update=%d, want only the new mention", pageLogCount)
	}
	var savedEntityName string
	if err := pool.QueryRow(ctx, `SELECT entity_name FROM page_logs WHERE page_id=$1 AND "transaction"=$2`, pageID, secondMentionID).Scan(&savedEntityName); err != nil {
		t.Fatal(err)
	}
	if savedEntityName != "page" {
		t.Fatalf("page mention entity_name=%q, want page", savedEntityName)
	}
	pageReplay, err := dispatcher.AdmitPage(ctx, uuid.NewString(), pageInput)
	if err != nil {
		t.Fatalf("admit duplicate page snapshot: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, pageReplay)
	pageCount, _, _ = queryPageVersion(t, ctx, pool, pageID, pageOwner)
	if pageCount != 1 {
		t.Fatalf("replayed page snapshot created %d rows, want coalesced one", pageCount)
	}

	issueSnapshot, _ := json.Marshal(map[string]string{"description_html": priorHTML})
	issueInput := IssueDescriptionVersionInput{IssueID: issueID.String(), UpdatedIssue: string(issueSnapshot), UserID: issueOwner.String()}
	issueInstance, err := dispatcher.AdmitIssueDescription(ctx, uuid.NewString(), issueInput)
	if err != nil {
		t.Fatalf("admit issue description version: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, issueInstance)
	var issueCount int
	var gotIssueHTML, gotIssueText string
	if err := pool.QueryRow(ctx, `SELECT COUNT(*),MAX(description_html),MAX(description_stripped) FROM issue_description_versions WHERE issue_id=$1 AND owned_by_id=$2`, issueID, issueOwner).Scan(&issueCount, &gotIssueHTML, &gotIssueText); err != nil {
		t.Fatal(err)
	}
	if issueCount != 1 || gotIssueHTML != "<p>"+issueMarker+"</p>" || gotIssueText != issueMarker {
		t.Fatalf("issue description versions count=%d html=%q text=%q", issueCount, gotIssueHTML, gotIssueText)
	}
	issueReplay, err := dispatcher.AdmitIssueDescription(ctx, uuid.NewString(), issueInput)
	if err != nil {
		t.Fatalf("admit duplicate issue snapshot: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, issueReplay)
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_description_versions WHERE issue_id=$1 AND owned_by_id=$2`, issueID, issueOwner).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if issueCount != 1 {
		t.Fatalf("replayed issue snapshot created %d rows, want coalesced one", issueCount)
	}
	versionAdmission, err := NewIssueDescriptionVersionAdmission(dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	operationID := uuid.NewString()
	if err := versionAdmission.AdmitIssueDescriptionVersion(ctx, operationID, string(issueSnapshot), issueID.String(), issueOwner.String(), false); err != nil {
		t.Fatalf("admit issue description through native producer adapter: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, "issue-description-version:"+operationID)
	if err := versionAdmission.AdmitIssueDescriptionVersion(ctx, operationID, string(issueSnapshot), issueID.String(), issueOwner.String(), false); err != nil {
		t.Fatalf("reconcile repeated native producer admission: %v", err)
	}
	secondOperationID := uuid.NewString()
	if secondOperationID == operationID {
		t.Fatal("distinct operation IDs must not collide")
	}
	if err := versionAdmission.AdmitIssueDescriptionVersion(ctx, secondOperationID, string(issueSnapshot), issueID.String(), issueOwner.String(), false); err != nil {
		t.Fatalf("admit second identical native operation: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, "issue-description-version:"+secondOperationID)
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_description_versions WHERE issue_id=$1 AND owned_by_id=$2`, issueID, issueOwner).Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if issueCount != 1 {
		t.Fatalf("native producer replay created %d rows, want one coalesced row", issueCount)
	}

	syncRunID := uuid.NewString()
	var totalIssues int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issues WHERE deleted_at IS NULL`).Scan(&totalIssues); err != nil {
		t.Fatal(err)
	}
	syncInstance, err := syncDispatcher.Admit(ctx, syncRunID, 1, 0, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("admit delayed issue description sync: %v", err)
	}
	syncInstanceDuplicate, err := syncDispatcher.Admit(ctx, syncRunID, 1, 0, 20*time.Millisecond)
	if err != nil || syncInstanceDuplicate != syncInstance {
		t.Fatalf("reconcile duplicate sync admission id=%q want=%q err=%v", syncInstanceDuplicate, syncInstance, err)
	}
	syncResult, err := selvedge.WaitWorkflow[IssueDescriptionSyncOutput](ctx, app, syncInstance)
	if err != nil {
		t.Fatalf("wait for issue description sync: %v", err)
	}
	if syncResult.Status != "completed" || syncResult.Output.Batches != totalIssues || syncResult.Output.Inserted+syncResult.Output.Skipped != totalIssues {
		t.Fatalf("issue description sync status=%q output=%+v error=%v", syncResult.Status, syncResult.Output, syncResult.Error)
	}
	for _, currentIssueID := range []uuid.UUID{issueID, issueBatchID} {
		versionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(syncRunID+":"+currentIssueID.String()))
		var versions int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM issue_description_versions WHERE id=$1 AND issue_id=$2 AND owned_by_id=$3`, versionID, currentIssueID, issueOwner).Scan(&versions); err != nil {
			t.Fatal(err)
		}
		if versions != 1 {
			t.Fatalf("sync version for issue %s count=%d, want one", currentIssueID, versions)
		}
	}

	issueVersionRunID := uuid.NewString()
	issueVersionSyncInstance, err := issueVersionDispatcher.Admit(ctx, issueVersionRunID, 1, 0, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("admit delayed issue version sync: %v", err)
	}
	issueVersionResult, err := selvedge.WaitWorkflow[IssueVersionSyncOutput](ctx, app, issueVersionSyncInstance)
	if err != nil {
		t.Fatalf("wait for issue version sync: %v", err)
	}
	if issueVersionResult.Status != "completed" || issueVersionResult.Output.Batches != totalIssues || issueVersionResult.Output.Inserted+issueVersionResult.Output.Skipped != totalIssues {
		t.Fatalf("issue version sync status=%q output=%+v error=%v", issueVersionResult.Status, issueVersionResult.Output, issueVersionResult.Error)
	}
	for currentIssueID, expectedName := range map[uuid.UUID]string{issueID: "Version Test", issueBatchID: "Version Sync Batch"} {
		versionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(issueVersionRunID+":"+currentIssueID.String()))
		var name, priority string
		var assignees, labels, modules []uuid.UUID
		var owner uuid.UUID
		if err := pool.QueryRow(ctx, `SELECT name,priority,owned_by_id,assignees,labels,modules FROM issue_versions WHERE id=$1 AND issue_id=$2`, versionID, currentIssueID).Scan(&name, &priority, &owner, &assignees, &labels, &modules); err != nil {
			t.Fatal(err)
		}
		if name != expectedName || priority != "medium" || owner != pageOwner || len(assignees) != 0 || len(labels) != 0 || len(modules) != 0 {
			t.Fatalf("issue version %s name=%q priority=%q owner=%s assignees=%v labels=%v modules=%v", currentIssueID, name, priority, owner, assignees, labels, modules)
		}
	}

	issueTaskEvent := uuid.NewString()
	issueTaskInput := IssueVersionTaskInput{UpdatedIssue: `{"name":"Version Test before edit"}`, IssueID: issueID.String(), UserID: issueTaskOwner.String()}
	issueTaskInstance, err := issueVersionTaskDispatcher.Admit(ctx, issueTaskEvent, issueTaskInput)
	if err != nil {
		t.Fatalf("admit issue version coalescing task: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, issueTaskInstance)
	duplicateInstance, err := issueVersionTaskDispatcher.Admit(ctx, issueTaskEvent, issueTaskInput)
	if err != nil || duplicateInstance != issueTaskInstance {
		t.Fatalf("reconcile issue version task retry id=%q want=%q err=%v", duplicateInstance, issueTaskInstance, err)
	}
	issueTaskChangedInput := IssueVersionTaskInput{UpdatedIssue: `{"priority":"low"}`, IssueID: issueID.String(), UserID: issueTaskOwner.String()}
	issueTaskChangedEvent := uuid.NewString()
	issueTaskChangedInstance, err := issueVersionTaskDispatcher.Admit(ctx, issueTaskChangedEvent, issueTaskChangedInput)
	if err != nil {
		t.Fatalf("admit second issue version coalescing task: %v", err)
	}
	waitVersionWorkflow(t, ctx, app, issueTaskChangedInstance)
	var issueTaskVersionCount int
	var issueTaskName, issueTaskPriority string
	var issueTaskProperties, issueTaskMeta []byte
	var issueTaskActivityIsNull bool
	if err := pool.QueryRow(ctx, `SELECT COUNT(*),MAX(name),MAX(priority),MAX(properties::text)::jsonb,MAX(meta::text)::jsonb,BOOL_AND(activity_id IS NULL) FROM issue_versions WHERE issue_id=$1 AND owned_by_id=$2`, issueID, issueTaskOwner).Scan(&issueTaskVersionCount, &issueTaskName, &issueTaskPriority, &issueTaskProperties, &issueTaskMeta, &issueTaskActivityIsNull); err != nil {
		t.Fatal(err)
	}
	if issueTaskVersionCount != 1 || issueTaskName != "Version Test" || issueTaskPriority != "low" || string(issueTaskProperties) != "{}" || string(issueTaskMeta) != "{}" || !issueTaskActivityIsNull {
		t.Fatalf("coalesced issue version rows=%d name=%q priority=%q properties=%s meta=%s activity_is_null=%t", issueTaskVersionCount, issueTaskName, issueTaskPriority, issueTaskProperties, issueTaskMeta, issueTaskActivityIsNull)
	}
}

func waitVersionWorkflow(t *testing.T, ctx context.Context, app *selvedge.App, instanceID string) {
	t.Helper()
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, instanceID)
	if err != nil {
		t.Fatalf("wait for workflow %q: %v", instanceID, err)
	}
	if result.Status != "completed" {
		t.Fatalf("workflow %q status=%q error=%s", instanceID, result.Status, result.Error)
	}
}

func queryPageVersion(t *testing.T, ctx context.Context, pool *pgxpool.Pool, pageID, userID uuid.UUID) (int, string, string) {
	t.Helper()
	var count int
	var html, text string
	err := pool.QueryRow(ctx, `SELECT COUNT(*),MAX(description_html),MAX(description_stripped) FROM page_versions WHERE page_id=$1 AND owned_by_id=$2`, pageID, userID).Scan(&count, &html, &text)
	if err != nil {
		t.Fatalf("query page version: %v", err)
	}
	return count, html, text
}
