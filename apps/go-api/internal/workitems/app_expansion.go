package workitems

import (
	"context"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
)

// App DynamicBaseSerializer keeps its declared fields and appends only known
// relations. Its lite serializers intentionally differ from the public API.
func (r *Repository) expandAppIssue(ctx context.Context, row map[string]any, q Request, out map[string]any) (map[string]any, error) {
	for _, field := range strings.Split(q.Params["expand"], ",") {
		if field == "" {
			continue
		}
		value, known, err := r.appIssueExpansion(ctx, row, q, field)
		if err != nil {
			return nil, err
		}
		if known {
			out[field] = value
		} else if _, exists := out[field]; exists {
			out[field] = row[field+"_id"]
		}
	}
	return out, nil
}

func (r *Repository) appIssueExpansion(ctx context.Context, row map[string]any, q Request, field string) (any, bool, error) {
	query, id, many := "", row[field+"_id"], false
	switch field {
	case "user", "actor", "owned_by", "created_by", "default_assignee", "project_lead":
		query = `SELECT ` + appUserSQL + ` FROM users u WHERE u.id=$1`
	case "workspace":
		query = `SELECT jsonb_build_object('id',id,'name',name,'slug',slug) FROM workspaces WHERE id=$1`
	case "project":
		query = `SELECT ` + appProjectSQL + ` FROM projects p WHERE p.id=$1`
	case "state":
		query = `SELECT jsonb_build_object('id',id,'name',name,'color',color,'group',"group") FROM states WHERE id=$1`
	case "parent":
		query = `SELECT jsonb_build_object('id',id,'sequence_id',sequence_id,'project_id',project_id) FROM issues WHERE id=$1`
	case "assignees":
		query, id, many = `SELECT `+appUserSQL+` FROM users u WHERE u.id IN (SELECT assignee_id FROM issue_assignees WHERE issue_id=$1) ORDER BY u.created_at DESC`, row["id"], true
	case "labels":
		query, id, many = `SELECT jsonb_build_object('id',id,'parent',parent_id,'name',name,'color',color,'project_id',project_id,'workspace_id',workspace_id,'sort_order',sort_order) FROM labels WHERE id IN (SELECT label_id FROM issue_labels WHERE issue_id=$1) AND deleted_at IS NULL ORDER BY created_at DESC`, row["id"], true
	case "issue_link":
		query, id, many = `SELECT jsonb_build_object('id',id,'issue_id',issue_id,'title',title,'url',url,'metadata',metadata,'created_by_id',created_by_id,'created_at',created_at) FROM issue_links WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, row["id"], true
	case "issue_reactions":
		query, id, many = `SELECT jsonb_build_object('id',r.id,'actor',r.actor_id,'issue',r.issue_id,'reaction',r.reaction,'display_name',u.display_name) FROM issue_reactions r JOIN users u ON u.id=r.actor_id WHERE r.issue_id=$1 AND r.deleted_at IS NULL ORDER BY r.created_at DESC`, row["id"], true
	case "issue_intake":
		query, id, many = `SELECT jsonb_build_object('id',id,'status',status,'duplicate_to',duplicate_to_id,'snoozed_till',snoozed_till,'source',source) FROM intake_issues WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, row["id"], true
	case "sub_issues":
		query, id, many = `SELECT jsonb_build_object('id',id,'sequence_id',sequence_id,'project_id',project_id) FROM issues WHERE parent_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, row["id"], true
	case "issue_relation", "issue_related":
		anchor, related := "issue_id", "related_issue_id"
		if field == "issue_related" {
			anchor, related = related, anchor
		}
		query, id, many = `SELECT jsonb_build_object('id',i.id,'project_id',i.project_id,'sequence_id',i.sequence_id,'relation_type',rel.relation_type,'name',i.name,'state_id',i.state_id,'priority',i.priority,'created_by',rel.created_by_id,'created_at',rel.created_at,'updated_at',rel.updated_at,'updated_by',rel.updated_by_id) FROM issue_relations rel JOIN issues i ON i.id=rel.`+related+` WHERE rel.`+anchor+`=$1 AND rel.deleted_at IS NULL ORDER BY rel.created_at DESC`, row["id"], true
	case "issue_cycle":
		links, err := r.rows(ctx, `SELECT to_jsonb(c) FROM cycle_issues c WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, row["id"])
		if err != nil {
			return nil, true, err
		}
		out := []any{}
		for _, link := range links {
			value := planningProjection(link, "cycle")
			value["issue_detail"], err = r.appIssueState(ctx, row, q)
			if err != nil {
				return nil, true, err
			}
			out = append(out, value)
		}
		return out, true, nil
	case "issue_attachments":
		query, id, many = `SELECT jsonb_build_object('id',f.id,'asset',f.asset,'attributes',f.attributes,'created_by',f.created_by_id,'updated_at',f.updated_at,'updated_by',f.updated_by_id)||`+djangoapi.FileAssetURLFieldSQL("f.id", "NULL", "asset_url")+` FROM file_assets f WHERE f.issue_id=$1 AND f.entity_type='ISSUE_ATTACHMENT' AND f.deleted_at IS NULL ORDER BY f.created_at DESC`, row["id"], true
	default:
		return nil, false, nil
	}
	if many {
		rows, err := r.rows(ctx, query, id)
		if field == "labels" {
			for _, label := range rows {
				label["sort_order"] = projectFields(label, []string{"sort_order"})["sort_order"]
			}
		}
		return rows, true, err
	}
	if id == nil {
		return map[string]any{}, true, nil
	}
	value, err := djangoapi.JSON(ctx, r.db, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return map[string]any{}, true, nil
	}
	return value, true, err
}

