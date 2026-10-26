package workitems

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

var compactFields = strings.Fields("id name state_id sort_order completed_at estimate_point priority start_date target_date sequence_id project_id parent_id cycle_id module_ids label_ids assignee_ids sub_issues_count created_at updated_at created_by updated_by attachment_count link_count is_draft archived_at type_id custom_properties")
var detailFields = append(append([]string{}, compactFields...), "description_html", "is_subscribed", "is_intake")
var apiIssueOrder = strings.Fields("id type_id created_at updated_at deleted_at point name description_html description_binary priority start_date target_date sequence_id sort_order completed_at archived_at is_draft external_source external_id created_by updated_by project workspace parent state estimate_point type assignees labels custom_properties")

// Only authenticated work-item handlers use this projection. The shared public
// serializer retains its explicit field allowlist, including expanded relations.
func (r *Repository) serializeAuthorizedIssue(ctx context.Context, row map[string]any, fields, expand string) (map[string]any, error) {
	result, err := djangoapi.SerializeIssue(ctx, r.db, row, fields, expand)
	if err != nil {
		return nil, err
	}
	if fields == "" || slices.Contains(strings.Split(fields, ","), "custom_properties") {
		result["custom_properties"] = row["custom_properties"]
	}
	return result, nil
}

func (r *Repository) serializeVisibleIssue(ctx context.Context, row map[string]any, q Request, fields, expand string) (map[string]any, error) {
	row, err := djangoapi.MaskIssueParent(ctx, r.db, row, q.ActorID)
	if err != nil {
		return nil, err
	}
	return r.serializeAuthorizedIssue(ctx, row, fields, expand)
}

func projectFields(row map[string]any, fields []string) map[string]any {
	out := map[string]any{}
	for _, key := range fields {
		source := key
		switch key {
		case "created_by", "updated_by", "estimate_point":
			source += "_id"
		}
		if value, ok := row[source]; ok {
			if key == "sort_order" {
				if number, ok := value.(json.Number); ok {
					f, _ := number.Float64()
					value = json.Number(pythoncompat.FloatString(f))
				}
			}
			out[key] = value
		}
	}
	return out
}

func (r *Repository) annotateIssue(ctx context.Context, row map[string]any, actor string, identifier bool) (map[string]any, error) {
	value, err := r.row(ctx, `SELECT `+issueAnnotationsSQL("$2", true)+` FROM issues i WHERE i.id=$1`, row["id"], actor)
	if err != nil {
		return nil, err
	}
	for k, v := range value {
		row[k] = v
	}
	if !identifier {
		for _, k := range []string{"sub_issues_count", "attachment_count", "link_count"} {
			if row[k] == json.Number("0") {
				row[k] = nil
			}
		}
		delete(row, "is_intake")
	}
	return row, nil
}

func (r *Repository) detail(ctx context.Context, row map[string]any, q Request, subscribed, identifier bool) (map[string]any, error) {
	row, err := r.annotateIssue(ctx, row, q.ActorID, identifier)
	if err != nil {
		return nil, err
	}
	if !subscribed {
		delete(row, "is_subscribed")
	}
	result := projectFields(row, detailFields)
	return r.expandAppIssue(ctx, row, q, result)
}

