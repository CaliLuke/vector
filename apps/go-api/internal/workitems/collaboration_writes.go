package workitems

import (
	"context"
	"errors"
	"io"
	"maps"
	"reflect"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) collaborationActivity(ctx context.Context, q Request, kind, verb string, requested, previous any) error {
	category := map[string]string{"comment": "comment", "link": "link", "issueReaction": "issue_reaction", "commentReaction": "comment_reaction"}[kind]
	var issue any = q.IssueID
	project := q.ProjectID
	if kind == "commentReaction" {
		issue = nil
		if collaborationPublic(q) && verb == "created" {
			project = "None"
		}
	}
	kwargs := map[string]any{"type": category + ".activity." + verb, "requested_data": requested, "current_instance": previous, "actor_id": q.ActorID, "issue_id": issue, "project_id": project, "epoch": r.now().Unix()}
	if !q.API && !collaborationPublic(q) {
		kwargs["notification"], kwargs["origin"] = true, r.origin
	}
	return r.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, kwargs)
}
func (r *Repository) collaborationWebhook(ctx context.Context, q Request, row map[string]any, input djangoinput.Input, previous any) error {
	return r.publisher.Task(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "issue_comment", "model_id": row["id"], "requested_data": input.Fields, "current_instance": previous, "actor_id": q.ActorID, "slug": q.Slug, "origin": r.origin})
}
func (r *Repository) collaborationExternalConflict(ctx context.Context, q Request, input djangoinput.Input, old map[string]any) error {
	// The API source calls request.data.get before serializer validation.
	// Arrays and JSON null therefore fail before any domain write or job.
	if q.API && collaborationComment(q) && (input.Null || input.Kind != "") {
		return errors.New("API comment request data is not an object")
	}
	if !q.API || !collaborationComment(q) || !truthy(input.Fields["external_id"]) {
		return nil
	}
	source, present := input.Fields["external_source"]
	if old == nil && !truthy(source) {
		return nil
	}
	if old != nil {
		if old["external_id"] == djangoinput.ScalarString(input.Fields["external_id"]) {
			return nil
		}
		if !present {
			source = old["external_source"]
		}
	}
	var exists bool
	var visibleID *string
	query := `WITH conflicts AS (
SELECT c.id,` + djangoapi.IssueVisibleSQL("visible_issue", "NULLIF($5,'')::uuid", "w.id") + ` AS visible
FROM issue_comments c JOIN workspaces w ON w.id=c.workspace_id
JOIN issues visible_issue ON visible_issue.id=c.issue_id AND visible_issue.project_id=c.project_id AND visible_issue.workspace_id=c.workspace_id
WHERE c.project_id=$1 AND w.slug=$2 AND c.external_source=$3 AND c.external_id=$4 AND c.deleted_at IS NULL
)
SELECT EXISTS(SELECT 1 FROM conflicts), (SELECT id::text FROM conflicts WHERE visible ORDER BY id LIMIT 1)`
	if err := r.db.QueryRow(ctx, query, q.ProjectID, q.Slug, source, input.Fields["external_id"], q.ActorID).Scan(&exists, &visibleID); err != nil {
		return err
	}
	if exists {
		body := map[string]any{"error": "Work item comment with the same external id and external source already exists"}
		if visibleID != nil {
			body["id"] = *visibleID
		}
		if old != nil {
			body["id"] = old["id"]
		}
		return &djangoapi.HTTPError{Status: 409, Body: body}
	}
	return nil
}

