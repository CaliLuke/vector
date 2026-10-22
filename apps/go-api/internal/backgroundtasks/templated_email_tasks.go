package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	forgotPasswordTask           = "plane.bgtasks.forgot_password_task.forgot_password"
	magicLinkTask                = "plane.bgtasks.magic_link_code_task.magic_link"
	emailUpdateMagicCodeTask     = "plane.bgtasks.user_email_update_task.send_email_update_magic_code"
	emailUpdateConfirmationTask  = "plane.bgtasks.user_email_update_task.send_email_update_confirmation"
	userActivationEmailTask      = "plane.bgtasks.user_activation_email_task.user_activation_email"
	userDeactivationEmailTask    = "plane.bgtasks.user_deactivation_email_task.user_deactivation_email"
	projectAddUserEmailTask      = "plane.bgtasks.project_add_user_email_task.project_add_user_email"
	webhookDeactivationEmailTask = "plane.bgtasks.webhook_task.send_webhook_deactivation_email"
	workspaceInvitationEmailTask = "plane.bgtasks.workspace_invitation_task.workspace_invitation"
	projectInvitationEmailTask   = "plane.bgtasks.project_invitation_task.project_invitation"
	templatedEmailWorkflowName   = "plane.notifications.send_templated_email"
	templatedEmailActivityName   = "plane.notifications.render_and_send_templated_email"
)

var supportedTemplatedEmailTasks = map[string]struct{}{
	forgotPasswordTask: {}, magicLinkTask: {}, emailUpdateMagicCodeTask: {}, emailUpdateConfirmationTask: {},
	userActivationEmailTask: {}, userDeactivationEmailTask: {}, projectAddUserEmailTask: {}, webhookDeactivationEmailTask: {},
	workspaceInvitationEmailTask: {}, projectInvitationEmailTask: {},
}

type TemplatedEmailTaskDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[templatedEmailTaskInput, struct{}]
}

type TemplatedEmailTaskAdmission struct{ dispatcher *TemplatedEmailTaskDispatcher }

type TemplatedEmailTaskAdmitter interface {
	AdmitTemplatedEmailTask(context.Context, string, string, []any) error
}

type templatedEmailTaskInput struct {
	EventID  string            `json:"event_id"`
	TaskName string            `json:"task_name"`
	Args     []json.RawMessage `json:"args"`
}

type templatedEmailStore struct {
	pool   *pgxpool.Pool
	config issueEmailSMTPConfigProvider
	sender issueEmailSender
}

type preparedEmail struct {
	To, Subject, Template string
	Values                map[string]string
}

func NewPostgresTemplatedEmailStore(pool *pgxpool.Pool, config issueEmailSMTPConfigProvider, sender issueEmailSender) (*templatedEmailStore, error) {
	if pool == nil || config == nil || sender == nil {
		return nil, errors.New("templated email PostgreSQL pool, SMTP configuration and sender are required")
	}
	return &templatedEmailStore{pool: pool, config: config, sender: sender}, nil
}

