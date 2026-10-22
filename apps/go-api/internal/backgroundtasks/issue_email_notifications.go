package backgroundtasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/html"
)

const (
	issueEmailDispatchWorkflowName = "plane.notifications.dispatch_issue_emails"
	issueEmailSendWorkflowName     = "plane.notifications.send_issue_email_batch"
	issueEmailCollectActivityName  = "plane.notifications.collect_issue_email_batches"
	issueEmailSendActivityName     = "plane.notifications.send_issue_email_batch"
	issueEmailBeatScheduleName     = "plane.beat.stack_email_notification"
	issueEmailBeatScheduleID       = "plane-beat:stack-email-notification"
	issueEmailBeatInterval         = 5 * time.Minute
)

type issueEmailSMTPConfigProvider interface {
	SMTPConfiguration(context.Context) (instancemail.SMTPConfig, string, error)
}

type issueEmailSender interface {
	Send(context.Context, instancemail.SMTPConfig, instancemail.Message) error
}

type IssueEmailActivity struct {
	ID, ActorID, ActorName, ActorAvatar string
	Field, OldValue, NewValue, Time     string
	Comment                             string
}

type IssueEmailBatch struct {
	IDs                        []string
	IssueID, ReceiverID        string
	ReceiverEmail              string
	IssueName, IssueIdentifier string
	ProjectID, ProjectName     string
	WorkspaceSlug, Origin      string
	Activities                 []IssueEmailActivity
}

type issueEmailStore struct {
	pool   *pgxpool.Pool
	config issueEmailSMTPConfigProvider
	sender issueEmailSender
	origin *redis.Client
}

type IssueEmailDispatcher struct {
	app              *selvedge.App
	workflow         *selvedge.WorkflowFunc[issueEmailDispatchInput, struct{}]
	scheduleWorkflow *selvedge.WorkflowFunc[struct{}, struct{}]
}

type issueEmailDispatchInput struct {
	EventID string `json:"event_id"`
}
type issueEmailSendInput struct {
	Batch IssueEmailBatch `json:"batch"`
}

func NewPostgresIssueEmailStore(pool *pgxpool.Pool, config issueEmailSMTPConfigProvider, sender issueEmailSender) (*issueEmailStore, error) {
	if pool == nil || config == nil || sender == nil {
		return nil, errors.New("email PostgreSQL pool, SMTP configuration and sender are required")
	}
	store := &issueEmailStore{pool: pool, config: config, sender: sender}
	if redisURL := strings.TrimSpace(os.Getenv("REDIS_URL")); redisURL != "" {
		options, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("configure issue email origin Redis: %w", err)
		}
		store.origin = redis.NewClient(options)
	}
	return store, nil
}

func (store *issueEmailStore) Close() error {
	if store == nil || store.origin == nil {
		return nil
	}
	return store.origin.Close()
}

