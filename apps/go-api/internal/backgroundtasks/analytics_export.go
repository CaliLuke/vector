package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	csv "github.com/JensRantil/go-csv"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	analyticExportTask      = "plane.bgtasks.analytic_plot_export.analytic_export_task"
	analyticsExportWorkflow = "plane.exports.analytics_csv_email"
	analyticsExportActivity = "plane.exports.generate_analytics_csv_email"
)

var analyticsAxisExpressions = map[string]string{
	"state_id":                "issue.state_id::text",
	"state__group":            `state."group"`,
	"labels__id":              "label_link.label_id::text",
	"assignees__id":           "assignee_link.assignee_id::text",
	"estimate_point__value":   "estimate_point.value::text",
	"issue_cycle__cycle_id":   "cycle_link.cycle_id::text",
	"issue_module__module_id": "module_link.module_id::text",
	"priority":                "issue.priority",
	"start_date":              "extract(year FROM issue.start_date)::int::text||'-'||extract(month FROM issue.start_date)::int::text",
	"target_date":             "extract(year FROM issue.target_date)::int::text||'-'||extract(month FROM issue.target_date)::int::text",
	"created_at":              "extract(year FROM issue.created_at)::int::text||'-'||extract(month FROM issue.created_at)::int::text",
	"completed_at":            "extract(year FROM issue.completed_at)::int::text||'-'||extract(month FROM issue.completed_at)::int::text",
}

var analyticsAxisHeaders = map[string]string{
	"state__name": "State", "state__group": "State Group", "labels__id": "Label",
	"assignees__id": "Assignee Name", "start_date": "Start Date", "target_date": "Due Date",
	"completed_at": "Completed At", "created_at": "Created At", "issue_count": "Issue Count",
	"priority": "Priority", "estimate": "Estimate", "issue_cycle__cycle_id": "Cycle",
	"issue_module__module_id": "Module",
}

type AnalyticsExportInput struct {
	EventID string          `json:"event_id"`
	Email   string          `json:"email"`
	Slug    string          `json:"slug"`
	Data    json.RawMessage `json:"data"`
}

type analyticsExportStore struct {
	pool   *pgxpool.Pool
	config issueEmailSMTPConfigProvider
	sender issueEmailSender
}

type AnalyticsExportDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[AnalyticsExportInput, struct{}]
}

type AnalyticsExportTaskPublisher struct {
	fallback   TaskFallback
	dispatcher *AnalyticsExportDispatcher
}

func NewPostgresAnalyticsExportStore(pool *pgxpool.Pool, config issueEmailSMTPConfigProvider, sender issueEmailSender) (*analyticsExportStore, error) {
	if pool == nil || config == nil || sender == nil {
		return nil, errors.New("analytics export PostgreSQL pool, SMTP configuration and sender are required")
	}
	return &analyticsExportStore{pool: pool, config: config, sender: sender}, nil
}

func RegisterAnalyticsExportWorkflow(app *selvedge.App, store *analyticsExportStore, policy *retry.Policy) (*AnalyticsExportDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and analytics export store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[AnalyticsExportInput, struct{}]{Name: analyticsExportActivity}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input AnalyticsExportInput) (struct{}, error) {
		if _, err := uuid.Parse(input.EventID); err != nil {
			return struct{}{}, fmt.Errorf("invalid analytics export event identity: %w", err)
		}
		if err := store.ExportAndEmail(ctx, input); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: policy}); err != nil {
		return nil, fmt.Errorf("register analytics export activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(analyticsExportWorkflow, func(ctx *selvedge.WorkflowContext, input AnalyticsExportInput) (struct{}, error) {
		if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Slug) == "" || !json.Valid(input.Data) {
			return struct{}{}, errors.New("analytics export email, workspace slug and data are required")
		}
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &AnalyticsExportDispatcher{app: app, workflow: workflow}, nil
}

func NewAnalyticsExportTaskPublisher(fallback TaskFallback, dispatcher *AnalyticsExportDispatcher) (*AnalyticsExportTaskPublisher, error) {
	if fallback == nil || dispatcher == nil {
		return nil, errors.New("task fallback and analytics export dispatcher are required")
	}
	return &AnalyticsExportTaskPublisher{fallback: fallback, dispatcher: dispatcher}, nil
}

func (dispatcher *AnalyticsExportDispatcher) Admit(ctx context.Context, input AnalyticsExportInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable analytics export dispatcher is not configured")
	}
	if _, err := uuid.Parse(input.EventID); err != nil {
		return "", fmt.Errorf("analytics export event ID must be a UUID: %w", err)
	}
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Slug) == "" || !json.Valid(input.Data) {
		return "", errors.New("analytics export email, workspace slug and data are required")
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "plane-analytics-export:"+input.EventID, input)
}

