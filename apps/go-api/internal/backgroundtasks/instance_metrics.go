package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	instanceMetricsWorkflowName     = "plane.license.push_instance_metrics"
	instanceMetricsActivityName     = "plane.license.collect_instance_metrics"
	instanceMetricsScheduleWorkflow = "plane.beat.push_instance_metrics"
	instanceMetricsScheduleID       = "plane-beat:push-instance-metrics"
	instanceMetricsWorkspaceLimit   = 1000
	instanceMetricsExportInterval   = 20 * time.Second
	instanceMetricsExportTimeout    = 30 * time.Second
	instanceMetricsDefaultInterval  = 360
	instanceMetricsMinimumInterval  = 1
	instanceMetricsMaximumInterval  = 10_000_000
)

type InstanceMetricObservation struct {
	Name        string
	Description string
	Value       int64
	Attributes  map[string]string
}

type InstanceMetricsSnapshot struct {
	ServiceName  string
	InstanceID   string
	Observations []InstanceMetricObservation
}

type instanceMetricsExporter interface {
	Export(context.Context, InstanceMetricsSnapshot) error
}

type PostgresInstanceMetricsStore struct {
	pool     *pgxpool.Pool
	exporter instanceMetricsExporter
}

type InstanceMetricsDispatcher struct {
	app              *selvedge.App
	workflow         *selvedge.WorkflowFunc[instanceMetricsInput, struct{}]
	scheduleWorkflow *selvedge.WorkflowFunc[struct{}, struct{}]
}

type instanceMetricsInput struct {
	EventID string `json:"event_id"`
}

type instanceMetricsWorkspace struct {
	ID   uuid.UUID
	Slug string
}

type instanceMetricsCounts struct {
	Users, Workspaces, Projects, Issues, Modules int64
	Cycles, CycleIssues, ModuleIssues, Pages     int64
}

func NewPostgresInstanceMetricsStore(pool *pgxpool.Pool, exporter instanceMetricsExporter) (*PostgresInstanceMetricsStore, error) {
	if pool == nil || exporter == nil {
		return nil, errors.New("instance metrics PostgreSQL pool and exporter are required")
	}
	return &PostgresInstanceMetricsStore{pool: pool, exporter: exporter}, nil
}

func RegisterInstanceMetricsWorkflow(app *selvedge.App, store *PostgresInstanceMetricsStore, policy *retry.Policy) (*InstanceMetricsDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and instance metrics store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[instanceMetricsInput, struct{}]{Name: instanceMetricsActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input instanceMetricsInput) (struct{}, error) {
		if _, err := uuid.Parse(input.EventID); err != nil {
			return struct{}{}, fmt.Errorf("invalid instance metrics event identity: %w", err)
		}
		if err := store.Push(ctx); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 45 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register instance metrics activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(instanceMetricsWorkflowName, func(ctx *selvedge.WorkflowContext, input instanceMetricsInput) (struct{}, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	interval := instanceMetricsBeatIntervalFromEnv()
	schedule := selvedge.DefineWorkflow(instanceMetricsScheduleWorkflow, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
		for {
			now, err := selvedge.Now(ctx)
			if err != nil {
				return struct{}{}, err
			}
			next := now.UTC().Add(interval)
			if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("instance-metrics-beat:"+next.Format(time.RFC3339Nano))); err != nil {
				return struct{}{}, err
			}
			scheduledEvent := uuid.NewSHA1(uuid.NameSpaceOID, []byte(next.Format(time.RFC3339Nano))).String()
			for attempt := 0; ; attempt++ {
				eventID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%d", scheduledEvent, attempt))).String()
				child, err := selvedge.StartChild[struct{}](ctx, instanceMetricsWorkflowName, instanceMetricsInput{EventID: eventID}, selvedge.WithChildID("plane-instance-metrics:"+eventID))
				if err != nil {
					return struct{}{}, err
				}
				if _, err := child.Get(ctx); err == nil {
					break
				}
				if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("instance-metrics-retry:%s:%d", scheduledEvent, attempt))); err != nil {
					return struct{}{}, err
				}
			}
		}
	})
	selvedge.RegisterWorkflow(app, schedule)
	return &InstanceMetricsDispatcher{app: app, workflow: workflow, scheduleWorkflow: schedule}, nil
}

func (dispatcher *InstanceMetricsDispatcher) Admit(ctx context.Context, eventID string) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("instance metrics dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("instance metrics event identity must be a UUID: %w", err)
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "plane-instance-metrics:"+eventID, instanceMetricsInput{EventID: eventID})
}

func (dispatcher *InstanceMetricsDispatcher) StartBeatSchedule(ctx context.Context) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.scheduleWorkflow == nil {
		return "", errors.New("instance metrics Beat schedule is not configured")
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.scheduleWorkflow, instanceMetricsScheduleID, struct{}{})
}

