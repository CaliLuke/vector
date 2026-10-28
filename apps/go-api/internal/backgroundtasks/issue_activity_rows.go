package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type issueActivityEvent struct {
	IssueActivityInput
	requested map[string]json.RawMessage
	current   map[string]json.RawMessage
}

func (event issueActivityEvent) baseRow() issueActivityRow {
	return issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Epoch: event.Epoch}
}

func createdIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	var createdAt time.Time
	var creatorID *string
	if err := tx.QueryRow(ctx, `SELECT created_at,created_by_id FROM issues WHERE id=$1`, *event.IssueID).Scan(&createdAt, &creatorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: issue does not exist", ErrPermanentIssueActivity)
		}
		return nil, fmt.Errorf("load issue creation metadata: %w", err)
	}
	rows := []issueActivityRow{{IssueID: event.IssueID, ActorID: creatorID, Verb: "created", Comment: "created the issue", Epoch: event.Epoch, CreatedAt: &createdAt}}
	if _, ok := requested["assignee_ids"]; ok {
		requestedAssignees, currentAssignees := activityIDSet(requested, "assignee_ids"), activityIDSet(current, "assignee_ids")
		for _, id := range sortedActivityIDs(requestedAssignees) {
			if _, exists := currentAssignees[id]; exists {
				continue
			}
			name, err := issueActivityRelatedName(ctx, tx, "users", "id", "display_name", id)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "assignees", OldValue: stringPointerValue(""), NewValue: name, Comment: "added assignee ", NewIdentifier: stringPointerValue(id), Epoch: event.Epoch})
		}
	}
	return rows, nil
}

func updatedIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	rows := make([]issueActivityRow, 0, 13)
	milestoneRow, err := milestoneActivityRow(event.IssueActivityInput)
	if err != nil {
		return nil, err
	}
	if milestoneRow != nil {
		rows = append(rows, *milestoneRow)
	}
	// Description edits can refresh existing history before the remaining lookups.
	scalarRows, err := scalarIssueActivityRows(ctx, tx, event)
	if err != nil {
		return nil, err
	}
	rows = append(rows, scalarRows...)
	referenceRows, err := referenceIssueActivityRows(ctx, tx, event)
	if err != nil {
		return nil, err
	}
	rows = append(rows, referenceRows...)
	rows = append(rows, archiveIssueActivityRows(event)...)
	collectionRows, err := collectionIssueActivityRows(ctx, tx, event)
	if err != nil {
		return nil, err
	}
	return append(rows, collectionRows...), nil
}

func scalarIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	var rows []issueActivityRow
	for _, field := range []struct{ key, column, comment string }{
		{"name", "name", "updated the name to"},
		{"priority", "priority", "updated the priority to"},
		{"target_date", "target_date", "updated the target date to"},
		{"start_date", "start_date", "updated the start date to "},
		{"description_html", "description", "updated the description to"},
	} {
		requestedValue, requestedOK := requested[field.key]
		if !requestedOK || sameActivityJSON(current[field.key], requestedValue) {
			continue
		}
		oldValue := activityText(current[field.key])
		newValue := activityText(requestedValue)
		if field.key == "description_html" {
			refreshed, err := refreshLastDescriptionActivity(ctx, tx, event.IssueActivityInput)
			if err != nil {
				return nil, err
			}
			if refreshed {
				continue
			}
		}
		rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: field.column, OldValue: oldValue, NewValue: newValue, Comment: field.comment, Epoch: event.Epoch})
	}
	return rows, nil
}

func referenceIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	var rows []issueActivityRow
	if requestedParent, ok := firstActivityValue(requested, "parent_id", "parent"); ok && !sameActivityJSON(firstActivityValueRaw(current, "parent_id", "parent"), requestedParent) {
		oldID, newID := activityText(firstActivityValueRaw(current, "parent_id", "parent")), activityText(requestedParent)
		oldValue, err := issueActivityIdentifier(ctx, tx, oldID)
		if err != nil {
			return nil, err
		}
		newValue, err := issueActivityIdentifier(ctx, tx, newID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "parent", OldValue: oldValue, NewValue: newValue, Comment: "updated the parent issue to", OldIdentifier: oldID, NewIdentifier: newID, Epoch: event.Epoch})
	}
	if requestedState, ok := firstActivityValue(requested, "state_id", "state"); ok && !sameActivityJSON(firstActivityValueRaw(current, "state_id", "state"), requestedState) {
		oldID, newID := activityText(firstActivityValueRaw(current, "state_id", "state")), activityText(requestedState)
		oldValue, err := issueActivityStateName(ctx, tx, event.ProjectID, oldID)
		if err != nil {
			return nil, err
		}
		newValue, err := issueActivityStateName(ctx, tx, event.ProjectID, newID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "state", OldValue: oldValue, NewValue: newValue, Comment: "updated the state to", OldIdentifier: oldID, NewIdentifier: newID, Epoch: event.Epoch})
	}
	if requestedPriority, ok := requested["closed_to"]; ok && string(requestedPriority) != "null" {
		newID := activityText(requestedPriority)
		newValue, err := issueActivityStateName(ctx, tx, event.ProjectID, newID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "state", NewValue: newValue, Comment: "Plane updated the state to ", NewIdentifier: newID, Epoch: event.Epoch})
	}
	return rows, nil
}

func archiveIssueActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	current := event.current
	var rows []issueActivityRow
	if requestedArchivedAt, ok := requested["archived_at"]; ok && !sameActivityJSON(current["archived_at"], requestedArchivedAt) {
		if string(requestedArchivedAt) == "null" {
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "archived_at", OldValue: stringPointerValue("archive"), NewValue: stringPointerValue("restore"), Comment: "has restored the issue", Epoch: event.Epoch})
		} else if string(requested["automation"]) == "true" {
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "archived_at", NewValue: stringPointerValue("archive"), Comment: "Plane has archived the issue", Epoch: event.Epoch})
		} else {
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: "archived_at", NewValue: stringPointerValue("manual_archive"), Comment: "Actor has archived the issue", Epoch: event.Epoch})
		}
	}
	return rows
}

func collectionIssueActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	var rows []issueActivityRow
	for _, collection := range []struct {
		keys                                                          []string
		table, idColumn, nameColumn, field, addComment, removeComment string
	}{
		{[]string{"label_ids", "labels"}, "labels", "id", "name", "labels", "added label ", "removed label "},
		{[]string{"assignee_ids", "assignees"}, "users", "id", "display_name", "assignees", "added assignee ", "removed assignee "},
	} {
		requestedSet := activityIDSet(requested, collection.keys...)
		currentSet := activityIDSet(current, collection.keys...)
		for _, id := range sortedActivityIDs(requestedSet) {
			if _, exists := currentSet[id]; exists {
				continue
			}
			name, err := issueActivityRelatedName(ctx, tx, collection.table, collection.idColumn, collection.nameColumn, id)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: collection.field, OldValue: stringPointerValue(""), NewValue: name, Comment: collection.addComment, NewIdentifier: stringPointerValue(id), Epoch: event.Epoch})
		}
		for _, id := range sortedActivityIDs(currentSet) {
			if _, exists := requestedSet[id]; exists {
				continue
			}
			name, err := issueActivityRelatedName(ctx, tx, collection.table, collection.idColumn, collection.nameColumn, id)
			if err != nil {
				return nil, err
			}
			rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: collection.field, OldValue: name, NewValue: stringPointerValue(""), Comment: collection.removeComment, OldIdentifier: stringPointerValue(id), Epoch: event.Epoch})
		}
	}
	return rows, nil
}

func cycleActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	switch event.Type {
	case "cycle.activity.created":
		rows := make([]issueActivityRow, 0)
		for _, record := range rawObjectArray(current["updated_cycle_issues"]) {
			issueID := activityText(record["issue_id"])
			oldID := activityText(record["old_cycle_id"])
			newID := activityText(record["new_cycle_id"])
			oldName, err := lookupActivityName(ctx, tx, "cycles", oldID)
			if err != nil {
				return nil, err
			}
			newName, err := lookupActivityName(ctx, tx, "cycles", newID)
			if err != nil {
				return nil, err
			}
			oldText, newText := stringOrEmpty(oldName), stringOrEmpty(newName)
			rows = append(rows, issueActivityRow{IssueID: issueID, ActorID: event.ActorID, Verb: "updated", Field: "cycles", OldValue: oldText, NewValue: newText, Comment: fmt.Sprintf("updated cycle from %s\n                to %s", *oldText, *newText), OldIdentifier: oldID, NewIdentifier: newID, Epoch: event.Epoch})
		}
		for _, record := range rawObjectArray(current["created_cycle_issues"]) {
			fields := decodeActivityObject(record["fields"])
			issueID := activityText(fields["issue"])
			cycleID := activityText(fields["cycle"])
			name, err := lookupActivityName(ctx, tx, "cycles", cycleID)
			if err != nil {
				return nil, err
			}
			name = stringOrEmpty(name)
			rows = append(rows, issueActivityRow{IssueID: issueID, ActorID: event.ActorID, Verb: "created", Field: "cycles", OldValue: stringPointerValue(""), NewValue: name, Comment: "added cycle " + *name, NewIdentifier: cycleID, Epoch: event.Epoch})
		}
		return rows, nil
	case "cycle.activity.deleted":
		cycleID := activityText(requested["cycle_id"])
		cycleName, err := lookupActivityName(ctx, tx, "cycles", cycleID)
		if err != nil {
			return nil, err
		}
		if cycleName == nil {
			cycleName = stringOrEmpty(activityText(requested["cycle_name"]))
		}
		rows := make([]issueActivityRow, 0)
		for _, issue := range rawStringArray(requested["issues"]) {
			name := stringOrEmpty(cycleName)
			rows = append(rows, issueActivityRow{IssueID: &issue, ActorID: event.ActorID, Verb: "deleted", Field: "cycles", OldValue: name, NewValue: stringPointerValue(""), Comment: "removed this issue from " + *name, OldIdentifier: cycleID, Epoch: event.Epoch})
		}
		return rows, nil
	}
	return []issueActivityRow{event.baseRow()}, nil
}

func moduleActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "module.activity.created":
		moduleID := activityText(requested["module_id"])
		name, err := lookupActivityName(ctx, tx, "modules", moduleID)
		if err != nil {
			return nil, err
		}
		name = stringOrEmpty(name)
		row.Verb, row.Field, row.OldValue, row.NewValue = "created", "modules", stringPointerValue(""), name
		row.Comment = "added module " + *name
		row.NewIdentifier = moduleID
	case "module.activity.deleted":
		name := stringOrEmpty(activityText(current["module_name"]))
		row.Verb, row.Field, row.OldValue, row.NewValue = "deleted", "modules", name, stringPointerValue("")
		row.Comment = "removed this issue from " + *name
		row.OldIdentifier = activityText(requested["module_id"])
	}
	return []issueActivityRow{row}, nil
}

func legacyRelationActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	row := event.baseRow()
	switch event.Type {
	case "issue_relation.activity.created":
		if len(event.CurrentInstance) == 0 || string(event.CurrentInstance) == "null" {
			relation := stringOrEmpty(activityText(requested["relation_type"]))
			rows := make([]issueActivityRow, 0)
			for _, relatedID := range rawStringArray(requested["issues"]) {
				identifier, err := issueActivityIdentifier(ctx, tx, &relatedID)
				if err != nil {
					return nil, err
				}
				rows = append(rows, issueActivityRow{IssueID: event.IssueID, ActorID: event.ActorID, Verb: "updated", Field: *relation, OldValue: stringPointerValue(""), NewValue: identifier, Comment: "added " + *relation + " relation", OldIdentifier: &relatedID, Epoch: event.Epoch})
				currentIdentifier, err := issueActivityIdentifier(ctx, tx, event.IssueID)
				if err != nil {
					return nil, err
				}
				inverse := inverseActivityRelation(*relation)
				rows = append(rows, issueActivityRow{IssueID: &relatedID, ActorID: event.ActorID, Verb: "updated", Field: inverse, OldValue: stringPointerValue(""), NewValue: currentIdentifier, Comment: "added " + inverse + " relation", OldIdentifier: event.IssueID, Epoch: event.Epoch})
			}
			return rows, nil
		}
		return nil, nil
	case "issue_relation.activity.deleted":
		relation := stringOrEmpty(activityText(requested["relation_type"]))
		relatedID := activityText(requested["related_issue"])
		identifier, err := issueActivityIdentifier(ctx, tx, relatedID)
		if err != nil {
			return nil, err
		}
		row.Verb, row.Field, row.OldValue, row.NewValue = "deleted", *relation, identifier, stringPointerValue("")
		row.Comment = "deleted " + *relation + " relation"
		row.OldIdentifier = relatedID
		currentIdentifier, err := issueActivityIdentifier(ctx, tx, event.IssueID)
		if err != nil {
			return nil, err
		}
		inverse := inverseActivityRelation(*relation)
		return []issueActivityRow{row, {IssueID: relatedID, ActorID: event.ActorID, Verb: "deleted", Field: inverse, OldValue: currentIdentifier, NewValue: stringPointerValue(""), Comment: "deleted " + *relation + " relation", OldIdentifier: relatedID, Epoch: event.Epoch}}, nil
	}
	return []issueActivityRow{row}, nil
}

func commentActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "comment.activity.created":
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "comment", "created a comment", stringOrEmpty(activityString(requested["comment_html"]))
		row.CommentID = activityString(requested["id"])
		row.NewIdentifier = row.CommentID
	case "comment.activity.updated":
		oldValue, newValue := stringOrEmpty(activityString(current["comment_html"])), stringOrEmpty(activityString(requested["comment_html"]))
		if oldValue == newValue {
			return nil
		}
		row.Verb, row.Field, row.Comment, row.OldValue, row.NewValue = "updated", "comment", "updated a comment", oldValue, newValue
		row.CommentID = activityString(current["id"])
		row.OldIdentifier = row.CommentID
		row.NewIdentifier = row.CommentID
	case "comment.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "comment", "deleted the comment"
		row.CommentID = activityString(requested["comment_id"])
	}
	return []issueActivityRow{row}
}

func linkActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "link.activity.created":
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "link", "created a link", stringOrEmpty(activityString(requested["url"]))
		row.NewIdentifier = activityString(requested["id"])
	case "link.activity.updated":
		oldValue, newValue := stringOrEmpty(activityString(current["url"])), stringOrEmpty(activityString(requested["url"]))
		if oldValue == newValue {
			return nil
		}
		row.Verb, row.Field, row.Comment, row.OldValue, row.NewValue = "updated", "link", "updated a link", oldValue, newValue
		row.OldIdentifier = activityString(current["id"])
		row.NewIdentifier = activityString(current["id"])
	case "link.activity.deleted":
		oldValue := stringOrEmpty(activityString(current["url"]))
		row.Verb, row.Field, row.Comment, row.OldValue = "deleted", "link", "deleted the link", oldValue
		empty := ""
		row.NewValue = &empty
	}
	return []issueActivityRow{row}
}

func attachmentActivityRows(event issueActivityEvent) []issueActivityRow {
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "attachment.activity.created":
		row.Verb, row.Field, row.Comment = "created", "attachment", "created an attachment"
		row.NewValue = activityString(current["asset"])
		row.NewIdentifier = activityString(current["id"])
	case "attachment.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "attachment", "deleted the attachment"
	}
	return []issueActivityRow{row}
}