func RegisterIssueEmailNotificationWorkflow(app *selvedge.App, store *issueEmailStore, policy *retry.Policy) (*IssueEmailDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and issue email store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	collect := execution.Activity[issueEmailDispatchInput, []IssueEmailBatch]{Name: issueEmailCollectActivityName}
	if err := executionSelvedge.Register(activities, collect, func(ctx context.Context, input issueEmailDispatchInput) ([]IssueEmailBatch, error) {
		if _, err := uuid.Parse(input.EventID); err != nil {
			return nil, fmt.Errorf("invalid issue email dispatch identity: %w", err)
		}
		return store.pendingBatches(ctx)
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 30 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue email collection activity: %w", err)
	}
	send := execution.Activity[issueEmailSendInput, struct{}]{Name: issueEmailSendActivityName}
	if err := executionSelvedge.Register(activities, send, func(ctx context.Context, input issueEmailSendInput) (struct{}, error) {
		if err := store.sendBatch(ctx, input.Batch); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue email delivery activity: %w", err)
	}
	sendWorkflow := selvedge.DefineWorkflow(issueEmailSendWorkflowName, func(ctx *selvedge.WorkflowContext, input issueEmailSendInput) (struct{}, error) {
		return send.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, sendWorkflow)
	workflow := selvedge.DefineWorkflow(issueEmailDispatchWorkflowName, func(ctx *selvedge.WorkflowContext, input issueEmailDispatchInput) (struct{}, error) {
		batches, err := collect.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		for _, batch := range batches {
			childID := issueEmailBatchID(input.EventID, batch)
			child, err := selvedge.StartChild[struct{}](ctx, issueEmailSendWorkflowName, issueEmailSendInput{Batch: batch}, selvedge.WithChildID(childID))
			if err != nil {
				return struct{}{}, err
			}
			if _, err := child.Get(ctx); err != nil {
				return struct{}{}, err
			}
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, workflow)
	schedule := selvedge.DefineWorkflow(issueEmailBeatScheduleName, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
		for {
			now, err := selvedge.Now(ctx)
			if err != nil {
				return struct{}{}, err
			}
			next := nextIssueEmailBeat(now)
			if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("email-beat:"+next.Format(time.RFC3339))); err != nil {
				return struct{}{}, err
			}
			scheduledEvent := uuid.NewSHA1(uuid.NameSpaceOID, []byte(next.Format(time.RFC3339))).String()
			for attempt := 0; ; attempt++ {
				eventID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%d", scheduledEvent, attempt))).String()
				child, err := selvedge.StartChild[struct{}](ctx, issueEmailDispatchWorkflowName, issueEmailDispatchInput{EventID: eventID}, selvedge.WithChildID("plane-email-dispatch:"+eventID))
				if err != nil {
					return struct{}{}, err
				}
				if _, err := child.Get(ctx); err == nil {
					break
				}
				if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("email-beat-retry:%s:%d", scheduledEvent, attempt))); err != nil {
					return struct{}{}, err
				}
			}
		}
	})
	selvedge.RegisterWorkflow(app, schedule)
	return &IssueEmailDispatcher{app: app, workflow: workflow, scheduleWorkflow: schedule}, nil
}

// StartBeatSchedule durably admits the source stack_email_notification task's
// five-minute Beat schedule. The stable instance ID reconciles repeated starts
// and the framework timer survives process restarts.
func (dispatcher *IssueEmailDispatcher) StartBeatSchedule(ctx context.Context) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.scheduleWorkflow == nil {
		return "", errors.New("issue email Beat schedule is not configured")
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.scheduleWorkflow, issueEmailBeatScheduleID, struct{}{})
}

func nextIssueEmailBeat(now time.Time) time.Time {
	now = now.UTC()
	return now.Truncate(issueEmailBeatInterval).Add(issueEmailBeatInterval)
}

func (dispatcher *IssueEmailDispatcher) Admit(ctx context.Context, eventID string) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("issue email dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("issue email dispatch identity must be a UUID: %w", err)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "issue-email-dispatch:"+eventID, issueEmailDispatchInput{EventID: eventID})
}

