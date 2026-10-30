package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"
)

type issueNotificationEvent struct {
	id uuid.UUID
	// The producer's optional workspace binding differs from the stored
	// workspaceID below; recipient filtering and email payloads use the binding.
	input                                       IssueActivityInput
	issueID, projectID, workspaceID             uuid.UUID
	workspaceSlug, issueName, projectIdentifier string
	sequenceID                                  int64
	creatorID, stateName, stateGroup            *string
}

func (store *PostgresIssueActivityStore) ProcessIssueNotifications(ctx context.Context, eventID string, input IssueActivityInput, activityIDs []string) (resultErr error) {
	if err := validateIssueActivityInput(eventID, input); err != nil {
		return err
	}
	if input.IssueID == nil || len(activityIDs) == 0 || notificationsSkipActivityType(input.Type) {
		return nil
	}
	projectID, _ := uuid.Parse(input.ProjectID)
	issueID, _ := uuid.Parse(*input.IssueID)
	eventUUID, _ := uuid.Parse(eventID)
	event := issueNotificationEvent{id: eventUUID, input: input, issueID: issueID, projectID: projectID}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin issue notification transaction: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if input.WorkspaceID != "" {
		if scopeErr := lockBoundIssueActivity(ctx, tx, input, uuid.MustParse(input.WorkspaceID), false); scopeErr != nil {
			return scopeErr
		}
	}
	err = tx.QueryRow(ctx, `SELECT project.workspace_id,workspace.slug,issue.name,project.identifier,issue.sequence_id,issue.created_by_id,state.name,state."group" FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=project.workspace_id LEFT JOIN states state ON state.id=issue.state_id WHERE issue.id=$1 AND issue.project_id=$2 AND issue.deleted_at IS NULL`, event.issueID, event.projectID).Scan(&event.workspaceID, &event.workspaceSlug, &event.issueName, &event.projectIdentifier, &event.sequenceID, &event.creatorID, &event.stateName, &event.stateGroup)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load issue for notifications: %w", err)
	}
	if input.Subscriber == nil || *input.Subscriber {
		if input.ActorID != nil {
			if _, subscribeErr := tx.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,subscriber_id) VALUES ($1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$2) ON CONFLICT (issue_id,subscriber_id) WHERE deleted_at IS NULL DO NOTHING`, uuid.New(), *input.ActorID, event.projectID, event.workspaceID, event.issueID); subscribeErr != nil {
				return fmt.Errorf("subscribe issue activity actor: %w", subscribeErr)
			}
		}
	}
	activities, err := loadIssueActivities(ctx, tx, activityIDs, event.issueID, event.projectID, event.workspaceID)
	if err != nil {
		return err
	}
	if len(activities) == 0 {
		return tx.Commit(ctx)
	}
	newDescriptionMentions, removedDescriptionMentions, newCommentMentions, allCommentMentions := issueActivityMentions(input, activities)
	newCommentRecipientIDs := make([]uuid.UUID, 0)
	for _, mentions := range newCommentMentions {
		newCommentRecipientIDs = append(newCommentRecipientIDs, mentions...)
	}
	newCommentRecipientIDs = uniqueUUIDs(newCommentRecipientIDs)
	if input.Type == "issue.activity.updated" {
		if mentionErr := event.syncDescriptionMentions(ctx, tx, newDescriptionMentions, removedDescriptionMentions); mentionErr != nil {
			return mentionErr
		}
	}
	activeMembers, err := activeProjectMembers(ctx, tx, event.projectID)
	if err != nil {
		return err
	}
	for _, mentionID := range unionUUIDs(newDescriptionMentions, allCommentMentions) {
		if _, subscribeErr := tx.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,subscriber_id) SELECT $1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$6 WHERE EXISTS(SELECT 1 FROM project_members WHERE project_id=$3 AND member_id=$6 AND is_active=true) AND NOT EXISTS(SELECT 1 FROM issue_assignees WHERE project_id=$3 AND issue_id=$5 AND assignee_id=$6 AND deleted_at IS NULL) AND NOT EXISTS(SELECT 1 FROM issues WHERE id=$5 AND created_by_id=$6) ON CONFLICT (issue_id,subscriber_id) WHERE deleted_at IS NULL DO NOTHING`, uuid.New(), nullableUUIDString(input.ActorID), event.projectID, event.workspaceID, event.issueID, mentionID); subscribeErr != nil {
			return fmt.Errorf("subscribe issue mention: %w", subscribeErr)
		}
	}
	recipients, err := event.loadRecipients(ctx, tx)
	if err != nil {
		return err
	}
	for _, activity := range activities {
		for _, receiver := range recipients {
			if containsUUID(newDescriptionMentions, receiver.id) || containsUUID(newCommentRecipientIDs, receiver.id) {
				continue
			}
			if err := event.insertSubscriberNotification(ctx, tx, activity, receiver); err != nil {
				return err
			}
		}
	}
	for activityID, mentionIDs := range newCommentMentions {
		activity, ok := issueActivityByID(activities, activityID)
		if !ok || input.ActorID == nil {
			continue
		}
		for _, mentionID := range mentionIDs {
			if mentionID == *mustUUID(input.ActorID) || !activeMembers[mentionID] {
				continue
			}
			if err := event.insertMentionNotification(ctx, tx, activity, mentionID, "You have been mentioned in the issue "+event.issueName); err != nil {
				return err
			}
		}
	}
	if input.ActorID != nil && len(newDescriptionMentions) > 0 && input.Type == "issue.activity.updated" {
		last := activities[len(activities)-1]
		if last.field == "description" && last.actorID != nil && *last.actorID == *mustUUID(input.ActorID) {
			for _, mentionID := range newDescriptionMentions {
				if mentionID == *mustUUID(input.ActorID) || !activeMembers[mentionID] {
					continue
				}
				if err := event.insertMentionNotification(ctx, tx, last, mentionID, "You have been mentioned in the issue "+event.issueName); err != nil {
					return err
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit issue notifications: %w", err)
	}
	return nil
}

type issueNotificationRecipient struct {
	id                                  uuid.UUID
	createdBy                           *uuid.UUID
	assignee                            bool
	property, state, comment, completed bool
}

func (event issueNotificationEvent) loadRecipients(ctx context.Context, tx pgx.Tx) ([]issueNotificationRecipient, error) {
	var boundWorkspace any
	if event.input.WorkspaceID != "" {
		boundWorkspace = event.input.WorkspaceID
	}
	rows, err := tx.Query(ctx, `SELECT DISTINCT subscriber.id,issue.created_by_id,
 EXISTS(SELECT 1 FROM issue_assignees assignee WHERE assignee.issue_id=issue.id AND assignee.project_id=project.id AND assignee.assignee_id=subscriber.id AND assignee.deleted_at IS NULL),
 COALESCE(preference.property_change,true),COALESCE(preference.state_change,true),COALESCE(preference.comment,true),COALESCE(preference.issue_completed,true)
 FROM issue_subscribers subscription JOIN users subscriber ON subscriber.id=subscription.subscriber_id
 JOIN issues issue ON issue.id=subscription.issue_id JOIN projects project ON project.id=subscription.project_id
 JOIN project_members member ON member.project_id=project.id AND member.member_id=subscriber.id AND member.is_active=true
 LEFT JOIN LATERAL (SELECT property_change,state_change,comment,issue_completed FROM user_notification_preferences WHERE user_id=subscriber.id AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1) preference ON true
 WHERE subscription.issue_id=$1 AND subscription.project_id=$2 AND subscription.deleted_at IS NULL AND ($3::uuid IS NULL OR subscription.subscriber_id<>$3::uuid)
 AND ($4::uuid IS NULL OR (subscription.workspace_id=$4 AND issue.workspace_id=$4 AND project.workspace_id=$4
 AND member.workspace_id=$4 AND member.deleted_at IS NULL AND subscriber.is_active=true AND project.archived_at IS NULL
 AND (member.role IN (15,20) OR member.role=5 AND project.guest_view_all_features)
 AND EXISTS(SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=$4 AND wm.member_id=subscriber.id AND wm.is_active=true AND wm.deleted_at IS NULL AND wm.role IN (5,15,20))))
 ORDER BY subscriber.id`, event.issueID, event.projectID, nullableUUIDString(event.input.ActorID), boundWorkspace)
	if err != nil {
		return nil, fmt.Errorf("load issue notification recipients: %w", err)
	}

	recipients := make([]issueNotificationRecipient, 0)
	for rows.Next() {
		var value issueNotificationRecipient
		if err := rows.Scan(&value.id, &value.createdBy, &value.assignee, &value.property, &value.state, &value.comment, &value.completed); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read issue notification recipient: %w", err)
		}
		recipients = append(recipients, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate issue notification recipients: %w", err)
	}
	rows.Close()
	return recipients, nil
}

func (event issueNotificationEvent) insertSubscriberNotification(ctx context.Context, tx pgx.Tx, activity issueNotificationActivity, receiver issueNotificationRecipient) error {
	sender := "in_app:issue_activities:subscribed"
	if event.creatorID != nil && *event.creatorID == receiver.id.String() {
		sender = "in_app:issue_activities:created"
	} else if receiver.assignee && (receiver.createdBy == nil || *receiver.createdBy != receiver.id) {
		sender = "in_app:issue_activities:assigned"
	}
	issueData := map[string]any{"id": event.issueID.String(), "name": event.issueName, "identifier": event.projectIdentifier, "sequence_id": event.sequenceID, "state_name": event.stateName, "state_group": event.stateGroup}
	commentText := ""
	if activity.commentID != nil {
		if err := tx.QueryRow(ctx, `SELECT comment_stripped FROM issue_comments WHERE id=$1 AND issue_id=$2 AND project_id=$3 AND workspace_id=$4`, *activity.commentID, event.issueID, event.projectID, event.workspaceID).Scan(&commentText); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load notification comment text: %w", err)
		}
	}
	activityData := map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "issue_comment": commentText, "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier)}
	data, err := json.Marshal(map[string]any{"issue": issueData, "issue_activity": activityData})
	if err != nil {
		return fmt.Errorf("encode issue notification: %w", err)
	}
	notificationID := uuid.NewSHA1(event.id, []byte("plane.notification:"+activity.id.String()+":"+receiver.id.String()))
	if _, err := tx.Exec(ctx, `INSERT INTO notifications (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,data,entity_identifier,entity_name,title,message,message_html,message_stripped,sender,triggered_by_id,receiver_id,read_at,snoozed_till,archived_at) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4::jsonb,$5,'issue',$6,NULL,'<p></p>','',$7,$8,$9,NULL,NULL,NULL) ON CONFLICT (id) DO NOTHING`, notificationID, event.workspaceID, event.projectID, data, event.issueID, activity.comment, sender, nullableUUIDString(event.input.ActorID), receiver.id); err != nil {
		return fmt.Errorf("insert issue notification: %w", err)
	}
	sendEmail := false
	if activity.field != "description" {
		switch activity.field {
		case "state":
			sendEmail = receiver.state
			if !sendEmail && receiver.completed && activity.newIdentifier != nil {
				var completed bool
				if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM states WHERE id=$1 AND project_id=$2 AND "group"='completed')`, *activity.newIdentifier, event.projectID).Scan(&completed); err != nil {
					return fmt.Errorf("check completed issue state preference: %w", err)
				}
				sendEmail = completed
			}
		case "comment":
			sendEmail = receiver.comment
		default:
			sendEmail = receiver.property
		}
	}
	if sendEmail {
		emailData := map[string]any{"issue": map[string]any{"id": event.issueID.String(), "name": event.issueName, "identifier": event.projectIdentifier, "project_id": event.projectID.String(), "workspace_slug": event.workspaceSlug, "workspace_id": event.input.WorkspaceID, "sequence_id": event.sequenceID, "state_name": event.stateName, "state_group": event.stateGroup}, "issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "issue_comment": commentText, "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier), "activity_time": activity.createdAt}}
		emailJSON, err := json.Marshal(emailData)
		if err != nil {
			return fmt.Errorf("encode email notification log: %w", err)
		}
		logID := uuid.NewSHA1(event.id, []byte("plane.email_notification:"+activity.id.String()+":"+receiver.id.String()))
		if _, err := tx.Exec(ctx, `INSERT INTO email_notification_logs (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,receiver_id,triggered_by_id,entity_identifier,entity_name,data,processed_at,sent_at,entity,old_value,new_value) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4,'issue',$5::jsonb,NULL,NULL,'',NULL,NULL) ON CONFLICT (id) DO NOTHING`, logID, receiver.id, nullableUUIDString(event.input.ActorID), event.issueID, emailJSON); err != nil {
			return fmt.Errorf("insert email notification log: %w", err)
		}
	}
	return nil
}

func (event issueNotificationEvent) insertMentionNotification(ctx context.Context, tx pgx.Tx, activity issueNotificationActivity, receiver uuid.UUID, message string) error {
	actorID := nullableUUIDString(event.input.ActorID)
	data, err := json.Marshal(map[string]any{
		"issue":          map[string]any{"id": event.issueID.String(), "name": event.issueName, "identifier": event.projectIdentifier, "sequence_id": event.sequenceID, "state_name": event.stateName, "state_group": event.stateGroup},
		"issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier)},
	})
	if err != nil {
		return fmt.Errorf("encode mention notification: %w", err)
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode mention message: %w", err)
	}
	notificationID := uuid.NewSHA1(event.id, []byte("plane.notification:mention:"+activity.id.String()+":"+receiver.String()))
	if _, insertErr := tx.Exec(ctx, `INSERT INTO notifications (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,data,entity_identifier,entity_name,title,message,message_html,message_stripped,sender,triggered_by_id,receiver_id,read_at,snoozed_till,archived_at) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4::jsonb,$5,'issue',$6,$7::jsonb,'<p></p>','',$8,$9,$10,NULL,NULL,NULL) ON CONFLICT (id) DO NOTHING`, notificationID, event.workspaceID, event.projectID, data, event.issueID, activity.comment, messageJSON, "in_app:issue_activities:mentioned", actorID, receiver); insertErr != nil {
		return fmt.Errorf("insert issue mention notification: %w", insertErr)
	}
	var sendEmail bool
	if preferenceErr := tx.QueryRow(ctx, `SELECT COALESCE((SELECT mention FROM user_notification_preferences WHERE user_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1),true)`, receiver).Scan(&sendEmail); preferenceErr != nil {
		return fmt.Errorf("load mention email preference: %w", preferenceErr)
	}
	if !sendEmail {
		return nil
	}
	emailData, err := json.Marshal(map[string]any{
		"issue":          map[string]any{"id": event.issueID.String(), "name": event.issueName, "identifier": event.projectIdentifier, "project_id": event.projectID.String(), "workspace_slug": event.workspaceSlug, "workspace_id": event.input.WorkspaceID, "sequence_id": event.sequenceID, "state_name": event.stateName, "state_group": event.stateGroup},
		"issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": "mention", "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier), "activity_time": activity.createdAt},
	})
	if err != nil {
		return fmt.Errorf("encode mention email log: %w", err)
	}
	logID := uuid.NewSHA1(event.id, []byte("plane.email_notification:mention:"+activity.id.String()+":"+receiver.String()))
	if _, err := tx.Exec(ctx, `INSERT INTO email_notification_logs (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,receiver_id,triggered_by_id,entity_identifier,entity_name,data,processed_at,sent_at,entity,old_value,new_value) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4,'issue',$5::jsonb,NULL,NULL,'',NULL,NULL) ON CONFLICT (id) DO NOTHING`, logID, receiver, actorID, event.issueID, emailData); err != nil {
		return fmt.Errorf("insert issue mention email log: %w", err)
	}
	return nil
}

type issueNotificationActivity struct {
	id                                               uuid.UUID
	commentID, actorID, oldIdentifier, newIdentifier *uuid.UUID
	verb, field, comment                             string
	oldValue, newValue                               *string
	createdAt                                        time.Time
}

func loadIssueActivities(ctx context.Context, tx pgx.Tx, activityIDs []string, issueID, projectID, workspaceID uuid.UUID) ([]issueNotificationActivity, error) {
	ids := make([]uuid.UUID, 0, len(activityIDs))
	for _, raw := range activityIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid generated activity ID: %w", err)
		}
		ids = append(ids, id)
	}
	rows, err := tx.Query(ctx, `SELECT id,issue_comment_id,actor_id,old_identifier,new_identifier,verb,COALESCE(field,''),comment,old_value,new_value,created_at FROM issue_activities WHERE id=ANY($1::uuid[]) AND issue_id=$2 AND project_id=$3 AND workspace_id=$4 AND deleted_at IS NULL ORDER BY array_position($1::uuid[],id)`, ids, issueID, projectID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load created issue activities for notifications: %w", err)
	}
	defer rows.Close()
	activities := make([]issueNotificationActivity, 0, len(ids))
	for rows.Next() {
		var value issueNotificationActivity
		if err := rows.Scan(&value.id, &value.commentID, &value.actorID, &value.oldIdentifier, &value.newIdentifier, &value.verb, &value.field, &value.comment, &value.oldValue, &value.newValue, &value.createdAt); err != nil {
			return nil, fmt.Errorf("read created issue activity for notification: %w", err)
		}
		activities = append(activities, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate issue activities for notifications: %w", err)
	}
	return activities, nil
}

func notificationsSkipActivityType(activityType string) bool {
	switch activityType {
	case "cycle.activity.created", "cycle.activity.deleted", "module.activity.created", "module.activity.deleted", "issue_reaction.activity.created", "issue_reaction.activity.deleted", "comment_reaction.activity.created", "comment_reaction.activity.deleted", "issue_vote.activity.created", "issue_vote.activity.deleted", "issue_draft.activity.created", "issue_draft.activity.updated", "issue_draft.activity.deleted":
		return true
	default:
		return false
	}
}

func issueActivityMentions(input IssueActivityInput, activities []issueNotificationActivity) ([]uuid.UUID, []uuid.UUID, map[uuid.UUID][]uuid.UUID, []uuid.UUID) {
	requestedDescription := descriptionUserMentions(input.RequestedData)
	currentDescription := descriptionUserMentions(input.CurrentInstance)
	newDescription := differenceUUIDs(requestedDescription, currentDescription)
	removedDescription := differenceUUIDs(currentDescription, requestedDescription)
	newComments := make(map[uuid.UUID][]uuid.UUID)
	allComments := make([]uuid.UUID, 0)
	for _, activity := range activities {
		if activity.field != "comment" {
			continue
		}
		newMentions := extractHTMLUserMentions(activity.newValue)
		oldMentions := extractHTMLUserMentions(activity.oldValue)
		newComments[activity.id] = differenceUUIDs(newMentions, oldMentions)
		allComments = append(allComments, newMentions...)
	}
	return newDescription, removedDescription, newComments, uniqueUUIDs(allComments)
}

func descriptionUserMentions(raw json.RawMessage) []uuid.UUID {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	var description *string
	if err := json.Unmarshal(value["description_html"], &description); err != nil || description == nil {
		return nil
	}
	return extractHTMLUserMentions(description)
}

func extractHTMLUserMentions(raw *string) []uuid.UUID {
	if raw == nil || *raw == "" {
		return nil
	}
	root, err := html.Parse(strings.NewReader(*raw))
	if err != nil {
		return nil
	}
	mentions := make([]uuid.UUID, 0)
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "mention-component" {
			attributes := make(map[string]string, len(node.Attr))
			for _, attribute := range node.Attr {
				attributes[strings.ToLower(attribute.Key)] = attribute.Val
			}
			if attributes["entity_name"] == "user_mention" {
				if id, err := uuid.Parse(attributes["entity_identifier"]); err == nil {
					mentions = append(mentions, id)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return uniqueUUIDs(mentions)
}

func (event issueNotificationEvent) syncDescriptionMentions(ctx context.Context, tx pgx.Tx, added, removed []uuid.UUID) error {
	for _, mentionID := range added {
		id := uuid.NewSHA1(event.id, []byte("plane.issue_mention:"+mentionID.String()))
		if _, err := tx.Exec(ctx, `INSERT INTO issue_mentions (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,issue_id,mention_id) VALUES ($1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$6) ON CONFLICT (issue_id,mention_id) WHERE deleted_at IS NULL DO NOTHING`, id, nullableUUIDString(event.input.ActorID), event.workspaceID, event.projectID, event.issueID, mentionID); err != nil {
			return fmt.Errorf("save issue description mention: %w", err)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE issue_mentions SET deleted_at=NOW(),updated_at=NOW() WHERE issue_id=$1 AND mention_id=ANY($2::uuid[]) AND deleted_at IS NULL`, event.issueID, removed); err != nil {
			return fmt.Errorf("remove issue description mention: %w", err)
		}
	}
	return nil
}

