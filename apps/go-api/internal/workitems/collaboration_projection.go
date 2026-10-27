package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

var (
	apiCommentFields        = strings.Fields("id created_at updated_at deleted_at comment_html attachments access external_source external_id edited_at created_by updated_by project workspace description issue actor parent")
	apiLinkFields           = strings.Fields("id created_at updated_at deleted_at title url metadata created_by updated_by project workspace issue")
	appCommentOrder         = strings.Fields("id actor_detail issue_detail project_detail workspace_detail comment_reactions created_at updated_at deleted_at comment_stripped comment_json comment_html attachments access external_source external_id edited_at created_by updated_by project workspace description issue actor parent")
	appLinkOrder            = strings.Fields("id created_by_detail created_at updated_at deleted_at title url metadata created_by updated_by project workspace issue")
	appReactionOrder        = strings.Fields("id actor_detail created_at updated_at deleted_at created_by updated_by project workspace actor issue reaction")
	appCommentReactionOrder = strings.Fields("id actor comment reaction display_name deleted_at workspace project created_at updated_at created_by updated_by")
	appSubscriberOrder      = strings.Fields("id created_at updated_at deleted_at created_by updated_by project workspace issue subscriber")
)

// collaborationProjection mirrors the API and app serializers for the shared
// collaboration models. App nested details are deliberately kept distinct
// from API expand serializers (notably UserLite's is_bot and no email field).
func (r *Repository) collaborationProjection(ctx context.Context, q Request, row map[string]any, kind string, annotated bool) (map[string]any, error) {
	if kind == "comment" && annotated && q.ExpectedWorkspaceID == nil {
		var member bool
		if q.ActorID != "" {
			if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL)`, row["project_id"], q.ActorID).Scan(&member); err != nil {
				return nil, err
			}
		}
		row = cloneCollaborationRow(row)
		row["is_member"] = member
	}
	if kind == "comment" && row["description_id"] != nil && q.ExpectedWorkspaceID == nil {
		owned, err := r.collaborationDescriptionOwned(ctx, q, row)
		if err != nil {
			return nil, err
		}
		if !owned {
			row = cloneCollaborationRow(row)
			row["description_id"] = nil
		}
	}
	if q.API {
		fields := apiCommentFields
		if kind == "link" {
			fields = apiLinkFields
		} else if kind == "comment" && annotated {
			fields = append(append([]string{}, apiCommentFields...), "is_member")
		}
		selected, expanded := "", ""
		if q.Method == "GET" || q.Method == "HEAD" {
			selected, expanded = q.Params["fields"], q.Params["expand"]
		}
		commentParent := false
		if kind == "comment" {
			other := []string{}
			for _, field := range strings.Split(expanded, ",") {
				if field == "parent" {
					commentParent = true
				} else {
					other = append(other, field)
				}
			}
			expanded = strings.Join(other, ",")
		}
		out, err := djangoapi.Serialize(ctx, r.db, row, fields, selected, expanded, false)
		if err != nil {
			return nil, err
		}
		if _, selected := out["parent"]; commentParent && selected {
			parent := map[string]any{}
			if row["parent_id"] != nil {
				parent, err = r.collaborationCommentParent(ctx, q, row)
				if err != nil {
					return nil, err
				}
				if parent == nil {
					parent = map[string]any{}
				}
			}
			out["parent"] = parent
		}
		collaborationTimestamps(out, q.Timezone)
		return out, nil
	}

	out := modelProjection(row, collaborationFieldOrder(kind))
	if kind == "comment" {
		parent, err := r.collaborationCommentParent(ctx, q, row)
		if err != nil {
			return nil, err
		}
		if row["parent_id"] != nil {
			if parent == nil {
				out["parent"] = nil
			} else {
				out["parent"] = parent["id"]
			}
		}
	}
	var err error
	switch kind {
	case "comment":
		out["actor_detail"], err = r.collaborationUser(ctx, row["actor_id"])
		if err != nil {
			return nil, err
		}
		out["issue_detail"], err = r.collaborationIssueDetail(ctx, row["issue_id"])
		if err != nil {
			return nil, err
		}
		out["project_detail"], err = r.collaborationProjectDetail(ctx, row["project_id"])
		if err != nil {
			return nil, err
		}
		out["workspace_detail"], err = r.collaborationWorkspaceDetail(ctx, row["workspace_id"])
		if err != nil {
			return nil, err
		}
		out["comment_reactions"], err = r.collaborationCommentReactions(ctx, row["id"], q.Timezone)
		if err != nil {
			return nil, err
		}
		if annotated {
			out["is_member"] = row["is_member"]
		}
	case "link":
		out["created_by_detail"], err = r.collaborationUser(ctx, row["created_by_id"])
		if err != nil {
			return nil, err
		}
	case "issueReaction":
		out["actor_detail"], err = r.collaborationUser(ctx, row["actor_id"])
		if err != nil {
			return nil, err
		}
	case "commentReaction":
		name, e := r.collaborationDisplayName(ctx, row["actor_id"])
		if e != nil {
			return nil, e
		}
		out["display_name"] = name
	case "subscriber":
	default:
		return nil, errors.New("unknown collaboration projection kind: " + kind)
	}
	collaborationTimestamps(out, q.Timezone)
	return out, nil
}

func (r *Repository) collaborationDescriptionOwned(ctx context.Context, q Request, row map[string]any) (bool, error) {
	if row["description_id"] == nil {
		return true, nil
	}
	var owned bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM descriptions d JOIN workspaces w ON w.id=d.workspace_id WHERE d.id=$1 AND d.project_id=$2 AND d.workspace_id=$3 AND w.slug=$4 AND d.deleted_at IS NULL)`, row["description_id"], row["project_id"], row["workspace_id"], q.Slug).Scan(&owned)
	return owned, err
}

