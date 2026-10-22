package backgroundtasks

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
)

const (
	IssueExportWorkflowName = "plane.exports.issue_export"
	issueExportActivityName = "plane.exports.generate_issue_export"
	issueExportFailureName  = "plane.exports.mark_issue_export_failed"
	issueExportStartTimeout = 10 * time.Minute
)

var issueExportFields = []string{
	"project_name", "project_identifier", "parent", "identifier", "sequence_id", "name", "state_name", "priority",
	"assignees", "subscribers", "created_by_name", "start_date", "target_date", "completed_at", "created_at",
	"updated_at", "archived_at", "estimate", "labels", "cycles", "modules", "links", "relations", "comments",
	"sub_issues_count", "link_count", "attachment_count", "is_draft",
}

type IssueExportInput struct {
	Provider    string   `json:"provider"`
	WorkspaceID string   `json:"workspace_id"`
	ProjectIDs  []string `json:"project_ids"`
	TokenID     string   `json:"token_id"`
	Multiple    bool     `json:"multiple"`
	Slug        string   `json:"slug"`
}

type issueExportFailureInput struct {
	TokenID string `json:"token_id"`
	Reason  string `json:"reason"`
}

type issueExportObjectStore interface {
	Put(context.Context, string, string, []byte) error
	PresignGet(context.Context, string, string, *string, string, string) (string, error)
}

type issueExportStore struct {
	pool    *pgxpool.Pool
	objects issueExportObjectStore
}

type IssueExportDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[IssueExportInput, struct{}]
}

type IssueExportTaskAdmitter interface {
	AdmitIssueExport(context.Context, IssueExportInput) error
}

type IssueExportTaskPublisher struct {
	fallback  TaskFallback
	admission IssueExportTaskAdmitter
}

func NewIssueExportTaskPublisher(fallback TaskFallback, admission IssueExportTaskAdmitter) (*IssueExportTaskPublisher, error) {
	if fallback == nil || admission == nil {
		return nil, errors.New("task fallback and issue export admission are required")
	}
	return &IssueExportTaskPublisher{fallback: fallback, admission: admission}, nil
}