func (r *Repository) validateCollaborationCommentReferences(ctx context.Context, q Request, changes, old map[string]any) error {
	if parentID, present := changes["parent_id"]; present && parentID != nil && parentID != "" {
		var selfID any
		if old != nil {
			selfID = old["id"]
		}
		var visible bool
		args := []any{parentID, q.IssueID, q.ProjectID, q.Slug, selfID}
		visibility := `parent.access='EXTERNAL' AND ` + djangoapi.PublicRelatedIssueSQL("parent_issue", "parent_issue")
		if !collaborationPublic(q) {
			args = append(args, q.ActorID)
			visibility = djangoapi.IssueVisibleSQL("parent_issue", "NULLIF($6,'')::uuid", "parent_workspace.id")
		}
		query := `SELECT EXISTS(SELECT 1 FROM issue_comments parent
JOIN issues parent_issue ON parent_issue.id=parent.issue_id
JOIN workspaces parent_workspace ON parent_workspace.id=parent.workspace_id
WHERE parent.id=$1 AND parent.id IS DISTINCT FROM $5::uuid
AND parent.issue_id=$2 AND parent.project_id=$3 AND parent.workspace_id=parent_workspace.id
AND parent_workspace.slug=$4 AND parent.deleted_at IS NULL
AND ` + visibility + `)`
		if err := r.db.QueryRow(ctx, query, args...).Scan(&visible); err != nil {
			return err
		}
		if !visible {
			return collaborationMissing(q)
		}
	}
	if old == nil {
		return nil
	}
	previousDescription := old["description_id"]
	if descriptionID, present := changes["description_id"]; present && !reflect.DeepEqual(descriptionID, previousDescription) {
		return collaborationMissing(q)
	}
	if previousDescription == nil {
		return nil
	}
	var shared bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_comments other WHERE other.description_id=$1 AND other.id<>$2)`, previousDescription, old["id"]).Scan(&shared); err != nil {
		return err
	}
	if shared {
		return collaborationMissing(q)
	}
	return nil
}

func (r *Repository) writeCollaboration(ctx context.Context, q Request, kind, table string, input djangoinput.Input, changes, old map[string]any) (Result, error) {
	if kind == "link" && (!q.API || q.Method == "POST") {
		var duplicate bool
		var id any
		if old != nil {
			id = old["id"]
		}
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_links WHERE issue_id=$1 AND url=$2 AND deleted_at IS NULL AND ($3::uuid IS NULL OR id<>$3))`, q.IssueID, changes["url"], id).Scan(&duplicate); err != nil {
			return Result{}, err
		}
		if duplicate {
			return Result{}, invalid(map[string]any{"error": "URL already exists for this Issue"})
		}
	}
	// Tenant checks follow source serializer/duplicate precedence and precede writes.
	if err := r.collaborationReference(ctx, q, "issues", q.IssueID); err != nil {
		return Result{}, err
	}
	if kind == "commentReaction" {
		if err := r.collaborationReference(ctx, q, "issue_comments", q.Params["comment_id"]); err != nil {
			return Result{}, err
		}
		if collaborationPublic(q) {
			if err := r.collaborationPublicCommentReference(ctx, q); err != nil {
				return Result{}, err
			}
		}
	}
	if kind == "comment" && !q.API {
		if err := r.collaborationReference(ctx, q, "issue_comments", changes["parent_id"]); err != nil {
			return Result{}, err
		}
	}
	if kind == "comment" && old != nil {
		// Saving content updates the attached description even when its ID
		// was omitted from the request, including the narrower API serializer.
		description := old["description_id"]
		if value, present := changes["description_id"]; present {
			description = value
		}
		if err := r.collaborationReference(ctx, q, "descriptions", description); err != nil {
			return Result{}, err
		}
	}
	if kind == "comment" {
		if err := r.validateCollaborationCommentReferences(ctx, q, changes, old); err != nil {
			return Result{}, err
		}
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	row := maps.Clone(old)
	if old == nil {
		row = collaborationDefault(table, q, project["workspace_id"])
	}
	maps.Copy(row, changes)
	if kind == "comment" && old == nil {
		row["actor_id"] = q.ActorID
		if collaborationPublic(q) {
			row["access"] = "EXTERNAL"
		}
	}
	var previous any
	if old != nil && q.Method != "PUT" {
		previous, err = r.collaborationDump(ctx, q, old, kind, false)
		if err != nil {
			return Result{}, err
		}
	}
	if kind == "comment" && q.Method == "PATCH" && !q.API && !collaborationPublic(q) {
		if raw, present := input.Fields["comment_html"]; present && !reflect.DeepEqual(raw, old["comment_html"]) {
			row["edited_at"] = timestamp(r.now())
		}
	}
	row, err = r.saveCollaboration(ctx, q, table, row, old)
	if err != nil {
		var pg *pgconn.PgError
		if kind == "commentReaction" && !collaborationPublic(q) && errors.As(err, &pg) && pg.Code == "23505" {
			return Result{}, failure(400, "error", "Reaction already exists for the user")
		}
		return Result{}, err
	}
	if q.Method == "PUT" {
		projected, err := r.collaborationProjection(ctx, q, row, kind, true)
		return Result{200, projected}, err
	}
	if kind == "link" {
		if err = r.publisher.Task(ctx, "plane.bgtasks.work_item_link_task.crawl_work_item_link_title", []any{row["id"], row["url"]}, map[string]any{}); err != nil {
			return Result{}, err
		}
	}
	// API creation has a second update_fields save. Preserve its deliberately
	// narrower persisted fields and its in-memory actor/updater representation.
	if q.API && old == nil {
		creator := any(q.ActorID)
		if value, present := input.Fields["created_by"]; present {
			creator = value
		}
		if kind == "comment" {
			created := any(timestamp(r.now()))
			if value, present := input.Fields["created_at"]; present {
				created = value
			}
			if _, err = r.db.Exec(ctx, `UPDATE issue_comments SET created_at=$2,created_by_id=$3 WHERE id=$1`, row["id"], created, creator); err != nil {
				return Result{}, err
			}
			row["created_at"], row["created_by_id"], row["actor_id"], row["updated_by_id"] = created, creator, creator, q.ActorID
		} else {
			if _, err = r.db.Exec(ctx, `UPDATE issue_links SET created_by_id=$2 WHERE id=$1`, row["id"], creator); err != nil {
				return Result{}, err
			}
			row["created_by_id"], row["updated_by_id"] = creator, q.ActorID
		}
	}
	requested, err := dumpRequest(input)
	if err != nil {
		return Result{}, err
	}
	verb := "updated"
	if old == nil {
		verb = "created"
		if q.API {
			fields := strings.Fields("comment_json comment_html access external_source external_id")
			if kind == "link" {
				fields = strings.Fields("title url issue_id")
			}
			projected := map[string]any{}
			for _, field := range fields {
				projected[field] = row[field]
				if field == "issue_id" {
					projected[field] = row["issue_id"]
				}
			}
			requested, err = orderedDump(projected, fields)
		} else if kind == "comment" || kind == "link" {
			requested, err = r.collaborationDump(ctx, q, row, kind, false)
		}
		if err != nil {
			return Result{}, err
		}
	}
	activityQ := q
	if q.API && old == nil {
		activityQ.ActorID = djangoinput.ScalarString(row["created_by_id"])
	}
	if collaborationPublic(q) && kind == "comment" && old != nil {
		previous, err = r.collaborationDump(ctx, q, row, kind, false)
		if err != nil {
			return Result{}, err
		}
	}
	if collaborationPublic(q) && kind != "comment" {
		if err = r.collaborationPublicMember(ctx, q, project["workspace_id"]); err != nil {
			return Result{}, err
		}
	}
	if err = r.collaborationActivity(ctx, activityQ, kind, verb, requested, previous); err != nil {
		return Result{}, err
	}
	if kind == "comment" && (q.API || !collaborationPublic(q)) {
		if err = r.collaborationWebhook(ctx, q, row, input, previous); err != nil {
			return Result{}, err
		}
	}
	if collaborationPublic(q) && kind == "comment" && old == nil {
		if err = r.collaborationPublicMember(ctx, q, project["workspace_id"]); err != nil {
			return Result{}, err
		}
	}
	if kind == "link" && !q.API {
		readQ := q
		readQ.Params = maps.Clone(q.Params)
		readQ.Params["pk"] = row["id"].(string)
		row, err = r.collaborationObject(ctx, readQ, kind, table, true)
		if err != nil {
			var missing *djangoapi.HTTPError
			if errors.As(err, &missing) && missing.Status == 404 {
				return Result{}, requiredObject()
			}
			return Result{}, err
		}
	}
	projected, err := r.collaborationProjection(ctx, q, row, kind, false)
	status := 200
	if old == nil {
		status = 201
	}
	return Result{status, projected}, err
}

