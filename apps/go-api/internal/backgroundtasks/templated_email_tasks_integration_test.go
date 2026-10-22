//go:build integration

package backgroundtasks

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

func TestPostgresTemplatedEmailWorkflowRetriesAndDeliversSMTP(t *testing.T) {
	_, executionURL, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	smtp := startRetryOnceEmailSMTP(t)
	store, err := NewPostgresTemplatedEmailStore(pool, fixedSMTPConfig{config: instancemail.SMTPConfig{Host: "127.0.0.1", Port: smtp.port}, from: "Plane <noreply@example.test>"}, instancemail.Sender{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	app := selvedge.NewApp(
		selvedge.WithDatabase(executionURL),
		selvedge.WithServiceName("plane-templated-email-integration-test"),
		selvedge.WithActivityPollInterval(20*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(20*time.Millisecond),
		selvedge.WithTimerCheckInterval(20*time.Millisecond),
	)
	dispatcher, err := RegisterTemplatedEmailTaskWorkflow(app, store, retry.Exponential(3, 10*time.Millisecond, 30*time.Millisecond, 2))
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
			t.Errorf("shutdown templated email runtime: %v", err)
		}
	}()

	admission, err := NewTemplatedEmailTaskAdmission(dispatcher)
	if err != nil {
		t.Fatal(err)
	}
	eventID := uuid.NewString()
	if err := admission.AdmitTemplatedEmailTask(ctx, eventID, magicLinkTask, []any{"person@example.test", "opaque-key", "314159"}); err != nil {
		t.Fatalf("admit magic-link email: %v", err)
	}
	instanceID := "plane-email-task:" + eventID
	result, err := selvedge.WaitWorkflow[struct{}](ctx, app, instanceID)
	if err != nil {
		t.Fatalf("wait for SMTP delivery workflow: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("templated email workflow status=%q error=%v", result.Status, result.Error)
	}
	if got := smtp.attempts(); got != 2 {
		t.Fatalf("SMTP delivery attempts=%d, want first transient rejection plus successful retry", got)
	}
	message := smtp.acceptedMessage()
	if !strings.Contains(message, "Your unique Vector login code is 314159") || !strings.Contains(message, "314159") || !strings.Contains(message, "person@example.test") {
		t.Fatalf("delivered SMTP message omitted rendered subject/body/recipient: %s", message)
	}
	plainBody, err := extractEmailPlainText(message)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plainBody, "font-family") || strings.Contains(plainBody, "<style") {
		t.Fatalf("plain text MIME part retained CSS: %s", plainBody)
	}
}

func TestPostgresTemplatedEmailPreparationUsesPlaneRows(t *testing.T) {
	_, _, pool := openBackgroundTaskDatabase(t)
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ownerID, memberID, workspaceID, projectID, projectMemberID, workspaceInviteID, projectInviteID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now().UTC()
	ownerName := "mail-owner-" + ownerID.String()
	memberName := "mail-member-" + memberID.String()
	ownerEmail := ownerName + "@example.test"
	memberEmail := memberName + "@example.test"
	workspaceSlug := "mail-" + ownerID.String()[:8]
	workspaceToken, projectToken := "workspace-"+workspaceID.String(), "project-"+projectID.String()
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspace_member_invites WHERE id=$1`, workspaceInviteID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_member_invites WHERE id=$1`, projectInviteID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM project_members WHERE id=$1`, projectMemberID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM projects WHERE id=$1`, projectID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM workspaces WHERE id=$1`, workspaceID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=ANY($1::uuid[])`, []uuid.UUID{ownerID, memberID})
	}()
	for _, user := range []struct {
		id                                      uuid.UUID
		username, email, firstName, displayName string
	}{
		{ownerID, ownerName, ownerEmail, "Owner", "Workspace Owner"},
		{memberID, memberName, memberEmail, "Member", "Project Member"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO users (id,password,username,email,display_name,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,is_password_reset_required,token,last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,user_timezone,is_email_valid) VALUES ($1,'!', $2,$3,$4,$5,'Test','',$6,$6,$6,'','',false,false,false,true,false,true,false,false,$2,'','','','',false,'UTC',true)`, user.id, user.username, user.email, user.displayName, user.firstName, now); err != nil {
			t.Fatalf("insert templated email user %s: %v", user.username, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES ($1,'Email Workspace',$2,$3,'UTC','',$4,$4)`, workspaceID, ownerID, workspaceSlug, now); err != nil {
		t.Fatalf("insert email workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id,name,description,network,workspace_id,identifier,module_view,cycle_view,issue_views_view,page_view,intake_view,is_time_tracking_enabled,is_issue_type_enabled,guest_view_all_features,archive_in,close_in,logo_props,timezone,created_at,updated_at) VALUES ($1,'Email Project','',0,$2,'MAIL',false,false,false,true,false,false,false,false,0,0,'{}'::jsonb,'UTC',$3,$3)`, projectID, workspaceID, now); err != nil {
		t.Fatalf("insert email project: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id,workspace_id,project_id,member_id,role,view_props,default_props,sort_order,preferences,is_active,created_at,updated_at,deleted_at) VALUES ($1,$2,$3,$4,15,'{}'::jsonb,'{}'::jsonb,0,'{}'::jsonb,true,$5,$5,NULL)`, projectMemberID, workspaceID, projectID, memberID, now); err != nil {
		t.Fatalf("insert email project member: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO workspace_member_invites (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,email,accepted,token,message,responded_at,role) VALUES ($1,$2,$2,NULL,$3,$3,$4,$5,false,$6,NULL,NULL,5)`, workspaceInviteID, now, ownerID, workspaceID, memberEmail, workspaceToken); err != nil {
		t.Fatalf("insert workspace invite: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_member_invites (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,email,accepted,token,message,responded_at,role) VALUES ($1,$2,$2,NULL,$3,$3,$4,$5,$6,false,$7,NULL,NULL,5)`, projectInviteID, now, ownerID, projectID, workspaceID, memberEmail, projectToken); err != nil {
		t.Fatalf("insert project invite: %v", err)
	}
	store, err := NewPostgresTemplatedEmailStore(pool, fixedSMTPConfig{config: instancemail.SMTPConfig{Host: "smtp.example.test", Port: 587}, from: "Plane <noreply@example.test>"}, instancemail.Sender{})
	if err != nil {
		t.Fatal(err)
	}
	prepare := func(task string, args ...string) *preparedEmail {
		t.Helper()
		encoded, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		var rawArgs []json.RawMessage
		if err := json.Unmarshal(encoded, &rawArgs); err != nil {
			t.Fatal(err)
		}
		prepared, err := store.prepare(ctx, templatedEmailTaskInput{EventID: uuid.NewString(), TaskName: task, Args: rawArgs})
		if err != nil {
			t.Fatalf("prepare %s: %v", task, err)
		}
		return prepared
	}
	projectAddition := prepare(projectAddUserEmailTask, "https://plane.example.test", projectMemberID.String(), ownerID.String())
	if projectAddition.To != memberEmail || projectAddition.Subject != "You have been invited to a Vector project" || projectAddition.Values["project_url"] != "https://plane.example.test/"+workspaceSlug+"/projects/"+projectID.String()+"/issues" {
		t.Fatalf("project addition email = %+v", projectAddition)
	}
	workspaceInvitation := prepare(workspaceInvitationEmailTask, memberEmail, workspaceID.String(), workspaceToken, "https://plane.example.test", ownerEmail)
	if workspaceInvitation.To != memberEmail || workspaceInvitation.Subject != "Owner has invited you to join them in Email Workspace on Vector" {
		t.Fatalf("workspace invitation email = %+v", workspaceInvitation)
	}
	var workspaceMessage string
	if err := pool.QueryRow(ctx, `SELECT message FROM workspace_member_invites WHERE id=$1`, workspaceInviteID).Scan(&workspaceMessage); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(workspaceMessage, "Email Workspace") || strings.Contains(workspaceMessage, "<html") {
		t.Fatalf("workspace invite message = %q", workspaceMessage)
	}
	projectInvitation := prepare(projectInvitationEmailTask, memberEmail, projectID.String(), projectToken, "https://plane.example.test", ownerEmail)
	if projectInvitation.To != memberEmail || projectInvitation.Subject != "Owner invited you to join Email Project on Vector" {
		t.Fatalf("project invitation email = %+v", projectInvitation)
	}
	var projectMessage string
	if err := pool.QueryRow(ctx, `SELECT message FROM project_member_invites WHERE id=$1`, projectInviteID).Scan(&projectMessage); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(projectMessage, "Email Project") || strings.Contains(projectMessage, "<html") {
		t.Fatalf("project invite message = %q", projectMessage)
	}
	activation := prepare(userActivationEmailTask, "https://plane.example.test", memberID.String())
	if activation.To != memberEmail || activation.Subject != "Member has been activated on Vector" {
		t.Fatalf("activation email = %+v", activation)
	}
	deactivation := prepare(userDeactivationEmailTask, "https://plane.example.test", memberID.String())
	if deactivation.To != memberEmail || deactivation.Subject != "Member has been deactivated on Vector" {
		t.Fatalf("deactivation email = %+v", deactivation)
	}
}

func extractEmailPlainText(raw string) (string, error) {
	message, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return "", err
	}
	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		return "", fmt.Errorf("expected multipart email, got %q", mediaType)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(strings.ToLower(part.Header.Get("Content-Type")), "text/plain") {
			decoded, err := io.ReadAll(quotedprintable.NewReader(part))
			_ = part.Close()
			return string(decoded), err
		}
		_ = part.Close()
	}
	return "", errors.New("email did not include a text/plain MIME part")
}

type fixedSMTPConfig struct {
	config instancemail.SMTPConfig
	from   string
}

func (config fixedSMTPConfig) SMTPConfiguration(context.Context) (instancemail.SMTPConfig, string, error) {
	return config.config, config.from, nil
}

type retryOnceEmailSMTP struct {
	listener net.Listener
	port     int
	mu       sync.Mutex
	count    int
	accepted string
}

func startRetryOnceEmailSMTP(t *testing.T) *retryOnceEmailSMTP {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	sink := &retryOnceEmailSMTP{listener: listener, port: listener.Addr().(*net.TCPAddr).Port}
	t.Cleanup(func() { _ = listener.Close() })
	go sink.serve()
	return sink
}

func (sink *retryOnceEmailSMTP) serve() {
	for {
		connection, err := sink.listener.Accept()
		if err != nil {
			return
		}
		sink.handle(connection)
	}
}

func (sink *retryOnceEmailSMTP) handle(connection net.Conn) {
	defer connection.Close()
	reader := bufio.NewReader(connection)
	_, _ = fmt.Fprint(connection, "220 localhost ESMTP test sink\r\n")
	var body strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			_, _ = fmt.Fprint(connection, "250-localhost\r\n250 SIZE 1000000\r\n")
		case strings.HasPrefix(command, "MAIL FROM:"):
			_, _ = fmt.Fprint(connection, "250 2.1.0 sender ok\r\n")
		case strings.HasPrefix(command, "RCPT TO:"):
			_, _ = fmt.Fprint(connection, "250 2.1.5 recipient ok\r\n")
		case command == "DATA":
			_, _ = fmt.Fprint(connection, "354 end with dot\r\n")
			body.Reset()
			for {
				part, readErr := reader.ReadString('\n')
				if readErr != nil {
					return
				}
				if strings.TrimSpace(part) == "." {
					break
				}
				body.WriteString(part)
			}
			sink.mu.Lock()
			sink.count++
			count := sink.count
			if count > 1 {
				sink.accepted = body.String()
			}
			sink.mu.Unlock()
			if count == 1 {
				_, _ = fmt.Fprint(connection, "451 4.3.0 temporary failure\r\n")
			} else {
				_, _ = fmt.Fprint(connection, "250 2.0.0 accepted\r\n")
			}
		case command == "RSET", command == "NOOP":
			_, _ = fmt.Fprint(connection, "250 2.0.0 ok\r\n")
		case command == "QUIT":
			_, _ = fmt.Fprint(connection, "221 2.0.0 bye\r\n")
			return
		default:
			_, _ = fmt.Fprint(connection, "250 2.0.0 ok\r\n")
		}
	}
}

func (sink *retryOnceEmailSMTP) attempts() int {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.count
}

func (sink *retryOnceEmailSMTP) acceptedMessage() string {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.accepted
}