var appUserSQL = `jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'avatar',u.avatar,'is_bot',u.is_bot,'display_name',u.display_name)||` + djangoapi.FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url")
var appProjectSQL = `jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'logo_props',p.logo_props,'description',p.description)||` + djangoapi.FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url")

func (r *Repository) appIssueState(ctx context.Context, row map[string]any, q Request) (map[string]any, error) {
	result, err := djangoapi.SerializeIssue(ctx, r.db, row, "", "")
	if err != nil {
		return nil, err
	}
	delete(result, "type_id")
	for _, relation := range []string{"assignees", "labels"} {
		result[relation], err = r.appIssueRelationIDs(ctx, row["id"], relation)
		if err != nil {
			return nil, err
		}
	}
	for _, field := range []string{"description_json", "description_stripped", "sub_issues_count", "attachment_count", "link_count"} {
		if value, ok := row[field]; ok {
			result[field] = value
		}
	}
	for field, name := range map[string]string{"assignees": "assignee_details", "labels": "label_details", "state": "state_detail", "project": "project_detail"} {
		value, _, err := r.appIssueExpansion(ctx, row, q, field)
		if err != nil {
			return nil, err
		}
		if field == "labels" {
			labels := []any{}
			for _, label := range value.([]map[string]any) {
				labels = append(labels, projectFields(label, []string{"id", "name", "color"}))
			}
			value = labels
		}
		result[name] = value
	}
	return result, nil
}

func (r *Repository) appIssueRelationIDs(ctx context.Context, issue any, field string) (any, error) {
	query := `SELECT COALESCE(jsonb_agg(u.id ORDER BY u.created_at DESC),'[]'::jsonb) FROM users u WHERE u.id IN (SELECT assignee_id FROM issue_assignees WHERE issue_id=$1)`
	if field == "labels" {
		query = `SELECT COALESCE(jsonb_agg(l.id ORDER BY l.created_at DESC),'[]'::jsonb) FROM labels l WHERE l.id IN (SELECT label_id FROM issue_labels WHERE issue_id=$1) AND l.deleted_at IS NULL`
	}
	return djangoapi.JSON(ctx, r.db, query, issue)
}