func instanceMetricsBeatIntervalFromEnv() time.Duration {
	minutes, err := strconv.Atoi(strings.TrimSpace(os.Getenv("METRICS_PUSH_INTERVAL_MINUTES")))
	if os.Getenv("METRICS_PUSH_INTERVAL_MINUTES") == "" || err != nil || minutes < instanceMetricsMinimumInterval || minutes > instanceMetricsMaximumInterval {
		minutes = instanceMetricsDefaultInterval
	}
	return time.Duration(minutes) * time.Minute
}

func (store *PostgresInstanceMetricsStore) Push(ctx context.Context) error {
	snapshot, enabled, err := store.collect(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	if err := store.exporter.Export(ctx, snapshot); err != nil {
		return fmt.Errorf("export Plane instance metrics: %w", err)
	}
	return nil
}

func (store *PostgresInstanceMetricsStore) collect(ctx context.Context) (InstanceMetricsSnapshot, bool, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return InstanceMetricsSnapshot{}, false, fmt.Errorf("begin instance metrics snapshot: %w", err)
	}
	defer tx.Rollback(ctx)
	var instanceID, instanceName, currentVersion, edition string
	var latestVersion *string
	var isTelemetryEnabled, isVerified, isSetupDone bool
	err = tx.QueryRow(ctx, `SELECT instance_id,instance_name,current_version,latest_version,edition,is_telemetry_enabled,is_verified,is_setup_done FROM instances WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 1`).Scan(&instanceID, &instanceName, &currentVersion, &latestVersion, &edition, &isTelemetryEnabled, &isVerified, &isSetupDone)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !isTelemetryEnabled {
		return InstanceMetricsSnapshot{}, false, nil
	}
	if err != nil {
		return InstanceMetricsSnapshot{}, false, fmt.Errorf("load telemetry-enabled Plane instance: %w", err)
	}
	counts, err := collectInstanceMetricCounts(ctx, tx)
	if err != nil {
		return InstanceMetricsSnapshot{}, false, err
	}
	workspaces, err := collectInstanceMetricWorkspaces(ctx, tx)
	if err != nil {
		return InstanceMetricsSnapshot{}, false, err
	}
	workspaceCounts, err := collectWorkspaceMetricCounts(ctx, tx, workspaces)
	if err != nil {
		return InstanceMetricsSnapshot{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InstanceMetricsSnapshot{}, false, fmt.Errorf("commit instance metrics snapshot: %w", err)
	}
	if latestVersion == nil {
		latestVersion = new(string)
	}
	attrs := map[string]string{
		"instance_id": instanceID, "instance_name": instanceName, "current_version": currentVersion,
		"latest_version": *latestVersion, "edition": edition, "domain": instanceMetricDomain(os.Getenv("WEB_URL")),
		"is_verified": strconv.FormatBool(isVerified), "is_setup_done": strconv.FormatBool(isSetupDone),
	}
	snapshot := InstanceMetricsSnapshot{ServiceName: envOrDefault("SERVICE_NAME", "vector-ce-api"), InstanceID: instanceID}
	addInstance := func(name, description string, value int64) {
		snapshot.Observations = append(snapshot.Observations, InstanceMetricObservation{Name: name, Description: description, Value: value, Attributes: attrs})
	}
	addInstance("plane_instance_users_total", "Total number of users in the Plane instance", counts.Users)
	addInstance("plane_instance_workspaces_total", "Total number of workspaces", counts.Workspaces)
	addInstance("plane_instance_projects_total", "Total number of projects across all workspaces", counts.Projects)
	addInstance("plane_instance_issues_total", "Total number of issues across all projects", counts.Issues)
	addInstance("plane_instance_modules_total", "Total number of modules", counts.Modules)
	addInstance("plane_instance_cycles_total", "Total number of cycles", counts.Cycles)
	addInstance("plane_instance_cycle_issues_total", "Total number of issues in cycles", counts.CycleIssues)
	addInstance("plane_instance_module_issues_total", "Total number of issues in modules", counts.ModuleIssues)
	addInstance("plane_instance_pages_total", "Total number of pages", counts.Pages)
	for _, workspace := range workspaces {
		workspaceAttrs := map[string]string{"workspace_id": workspace.ID.String(), "workspace_slug": workspace.Slug, "instance_id": instanceID}
		for _, metric := range []struct {
			name, description string
			value             int64
		}{
			{"plane_workspace_projects_total", "Number of projects per workspace", workspaceCounts[workspace.ID].Projects},
			{"plane_workspace_issues_total", "Number of issues per workspace", workspaceCounts[workspace.ID].Issues},
			{"plane_workspace_modules_total", "Number of modules per workspace", workspaceCounts[workspace.ID].Modules},
			{"plane_workspace_cycles_total", "Number of cycles per workspace", workspaceCounts[workspace.ID].Cycles},
			{"plane_workspace_members_total", "Number of members per workspace", workspaceCounts[workspace.ID].Members},
			{"plane_workspace_pages_total", "Number of pages per workspace", workspaceCounts[workspace.ID].Pages},
		} {
			snapshot.Observations = append(snapshot.Observations, InstanceMetricObservation{Name: metric.name, Description: metric.description, Value: metric.value, Attributes: workspaceAttrs})
		}
	}
	return snapshot, true, nil
}

func collectInstanceMetricCounts(ctx context.Context, tx pgx.Tx) (instanceMetricsCounts, error) {
	var counts instanceMetricsCounts
	err := tx.QueryRow(ctx, `SELECT
 (SELECT COUNT(*) FROM users WHERE is_bot=false),
 (SELECT COUNT(*) FROM workspaces WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM projects WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM issues WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM modules WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM cycles WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM cycle_issues WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM module_issues WHERE deleted_at IS NULL),
 (SELECT COUNT(*) FROM pages page WHERE page.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM users owner WHERE owner.id=page.owned_by_id AND owner.is_bot=true AND page.access=1))`).Scan(&counts.Users, &counts.Workspaces, &counts.Projects, &counts.Issues, &counts.Modules, &counts.Cycles, &counts.CycleIssues, &counts.ModuleIssues, &counts.Pages)
	if err != nil {
		return instanceMetricsCounts{}, fmt.Errorf("collect Plane instance metric counts: %w", err)
	}
	return counts, nil
}

func collectInstanceMetricWorkspaces(ctx context.Context, tx pgx.Tx) ([]instanceMetricsWorkspace, error) {
	rows, err := tx.Query(ctx, `SELECT id,slug FROM workspaces WHERE deleted_at IS NULL ORDER BY created_at LIMIT $1`, instanceMetricsWorkspaceLimit)
	if err != nil {
		return nil, fmt.Errorf("list workspaces for instance metrics: %w", err)
	}
	defer rows.Close()
	workspaces := make([]instanceMetricsWorkspace, 0, instanceMetricsWorkspaceLimit)
	for rows.Next() {
		var workspace instanceMetricsWorkspace
		if err := rows.Scan(&workspace.ID, &workspace.Slug); err != nil {
			return nil, fmt.Errorf("read workspace for instance metrics: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instance metrics workspaces: %w", err)
	}
	rows.Close()
	return workspaces, nil
}

type instanceMetricsWorkspaceCounts struct{ Projects, Issues, Modules, Cycles, Members, Pages int64 }

func collectWorkspaceMetricCounts(ctx context.Context, tx pgx.Tx, workspaces []instanceMetricsWorkspace) (map[uuid.UUID]instanceMetricsWorkspaceCounts, error) {
	counts := make(map[uuid.UUID]instanceMetricsWorkspaceCounts, len(workspaces))
	if len(workspaces) == 0 {
		return counts, nil
	}
	ids := make([]uuid.UUID, 0, len(workspaces))
	for _, workspace := range workspaces {
		ids = append(ids, workspace.ID)
		counts[workspace.ID] = instanceMetricsWorkspaceCounts{}
	}
	for _, query := range []struct {
		table  string
		assign func(*instanceMetricsWorkspaceCounts, int64)
	}{
		{"projects", func(c *instanceMetricsWorkspaceCounts, n int64) { c.Projects = n }},
		{"issues", func(c *instanceMetricsWorkspaceCounts, n int64) { c.Issues = n }},
		{"modules", func(c *instanceMetricsWorkspaceCounts, n int64) { c.Modules = n }},
		{"cycles", func(c *instanceMetricsWorkspaceCounts, n int64) { c.Cycles = n }},
		{"workspace_members", func(c *instanceMetricsWorkspaceCounts, n int64) { c.Members = n }},
	} {
		rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT workspace_id,COUNT(*) FROM %s WHERE deleted_at IS NULL AND workspace_id=ANY($1::uuid[]) GROUP BY workspace_id`, query.table), ids)
		if err != nil {
			return nil, fmt.Errorf("collect per-workspace %s metrics: %w", query.table, err)
		}
		for rows.Next() {
			var workspaceID uuid.UUID
			var count int64
			if err := rows.Scan(&workspaceID, &count); err != nil {
				rows.Close()
				return nil, fmt.Errorf("read per-workspace %s metric: %w", query.table, err)
			}
			value := counts[workspaceID]
			query.assign(&value, count)
			counts[workspaceID] = value
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate per-workspace %s metrics: %w", query.table, err)
		}
		rows.Close()
	}
	rows, err := tx.Query(ctx, `SELECT page.workspace_id,COUNT(*) FROM pages page WHERE page.deleted_at IS NULL AND page.workspace_id=ANY($1::uuid[]) AND NOT EXISTS(SELECT 1 FROM users owner WHERE owner.id=page.owned_by_id AND owner.is_bot=true AND page.access=1) GROUP BY page.workspace_id`, ids)
	if err != nil {
		return nil, fmt.Errorf("collect per-workspace page metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var workspaceID uuid.UUID
		var count int64
		if err := rows.Scan(&workspaceID, &count); err != nil {
			return nil, fmt.Errorf("read per-workspace page metrics: %w", err)
		}
		value := counts[workspaceID]
		value.Pages = count
		counts[workspaceID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate per-workspace page metrics: %w", err)
	}
	return counts, nil
}

type otlpInstanceMetricsExporter struct{}

func (otlpInstanceMetricsExporter) Export(ctx context.Context, snapshot InstanceMetricsSnapshot) error {
	if strings.TrimSpace(os.Getenv("OTLP_ENDPOINT")) == "" {
		return nil
	}
	exporter, err := newInstanceOTLPExporter(ctx)
	if err != nil {
		return fmt.Errorf("configure OTLP metrics exporter: %w", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(instanceMetricsExportInterval), sdkmetric.WithTimeout(instanceMetricsExportTimeout))
	resourceAttrs := []attribute.KeyValue{
		attribute.String("service.name", snapshot.ServiceName),
		attribute.String("instance_id", snapshot.InstanceID),
		attribute.String("plane.instance.type", "self-hosted"),
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithResource(resource.NewWithAttributes("", resourceAttrs...)), sdkmetric.WithReader(reader))
	meter := provider.Meter("plane.license.telemetry_metrics")
	byName := make(map[string][]InstanceMetricObservation)
	order := make([]string, 0)
	for _, observation := range snapshot.Observations {
		if _, exists := byName[observation.Name]; !exists {
			order = append(order, observation.Name)
		}
		byName[observation.Name] = append(byName[observation.Name], observation)
	}
	for _, name := range order {
		observations := byName[name]
		gauge, err := meter.Int64ObservableGauge(name, metric.WithDescription(observations[0].Description))
		if err != nil {
			_ = provider.Shutdown(context.Background())
			return fmt.Errorf("register metric %q: %w", name, err)
		}
		registration, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
			for _, observation := range observations {
				attributes := make([]attribute.KeyValue, 0, len(observation.Attributes))
				for key, value := range observation.Attributes {
					attributes = append(attributes, attribute.String(key, value))
				}
				observer.ObserveInt64(gauge, observation.Value, metric.WithAttributes(attributes...))
			}
			return nil
		}, gauge)
		if err != nil {
			_ = provider.Shutdown(context.Background())
			return fmt.Errorf("register metric callback %q: %w", name, err)
		}
		defer registration.Unregister()
	}
	if err := provider.ForceFlush(ctx); err != nil {
		_ = provider.Shutdown(context.Background())
		return fmt.Errorf("flush instance metrics to OTLP: %w", err)
	}
	if err := provider.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown OTLP metrics provider: %w", err)
	}
	return nil
}

func newInstanceOTLPExporter(ctx context.Context) (sdkmetric.Exporter, error) {
	protocol := strings.ToLower(strings.TrimSpace(envOrDefault("OTLP_METRICS_PROTOCOL", "grpc")))
	baseEndpoint := strings.TrimSpace(os.Getenv("OTLP_ENDPOINT"))
	if baseEndpoint == "" {
		return nil, errors.New("OTLP_ENDPOINT is required to export instance metrics")
	}
	switch protocol {
	case "grpc":
		endpoint, err := instanceMetricsGRPCEndpoint(baseEndpoint)
		if err != nil {
			return nil, err
		}
		insecure := strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_METRICS_INSECURE"), "true")
		scheme := "https"
		if insecure {
			scheme = "http"
		}
		options := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpointURL(scheme + "://" + endpoint)}
		return otlpmetricgrpc.New(ctx, options...)
	case "http":
		endpoint, err := instanceMetricsHTTPEndpoint(baseEndpoint)
		if err != nil {
			return nil, err
		}
		return otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(endpoint))
	default:
		return nil, fmt.Errorf("OTLP_METRICS_PROTOCOL must be grpc or http, got %q", protocol)
	}
}

func instanceMetricsGRPCEndpoint(raw string) (string, error) {
	parsed, err := parseInstanceMetricsEndpoint(raw)
	if err != nil {
		return "", err
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "4317"
		}
	}
	return net.JoinHostPort(parsed.Hostname(), port), nil
}

func instanceMetricsHTTPEndpoint(raw string) (string, error) {
	parsed, err := parseInstanceMetricsEndpoint(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1/metrics"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func parseInstanceMetricsEndpoint(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid OTLP metrics endpoint %q", raw)
	}
	return parsed, nil
}

func instanceMetricDomain(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "//" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
