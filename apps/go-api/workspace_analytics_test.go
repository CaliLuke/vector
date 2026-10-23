package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"uuid"

	store "github.com/CaliLuke/go-plane/internal/workspaceanalytics"
	api "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
)

type workspaceAnalyticsRepositoryStub struct {
	authorized bool
	body       json.RawMessage
	savedBody  json.RawMessage
	query      string
	args       []any
	chartErr   error
}

func (r *workspaceAnalyticsRepositoryStub) Authorized(context.Context, uuid.UUID, string, string) (bool, error) {
	return r.authorized, nil
}
func (r *workspaceAnalyticsRepositoryStub) AuthorizedProjectChart(context.Context, uuid.UUID, string, string) (bool, error) {
	return r.authorized, nil
}
func (r *workspaceAnalyticsRepositoryStub) AuthorizedDefault(context.Context, uuid.UUID, string) (bool, error) {
	return r.authorized, nil
}
func (r *workspaceAnalyticsRepositoryStub) Overview(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) WorkItemStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) ProjectStats(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) ProjectAssigneeStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) ProjectChart(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error) {
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) SavedView(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	if r.savedBody != nil {
		return r.savedBody, nil
	}
	return r.body, nil
}
func (r *workspaceAnalyticsRepositoryStub) Chart(context.Context, uuid.UUID, string, string, string, string, string, string, string, string) (json.RawMessage, error) {
	return r.body, r.chartErr
}
func (r *workspaceAnalyticsRepositoryStub) JSON(_ context.Context, q string, args ...any) (json.RawMessage, error) {
	r.query = q
	r.args = args
	return r.body, nil
}

func TestWorkspaceAnalyticsValidatesAxesBeforeQuery(t *testing.T) {
	repository := &workspaceAnalyticsRepositoryStub{authorized: true, body: json.RawMessage(`{"total":1,"distribution":{}}`)}
	service := &workspaceAnalyticsService{repository: repository}
	ctx := context.WithValue(context.Background(), workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{id: uuid.MustParse("11111111-1111-4111-8111-111111111111")})
	_, _, err := service.analytics(ctx, "acme", "state_id", "bogus", "", nil)
	if err == nil {
		t.Fatal("invalid y-axis was accepted")
	}
	response, ok := err.(interface{ LoomErrorName() string })
	if !ok || response.LoomErrorName() != "status_400" {
		t.Fatalf("error = %T %v", err, err)
	}
	if repository.query != "" {
		t.Fatal("invalid axes queried the database")
	}
}