func (r *Repository) guestIssue(ctx context.Context, q Request, row, project map[string]any) error {
	var guest bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members m JOIN workspaces w ON w.id=m.workspace_id WHERE m.project_id=$1 AND w.slug=$2 AND m.member_id=$3 AND m.is_active AND m.deleted_at IS NULL AND m.role=5)`, q.ProjectID, q.Slug, q.ActorID).Scan(&guest); err != nil {
		return err
	}
	if guest && project["guest_view_all_features"] != true && row["created_by_id"] != q.ActorID {
		return failure(403, "error", "You are not allowed to view this issue")
	}
	return nil
}

func (r *Repository) authorizeGuestItemRead(ctx context.Context, q Request, row map[string]any) error {
	q.ProjectID = stringValue(row["project_id"])
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return err
	}
	return r.guestIssue(ctx, q, row, project)
}

func (r *Repository) recentVisit(ctx context.Context, q Request, entity, id string) error {
	if r.recentVisits != nil {
		return r.recentVisits.AdmitRecentVisit(ctx, uuid.NewString(), q.Slug, entity, id, q.ActorID, q.ProjectID)
	}
	return r.publisher.Task(ctx, "plane.bgtasks.recent_visited_task.recent_visited_task", []any{}, map[string]any{"slug": q.Slug, "entity_name": entity, "entity_identifier": id, "user_id": q.ActorID, "project_id": q.ProjectID})
}

func (r *Repository) readIssue(ctx context.Context, q Request) (Result, error) {
	if !q.API {
		if err := r.allowProject(ctx, q, []int{5, 15, 20}, true); err != nil {
			return Result{}, err
		}
	}
	row, err := r.issue(ctx, q, q.API)
	if err != nil {
		return Result{}, err
	}
	if q.API {
		if err = r.authorizeGuestItemRead(ctx, q, row); err != nil {
			return Result{}, err
		}
		body, err := r.serializeVisibleIssue(ctx, row, q, q.Params["fields"], q.Params["expand"])
		return Result{200, body}, err
	}
	project, err := r.project(ctx, q.ProjectID)
	if err != nil {
		return Result{}, err
	}
	if err = r.guestIssue(ctx, q, row, project); err != nil {
		return Result{}, err
	}
	if err = r.recentVisit(ctx, q, "issue", q.IssueID); err != nil {
		return Result{}, err
	}
	body, err := r.detail(ctx, row, q, true, false)
	return Result{200, body}, err
}

func orderedDump(value map[string]any, keys []string) (string, error) {
	return pythoncompat.Dumps(value, jsonOrder(keys, nil))
}

func jsonOrder(keys []string, children map[string][]byte) []byte {
	var order strings.Builder
	order.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			order.WriteByte(',')
		}
		encoded, _ := json.Marshal(k)
		order.Write(encoded)
		order.WriteByte(':')
		if child := children[k]; len(child) > 0 {
			order.Write(child)
		} else {
			order.WriteString("null")
		}
	}
	order.WriteByte('}')
	return []byte(order.String())
}

func issueAnnotationsSQL(actor string, activeAssignees bool) string {
	return issueAnnotationsSQLWithModuleConditions(actor, activeAssignees, " AND m.deleted_at IS NULL AND md.deleted_at IS NULL AND md.archived_at IS NULL")
}

// issueCollectionAnnotationsSQL keeps archived modules in public issue
// annotations while excluding deleted relation rows and modules.
func issueCollectionAnnotationsSQL(actor string) string {
	return publicIssueAnnotationsSQL(actor, " AND m.deleted_at IS NULL AND md.deleted_at IS NULL")
}

func issueAnnotationsSQLWithModuleConditions(actor string, activeAssignees bool, moduleConditions string) string {
	return issueAnnotationsSQLWithVisibility(actor, activeAssignees, moduleConditions, djangoapi.VisibleParentSQL("i", actor), djangoapi.IssueVisibleSQL("child", actor, "i.workspace_id"))
}

func publicIssueAnnotationsSQL(actor, moduleConditions string) string {
	return issueAnnotationsSQLWithVisibility(actor, false, moduleConditions, djangoapi.PublicParentSQL("i"), djangoapi.PublicRelatedIssueSQL("child", "i"))
}

func issueAnnotationsSQLWithVisibility(actor string, activeAssignees bool, moduleConditions, parentSQL, childSQL string) string {
	assigneeFilter := ""
	if activeAssignees {
		assigneeFilter = " AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.member_id=a.assignee_id AND pm.is_active)"
	}
	return strings.NewReplacer("$2", actor, "$assigneeFilter", assigneeFilter, "$moduleConditions", moduleConditions, "$visibleParent", parentSQL, "$visibleChild", childSQL).Replace(`jsonb_build_object(
 'parent_id',$visibleParent,
 'cycle_id',(SELECT cycle_issue.cycle_id FROM cycle_issues cycle_issue JOIN cycles cycle ON cycle.id=cycle_issue.cycle_id AND cycle.deleted_at IS NULL AND cycle.project_id=i.project_id AND cycle.workspace_id=i.workspace_id WHERE cycle_issue.issue_id=i.id AND cycle_issue.deleted_at IS NULL ORDER BY cycle_issue.created_at DESC LIMIT 1),
 'module_ids',COALESCE((SELECT jsonb_agg(DISTINCT m.module_id) FROM module_issues m JOIN modules md ON md.id=m.module_id WHERE m.issue_id=i.id$moduleConditions),'[]'::jsonb),
 'label_ids',COALESCE((SELECT jsonb_agg(DISTINCT issue_label.label_id) FROM issue_labels issue_label WHERE ` + issuequery.LiveIssueLabelPredicate("i", "issue_label") + `),'[]'::jsonb),
 'assignee_ids',COALESCE((SELECT jsonb_agg(DISTINCT a.assignee_id) FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL$assigneeFilter),'[]'::jsonb),
 'sub_issues_count',(SELECT COUNT(*) FROM issues child WHERE child.parent_id=i.id AND $visibleChild AND child.deleted_at IS NULL AND child.archived_at IS NULL AND NOT child.is_draft AND NOT EXISTS(SELECT 1 FROM states s WHERE s.id=child.state_id AND s."group"='triage') AND NOT EXISTS(SELECT 1 FROM projects p WHERE p.id=child.project_id AND p.archived_at IS NOT NULL)),
 'attachment_count',(SELECT COUNT(*) FROM file_assets a WHERE a.issue_id=i.id AND a.entity_type='ISSUE_ATTACHMENT' AND a.deleted_at IS NULL),
 'link_count',(SELECT COUNT(*) FROM issue_links l WHERE l.issue_id=i.id AND l.deleted_at IS NULL),
 'is_subscribed',EXISTS(SELECT 1 FROM issue_subscribers s WHERE s.issue_id=i.id AND s.project_id=i.project_id AND s.workspace_id=i.workspace_id AND s.subscriber_id=$2 AND s.deleted_at IS NULL),
 'is_intake',EXISTS(SELECT 1 FROM intake_issues t WHERE t.issue_id=i.id AND t.status IN (-2,0) AND t.project_id=i.project_id AND t.workspace_id=i.workspace_id AND t.deleted_at IS NULL)
 )`)
}