func (r *Repository) collaborationCommentParent(ctx context.Context, q Request, row map[string]any) (map[string]any, error) {
	if row["parent_id"] == nil {
		return nil, nil
	}
	query := `SELECT jsonb_build_object('id',parent.id,'project_id',parent.project_id)
FROM issue_comments parent
JOIN issues parent_issue ON parent_issue.id=parent.issue_id AND parent_issue.project_id=parent.project_id AND parent_issue.workspace_id=parent.workspace_id
JOIN workspaces parent_workspace ON parent_workspace.id=parent.workspace_id
WHERE parent.id=$1 AND parent.issue_id=$2 AND parent.project_id=$3 AND parent_workspace.slug=$4 AND parent.deleted_at IS NULL AND `
	args := []any{row["parent_id"], q.IssueID, q.ProjectID, q.Slug}
	if collaborationPublic(q) {
		query += `parent.access='EXTERNAL' AND ` + djangoapi.PublicRelatedIssueSQL("parent_issue", "parent_issue")
	} else {
		query += djangoapi.IssueVisibleSQL("parent_issue", "NULLIF($5,'')::uuid", "parent_workspace.id")
		args = append(args, q.ActorID)
	}
	return r.collaborationOptionalRow(ctx, query, args...)
}

func cloneCollaborationRow(row map[string]any) map[string]any {
	copy := make(map[string]any, len(row)+1)
	for key, value := range row {
		copy[key] = value
	}
	return copy
}

func collaborationFieldOrder(kind string) []string {
	switch kind {
	case "comment":
		return appCommentOrder
	case "link":
		return appLinkOrder
	case "issueReaction":
		return appReactionOrder
	case "commentReaction":
		return appCommentReactionOrder
	case "subscriber":
		return appSubscriberOrder
	default:
		return nil
	}
}

func modelProjection(row map[string]any, order []string) map[string]any {
	out := make(map[string]any, len(order))
	for _, field := range order {
		key := field
		if field == "created_by" || field == "updated_by" || field == "project" || field == "workspace" || field == "issue" || field == "actor" || field == "parent" || field == "description" || field == "subscriber" || field == "comment" {
			key += "_id"
		}
		if value, ok := row[key]; ok {
			out[field] = value
		}
	}
	return out
}