func RegisterTemplatedEmailTaskWorkflow(app *selvedge.App, store *templatedEmailStore, policy *retry.Policy) (*TemplatedEmailTaskDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and templated email store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[templatedEmailTaskInput, struct{}]{Name: templatedEmailActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input templatedEmailTaskInput) (struct{}, error) {
		if _, err := uuid.Parse(input.EventID); err != nil {
			return struct{}{}, fmt.Errorf("invalid templated email event identity: %w", err)
		}
		if err := store.Deliver(ctx, input); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register templated email activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(templatedEmailWorkflowName, func(ctx *selvedge.WorkflowContext, input templatedEmailTaskInput) (struct{}, error) {
		if _, ok := supportedTemplatedEmailTasks[input.TaskName]; !ok {
			return struct{}{}, fmt.Errorf("unsupported templated email task %q", input.TaskName)
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &TemplatedEmailTaskDispatcher{app: app, workflow: workflow}, nil
}

func NewTemplatedEmailTaskAdmission(dispatcher *TemplatedEmailTaskDispatcher) (*TemplatedEmailTaskAdmission, error) {
	if dispatcher == nil {
		return nil, errors.New("templated email task dispatcher is required")
	}
	return &TemplatedEmailTaskAdmission{dispatcher: dispatcher}, nil
}

func (admission *TemplatedEmailTaskAdmission) AdmitTemplatedEmailTask(ctx context.Context, eventID, taskName string, args []any) error {
	if admission == nil || admission.dispatcher == nil {
		return errors.New("native templated email admission is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return fmt.Errorf("templated email event ID must be a UUID: %w", err)
	}
	if _, ok := supportedTemplatedEmailTasks[taskName]; !ok {
		return fmt.Errorf("unsupported templated email task %q", taskName)
	}
	encoded, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("encode %s arguments: %w", taskName, err)
	}
	var rawArgs []json.RawMessage
	if err := json.Unmarshal(encoded, &rawArgs); err != nil {
		return fmt.Errorf("decode %s arguments: %w", taskName, err)
	}
	input := templatedEmailTaskInput{EventID: eventID, TaskName: taskName, Args: rawArgs}
	if _, err := admitOrReconcile(ctx, admission.dispatcher.app, admission.dispatcher.workflow, "plane-email-task:"+eventID, input); err != nil {
		return fmt.Errorf("admit %s workflow: %w", taskName, err)
	}
	return nil
}

// TemplatedEmailTaskPublisher routes the selected source mail tasks to durable
// execution while retaining the current publisher for all other tasks.
type TemplatedEmailTaskPublisher struct {
	fallback  TaskFallback
	admission TemplatedEmailTaskAdmitter
}

func NewTemplatedEmailTaskPublisher(fallback TaskFallback, admission TemplatedEmailTaskAdmitter) (*TemplatedEmailTaskPublisher, error) {
	if fallback == nil || admission == nil {
		return nil, errors.New("task fallback and templated email admission are required")
	}
	return &TemplatedEmailTaskPublisher{fallback: fallback, admission: admission}, nil
}

func (publisher *TemplatedEmailTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.admission == nil {
		return errors.New("templated email task publisher is not configured")
	}
	if _, ok := supportedTemplatedEmailTasks[name]; !ok {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	if len(kwargs) != 0 {
		return fmt.Errorf("templated email task %s does not accept keyword arguments", name)
	}
	return publisher.admission.AdmitTemplatedEmailTask(ctx, uuid.NewString(), name, args)
}

func (publisher *TemplatedEmailTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("templated email task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func (store *templatedEmailStore) Deliver(ctx context.Context, input templatedEmailTaskInput) error {
	prepared, err := store.prepare(ctx, input)
	if err != nil {
		if errors.Is(err, errEmailTaskSourceRowMissing) {
			return nil
		}
		return err
	}
	if prepared == nil {
		return nil
	}
	config, from, err := store.config.SMTPConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("load SMTP configuration for %s: %w", input.TaskName, err)
	}
	if config.Host == "" || strings.TrimSpace(prepared.To) == "" {
		return fmt.Errorf("SMTP host and recipient are required for %s", input.TaskName)
	}
	htmlBody, textBody, err := renderPlaneEmail(prepared.Template, prepared.Values)
	if err != nil {
		return err
	}
	subject := stripEmailControls(prepared.Subject)
	if err := store.sender.Send(ctx, config, instancemail.Message{From: from, To: prepared.To, Subject: subject, Body: textBody, HTMLBody: htmlBody}); err != nil {
		return fmt.Errorf("send %s email: %w", input.TaskName, err)
	}
	return nil
}

var errEmailTaskSourceRowMissing = errors.New("email source row is missing")

func (store *templatedEmailStore) prepare(ctx context.Context, input templatedEmailTaskInput) (*preparedEmail, error) {
	args := input.Args
	arg := func(index int) (string, error) {
		if index >= len(args) {
			return "", fmt.Errorf("%s requires argument %d", input.TaskName, index+1)
		}
		if string(args[index]) == "null" {
			return "", nil
		}
		var value string
		if err := json.Unmarshal(args[index], &value); err != nil {
			return "", fmt.Errorf("%s argument %d must be a string: %w", input.TaskName, index+1, err)
		}
		return value, nil
	}
	exact := func(count int) error {
		if len(args) != count {
			return fmt.Errorf("%s received %d arguments, want %d", input.TaskName, len(args), count)
		}
		return nil
	}
	makeEmail := func(to, subject, template string, values map[string]string) *preparedEmail {
		return &preparedEmail{To: to, Subject: subject, Template: template, Values: values}
	}
	switch input.TaskName {
	case forgotPasswordTask:
		if err := exact(5); err != nil {
			return nil, err
		}
		firstName, err := arg(0)
		if err != nil {
			return nil, err
		}
		email, err := arg(1)
		if err != nil {
			return nil, err
		}
		uid, err := arg(2)
		if err != nil {
			return nil, err
		}
		token, err := arg(3)
		if err != nil {
			return nil, err
		}
		currentSite, err := arg(4)
		if err != nil {
			return nil, err
		}
		absURL := currentSite + "/accounts/reset-password/?uidb64=" + uid + "&token=" + token + "&email=" + email
		return makeEmail(email, "A new password to your Vector account has been requested", "auth/forgot_password.html", map[string]string{"first_name": firstName, "forgot_password_url": absURL, "email": email}), nil
	case magicLinkTask:
		if err := exact(3); err != nil {
			return nil, err
		}
		email, err := arg(0)
		if err != nil {
			return nil, err
		}
		_, err = arg(1)
		if err != nil {
			return nil, err
		}
		token, err := arg(2)
		if err != nil {
			return nil, err
		}
		return makeEmail(email, "Your unique Vector login code is "+token, "auth/magic_signin.html", map[string]string{"code": token, "email": email}), nil
	case emailUpdateMagicCodeTask:
		if err := exact(2); err != nil {
			return nil, err
		}
		email, err := arg(0)
		if err != nil {
			return nil, err
		}
		token, err := arg(1)
		if err != nil {
			return nil, err
		}
		return makeEmail(email, "Verify your new email address", "auth/magic_signin.html", map[string]string{"code": token, "email": email}), nil
	case emailUpdateConfirmationTask:
		if err := exact(1); err != nil {
			return nil, err
		}
		email, err := arg(0)
		if err != nil {
			return nil, err
		}
		return makeEmail(email, "Vector email address successfully updated", "user/email_updated.html", map[string]string{"email": email}), nil
	case userActivationEmailTask, userDeactivationEmailTask:
		if err := exact(2); err != nil {
			return nil, err
		}
		currentSite, err := arg(0)
		if err != nil {
			return nil, err
		}
		userID, err := arg(1)
		if err != nil {
			return nil, err
		}
		user, err := store.userForEmail(ctx, userID)
		if err != nil {
			return nil, err
		}
		if input.TaskName == userActivationEmailTask {
			return makeEmail(user.email, user.displayName+" has been activated on Vector", "user/user_activation.html", map[string]string{"email": user.email, "profile_url": currentSite + "/profile"}), nil
		}
		return makeEmail(user.email, user.displayName+" has been deactivated on Vector", "user/user_deactivation.html", map[string]string{"email": user.email, "login_url": currentSite + "/login"}), nil
	case projectAddUserEmailTask:
		if err := exact(3); err != nil {
			return nil, err
		}
		currentSite, err := arg(0)
		if err != nil {
			return nil, err
		}
		memberID, err := arg(1)
		if err != nil {
			return nil, err
		}
		inviterID, err := arg(2)
		if err != nil {
			return nil, err
		}
		var projectName, workspaceName, workspaceSlug, projectID, memberEmail, inviterFirstName string
		err = store.pool.QueryRow(ctx, `SELECT p.name,w.name,w.slug,p.id::text,member.email,inviter.first_name FROM project_members pm JOIN projects p ON p.id=pm.project_id JOIN workspaces w ON w.id=pm.workspace_id JOIN users member ON member.id=pm.member_id JOIN users inviter ON inviter.id=$2 WHERE pm.id=$1`, memberID, inviterID).Scan(&projectName, &workspaceName, &workspaceSlug, &projectID, &memberEmail, &inviterFirstName)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errEmailTaskSourceRowMissing
		}
		if err != nil {
			return nil, fmt.Errorf("load project addition email data: %w", err)
		}
		projectURL := currentSite + "/" + workspaceSlug + "/projects/" + projectID + "/issues"
		return makeEmail(memberEmail, "You have been invited to a Vector project", "notifications/project_addition.html", map[string]string{"project_name": projectName, "workspace_name": workspaceName, "email": memberEmail, "inviter_first_name": inviterFirstName, "project_url": projectURL}), nil
	case webhookDeactivationEmailTask:
		if err := exact(4); err != nil {
			return nil, err
		}
		webhookID, err := arg(0)
		if err != nil {
			return nil, err
		}
		receiverID, err := arg(1)
		if err != nil {
			return nil, err
		}
		currentSite, err := arg(2)
		if err != nil {
			return nil, err
		}
		// The source task accepts this reason but its current template/message
		// intentionally uses the fixed failed-request wording.
		if _, err := arg(3); err != nil {
			return nil, err
		}
		user, err := store.userForEmail(ctx, receiverID)
		if err != nil {
			return nil, err
		}
		var webhookURL, slug string
		err = store.pool.QueryRow(ctx, `SELECT webhook.url,workspace.slug FROM webhooks webhook JOIN workspaces workspace ON workspace.id=webhook.workspace_id WHERE webhook.id=$1`, webhookID).Scan(&webhookURL, &slug)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errEmailTaskSourceRowMissing
		}
		if err != nil {
			return nil, fmt.Errorf("load webhook deactivation email data: %w", err)
		}
		message := "Webhook " + webhookURL + " has been deactivated due to failed requests."
		settingsURL := currentSite + "/" + slug + "/settings/webhooks/" + webhookID
		return makeEmail(user.email, "Webhook Deactivated", "notifications/webhook-deactivate.html", map[string]string{"email": user.email, "message": message, "webhook_url": settingsURL}), nil
	case workspaceInvitationEmailTask, projectInvitationEmailTask:
		if err := exact(5); err != nil {
			return nil, err
		}
		email, err := arg(0)
		if err != nil {
			return nil, err
		}
		entityID, err := arg(1)
		if err != nil {
			return nil, err
		}
		token, err := arg(2)
		if err != nil {
			return nil, err
		}
		currentSite, err := arg(3)
		if err != nil {
			return nil, err
		}
		inviterEmail, err := arg(4)
		if err != nil {
			return nil, err
		}
		inviter, err := store.userByEmail(ctx, inviterEmail)
		if err != nil {
			return nil, err
		}
		if input.TaskName == workspaceInvitationEmailTask {
			var invitationID, slug, name string
			err = store.pool.QueryRow(ctx, `SELECT invite.id::text,workspace.slug,workspace.name FROM workspace_member_invites invite JOIN workspaces workspace ON workspace.id=invite.workspace_id WHERE invite.token=$1 AND invite.email=$2 AND invite.workspace_id=$3 AND invite.deleted_at IS NULL`, token, email, entityID).Scan(&invitationID, &slug, &name)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errEmailTaskSourceRowMissing
			}
			if err != nil {
				return nil, fmt.Errorf("load workspace invitation email data: %w", err)
			}
			absURL := currentSite + "/workspace-invitations/?invitation_id=" + invitationID + "&slug=" + slug + "&token=" + token
			firstName := inviter.displayName
			prepared := makeEmail(email, firstName+" has invited you to join them in "+name+" on Vector", "invitations/workspace_invitation.html", map[string]string{"email": email, "first_name": firstName, "workspace_name": name, "abs_url": absURL})
			return store.saveInvitationMessage(ctx, "workspace_member_invites", invitationID, prepared)
		}
		var invitationID, projectName, workspaceSlug, projectID string
		err = store.pool.QueryRow(ctx, `SELECT invite.id::text,project.name,workspace.slug,project.id::text FROM project_member_invites invite JOIN projects project ON project.id=invite.project_id JOIN workspaces workspace ON workspace.id=project.workspace_id WHERE invite.token=$1 AND invite.email=$2 AND invite.project_id=$3 AND invite.deleted_at IS NULL`, token, email, entityID).Scan(&invitationID, &projectName, &workspaceSlug, &projectID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errEmailTaskSourceRowMissing
		}
		if err != nil {
			return nil, fmt.Errorf("load project invitation email data: %w", err)
		}
		absURL := currentSite + "/project-invitations/?invitation_id=" + invitationID + "&email=" + email + "&slug=" + workspaceSlug + "&project_id=" + projectID
		prepared := makeEmail(email, inviter.displayName+" invited you to join "+projectName+" on Vector", "invitations/project_invitation.html", map[string]string{"email": email, "first_name": inviter.firstName, "project_name": projectName, "invitation_url": absURL, "current_site": currentSite, "mirror": "", "unsubscribe": ""})
		return store.saveInvitationMessage(ctx, "project_member_invites", invitationID, prepared)
	default:
		return nil, fmt.Errorf("unsupported templated email task %q", input.TaskName)
	}
}

type emailUser struct{ email, firstName, displayName string }

func (store *templatedEmailStore) userForEmail(ctx context.Context, id string) (emailUser, error) {
	var user emailUser
	err := store.pool.QueryRow(ctx, `SELECT email,COALESCE(first_name,''),COALESCE(display_name,'') FROM users WHERE id=$1`, id).Scan(&user.email, &user.firstName, &user.displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return emailUser{}, errEmailTaskSourceRowMissing
	}
	if err != nil {
		return emailUser{}, fmt.Errorf("load account email recipient: %w", err)
	}
	user.displayName = firstNonEmpty(user.firstName, user.displayName, user.email)
	return user, nil
}

func (store *templatedEmailStore) userByEmail(ctx context.Context, email string) (emailUser, error) {
	var user emailUser
	err := store.pool.QueryRow(ctx, `SELECT email,COALESCE(first_name,''),COALESCE(display_name,'') FROM users WHERE email=$1`, email).Scan(&user.email, &user.firstName, &user.displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return emailUser{}, errEmailTaskSourceRowMissing
	}
	if err != nil {
		return emailUser{}, fmt.Errorf("load invitation email sender: %w", err)
	}
	user.displayName = firstNonEmpty(user.firstName, user.displayName, user.email)
	return user, nil
}

func (store *templatedEmailStore) saveInvitationMessage(ctx context.Context, table, id string, email *preparedEmail) (*preparedEmail, error) {
	htmlBody, textBody, err := renderPlaneEmail(email.Template, email.Values)
	if err != nil {
		return nil, err
	}
	query := `UPDATE ` + table + ` SET message=$2,updated_at=NOW() WHERE id=$1`
	if _, err := store.pool.Exec(ctx, query, id, textBody); err != nil {
		return nil, fmt.Errorf("save invitation email body: %w", err)
	}
	_ = htmlBody // Rendering both representations here matches Django's pre-send save order.
	return email, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
