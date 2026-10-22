package cyclereads

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type lifecycleTaskCapture struct {
	name   string
	args   []any
	kwargs map[string]any
}

type lifecycleEffectsCapture struct {
	tasks []lifecycleTaskCapture
}

func (effects *lifecycleEffectsCapture) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	effects.tasks = append(effects.tasks, lifecycleTaskCapture{name: name, args: args, kwargs: kwargs})
	return nil
}

func (*lifecycleEffectsCapture) SoftDelete(context.Context, string, string, string) error {
	return nil
}

func TestLifecyclePatchPersistsPrevalidatedSortOrderAndFiltersCompletedActivity(t *testing.T) {
	const sortOrder = 42.5
	fixedNow := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	database := &databaseStub{rows: []pgx.Row{
		jsonRow(`{"id":"cycle-1","archived_at":null,"end_date":"2020-01-02T00:00:00Z"}`),
		jsonRow(`{"id":"cycle-1","start_date":null,"end_date":"2020-01-02T00:00:00-08:00"}`),
		rowFunc(func(destinations ...any) error {
			*destinations[0].(*string) = "UTC"
			*destinations[1].(*string) = "America/Los_Angeles"
			return nil
		}),
		jsonRow(`{"id":"cycle-1","sort_order":42.5,"end_date":"2020-01-02T00:00:00Z"}`),
	}}
	effects := &lifecycleEffectsCapture{}
	repository, err := NewLifecycleRepository(database, effects, "")
	if err != nil {
		t.Fatal(err)
	}
	repository.now = func() time.Time { return fixedNow }

	requested := map[string]any{"sort_order": float64(sortOrder), "name": "must be ignored"}
	activityRequested := map[string]any{"sort_order": float64(sortOrder), "name": "must be ignored"}
	_, err = repository.Patch(context.Background(), "workspace", "project-1", "cycle-1", "actor-1", true, requested, activityRequested)
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}

	if len(database.args) != 4 {
		t.Fatalf("database calls = %d, want raw lookup, serializer snapshot, timezone lookup, and update", len(database.args))
	}
	updateArgs := database.args[3]
	var persistedValue any
	for _, argument := range updateArgs {
		if value, ok := argument.(float64); ok {
			persistedValue = value
		}
		if value, ok := argument.(string); ok && value == "must be ignored" {
			t.Fatalf("completed-cycle patch persisted an unrelated field: %#v", updateArgs)
		}
	}
	if persistedValue != float64(sortOrder) {
		t.Fatalf("persisted sort_order argument = %#v, want float64(%v); update args: %#v", persistedValue, sortOrder, updateArgs)
	}

	if len(effects.tasks) != 1 {
		t.Fatalf("queued tasks = %d, want one model-activity task", len(effects.tasks))
	}
	task := effects.tasks[0]
	if task.name != "plane.bgtasks.webhook_task.model_activity" {
		t.Fatalf("queued task = %q, want model_activity", task.name)
	}
	requestedData, ok := task.kwargs["requested_data"].(map[string]any)
	if !ok {
		t.Fatalf("queued requested_data has type %T, want map[string]any", task.kwargs["requested_data"])
	}
	if !reflect.DeepEqual(requestedData, map[string]any{"sort_order": float64(sortOrder)}) {
		t.Fatalf("queued requested_data = %#v, want only sort_order=%v", requestedData, sortOrder)
	}
}
