package djangoapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

var apiAuditFields = strings.Fields("id created_at updated_at deleted_at created_by updated_by workspace project")
var apiIssueFields = strings.Fields("name description_html description_binary priority start_date target_date sequence_id sort_order completed_at archived_at is_draft external_source external_id point state parent estimate_point type type_id assignees labels")

func apiForeignKey(field string) bool {
	switch field {
	case "default_assignee", "project_lead", "created_by", "updated_by", "workspace", "project", "owned_by", "issue", "cycle", "parent", "estimate_point", "type", "state", "estimate", "lead", "module", "actor", "comment", "description", "subscriber":
		return true
	}
	return false
}

// Serialize projects the source BaseSerializer fields and resolves its API relations.
// Callers retain ownership of their model's declared fields and final overrides.
func Serialize(ctx context.Context, db Database, row map[string]any, declared []string, fields, expand string, issue bool) (map[string]any, error) {
	selected := map[string]bool{}
	for _, field := range strings.Split(fields, ",") {
		if field != "" {
			selected[field] = true
		}
	}
	expansions := map[string]bool{}
	for _, field := range strings.Split(expand, ",") {
		if field != "" {
			expansions[field] = true
		}
	}
	result := map[string]any{}
	for _, field := range declared {
		if len(selected) > 0 && !selected[field] {
			continue
		}
		source := field
		if apiForeignKey(field) {
			foreignKey := source + "_id"
			if _, present := row[foreignKey]; present {
				source = foreignKey
			}
		}
		value, present := row[source]
		if number, ok := value.(json.Number); ok && (field == "sort_order" || strings.HasSuffix(field, "_estimates")) {
			float, err := number.Float64()
			if err != nil {
				return nil, err
			}
			value = json.Number(pythoncompat.FloatString(float))
		}
		if present {
			result[field] = value
		}
		if expansions[field] {
			// API IssueSerializer replaces BaseSerializer's write-only expansion with
			// relation-specific serializers, using live relation-manager rows.
			if issue && (field == "assignees" || field == "labels") {
				expanded, err := apiIssueRelations(ctx, db, row["id"], field)
				if err != nil {
					return nil, err
				}
				result[field] = expanded
				continue
			}
			expanded, known, err := apiExpand(ctx, db, field, value)
			if err != nil {
				return nil, err
			}
			if known {
				result[field] = expanded
			} else {
				result[field] = row[field+"_id"]
			}
		}
	}
	return result, nil
}

func apiExpand(ctx context.Context, db Database, field string, id any) (any, bool, error) {
	var query, kind string
	switch field {
	case "default_assignee", "project_lead", "owned_by", "created_by", "updated_by", "actor":
		query = `SELECT jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'avatar',u.avatar,'display_name',u.display_name)||` + FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + ` FROM users u WHERE u.id=$1`
	case "workspace":
		query = `SELECT jsonb_build_object('id',id,'name',name,'slug',slug) FROM workspaces WHERE id=$1`
	case "project":
		query = `SELECT jsonb_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'description',p.description,'cover_image',p.cover_image,'emoji',p.emoji,'icon_prop',p.icon_prop,'archived_at',p.archived_at)||` + FileAssetURLFieldSQL("p.cover_image_asset_id", "p.cover_image", "cover_image_url") + ` FROM projects p WHERE p.id=$1`
	case "state":
		query = `SELECT jsonb_build_object('id',id,'name',name,'color',color,'group',"group") FROM states WHERE id=$1`
	case "parent":
		query = `SELECT jsonb_build_object('id',id,'sequence_id',sequence_id,'project_id',project_id) FROM issues WHERE id=$1`
	case "estimate_point":
		query = `SELECT to_jsonb(ep) FROM estimate_points ep WHERE id=$1`
		kind = "estimate"
	case "issue":
		query = `SELECT ` + IssueRowSQL("i") + ` FROM issues i WHERE i.id=$1`
		kind = "issue"
	default:
		return nil, false, nil
	}
	if id == nil {
		if kind == "estimate" {
			// EstimatePointSerializer(None).data uses DRF's initial writable
			// fields; unlike the read-only lite serializers it is not empty.
			return map[string]any{"deleted_at": nil, "key": nil, "description": "", "value": "", "created_by": nil, "updated_by": nil}, true, nil
		}
		return map[string]any{}, true, nil
	}
	value, err := JSON(ctx, db, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return map[string]any{}, true, nil
	}
	if err != nil {
		return nil, true, err
	}
	if kind != "" {
		value, err = serializeRelation(ctx, db, value.(map[string]any), kind, "", "")
	}
	return value, true, err
}

func apiIssueRelations(ctx context.Context, db Database, issue any, relation string) (any, error) {
	query := `SELECT COALESCE(jsonb_agg((jsonb_build_object('id',u.id,'first_name',u.first_name,'last_name',u.last_name,'email',u.email,'avatar',u.avatar,'display_name',u.display_name)||` + FileAssetURLFieldSQL("u.avatar_asset_id", "u.avatar", "avatar_url") + `) ORDER BY u.created_at DESC),'[]'::jsonb) FROM users u WHERE u.id IN (SELECT assignee_id FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL)`
	if relation == "labels" {
		query = `SELECT COALESCE(jsonb_agg(to_jsonb(l) ORDER BY l.created_at DESC),'[]'::jsonb) FROM labels l WHERE l.deleted_at IS NULL AND l.id IN (SELECT label_id FROM issue_labels WHERE issue_id=$1 AND deleted_at IS NULL)`
	}
	value, err := JSON(ctx, db, query, issue)
	if err != nil {
		return nil, err
	}
	if relation == "labels" {
		for i, row := range value.([]any) {
			value.([]any)[i], err = serializeRelation(ctx, db, row.(map[string]any), "label", "", "")
			if err != nil {
				return nil, err
			}
		}
	}
	return value, nil
}

func SerializeIssue(ctx context.Context, db Database, row map[string]any, fields, expand string) (map[string]any, error) {
	return serializeRelation(ctx, db, row, "issue", fields, expand)
}
func serializeRelation(ctx context.Context, db Database, row map[string]any, kind, fields, expand string) (map[string]any, error) {
	declared := append([]string{}, apiAuditFields...)
	switch kind {
	case "issue":
		declared = append(declared, apiIssueFields...)
	case "label":
		declared = append(declared, strings.Fields("name description color sort_order external_source external_id parent")...)
	case "estimate":
		declared = append(declared, "estimate", "key", "description", "value")
	}
	return Serialize(ctx, db, row, declared, fields, expand, kind == "issue")
}