func (publisher *IssueExportTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.admission == nil {
		return errors.New("issue export task publisher is not configured")
	}
	if name != "plane.bgtasks.export_task.issue_export_task" {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	if len(args) != 0 {
		return errors.New("issue export task requires keyword arguments")
	}
	encoded, err := json.Marshal(kwargs)
	if err != nil {
		return fmt.Errorf("encode issue export task arguments: %w", err)
	}
	var input IssueExportInput
	if err := json.Unmarshal(encoded, &input); err != nil {
		return fmt.Errorf("decode issue export task arguments: %w", err)
	}
	return publisher.admission.AdmitIssueExport(ctx, input)
}

func (publisher *IssueExportTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("issue export task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func (dispatcher *IssueExportDispatcher) AdmitIssueExport(ctx context.Context, input IssueExportInput) error {
	_, err := dispatcher.Admit(ctx, input)
	return err
}

func NewPostgresIssueExportStore(pool *pgxpool.Pool, objects issueExportObjectStore) (*issueExportStore, error) {
	if pool == nil || objects == nil {
		return nil, errors.New("issue export PostgreSQL pool and object storage are required")
	}
	return &issueExportStore{pool: pool, objects: objects}, nil
}

func RegisterIssueExportWorkflow(app *selvedge.App, store *issueExportStore, policy *retry.Policy) (*IssueExportDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and issue export store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	generate := execution.Activity[IssueExportInput, struct{}]{Name: issueExportActivityName}
	if err := executionSelvedge.Register(activities, generate, func(ctx context.Context, input IssueExportInput) (struct{}, error) {
		if err := store.Generate(ctx, input); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: issueExportStartTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register issue export activity: %w", err)
	}
	markFailed := execution.Activity[issueExportFailureInput, struct{}]{Name: issueExportFailureName}
	if err := executionSelvedge.Register(activities, markFailed, func(ctx context.Context, input issueExportFailureInput) (struct{}, error) {
		if err := store.MarkFailed(ctx, input.TokenID, input.Reason); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 20 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register failed issue export activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(IssueExportWorkflowName, func(ctx *selvedge.WorkflowContext, input IssueExportInput) (struct{}, error) {
		if _, err := uuid.Parse(input.TokenID); err != nil {
			return struct{}{}, fmt.Errorf("invalid exporter token: %w", err)
		}
		if _, err := uuid.Parse(input.WorkspaceID); err != nil {
			return struct{}{}, fmt.Errorf("invalid exporter workspace: %w", err)
		}
		_, err := generate.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err == nil {
			return struct{}{}, nil
		}
		_, statusErr := markFailed.Execute(ctx.Context(), activities.Scope(ctx), issueExportFailureInput{TokenID: input.TokenID, Reason: err.Error()}, execution.ActivityOptions{})
		if statusErr != nil {
			return struct{}{}, fmt.Errorf("export failed (%v) and failure status could not be recorded: %w", err, statusErr)
		}
		return struct{}{}, err
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &IssueExportDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *IssueExportDispatcher) Admit(ctx context.Context, input IssueExportInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable issue export dispatcher is not configured")
	}
	if _, err := uuid.Parse(input.TokenID); err != nil {
		return "", fmt.Errorf("issue export token must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.WorkspaceID); err != nil {
		return "", fmt.Errorf("issue export workspace must be a UUID: %w", err)
	}
	if input.Provider != "csv" && input.Provider != "json" && input.Provider != "xlsx" {
		return "", fmt.Errorf("unsupported issue export provider %q", input.Provider)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "plane-issue-export:"+input.TokenID, input)
}

func (store *issueExportStore) Generate(ctx context.Context, input IssueExportInput) error {
	result, err := store.pool.Exec(ctx, `UPDATE exporters SET status='processing',updated_at=NOW() WHERE token=$1 AND workspace_id=$2::uuid AND type='issue_exports'`, input.TokenID, input.WorkspaceID)
	if err != nil {
		return fmt.Errorf("mark exporter processing: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("exporter history row is missing")
	}
	if input.Provider != "csv" && input.Provider != "json" && input.Provider != "xlsx" {
		return fmt.Errorf("unsupported exporter format %q", input.Provider)
	}
	projectIDs := make([]uuid.UUID, 0, len(input.ProjectIDs))
	for _, raw := range input.ProjectIDs {
		projectID, err := uuid.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid export project id: %w", err)
		}
		projectIDs = append(projectIDs, projectID)
	}
	groups := make([]struct {
		name string
		rows []map[string]any
	}, 0, 1)
	if input.Multiple {
		for _, projectID := range projectIDs {
			rows, err := store.issues(ctx, input, []uuid.UUID{projectID})
			if err != nil {
				return err
			}
			groups = append(groups, struct {
				name string
				rows []map[string]any
			}{name: input.Slug + "-" + projectID.String(), rows: rows})
		}
	} else {
		rows, err := store.issues(ctx, input, projectIDs)
		if err != nil {
			return err
		}
		groups = append(groups, struct {
			name string
			rows []map[string]any
		}{name: input.Slug + "-" + input.WorkspaceID, rows: rows})
	}
	files := make([]struct {
		name string
		data []byte
	}, 0, len(groups))
	for _, group := range groups {
		name, data, err := formatIssueExport(group.name, input.Provider, group.rows)
		if err != nil {
			return err
		}
		files = append(files, struct {
			name string
			data []byte
		}{name: name, data: data})
	}
	archive, err := zipIssueExport(files)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s/export-%s-%s-%s.zip", input.WorkspaceID, input.Slug, input.TokenID[:6], time.Now().UTC().Format("2006-01-02"))
	if err := store.objects.Put(ctx, key, "application/zip", archive); err != nil {
		return fmt.Errorf("upload issue export: %w", err)
	}
	filename := fmt.Sprintf("export-%s-%s-%s.zip", input.Slug, input.TokenID[:6], time.Now().UTC().Format("2006-01-02"))
	url, err := store.objects.PresignGet(ctx, key, "attachment", &filename, "", "")
	if err != nil {
		return fmt.Errorf("presign issue export download: %w", err)
	}
	result, err = store.pool.Exec(ctx, `UPDATE exporters SET status='completed',reason='',key=$2,url=$3,updated_at=NOW() WHERE token=$1 AND workspace_id=$4::uuid AND type='issue_exports'`, input.TokenID, key, url, input.WorkspaceID)
	if err != nil {
		return fmt.Errorf("mark issue export complete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("exporter history row disappeared before completion")
	}
	return nil
}

func (store *issueExportStore) MarkFailed(ctx context.Context, token, reason string) error {
	result, err := store.pool.Exec(ctx, `UPDATE exporters SET status='failed',reason=$2,updated_at=NOW() WHERE token=$1 AND type='issue_exports'`, token, reason)
	if err != nil {
		return fmt.Errorf("mark issue export failed: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("exporter history row is missing")
	}
	return nil
}

func (store *issueExportStore) issues(ctx context.Context, input IssueExportInput, projectIDs []uuid.UUID) ([]map[string]any, error) {
	rows, err := store.pool.Query(ctx, issueExportQuery, input.WorkspaceID, projectIDs, input.TokenID)
	if err != nil {
		return nil, fmt.Errorf("query issue export rows: %w", err)
	}
	defer rows.Close()
	result := make([]map[string]any, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("read issue export row: %w", err)
		}
		var record map[string]any
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, fmt.Errorf("decode issue export row: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate issue export rows: %w", err)
	}
	return result, nil
}

const issueExportQuery = `SELECT json_build_object(
 'project_name',p.name,'project_identifier',p.identifier,
 'parent',COALESCE(parent_project.identifier||'-'||parent.sequence_id::text,''),
 'identifier',p.identifier||'-'||i.sequence_id::text,'sequence_id',i.sequence_id,'name',i.name,
 'state_name',COALESCE(state.name,''),'priority',i.priority,
 'assignees',COALESCE((SELECT json_agg(COALESCE(NULLIF(u.display_name,''),trim(concat_ws(' ',u.first_name,u.last_name))) ORDER BY u.id) FROM issue_assignees a JOIN users u ON u.id=a.assignee_id WHERE a.issue_id=i.id AND a.deleted_at IS NULL AND u.is_active=true),'[]'::json),
 'subscribers',COALESCE((SELECT json_agg(COALESCE(NULLIF(u.display_name,''),trim(concat_ws(' ',u.first_name,u.last_name))) ORDER BY u.id) FROM issue_subscribers s JOIN users u ON u.id=s.subscriber_id WHERE s.issue_id=i.id AND s.deleted_at IS NULL),'[]'::json),
 'created_by_name',COALESCE(NULLIF(creator.display_name,''),trim(concat_ws(' ',creator.first_name,creator.last_name))),
 'start_date',i.start_date,'target_date',i.target_date,'completed_at',i.completed_at,'created_at',i.created_at,'updated_at',i.updated_at,'archived_at',i.archived_at,
 'estimate',COALESCE(estimate_point.value::text,''),
 'labels',COALESCE((SELECT json_agg(label.name ORDER BY label.name) FROM issue_labels il JOIN labels label ON label.id=il.label_id WHERE il.issue_id=i.id AND il.deleted_at IS NULL),'[]'::json),
 'cycles',COALESCE((SELECT json_agg(cycle.name ORDER BY cycle.name) FROM cycle_issues ci JOIN cycles cycle ON cycle.id=ci.cycle_id WHERE ci.issue_id=i.id),'[]'::json),
 'modules',COALESCE((SELECT json_agg(module.name ORDER BY module.name) FROM module_issues mi JOIN modules module ON module.id=mi.module_id WHERE mi.issue_id=i.id),'[]'::json),
 'links',COALESCE((SELECT json_agg(json_build_object('url',link.url,'title',COALESCE(NULLIF(link.title,''),link.url)) ORDER BY link.created_at) FROM issue_links link WHERE link.issue_id=i.id),'[]'::json),
 'relations',COALESCE((SELECT json_agg(relation_value ORDER BY relation_value->>'direction',relation_value->>'issue') FROM (
   SELECT json_build_object('type',rel.relation_type,'issue',related_project.identifier||'-'||related.sequence_id::text,'direction','outgoing') relation_value FROM issue_relations rel JOIN issues related ON related.id=rel.related_issue_id JOIN projects related_project ON related_project.id=related.project_id WHERE rel.issue_id=i.id
   UNION ALL SELECT json_build_object('type',rel.relation_type,'issue',source_project.identifier||'-'||source.sequence_id::text,'direction','incoming') FROM issue_relations rel JOIN issues source ON source.id=rel.issue_id JOIN projects source_project ON source_project.id=source.project_id WHERE rel.related_issue_id=i.id
 ) all_relations),'[]'::json),
 'comments',COALESCE((SELECT json_agg(json_build_object('comment',comment.comment_stripped,'created_by',COALESCE(NULLIF(actor.display_name,''),trim(concat_ws(' ',actor.first_name,actor.last_name))),'created_at',to_char(comment.created_at,'YYYY-MM-DD HH24:MI:SS')) ORDER BY comment.created_at) FROM issue_comments comment LEFT JOIN users actor ON actor.id=comment.actor_id WHERE comment.issue_id=i.id),'[]'::json),
 'sub_issues_count',(SELECT count(*) FROM issues child WHERE child.parent_id=i.id AND child.deleted_at IS NULL AND child.archived_at IS NULL AND child.is_draft=false),
 'link_count',(SELECT count(*) FROM issue_links link WHERE link.issue_id=i.id),
 'attachment_count',(SELECT count(*) FROM file_assets asset WHERE asset.issue_id=i.id AND asset.entity_type='ISSUE_ATTACHMENT'),
 'is_draft',i.is_draft
) FROM issues i JOIN projects p ON p.id=i.project_id JOIN users initiator ON initiator.id=(SELECT initiated_by_id FROM exporters WHERE token=$3 AND type='issue_exports') LEFT JOIN states state ON state.id=i.state_id LEFT JOIN users creator ON creator.id=i.created_by_id LEFT JOIN estimate_points estimate_point ON estimate_point.id=i.estimate_point_id LEFT JOIN issues parent ON parent.id=i.parent_id LEFT JOIN projects parent_project ON parent_project.id=parent.project_id WHERE i.workspace_id=$1::uuid AND i.project_id=ANY($2::uuid[]) AND i.deleted_at IS NULL AND p.archived_at IS NULL AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=i.project_id AND pm.member_id=initiator.id AND pm.is_active=true) ORDER BY i.created_at,i.id`

func formatIssueExport(filename, format string, records []map[string]any) (string, []byte, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(records, "", "  ")
		return filename + ".json", data, err
	case "csv":
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		headers := make([]string, len(issueExportFields))
		for i, field := range issueExportFields {
			headers[i] = strings.Title(strings.ReplaceAll(field, "_", " "))
		}
		if err := writeSanitizedCSVRow(writer, headers); err != nil {
			return "", nil, err
		}
		for _, record := range records {
			row := make([]string, len(issueExportFields))
			for i, field := range issueExportFields {
				formatted, err := exportCellValue(record[field], false)
				if err != nil {
					return "", nil, err
				}
				if value, ok := formatted.(string); ok {
					row[i] = value
				} else if formatted != nil {
					row[i] = fmt.Sprint(formatted)
				}
			}
			if err := writeSanitizedCSVRow(writer, row); err != nil {
				return "", nil, err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return "", nil, err
		}
		return filename + ".csv", buffer.Bytes(), nil
	case "xlsx":
		workbook := excelize.NewFile()
		defer workbook.Close()
		sheet := workbook.GetSheetName(0)
		for column, field := range issueExportFields {
			cell, _ := excelize.CoordinatesToCellName(column+1, 1)
			if err := workbook.SetCellValue(sheet, cell, strings.Title(strings.ReplaceAll(field, "_", " "))); err != nil {
				return "", nil, err
			}
		}
		for rowIndex, record := range records {
			for column, field := range issueExportFields {
				value, err := exportCellValue(record[field], true)
				if err != nil {
					return "", nil, err
				}
				cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+2)
				if err := workbook.SetCellValue(sheet, cell, value); err != nil {
					return "", nil, err
				}
			}
		}
		buffer, err := workbook.WriteToBuffer()
		if err != nil {
			return "", nil, err
		}
		return filename + ".xlsx", buffer.Bytes(), nil
	default:
		return "", nil, fmt.Errorf("unsupported issue export format %q", format)
	}
}

func exportCellValue(value any, joinLists bool) (any, error) {
	if value == nil {
		return "", nil
	}
	switch item := value.(type) {
	case string:
		return sanitizeExportCell(item), nil
	case bool, float64, int64, int, json.Number:
		return item, nil
	case []any:
		if joinLists {
			parts := make([]string, len(item))
			for i, value := range item {
				formatted, err := exportCellValue(value, false)
				if err != nil {
					return nil, err
				}
				if formatted != nil {
					parts[i] = fmt.Sprint(formatted)
				}
			}
			return strings.Join(parts, ", "), nil
		}
	case map[string]any:
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return sanitizeExportCell(string(data)), nil
}

func sanitizeExportCell(value string) string {
	if value != "" && strings.ContainsRune("=+-@\t\r\n", rune(value[0])) {
		return "'" + value
	}
	return value
}

func writeSanitizedCSVRow(writer *csv.Writer, row []string) error {
	return writer.Write(row)
}

func zipIssueExport(files []struct {
	name string
	data []byte
}) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range files {
		entry, err := writer.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(file.data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