func (store *issueEmailStore) pendingBatches(ctx context.Context) ([]IssueEmailBatch, error) {
	// Leave rows pending until each deterministic child is admitted. Marking them
	// before StartChild would create a lost-work window across the independent
	// business and Selvedge transactions.
	rows, err := store.pool.Query(ctx, `SELECT log.id,log.receiver_id,log.entity_identifier,log.data,receiver.email,issue.name,issue.sequence_id,project.id,project.identifier,project.name,workspace.slug,actor.id,actor.display_name,actor.avatar FROM email_notification_logs log JOIN users receiver ON receiver.id=log.receiver_id JOIN issues issue ON issue.id=log.entity_identifier JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=project.workspace_id LEFT JOIN users actor ON actor.id=log.triggered_by_id WHERE log.deleted_at IS NULL AND log.processed_at IS NULL AND log.entity_name='issue' AND log.entity_identifier IS NOT NULL ORDER BY log.receiver_id,log.entity_identifier,log.created_at,log.id LIMIT 1000`)
	if err != nil {
		return nil, fmt.Errorf("load pending issue email notifications: %w", err)
	}
	defer rows.Close()
	type key struct{ receiver, issue string }
	batches := make(map[key]*IssueEmailBatch)
	order := make([]key, 0)
	for rows.Next() {
		var id, receiverID, issueID, projectID uuid.UUID
		var dataJSON json.RawMessage
		var receiverEmail, issueName string
		var sequenceID int64
		var identifier, projectName, workspaceSlug string
		var actorID *uuid.UUID
		var actorName, actorAvatar *string
		if err := rows.Scan(&id, &receiverID, &issueID, &dataJSON, &receiverEmail, &issueName, &sequenceID, &projectID, &identifier, &projectName, &workspaceSlug, &actorID, &actorName, &actorAvatar); err != nil {
			return nil, fmt.Errorf("read pending issue email notification: %w", err)
		}
		batchKey := key{receiver: receiverID.String(), issue: issueID.String()}
		batch := batches[batchKey]
		if batch == nil {
			batch = &IssueEmailBatch{ReceiverID: receiverID.String(), IssueID: issueID.String(), ReceiverEmail: receiverEmail, IssueName: issueName, IssueIdentifier: fmt.Sprintf("%s-%d", identifier, sequenceID), ProjectID: projectID.String(), ProjectName: projectName, WorkspaceSlug: workspaceSlug}
			batches[batchKey] = batch
			order = append(order, batchKey)
		}
		batch.IDs = append(batch.IDs, id.String())
		activity, err := decodeIssueEmailActivity(dataJSON)
		if err != nil {
			return nil, fmt.Errorf("decode issue email notification %s: %w", id, err)
		}
		if actorID != nil {
			activity.ActorID = actorID.String()
		}
		if actorName != nil {
			activity.ActorName = *actorName
		}
		if actorAvatar != nil {
			activity.ActorAvatar = *actorAvatar
		}
		batch.Activities = append(batch.Activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending issue email notifications: %w", err)
	}
	rows.Close()
	result := make([]IssueEmailBatch, 0, len(order))
	for _, batchKey := range order {
		result = append(result, *batches[batchKey])
	}
	return result, nil
}

func decodeIssueEmailActivity(data json.RawMessage) (IssueEmailActivity, error) {
	var envelope struct {
		Activity struct {
			ID           string `json:"id"`
			Actor        string `json:"actor"`
			Field        string `json:"field"`
			OldValue     string `json:"old_value"`
			NewValue     string `json:"new_value"`
			ActivityTime string `json:"activity_time"`
			Comment      string `json:"issue_comment"`
		} `json:"issue_activity"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return IssueEmailActivity{}, err
	}
	return IssueEmailActivity{ID: envelope.Activity.ID, ActorID: envelope.Activity.Actor, Field: envelope.Activity.Field, OldValue: envelope.Activity.OldValue, NewValue: envelope.Activity.NewValue, Time: formatIssueEmailTime(envelope.Activity.ActivityTime), Comment: envelope.Activity.Comment}, nil
}

func formatIssueEmailTime(value string) string {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.Local().Format("03:04 PM")
	}
	return value
}

func (store *issueEmailStore) sendBatch(ctx context.Context, batch IssueEmailBatch) error {
	if len(batch.IDs) == 0 {
		return nil
	}
	needsDelivery, err := store.batchNeedsDelivery(ctx, batch.IDs)
	if err != nil || !needsDelivery {
		return err
	}
	if store.origin == nil {
		return store.markEmailBatchProcessed(ctx, batch.IDs)
	}
	baseURL, err := store.origin.Get(ctx, batch.IssueID).Result()
	if errors.Is(err, redis.Nil) || baseURL == "" {
		return store.markEmailBatchProcessed(ctx, batch.IDs)
	}
	if err != nil {
		return fmt.Errorf("load issue email origin: %w", err)
	}
	lockID := emailNotificationLockID(batch)
	lockToken := uuid.NewString()
	locked, err := store.origin.SetNX(ctx, lockID, lockToken, 5*time.Minute).Result()
	if err != nil {
		return fmt.Errorf("acquire issue email delivery lock: %w", err)
	}
	if !locked {
		return errors.New("issue email delivery is already in progress")
	}
	defer releaseEmailNotificationLock(context.WithoutCancel(ctx), store.origin, lockID, lockToken)
	issueURL, projectURL, preferenceURL, err := issueEmailURLs(baseURL, batch)
	if err != nil {
		return err
	}
	config, from, err := store.config.SMTPConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("load issue email SMTP configuration: %w", err)
	}
	if config.Host == "" || batch.ReceiverEmail == "" {
		return errors.New("issue email SMTP host and receiver address are required")
	}
	activities, err := store.processMentionValues(ctx, batch.Activities)
	if err != nil {
		return err
	}
	data := issueEmailTemplateData{IssueIdentifier: batch.IssueIdentifier, IssueName: batch.IssueName, IssueURL: issueURL, ProjectURL: projectURL, PreferenceURL: preferenceURL, Workspace: batch.WorkspaceSlug, Project: batch.ProjectName, ReceiverEmail: batch.ReceiverEmail, Activities: activities}
	htmlBody, textBody, err := renderIssueEmail(data)
	if err != nil {
		return err
	}
	subject := stripEmailControls(batch.IssueIdentifier + " " + batch.IssueName)
	if err := store.sender.Send(ctx, config, instancemail.Message{From: from, To: batch.ReceiverEmail, Subject: subject, Body: textBody, HTMLBody: htmlBody}); err != nil {
		return fmt.Errorf("send issue update email: %w", err)
	}
	return store.markEmailBatchSent(ctx, batch.IDs)
}

func (store *issueEmailStore) processMentionValues(ctx context.Context, activities []IssueEmailActivity) ([]IssueEmailActivity, error) {
	users := make(map[uuid.UUID]string)
	for _, activity := range activities {
		for _, value := range []string{activity.OldValue, activity.NewValue} {
			if !strings.Contains(value, "mention-component") {
				continue
			}
			root, err := html.Parse(strings.NewReader(value))
			if err != nil {
				return nil, fmt.Errorf("parse issue email mention markup: %w", err)
			}
			var collect func(*html.Node) error
			collect = func(node *html.Node) error {
				if node.Type == html.ElementNode && node.Data == "mention-component" {
					var raw string
					for _, attribute := range node.Attr {
						if attribute.Key == "entity_identifier" {
							raw = attribute.Val
							break
						}
					}
					id, err := uuid.Parse(raw)
					if err != nil {
						return fmt.Errorf("invalid email mention recipient ID: %w", err)
					}
					users[id] = ""
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					if err := collect(child); err != nil {
						return err
					}
				}
				return nil
			}
			if err := collect(root); err != nil {
				return nil, err
			}
		}
	}
	if len(users) > 0 {
		ids := make([]uuid.UUID, 0, len(users))
		for id := range users {
			ids = append(ids, id)
		}
		rows, err := store.pool.Query(ctx, `SELECT id,display_name FROM users WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return nil, fmt.Errorf("load issue email mention names: %w", err)
		}
		for rows.Next() {
			var id uuid.UUID
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				rows.Close()
				return nil, fmt.Errorf("read issue email mention name: %w", err)
			}
			users[id] = name
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate issue email mention names: %w", err)
		}
		rows.Close()
		for id, name := range users {
			if name == "" {
				return nil, fmt.Errorf("issue email mention user %s does not exist", id)
			}
		}
	}
	for index := range activities {
		for _, destination := range []*string{&activities[index].OldValue, &activities[index].NewValue} {
			if strings.Contains(*destination, "mention-component") {
				processed, err := replaceEmailMentions(*destination, users)
				if err != nil {
					return nil, err
				}
				*destination = processed
			}
		}
	}
	return activities, nil
}