func TestWorkspaceAnalyticsBindsFiltersAndPreservesJSON(t *testing.T) {
	repository := &workspaceAnalyticsRepositoryStub{authorized: true, body: json.RawMessage(`{"total":2,"distribution":{"started":[]}}`)}
	service := &workspaceAnalyticsService{repository: repository}
	ctx := context.WithValue(context.Background(), workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{id: uuid.MustParse("11111111-1111-4111-8111-111111111111")})
	result, body, err := service.analytics(ctx, "acme", "state__group", "issue_count", "", map[string]string{"priority": "high,urgent"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if result.ContentType != "application/json" || string(encoded) != string(repository.body) {
		t.Fatalf("result = %#v body=%s", result, encoded)
	}
	if len(repository.args) != 2 || repository.args[0] != "acme" || repository.args[1] != "high,urgent" {
		t.Fatalf("bound args = %#v", repository.args)
	}
	if !strings.Contains(repository.query, `state."group"`) || strings.Contains(repository.query, "high,urgent") {
		t.Fatalf("query does not use fixed dimension and bound filter: %s", repository.query)
	}
	if !strings.Contains(repository.query, "'extras'") {
		t.Fatalf("direct analytics query omitted its extras projection: %s", repository.query)
	}
}

func TestSavedAnalyticsViewUsesSavedResponseProjection(t *testing.T) {
	repository := &workspaceAnalyticsRepositoryStub{
		authorized: true,
		body:       json.RawMessage(`{"total":1,"distribution":{}}`),
		savedBody:  json.RawMessage(`{"query":{},"query_dict":{"x_axis":"state_id","y_axis":"issue_count"}}`),
	}
	service := &workspaceAnalyticsService{repository: repository}
	ctx := context.WithValue(context.Background(), workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{id: uuid.MustParse("11111111-1111-4111-8111-111111111111")})
	_, body, err := service.SavedAnalyticView(ctx, &api.SavedAnalyticViewPayload{Slug: "acme", AnalyticID: "11111111-1111-4111-8111-111111111112"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"total":1,"distribution":{}}` {
		t.Fatalf("saved analytics response = %s", encoded)
	}
	if strings.Contains(repository.query, "'extras'") {
		t.Fatalf("saved analytics query included the direct analytics extras projection: %s", repository.query)
	}
}

func TestWorkspaceAnalyticsRawCompactsPostgresJSONBelowCompressionThreshold(t *testing.T) {
	postgresJSON := json.RawMessage(`{ "data" : [ { "backlog" : 1, "completed" : 1, "count" : 2, "key" : "high", "name" : "high" }, { "count" : 1, "key" : "low", "name" : "low", "started" : 1 } ], "schema" : { "backlog" : "backlog", "completed" : "completed", "started" : "started" } }`)
	if len(postgresJSON) < 200 {
		t.Fatalf("regression input is %d bytes; need PostgreSQL-spaced JSON above the gzip threshold", len(postgresJSON))
	}

	metadata, body, err := workspaceAnalyticsRaw(postgresJSON)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) >= 200 {
		t.Fatalf("compacted response remains %d bytes, want below the source gzip threshold", len(encoded))
	}
	if metadata.ContentType != "application/json" {
		t.Fatalf("content type = %q", metadata.ContentType)
	}
	var got, want any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(postgresJSON, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compaction changed JSON value: got %#v, want %#v", got, want)
	}
}

func TestWorkspaceAnalyticsRawPreservesUnicodeAndRejectsInvalidJSON(t *testing.T) {
	metadata, body, err := workspaceAnalyticsRaw(json.RawMessage(`{ "label" : "café 東京" }`))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ContentType != "application/json" || string(encoded) != `{"label":"café 東京"}` {
		t.Fatalf("metadata/body = %#v / %s", metadata, encoded)
	}
	if _, _, err := workspaceAnalyticsRaw(json.RawMessage(`{"broken":`)); err == nil {
		t.Fatal("invalid JSON was accepted")
	}
}

func TestWorkspaceAnalyticsRepositoryRequired(t *testing.T) {
	if _, err := NewWorkspaceAnalytics(nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
	if _, err := store.NewRepository(nil); err == nil {
		t.Fatal("nil database accepted")
	}
}

func TestProjectChartMapsOnlyInvalidDimensionsToBadRequest(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "invalid x axis", err: store.ErrInvalidChartXAxis, want: "status_400"},
		{name: "invalid group by", err: store.ErrInvalidChartGroupBy, want: "status_400"},
		{name: "database error", err: errors.New("postgres syntax error"), want: "status_500"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &workspaceAnalyticsRepositoryStub{authorized: true, chartErr: test.err}
			service := &workspaceAnalyticsService{repository: repository}
			ctx := context.WithValue(context.Background(), workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{
				id: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			})
			_, _, err := service.ProjectAdvanceAnalyticsCharts(ctx, &api.ProjectAdvanceAnalyticsChartsPayload{
				Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111112",
				Type: workspaceAnalyticsTestStringPointer("custom-work-items"), XAxis: workspaceAnalyticsTestStringPointer("PRIORITY"),
			})
			if err == nil {
				t.Fatal("expected chart error")
			}
			failure, ok := err.(interface{ LoomErrorName() string })
			if !ok || failure.LoomErrorName() != test.want {
				t.Fatalf("error = %T %v, want %s", err, err, test.want)
			}
		})
	}
}

func TestProjectMonthlyChartUsesFilteredWindowAndLinkDailySeries(t *testing.T) {
	for _, test := range []struct {
		name      string
		payload   *api.ProjectAdvanceAnalyticsChartsPayload
		wantQuery []string
		wantArgs  int
	}{
		{
			name: "monthly date filter",
			payload: &api.ProjectAdvanceAnalyticsChartsPayload{
				Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111112",
				Type: workspaceAnalyticsTestStringPointer("work-items"), DateFilter: workspaceAnalyticsTestStringPointer("last_30_days"),
			},
			wantQuery: []string{"generate_series(", "project.created_at", "filter_window.start_date", "i.created_at::date BETWEEN CURRENT_DATE-30"},
			wantArgs:  7,
		},
		{
			name: "cycle daily bounds",
			payload: &api.ProjectAdvanceAnalyticsChartsPayload{
				Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111112",
				Type: workspaceAnalyticsTestStringPointer("work-items"), CycleID: workspaceAnalyticsTestStringPointer("11111111-1111-4111-8111-111111111113"),
			},
			wantQuery: []string{"generate_series(bounds.start_day,bounds.end_day,interval '1 day')", "cycle_issue.created_at::date", `state."group"='completed'`, "cycle.start_date::date"},
			wantArgs:  6,
		},
		{
			name: "module daily bounds",
			payload: &api.ProjectAdvanceAnalyticsChartsPayload{
				Slug: "acme", ProjectID: "11111111-1111-4111-8111-111111111112",
				Type: workspaceAnalyticsTestStringPointer("work-items"), ModuleID: workspaceAnalyticsTestStringPointer("11111111-1111-4111-8111-111111111114"),
			},
			wantQuery: []string{"generate_series(bounds.start_day,bounds.end_day,interval '1 day')", "module_issue.created_at::date", `state."group"='completed'`, "module.start_date"},
			wantArgs:  6,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &workspaceAnalyticsRepositoryStub{authorized: true, body: json.RawMessage(`{"data":[]}`)}
			service := &workspaceAnalyticsService{repository: repository}
			ctx := context.WithValue(context.Background(), workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{
				id: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			})
			_, response, err := service.ProjectAdvanceAnalyticsCharts(ctx, test.payload)
			if err != nil {
				t.Fatal(err)
			}
			response.Close()
			for _, fragment := range test.wantQuery {
				if !strings.Contains(repository.query, fragment) {
					t.Errorf("query does not contain %q", fragment)
				}
			}
			if len(repository.args) != test.wantArgs {
				t.Fatalf("query args = %d, want %d", len(repository.args), test.wantArgs)
			}
		})
	}
}

func workspaceAnalyticsTestStringPointer(value string) *string { return &value }