func (r *Repository) collaborationPublicCommentReference(ctx context.Context, q Request) error {
	var valid bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_comments c JOIN workspaces w ON w.id=c.workspace_id WHERE c.id=$1 AND c.project_id=$2 AND w.slug=$3 AND c.access='EXTERNAL' AND c.deleted_at IS NULL)`, q.Params["comment_id"], q.ProjectID, q.Slug).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return requiredObject()
	}
	return nil
}
func (r *Repository) collaborationPublicMember(ctx context.Context, q Request, workspace any) error {
	var existing bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL) OR EXISTS(SELECT 1 FROM project_public_members WHERE project_id=$1 AND member_id=$2 AND deleted_at IS NULL)`, q.ProjectID, q.ActorID).Scan(&existing); err != nil {
		return err
	}
	if existing {
		return nil
	}
	row := collaborationDefault("project_public_members", q, workspace)
	_, err := r.saveCollaboration(ctx, q, "project_public_members", row, nil)
	return err
}

func (r *Repository) destroyCollaboration(ctx context.Context, q Request, kind, table string) (Result, error) {
	if err := r.collaborationReference(ctx, q, "issues", q.IssueID); err != nil {
		return Result{}, err
	}
	if kind == "commentReaction" {
		if err := r.collaborationReference(ctx, q, "issue_comments", q.Params["comment_id"]); err != nil {
			return Result{}, err
		}
		if collaborationPublic(q) {
			if err := r.collaborationPublicCommentReference(ctx, q); err != nil {
				return Result{}, err
			}
		}
	}
	row, err := r.collaborationObject(ctx, q, kind, table, false)
	if err != nil {
		return Result{}, err
	}
	var previous any
	var requested any
	if kind == "comment" || kind == "link" {
		previous, err = r.collaborationDump(ctx, q, row, kind, false)
		field := "comment_id"
		if kind == "link" {
			field = "link_id"
		}
		requested, _ = orderedDump(map[string]any{field: row["id"]}, []string{field})
	} else {
		current := map[string]any{"reaction": q.Params["reaction_code"], "identifier": row["id"]}
		fields := []string{"reaction", "identifier"}
		if kind == "commentReaction" {
			current["comment_id"] = q.Params["comment_id"]
			fields = append(fields, "comment_id")
		}
		previous, err = orderedDump(current, fields)
	}
	if err != nil {
		return Result{}, err
	}
	// Private child writes must pass their atomic visibility guard before any
	// activity is published. Public-board actions retain their existing ordering.
	before := !collaborationPublic(q)
	if before {
		if err = r.deleteCollaboration(ctx, q, table, row); err != nil {
			return Result{}, err
		}
	}
	if err = r.collaborationActivity(ctx, q, kind, "deleted", requested, previous); err != nil {
		return Result{}, err
	}
	if !before {
		if err = r.deleteCollaboration(ctx, q, table, row); err != nil {
			return Result{}, err
		}
	}
	return Result{204, nil}, nil
}