func (r *Repository) collaborationUser(ctx context.Context, id any) (any, error) {
	if id == nil {
		return nil, nil
	}
	value, err := djangoapi.JSON(ctx, r.db, `SELECT `+appUserSQL+` FROM users u WHERE u.id=$1`, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}

func (r *Repository) collaborationDisplayName(ctx context.Context, id any) (any, error) {
	if id == nil {
		return nil, nil
	}
	var value *string
	err := r.db.QueryRow(ctx, `SELECT display_name FROM users WHERE id=$1`, id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	return *value, nil
}

func (r *Repository) collaborationIssueDetail(ctx context.Context, id any) (map[string]any, error) {
	if id == nil {
		return map[string]any{}, nil
	}
	row, err := r.collaborationOptionalRow(ctx, `SELECT jsonb_build_object('id',i.id,'name',i.name,'description_json',i.description_json,'description_html',i.description_html,'priority',i.priority,'start_date',i.start_date,'target_date',i.target_date,'sequence_id',i.sequence_id,'sort_order',i.sort_order,'is_draft',i.is_draft) FROM issues i WHERE i.id=$1`, id)
	if err != nil || row == nil {
		return map[string]any{}, err
	}
	if number, ok := row["sort_order"].(json.Number); ok {
		f, _ := number.Float64()
		row["sort_order"] = json.Number(pythoncompat.FloatString(f))
	}
	return row, nil
}

func (r *Repository) collaborationProjectDetail(ctx context.Context, id any) (map[string]any, error) {
	if id == nil {
		return map[string]any{}, nil
	}
	row, err := r.collaborationOptionalRow(ctx, `SELECT `+appProjectSQL+` FROM projects p WHERE p.id=$1`, id)
	if err != nil || row == nil {
		return map[string]any{}, err
	}
	return row, nil
}

func (r *Repository) collaborationWorkspaceDetail(ctx context.Context, id any) (map[string]any, error) {
	if id == nil {
		return map[string]any{}, nil
	}
	row, err := r.collaborationOptionalRow(ctx, `SELECT jsonb_build_object('name',w.name,'slug',w.slug,'id',w.id)||jsonb_build_object('logo_url',`+djangoapi.FileAssetURLSQL("w.logo_asset_id", "w.logo")+`) FROM workspaces w WHERE w.id=$1`, id)
	if err != nil || row == nil {
		return map[string]any{}, err
	}
	return row, nil
}

func (r *Repository) collaborationOptionalRow(ctx context.Context, query string, args ...any) (map[string]any, error) {
	value, err := djangoapi.JSON(ctx, r.db, query, args...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}

func (r *Repository) collaborationCommentReactions(ctx context.Context, comment any, zone string) ([]any, error) {
	rows, err := r.rows(ctx, `SELECT to_jsonb(c)||jsonb_build_object('display_name',u.display_name) FROM comment_reactions c JOIN users u ON u.id=c.actor_id WHERE c.comment_id=$1 AND c.deleted_at IS NULL ORDER BY c.created_at DESC`, comment)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(rows))
	for _, row := range rows {
		value := modelProjection(row, appCommentReactionOrder)
		collaborationTimestamps(value, zone)
		out = append(out, value)
	}
	return out, nil
}

func collaborationTimestamps(value any, zone string) {
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for _, key := range []string{"created_at", "updated_at", "deleted_at", "completed_at", "snoozed_till", "edited_at"} {
				if raw, ok := typed[key].(string); ok {
					location, _ := time.LoadLocation(zone)
					if location == nil {
						location = time.UTC
					}
					if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
						format := time.RFC3339
						if parsed.Nanosecond() != 0 {
							format = "2006-01-02T15:04:05.000000Z07:00"
						}
						typed[key] = parsed.In(location).Format(format)
					}
				}
			}
			for key, child := range typed {
				switch key {
				case "description_json", "description_html", "comment_json", "comment_html", "icon_prop", "logo_props", "meta", "metadata", "source_data", "view_props", "progress_snapshot":
					continue
				}
				visit(child)
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		case []map[string]any:
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
}

func (r *Repository) collaborationDump(ctx context.Context, q Request, row map[string]any, kind string, annotated bool) (string, error) {
	projected, err := r.collaborationProjection(ctx, q, row, kind, annotated)
	if err != nil {
		return "", err
	}
	keys := collaborationFieldOrder(kind)
	if q.API {
		if kind == "link" {
			keys = apiLinkFields
		} else {
			keys = apiCommentFields
		}
	} else if kind == "issueReaction" {
		keys = appReactionOrder
	}
	children := map[string][]byte{}
	if !q.API && kind == "comment" {
		children["actor_detail"] = orderedJSON(projected["actor_detail"], strings.Fields("id first_name last_name avatar avatar_url is_bot display_name"))
		children["issue_detail"] = orderedJSON(projected["issue_detail"], strings.Fields("id name description_json description_html priority start_date target_date sequence_id sort_order is_draft"))
		children["project_detail"] = orderedJSON(projected["project_detail"], strings.Fields("id identifier name cover_image cover_image_url logo_props description"))
		children["workspace_detail"] = orderedJSON(projected["workspace_detail"], strings.Fields("name slug id logo_url"))
		children["comment_reactions"] = orderedArrayJSON(projected["comment_reactions"], appCommentReactionOrder)
	} else if !q.API && kind == "link" {
		children["created_by_detail"] = orderedJSON(projected["created_by_detail"], strings.Fields("id first_name last_name avatar avatar_url is_bot display_name"))
	} else if !q.API && kind == "issueReaction" {
		children["actor_detail"] = orderedJSON(projected["actor_detail"], strings.Fields("id first_name last_name avatar avatar_url is_bot display_name"))
	}
	return pythoncompat.Dumps(projected, jsonOrder(keys, children))
}

func orderedJSON(value any, keys []string) []byte {
	if value == nil {
		return []byte("null")
	}
	if object, ok := value.(map[string]any); ok {
		if object == nil {
			return []byte("null")
		}
		return jsonOrder(keys, nilWithNulls(object, keys))
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

func nilWithNulls(value map[string]any, keys []string) map[string][]byte {
	children := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if _, ok := value[key]; ok {
			children[key] = []byte("null")
		}
	}
	return children
}

func orderedArrayJSON(value any, keys []string) []byte {
	rows, ok := value.([]any)
	if !ok {
		encoded, _ := json.Marshal(value)
		return encoded
	}
	var output strings.Builder
	output.WriteByte('[')
	for i, item := range rows {
		if i > 0 {
			output.WriteByte(',')
		}
		encoded := orderedJSON(item, keys)
		output.Write(encoded)
	}
	output.WriteByte(']')
	return []byte(output.String())
}