func replaceEmailMentions(value string, users map[uuid.UUID]string) (string, error) {
	root, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return "", fmt.Errorf("parse issue email mention value: %w", err)
	}
	var replace func(*html.Node)
	replace = func(node *html.Node) {
		for child := node.FirstChild; child != nil; {
			next := child.NextSibling
			if child.Type == html.ElementNode && child.Data == "mention-component" {
				var raw string
				for _, attribute := range child.Attr {
					if attribute.Key == "entity_identifier" {
						raw = attribute.Val
						break
					}
				}
				id, _ := uuid.Parse(raw)
				name, exists := users[id]
				if !exists {
					continue
				}
				text := &html.Node{Type: html.TextNode, Data: "@" + name}
				text.Parent, text.NextSibling, text.PrevSibling = node, next, child.PrevSibling
				if child.PrevSibling != nil {
					child.PrevSibling.NextSibling = text
				} else {
					node.FirstChild = text
				}
				if next != nil {
					next.PrevSibling = text
				} else {
					node.LastChild = text
				}
				child.Parent = nil
				child.PrevSibling = nil
				child.NextSibling = nil
			} else {
				replace(child)
			}
			child = next
		}
	}
	replace(root)
	var output strings.Builder
	var text func(*html.Node)
	text = func(node *html.Node) {
		if node.Type == html.TextNode {
			output.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			text(child)
		}
	}
	text(root)
	return strings.TrimSpace(output.String()), nil
}

func (store *issueEmailStore) markEmailBatchSent(ctx context.Context, ids []string) error {
	parsed := make([]uuid.UUID, 0, len(ids))
	for _, value := range ids {
		id, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid email notification ID: %w", err)
		}
		parsed = append(parsed, id)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE email_notification_logs SET processed_at=COALESCE(processed_at,NOW()),sent_at=COALESCE(sent_at,NOW()),updated_at=NOW() WHERE id=ANY($1::uuid[]) AND deleted_at IS NULL`, parsed); err != nil {
		return fmt.Errorf("mark issue email notifications sent: %w", err)
	}
	return nil
}

func (store *issueEmailStore) markEmailBatchProcessed(ctx context.Context, ids []string) error {
	parsed := make([]uuid.UUID, 0, len(ids))
	for _, value := range ids {
		id, err := uuid.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid email notification ID: %w", err)
		}
		parsed = append(parsed, id)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE email_notification_logs SET processed_at=COALESCE(processed_at,NOW()),updated_at=NOW() WHERE id=ANY($1::uuid[]) AND deleted_at IS NULL`, parsed); err != nil {
		return fmt.Errorf("mark issue email notifications processed: %w", err)
	}
	return nil
}