func activeProjectMembers(ctx context.Context, tx pgx.Tx, projectID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := tx.Query(ctx, `SELECT member_id FROM project_members WHERE project_id=$1 AND is_active=true`, projectID)
	if err != nil {
		return nil, fmt.Errorf("load active project members for mentions: %w", err)
	}
	defer rows.Close()
	members := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read active project member for mentions: %w", err)
		}
		members[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active project members for mentions: %w", err)
	}
	return members, nil
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	set := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	result := make([]uuid.UUID, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func differenceUUIDs(values, excluded []uuid.UUID) []uuid.UUID {
	excludedSet := make(map[uuid.UUID]struct{}, len(excluded))
	for _, value := range excluded {
		excludedSet[value] = struct{}{}
	}
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, exists := excludedSet[value]; !exists {
			result = append(result, value)
		}
	}
	return uniqueUUIDs(result)
}

func unionUUIDs(groups ...[]uuid.UUID) []uuid.UUID {
	all := make([]uuid.UUID, 0)
	for _, group := range groups {
		all = append(all, group...)
	}
	return uniqueUUIDs(all)
}

func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func issueActivityByID(activities []issueNotificationActivity, target uuid.UUID) (issueNotificationActivity, bool) {
	for _, activity := range activities {
		if activity.id == target {
			return activity, true
		}
	}
	return issueNotificationActivity{}, false
}

func mustUUID(value *string) *uuid.UUID {
	if value == nil {
		return nil
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil
	}
	return &parsed
}

func uuidString(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return value.String()
}
func stringifiedNullable(value *string) string {
	if value == nil {
		return "None"
	}
	return *value
}
