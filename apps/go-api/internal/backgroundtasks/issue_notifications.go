package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/net/html"
)

func (store *PostgresIssueActivityStore) ProcessIssueNotifications(ctx context.Context, eventID string, input IssueActivityInput, activityIDs []string) error {
	if err := validateIssueActivityInput(eventID, input); err != nil {
		return err
	}
	if input.IssueID == nil || len(activityIDs) == 0 || notificationsSkipActivityType(input.Type) {
		return nil
	}
	projectID, _ := uuid.Parse(input.ProjectID)
	issueID, _ := uuid.Parse(*input.IssueID)
	eventUUID, _ := uuid.Parse(eventID)
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin issue notification transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if input.WorkspaceID != "" {
		if scopeErr := lockBoundIssueActivity(ctx, tx, input, uuid.MustParse(input.WorkspaceID), false); scopeErr != nil {
			return scopeErr
		}
	}
	var workspaceID uuid.UUID
	var workspaceSlug, issueName, projectIdentifier string
	var sequenceID int64
	var creatorID, stateName, stateGroup *string
	err = tx.QueryRow(ctx, `SELECT project.workspace_id,workspace.slug,issue.name,project.identifier,issue.sequence_id,issue.created_by_id,state.name,state."group" FROM issues issue JOIN projects project ON project.id=issue.project_id JOIN workspaces workspace ON workspace.id=project.workspace_id LEFT JOIN states state ON state.id=issue.state_id WHERE issue.id=$1 AND issue.project_id=$2 AND issue.deleted_at IS NULL`, issueID, projectID).Scan(&workspaceID, &workspaceSlug, &issueName, &projectIdentifier, &sequenceID, &creatorID, &stateName, &stateGroup)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return fmt.Errorf("load issue for notifications: %w", err)
	}
	if input.Subscriber == nil || *input.Subscriber {
		if input.ActorID != nil {
			if _, err := tx.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,subscriber_id) VALUES ($1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$2) ON CONFLICT (issue_id,subscriber_id) WHERE deleted_at IS NULL DO NOTHING`, uuid.New(), *input.ActorID, projectID, workspaceID, issueID); err != nil {
				return fmt.Errorf("subscribe issue activity actor: %w", err)
			}
		}
	}
	activities, err := loadIssueActivities(ctx, tx, activityIDs, issueID, projectID, workspaceID)
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
		if err := syncIssueDescriptionMentions(ctx, tx, eventUUID, issueID, projectID, workspaceID, input.ActorID, newDescriptionMentions, removedDescriptionMentions); err != nil {
			return err
		}
	}
	activeMembers, err := activeProjectMembers(ctx, tx, projectID)
	if err != nil {
		return err
	}
	for _, mentionID := range unionUUIDs(newDescriptionMentions, allCommentMentions) {
		if _, err := tx.Exec(ctx, `INSERT INTO issue_subscribers (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,project_id,workspace_id,issue_id,subscriber_id) SELECT $1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$6 WHERE EXISTS(SELECT 1 FROM project_members WHERE project_id=$3 AND member_id=$6 AND is_active=true) AND NOT EXISTS(SELECT 1 FROM issue_assignees WHERE project_id=$3 AND issue_id=$5 AND assignee_id=$6 AND deleted_at IS NULL) AND NOT EXISTS(SELECT 1 FROM issues WHERE id=$5 AND created_by_id=$6) ON CONFLICT (issue_id,subscriber_id) WHERE deleted_at IS NULL DO NOTHING`, uuid.New(), nullableUUIDString(input.ActorID), projectID, workspaceID, issueID, mentionID); err != nil {
			return fmt.Errorf("subscribe issue mention: %w", err)
		}
	}
	var boundWorkspace any
	if input.WorkspaceID != "" {
		boundWorkspace = input.WorkspaceID
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
 ORDER BY subscriber.id`, issueID, projectID, nullableUUIDString(input.ActorID), boundWorkspace)
	if err != nil {
		return fmt.Errorf("load issue notification recipients: %w", err)
	}
	type recipient struct {
		id                                  uuid.UUID
		createdBy                           *uuid.UUID
		assignee                            bool
		property, state, comment, completed bool
	}
	recipients := make([]recipient, 0)
	for rows.Next() {
		var value recipient
		if err := rows.Scan(&value.id, &value.createdBy, &value.assignee, &value.property, &value.state, &value.comment, &value.completed); err != nil {
			rows.Close()
			return fmt.Errorf("read issue notification recipient: %w", err)
		}
		recipients = append(recipients, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate issue notification recipients: %w", err)
	}
	rows.Close()
	for _, activity := range activities {
		for _, receiver := range recipients {
			if containsUUID(newDescriptionMentions, receiver.id) || containsUUID(newCommentRecipientIDs, receiver.id) {
				continue
			}
			sender := "in_app:issue_activities:subscribed"
			if creatorID != nil && *creatorID == receiver.id.String() {
				sender = "in_app:issue_activities:created"
			} else if receiver.assignee && !(receiver.createdBy != nil && *receiver.createdBy == receiver.id) {
				sender = "in_app:issue_activities:assigned"
			}
			issueData := map[string]any{"id": issueID.String(), "name": issueName, "identifier": projectIdentifier, "sequence_id": sequenceID, "state_name": stateName, "state_group": stateGroup}
			commentText := ""
			if activity.commentID != nil {
				if err := tx.QueryRow(ctx, `SELECT comment_stripped FROM issue_comments WHERE id=$1 AND issue_id=$2 AND project_id=$3 AND workspace_id=$4`, *activity.commentID, issueID, projectID, workspaceID).Scan(&commentText); err != nil && !errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("load notification comment text: %w", err)
				}
			}
			activityData := map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "issue_comment": commentText, "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier)}
			data, err := json.Marshal(map[string]any{"issue": issueData, "issue_activity": activityData})
			if err != nil {
				return fmt.Errorf("encode issue notification: %w", err)
			}
			notificationID := uuid.NewSHA1(eventUUID, []byte("plane.notification:"+activity.id.String()+":"+receiver.id.String()))
			if _, err := tx.Exec(ctx, `INSERT INTO notifications (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,data,entity_identifier,entity_name,title,message,message_html,message_stripped,sender,triggered_by_id,receiver_id,read_at,snoozed_till,archived_at) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4::jsonb,$5,'issue',$6,NULL,'<p></p>','',$7,$8,$9,NULL,NULL,NULL) ON CONFLICT (id) DO NOTHING`, notificationID, workspaceID, projectID, data, issueID, activity.comment, sender, nullableUUIDString(input.ActorID), receiver.id); err != nil {
				return fmt.Errorf("insert issue notification: %w", err)
			}
			sendEmail := false
			if activity.field != "description" {
				switch activity.field {
				case "state":
					sendEmail = receiver.state
					if !sendEmail && receiver.completed && activity.newIdentifier != nil {
						var completed bool
						if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM states WHERE id=$1 AND project_id=$2 AND "group"='completed')`, *activity.newIdentifier, projectID).Scan(&completed); err != nil {
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
				emailData := map[string]any{"issue": map[string]any{"id": issueID.String(), "name": issueName, "identifier": projectIdentifier, "project_id": projectID.String(), "workspace_slug": workspaceSlug, "workspace_id": input.WorkspaceID, "sequence_id": sequenceID, "state_name": stateName, "state_group": stateGroup}, "issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "issue_comment": commentText, "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier), "activity_time": activity.createdAt}}
				emailJSON, err := json.Marshal(emailData)
				if err != nil {
					return fmt.Errorf("encode email notification log: %w", err)
				}
				logID := uuid.NewSHA1(eventUUID, []byte("plane.email_notification:"+activity.id.String()+":"+receiver.id.String()))
				if _, err := tx.Exec(ctx, `INSERT INTO email_notification_logs (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,receiver_id,triggered_by_id,entity_identifier,entity_name,data,processed_at,sent_at,entity,old_value,new_value) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4,'issue',$5::jsonb,NULL,NULL,'',NULL,NULL) ON CONFLICT (id) DO NOTHING`, logID, receiver.id, nullableUUIDString(input.ActorID), issueID, emailJSON); err != nil {
					return fmt.Errorf("insert email notification log: %w", err)
				}
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
			if err := insertMentionNotification(ctx, tx, eventUUID, issueID, projectID, workspaceID, workspaceSlug, issueName, projectIdentifier, sequenceID, stateName, stateGroup, input, activity, mentionID, "You have been mentioned in the issue "+issueName); err != nil {
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
				if err := insertMentionNotification(ctx, tx, eventUUID, issueID, projectID, workspaceID, workspaceSlug, issueName, projectIdentifier, sequenceID, stateName, stateGroup, input, last, mentionID, "You have been mentioned in the issue "+issueName); err != nil {
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

func insertMentionNotification(ctx context.Context, tx pgx.Tx, eventID, issueID, projectID, workspaceID uuid.UUID, workspaceSlug, issueName, projectIdentifier string, sequenceID int64, stateName, stateGroup *string, input IssueActivityInput, activity issueNotificationActivity, receiver uuid.UUID, message string) error {
	actorID := nullableUUIDString(input.ActorID)
	data, err := json.Marshal(map[string]any{
		"issue":          map[string]any{"id": issueID.String(), "name": issueName, "identifier": projectIdentifier, "sequence_id": sequenceID, "state_name": stateName, "state_group": stateGroup},
		"issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": activity.field, "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier)},
	})
	if err != nil {
		return fmt.Errorf("encode mention notification: %w", err)
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode mention message: %w", err)
	}
	notificationID := uuid.NewSHA1(eventID, []byte("plane.notification:mention:"+activity.id.String()+":"+receiver.String()))
	if _, err := tx.Exec(ctx, `INSERT INTO notifications (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,data,entity_identifier,entity_name,title,message,message_html,message_stripped,sender,triggered_by_id,receiver_id,read_at,snoozed_till,archived_at) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4::jsonb,$5,'issue',$6,$7::jsonb,'<p></p>','',$8,$9,$10,NULL,NULL,NULL) ON CONFLICT (id) DO NOTHING`, notificationID, workspaceID, projectID, data, issueID, activity.comment, messageJSON, "in_app:issue_activities:mentioned", actorID, receiver); err != nil {
		return fmt.Errorf("insert issue mention notification: %w", err)
	}
	var sendEmail bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT mention FROM user_notification_preferences WHERE user_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1),true)`, receiver).Scan(&sendEmail); err != nil {
		return fmt.Errorf("load mention email preference: %w", err)
	}
	if !sendEmail {
		return nil
	}
	emailData, err := json.Marshal(map[string]any{
		"issue":          map[string]any{"id": issueID.String(), "name": issueName, "identifier": projectIdentifier, "project_id": projectID.String(), "workspace_slug": workspaceSlug, "workspace_id": input.WorkspaceID, "sequence_id": sequenceID, "state_name": stateName, "state_group": stateGroup},
		"issue_activity": map[string]any{"id": activity.id.String(), "verb": activity.verb, "field": "mention", "actor": uuidString(activity.actorID), "new_value": notificationActivityValue(activity.field, activity.newValue), "old_value": notificationActivityValue(activity.field, activity.oldValue), "old_identifier": uuidString(activity.oldIdentifier), "new_identifier": uuidString(activity.newIdentifier), "activity_time": activity.createdAt},
	})
	if err != nil {
		return fmt.Errorf("encode mention email log: %w", err)
	}
	logID := uuid.NewSHA1(eventID, []byte("plane.email_notification:mention:"+activity.id.String()+":"+receiver.String()))
	if _, err := tx.Exec(ctx, `INSERT INTO email_notification_logs (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,receiver_id,triggered_by_id,entity_identifier,entity_name,data,processed_at,sent_at,entity,old_value,new_value) VALUES ($1,NOW(),NOW(),NULL,NULL,NULL,$2,$3,$4,'issue',$5::jsonb,NULL,NULL,'',NULL,NULL) ON CONFLICT (id) DO NOTHING`, logID, receiver, actorID, issueID, emailData); err != nil {
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

func syncIssueDescriptionMentions(ctx context.Context, tx pgx.Tx, eventID, issueID, projectID, workspaceID uuid.UUID, actorID *string, added, removed []uuid.UUID) error {
	for _, mentionID := range added {
		id := uuid.NewSHA1(eventID, []byte("plane.issue_mention:"+mentionID.String()))
		if _, err := tx.Exec(ctx, `INSERT INTO issue_mentions (id,created_at,updated_at,deleted_at,created_by_id,updated_by_id,workspace_id,project_id,issue_id,mention_id) VALUES ($1,NOW(),NOW(),NULL,$2,$2,$3,$4,$5,$6) ON CONFLICT (issue_id,mention_id) WHERE deleted_at IS NULL DO NOTHING`, id, nullableUUIDString(actorID), workspaceID, projectID, issueID, mentionID); err != nil {
			return fmt.Errorf("save issue description mention: %w", err)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE issue_mentions SET deleted_at=NOW(),updated_at=NOW() WHERE issue_id=$1 AND mention_id=ANY($2::uuid[]) AND deleted_at IS NULL`, issueID, removed); err != nil {
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