func (store *issueEmailStore) batchNeedsDelivery(ctx context.Context, values []string) (bool, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return false, fmt.Errorf("invalid email notification ID: %w", err)
		}
		ids = append(ids, id)
	}
	var pending bool
	if err := store.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM email_notification_logs WHERE id=ANY($1::uuid[]) AND deleted_at IS NULL AND sent_at IS NULL)`, ids).Scan(&pending); err != nil {
		return false, fmt.Errorf("check issue email delivery status: %w", err)
	}
	return pending, nil
}

func emailNotificationLockID(batch IssueEmailBatch) string {
	ids := append([]string(nil), batch.IDs...)
	sort.Strings(ids)
	return "send_email_notif_" + batch.IssueID + "_" + batch.ReceiverID + "_" + strings.Join(ids, "_")
}

func releaseEmailNotificationLock(ctx context.Context, client *redis.Client, lockID, token string) {
	const release = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, _ = client.Eval(ctx, release, []string{lockID}, token).Result()
}

type issueEmailTemplateData struct {
	IssueIdentifier, IssueName, IssueURL, ProjectURL, PreferenceURL string
	Workspace, Project, ReceiverEmail                               string
	Activities                                                      []IssueEmailActivity
}

var issueUpdatesEmailTemplate = htmltemplate.Must(htmltemplate.New("issue-updates").Parse(`<!doctype html><html><body style="font-family:Arial,sans-serif;color:#1f2d5c;background:#f7f9ff;padding:20px"><main style="max-width:600px;margin:auto;background:#fff;padding:28px;border-radius:8px"><h2>{{.IssueIdentifier}} updates</h2><p>{{.Workspace}} / <a href="{{.ProjectURL}}">{{.Project}}</a> / <a href="{{.IssueURL}}">{{.IssueIdentifier}}: {{.IssueName}}</a></p><ul>{{range .Activities}}<li><strong>{{.ActorName}}</strong>{{if .Field}} {{.Field}} changed{{end}}{{if .OldValue}} from “{{.OldValue}}”{{end}}{{if .NewValue}} to “{{.NewValue}}”{{end}}{{if .Comment}}<p>{{.Comment}}</p>{{end}}{{if .Time}} <small>{{.Time}}</small>{{end}}</li>{{end}}</ul><p><a href="{{.IssueURL}}">View issue</a></p><hr><small>This email was sent to {{.ReceiverEmail}}. <a href="{{.PreferenceURL}}">Manage email preferences</a>.</small></main></body></html>`))

func renderIssueEmail(data issueEmailTemplateData) (string, string, error) {
	var output strings.Builder
	if err := issueUpdatesEmailTemplate.Execute(&output, data); err != nil {
		return "", "", fmt.Errorf("render issue update email: %w", err)
	}
	text, err := htmlEmailText(output.String())
	if err != nil {
		return "", "", err
	}
	return output.String(), text, nil
}

func htmlEmailText(body string) (string, error) {
	root, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parse rendered issue email: %w", err)
	}
	var output strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "p" || node.Data == "li" || node.Data == "h2" || node.Data == "hr") {
			output.WriteByte('\n')
		}
		if node.Type == html.TextNode {
			output.WriteString(strings.TrimSpace(node.Data))
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if node.Type == html.ElementNode && (node.Data == "p" || node.Data == "li" || node.Data == "h2") {
			output.WriteByte('\n')
		}
	}
	visit(root)
	return strings.TrimSpace(output.String()), nil
}

func issueEmailURLs(base string, batch IssueEmailBatch) (string, string, string, error) {
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", "", "", errors.New("issue email origin must be an absolute HTTP URL")
	}
	root := strings.TrimRight(base, "/") + "/" + url.PathEscape(batch.WorkspaceSlug) + "/projects/" + url.PathEscape(batch.ProjectID)
	issue := root + "/issues/" + url.PathEscape(batch.IssueID)
	return issue, root + "/issues/", strings.TrimRight(base, "/") + "/" + url.PathEscape(batch.WorkspaceSlug) + "/settings/account/notifications/", nil
}

func issueEmailBatchID(eventID string, batch IssueEmailBatch) string {
	ids := append([]string(nil), batch.IDs...)
	sort.Strings(ids)
	hash := sha256.Sum256([]byte(eventID + ":" + batch.ReceiverID + ":" + batch.IssueID + ":" + strings.Join(ids, ":")))
	return "issue-email-" + hex.EncodeToString(hash[:])
}

func stripEmailControls(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return -1
		}
		return r
	}, value)
}