func reactionActivityRows(ctx context.Context, tx pgx.Tx, event issueActivityEvent) ([]issueActivityRow, error) {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "issue_reaction.activity.created", "comment_reaction.activity.created":
		reaction := activityString(requested["reaction"])
		if reaction == nil {
			return nil, nil
		}
		row.Verb, row.Field, row.Comment, row.NewValue = "created", "reaction", "added the reaction", reaction
		if event.Type == "issue_reaction.activity.created" {
			var err error
			row.NewIdentifier, err = findActivityReactionID(ctx, tx, event.ProjectID, event.ActorID, *reaction)
			if err != nil {
				return nil, err
			}
		} else {
			var commentID *string
			var err error
			row.NewIdentifier, commentID, err = findCommentReaction(ctx, tx, event.ProjectID, event.ActorID, *reaction)
			if err != nil {
				return nil, err
			}
			if commentID != nil {
				var issueID string
				if err := tx.QueryRow(ctx, `SELECT issue_id FROM issue_comments WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL`, *commentID, event.ProjectID).Scan(&issueID); err == nil {
					row.IssueID = &issueID
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("load comment reaction issue: %w", err)
				}
			}
		}
		if row.NewIdentifier == nil {
			return nil, nil
		}
	case "issue_reaction.activity.deleted", "comment_reaction.activity.deleted":
		reaction := activityString(current["reaction"])
		if reaction == nil {
			return nil, nil
		}
		id := activityString(current["identifier"])
		row.Verb, row.Field, row.Comment, row.OldValue, row.OldIdentifier = "deleted", "reaction", "removed the reaction", reaction, id
		if event.Type == "comment_reaction.activity.deleted" {
			if commentID := activityString(current["comment_id"]); commentID != nil {
				var issueID string
				if err := tx.QueryRow(ctx, `SELECT issue_id FROM issue_comments WHERE id=$1 AND project_id=$2 AND deleted_at IS NULL`, *commentID, event.ProjectID).Scan(&issueID); err == nil {
					row.IssueID = &issueID
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("load comment reaction issue: %w", err)
				}
			}
		}
	}
	return []issueActivityRow{row}, nil
}

func voteActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	switch event.Type {
	case "issue_vote.activity.created":
		if vote := activityString(requested["vote"]); vote != nil {
			row.Verb, row.Field, row.Comment, row.NewValue = "updated", "vote", "added the vote", vote
		} else {
			return nil
		}
	case "issue_vote.activity.deleted":
		if vote := activityString(current["vote"]); vote != nil {
			row.Verb, row.Field, row.Comment, row.OldValue = "deleted", "vote", "removed the vote", vote
			row.OldIdentifier = activityString(current["identifier"])
		} else {
			return nil
		}
	}
	return []issueActivityRow{row}
}

func draftActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	row := event.baseRow()
	switch event.Type {
	case "issue_draft.activity.created":
		row.Verb, row.Field, row.Comment = "created", "draft", "drafted the issue"
	case "issue_draft.activity.updated":
		row.Verb = "updated"
		if raw := requested["is_draft"]; len(raw) > 0 && string(raw) == "false" {
			row.Comment = "created the issue"
		} else {
			row.Field, row.Comment = "draft", "updated the draft issue"
		}
	case "issue_draft.activity.deleted":
		row.Verb, row.Field, row.Comment = "deleted", "draft", "deleted the draft issue"
		row.IssueID = nil
	}
	return []issueActivityRow{row}
}

func intakeActivityRows(event issueActivityEvent) []issueActivityRow {
	requested := event.requested
	current := event.current
	row := event.baseRow()
	status := activityString(requested["status"])
	if status == nil {
		return nil
	}
	row.Verb, row.Field, row.Comment = "updated", "intake", "updated the intake status"
	row.Verb = *status
	row.OldValue = intakeStatus(activityString(current["status"]))
	row.NewValue = intakeStatus(status)
	return []issueActivityRow{row}
}

func refreshLastDescriptionActivity(ctx context.Context, tx pgx.Tx, input IssueActivityInput) (bool, error) {
	var lastID string
	err := tx.QueryRow(ctx, `SELECT id FROM issue_activities WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC,id DESC LIMIT 1`, *input.IssueID).Scan(&lastID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load last issue activity: %w", err)
	}
	var sameActor bool
	if err := tx.QueryRow(ctx, `SELECT field='description' AND actor_id IS NOT DISTINCT FROM $2::uuid FROM issue_activities WHERE id=$1`, lastID, nullableUUIDString(input.ActorID)).Scan(&sameActor); err != nil {
		return false, fmt.Errorf("inspect last description activity: %w", err)
	}
	if !sameActor {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE issue_activities SET created_at=NOW(),updated_at=NOW() WHERE id=$1`, lastID); err != nil {
		return false, fmt.Errorf("refresh description activity timestamp: %w", err)
	}
	return true, nil
}