func (publisher *AnalyticsExportTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.dispatcher == nil {
		return errors.New("analytics export task publisher is not configured")
	}
	if name != analyticExportTask {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	if len(args) != 0 {
		return errors.New("analytics export task requires keyword arguments")
	}
	encoded, err := json.Marshal(kwargs)
	if err != nil {
		return fmt.Errorf("encode analytics export arguments: %w", err)
	}
	var input AnalyticsExportInput
	if err := json.Unmarshal(encoded, &input); err != nil {
		return fmt.Errorf("decode analytics export arguments: %w", err)
	}
	input.EventID = uuid.NewString()
	if _, err := publisher.dispatcher.Admit(ctx, input); err != nil {
		return fmt.Errorf("admit analytics export workflow: %w", err)
	}
	return nil
}

func (publisher *AnalyticsExportTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("analytics export task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func (store *analyticsExportStore) ExportAndEmail(ctx context.Context, input AnalyticsExportInput) error {
	rows, err := store.rows(ctx, input.Slug, input.Data)
	if err != nil {
		return fmt.Errorf("build analytics export rows: %w", err)
	}
	var csvData strings.Builder
	for rowIndex, row := range rows {
		clean := make([]string, len(row))
		for i, value := range row {
			clean[i] = value
			// Metrics are numeric output; sanitize user-controlled labels and headers.
			if rowIndex == 0 || i == 0 {
				clean[i] = sanitizeExportCell(value)
			}
		}
		if err := writeAnalyticsCSVRow(&csvData, clean); err != nil {
			return fmt.Errorf("write analytics CSV row %d: %w", rowIndex+1, err)
		}
	}
	config, from, err := store.config.SMTPConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("load SMTP configuration for analytics export: %w", err)
	}
	if config.Host == "" || strings.TrimSpace(input.Email) == "" {
		return errors.New("SMTP host and analytics export recipient are required")
	}
	htmlBody, textBody, err := renderPlaneEmail("emails/exports/analytics.html", nil)
	if err != nil {
		return err
	}
	message := instancemail.Message{
		From: from, To: input.Email, Subject: "Your Export is ready", Body: textBody, HTMLBody: htmlBody,
		Attachments: []instancemail.Attachment{{Name: input.Slug + "-analytics.csv", ContentType: "text/csv; charset=utf-8", Data: []byte(csvData.String())}},
	}
	if err := store.sender.Send(ctx, config, message); err != nil {
		return fmt.Errorf("send analytics CSV export: %w", err)
	}
	return nil
}

// Django's source writer uses csv.QUOTE_ALL with CRLF record endings.
func writeAnalyticsCSVRow(output io.Writer, row []string) error {
	writer := csv.NewDialectWriter(output, csv.Dialect{Quoting: csv.QuoteAll, LineTerminator: "\r\n"})
	if err := writer.Write(row); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func (store *analyticsExportStore) rows(ctx context.Context, slug string, raw json.RawMessage) ([][]string, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode analytics filter data: %w", err)
	}
	stringValue := func(key string) string {
		var value string
		if json.Unmarshal(data[key], &value) == nil {
			return value
		}
		var values []any
		if json.Unmarshal(data[key], &values) == nil {
			parts := make([]string, 0, len(values))
			for _, item := range values {
				if text, ok := item.(string); ok {
					parts = append(parts, text)
				}
			}
			return strings.Join(parts, ",")
		}
		return ""
	}
	xAxis, yAxis, segment := stringValue("x_axis"), stringValue("y_axis"), stringValue("segment")
	xExpr := analyticsAxisExpressions[xAxis]
	if xExpr == "" || (yAxis != "issue_count" && yAxis != "estimate") || (segment != "" && (analyticsAxisExpressions[segment] == "" || segment == xAxis)) {
		return nil, errors.New("invalid analytics axis configuration")
	}
	filters := make(map[string]string)
	present := make(map[string]bool)
	for _, key := range []string{"state", "state_group", "estimate_point", "priority", "parent", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "labels", "assignees", "mentions", "intake_status", "inbox_status", "cycle", "module", "subscriber", "sub_issue", "start_target_date"} {
		if _, ok := data[key]; ok {
			filters[key] = stringValue(key)
			present[key] = true
		}
	}
	where := []string{"workspace.slug=$1", "issue.deleted_at IS NULL", "issue.archived_at IS NULL", "issue.is_draft IS NOT TRUE", "project.deleted_at IS NULL", "project.archived_at IS NULL", `NOT EXISTS (SELECT 1 FROM states manager_state WHERE manager_state.id=issue.state_id AND manager_state."group"='triage')`}
	where, args, err := workitems.BuildAppIssueFilters(filters, present, where, []any{slug})
	if err != nil {
		return nil, err
	}
	joins := ` LEFT JOIN states state ON state.id=issue.state_id LEFT JOIN estimate_points estimate_point ON estimate_point.id=issue.estimate_point_id`
	if xAxis == "labels__id" || segment == "labels__id" {
		joins += ` LEFT JOIN issue_labels label_link ON ` + issuequery.LiveIssueLabelPredicate("issue", "label_link") + ` LEFT JOIN labels label ON label.id=label_link.label_id`
	}
	if xAxis == "assignees__id" || segment == "assignees__id" {
		joins += ` LEFT JOIN issue_assignees assignee_link ON assignee_link.issue_id=issue.id AND assignee_link.deleted_at IS NULL LEFT JOIN users assignee ON assignee.id=assignee_link.assignee_id`
	}
	if xAxis == "issue_cycle__cycle_id" || segment == "issue_cycle__cycle_id" {
		joins += ` LEFT JOIN cycle_issues cycle_link ON cycle_link.issue_id=issue.id AND cycle_link.deleted_at IS NULL LEFT JOIN cycles cycle ON cycle.id=cycle_link.cycle_id`
	}
	if xAxis == "issue_module__module_id" || segment == "issue_module__module_id" {
		joins += ` LEFT JOIN module_issues module_link ON module_link.issue_id=issue.id AND module_link.deleted_at IS NULL LEFT JOIN modules module ON module.id=module_link.module_id`
	}
	axisLabel := func(axis, prefix string) string {
		switch axis {
		case "labels__id":
			return "COALESCE(" + prefix + "label.name," + analyticsAxisExpressions[axis] + ")"
		case "state_id":
			return "COALESCE(state.name," + analyticsAxisExpressions[axis] + ")"
		case "assignees__id":
			return "COALESCE(assignee.first_name||' '||assignee.last_name," + analyticsAxisExpressions[axis] + ")"
		case "issue_cycle__cycle_id":
			return "COALESCE(cycle.name," + analyticsAxisExpressions[axis] + ")"
		case "issue_module__module_id":
			return "COALESCE(module.name," + analyticsAxisExpressions[axis] + ")"
		default:
			return analyticsAxisExpressions[axis]
		}
	}
	xLabel := axisLabel(xAxis, "")
	segmentExpr, segmentLabel := "NULL::text", "NULL::text"
	if segment != "" {
		segmentExpr, segmentLabel = analyticsAxisExpressions[segment], axisLabel(segment, "")
	}
	metric := "count(*)::double precision"
	if yAxis == "estimate" {
		metric = "sum(estimate_point.value::double precision)"
	}
	query := `SELECT ` + xExpr + `::text,` + xLabel + `::text,` + segmentExpr + `::text,` + segmentLabel + `::text,` + metric + ` FROM issues issue JOIN workspaces workspace ON workspace.id=issue.workspace_id JOIN projects project ON project.id=issue.project_id` + joins + ` WHERE ` + strings.Join(where, " AND ") + ` GROUP BY 1,2,3,4 ORDER BY 1 NULLS LAST,3 NULLS LAST`
	rowsResult, err := store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query analytics distribution: %w", err)
	}
	defer rowsResult.Close()
	type cell struct {
		value, label string
		metric       *float64
	}
	distribution := map[string][]cell{}
	segmentLabels := map[string]string{}
	segmentSet := map[string]struct{}{}
	for rowsResult.Next() {
		var dimension, label, segmentValue, segmentName *string
		var metricValue *float64
		if err := rowsResult.Scan(&dimension, &label, &segmentValue, &segmentName, &metricValue); err != nil {
			return nil, err
		}
		if dimension == nil {
			continue
		}
		item := cell{value: "", metric: metricValue}
		if label != nil {
			item.value = *label
		} else {
			item.value = *dimension
		}
		if segmentValue != nil {
			item.label = *segmentValue
			segmentSet[*segmentValue] = struct{}{}
			if segmentName != nil {
				segmentLabels[*segmentValue] = *segmentName
			}
		}
		distribution[*dimension] = append(distribution[*dimension], item)
	}
	if err := rowsResult.Err(); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(distribution))
	for key := range distribution {
		keys = append(keys, key)
	}
	if xAxis == "priority" {
		order := map[string]int{"low": 0, "medium": 1, "high": 2, "urgent": 3, "none": 4}
		sort.Slice(keys, func(i, j int) bool { return order[keys[i]] < order[keys[j]] })
	} else {
		sort.Strings(keys)
	}
	segments := make([]string, 0, len(segmentSet))
	for key := range segmentSet {
		segments = append(segments, key)
	}
	sort.Strings(segments)
	axisHeader := analyticsAxisHeaders[xAxis]
	if axisHeader == "" {
		axisHeader = "X-Axis"
	}
	yHeader := analyticsAxisHeaders[yAxis]
	header := []string{axisHeader, yHeader}
	if segment != "" {
		for _, key := range segments {
			if name := segmentLabels[key]; name != "" {
				header = append(header, name)
			} else {
				header = append(header, key)
			}
		}
	}
	result := [][]string{header}
	for _, key := range keys {
		items := distribution[key]
		row := []string{items[0].value, "0"}
		total := 0.0
		for _, item := range items {
			if item.metric != nil {
				total += *item.metric
			}
		}
		row[1] = fmt.Sprintf("%g", total)
		if segment != "" {
			for _, segmentKey := range segments {
				value := 0.0
				found := false
				for _, item := range items {
					if item.label == segmentKey && item.metric != nil {
						value = *item.metric
						found = true
						break
					}
				}
				if found {
					row = append(row, fmt.Sprintf("%g", value))
				} else {
					row = append(row, "0")
				}
			}
		}
		result = append(result, row)
	}
	return result, nil
}