func (r *Repository) writeCollaborationSubscriber(ctx context.Context, q Request, body io.Reader, contentType string) (Result, error) {
	self := strings.Contains(q.Operation, "_subscribe") && !strings.Contains(q.Operation, "_subscribers")
	if q.Method == "DELETE" {
		row, err := r.collaborationObject(ctx, q, "subscriber", "issue_subscribers", false)
		if err != nil {
			return Result{}, err
		}
		return Result{204, nil}, r.deleteCollaboration(ctx, q, "issue_subscribers", row)
	}
	input := djangoinput.Input{Fields: map[string]any{}}
	changes := map[string]any{}
	if !self {
		var err error
		input, err = djangoinput.Parse(body, contentType)
		if err != nil {
			return Result{}, err
		}
		changes, err = r.validateCollaboration(ctx, q, input, "issue_subscribers")
		if err != nil {
			return Result{}, err
		}
	} else {
		if _, err := r.collaborationObject(ctx, q, "subscriber", "issue_subscribers", false); err == nil {
			return Result{}, failure(400, "message", "User already subscribed to the issue.")
		} else {
			var missing *djangoapi.HTTPError
			if !errors.As(err, &missing) || missing.Status != 404 {
				return Result{}, err
			}
		}
	}
	if err := r.collaborationReference(ctx, q, "issues", q.IssueID); err != nil {
		return Result{}, err
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	row := collaborationDefault("issue_subscribers", q, project["workspace_id"])
	maps.Copy(row, changes)
	row, err = r.saveCollaboration(ctx, q, "issue_subscribers", row, nil)
	if err != nil {
		return Result{}, err
	}
	projected, err := r.collaborationProjection(ctx, q, row, "subscriber", false)
	return Result{201, projected}, err
}
