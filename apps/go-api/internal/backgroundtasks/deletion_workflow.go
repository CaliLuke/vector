package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/CaliLuke/go-plane/internal/deletion"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

const (
	SoftDeletionWorkflowName = "plane.deletion.soft_cascade"
	HardDeletionWorkflowName = "plane.deletion.hard_delete"
	deletionScheduleName     = "plane.beat.hard-delete"
	deletionTaskName         = "plane.bgtasks.deletion_task.soft_delete_related_objects"
)

type DeletionDispatcher struct {
	policy   deletion.Policy
	app      *selvedge.App
	soft     *selvedge.WorkflowFunc[DeletionInput, struct{}]
	hard     *selvedge.WorkflowFunc[time.Time, struct{}]
	schedule *selvedge.WorkflowFunc[struct{}, struct{}]
}

func HardDeleteDaysFromEnv() (int, error) {
	value := os.Getenv("HARD_DELETE_AFTER_DAYS")
	if value == "" {
		return 60, nil
	}
	days, err := strconv.Atoi(value)
	if err != nil || days < 1 || days > 36500 {
		return 0, errors.New("HARD_DELETE_AFTER_DAYS must be between 1 and 36500")
	}
	return days, nil
}
func RegisterDeletionWorkflows(app *selvedge.App, store deletionStore, days int, policy *retry.Policy) (*DeletionDispatcher, error) {
	if app == nil || store == nil || days < 1 || days > 36500 {
		return nil, errors.New("deletion app, store and valid retention are required")
	}
	if policy == nil {
		policy = retry.Exponential(12, time.Second, time.Minute, 2)
	}
	copyPolicy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	options := selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 5 * time.Minute, RetryPolicy: &copyPolicy}
	mark := execution.Activity[DeletionInput, bool]{Name: "plane.deletion.mark_root"}
	plan := execution.Activity[deletionPlanInput, deletionPlan]{Name: "plane.deletion.plan_related"}
	apply := execution.Activity[deletionPlan, []DeletionInput]{Name: "plane.deletion.apply_related"}
	hardBatch := execution.Activity[hardDeletionInput, int]{Name: "plane.deletion.hard_batch"}
	if err := executionSelvedge.Register(activities, mark, store.MarkRoot, options); err != nil {
		return nil, err
	}
	if err := executionSelvedge.Register(activities, plan, store.Plan, options); err != nil {
		return nil, err
	}
	if err := executionSelvedge.Register(activities, apply, store.Apply, options); err != nil {
		return nil, err
	}
	if err := executionSelvedge.Register(activities, hardBatch, store.HardDelete, options); err != nil {
		return nil, err
	}
	soft := selvedge.DefineWorkflow(SoftDeletionWorkflowName, func(ctx *selvedge.WorkflowContext, in DeletionInput) (struct{}, error) {
		model, err := store.Policy().Lookup(in.App, in.Model)
		if err != nil {
			return struct{}{}, err
		}
		exists, err := mark.Execute(ctx.Context(), activities.Scope(ctx), in, execution.ActivityOptions{})
		if err != nil || !exists {
			return struct{}{}, err
		}
		for index := range deletionRelations(store.Policy(), model) {
			after := ""
			for {
				batch, err := plan.Execute(ctx.Context(), activities.Scope(ctx), deletionPlanInput{Root: in, Relation: index, AfterID: after}, execution.ActivityOptions{})
				if err != nil {
					return struct{}{}, err
				}
				children, err := apply.Execute(ctx.Context(), activities.Scope(ctx), batch, execution.ActivityOptions{})
				if err != nil {
					return struct{}{}, err
				}
				for _, childInput := range children {
					child, err := selvedge.StartChild[struct{}](ctx, SoftDeletionWorkflowName, childInput, selvedge.WithChildID(deletionInstanceID(childInput)))
					if err != nil {
						return struct{}{}, err
					}
					if _, err = child.Get(ctx); err != nil {
						return struct{}{}, err
					}
				}
				if !batch.More {
					break
				}
				after = batch.IDs[len(batch.IDs)-1]
			}
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, soft)
	hard := selvedge.DefineWorkflow(HardDeletionWorkflowName, func(ctx *selvedge.WorkflowContext, at time.Time) (struct{}, error) {
		if at.IsZero() {
			return struct{}{}, errors.New("hard deletion schedule time is required")
		}
		cutoff := at.UTC().Add(-time.Duration(days) * 24 * time.Hour)
		for index, m := range store.Policy().Models {
			if !m.SoftDelete {
				continue
			}
			for {
				count, err := hardBatch.Execute(ctx.Context(), activities.Scope(ctx), hardDeletionInput{Cutoff: cutoff, Model: index}, execution.ActivityOptions{})
				if err != nil {
					return struct{}{}, err
				}
				if count < deletionBatchSize {
					break
				}
			}
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, hard)
	schedule := selvedge.DefineWorkflow(deletionScheduleName, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
		for {
			now, err := selvedge.Now(ctx)
			if err != nil {
				return struct{}{}, err
			}
			next := nextDailyUTC(now, 0, 0)
			if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("hard-delete:"+next.Format(time.RFC3339))); err != nil {
				return struct{}{}, err
			}
			for attempt := 0; ; attempt++ {
				child, err := selvedge.StartChild[struct{}](ctx, HardDeletionWorkflowName, next, selvedge.WithChildID(fmt.Sprintf("plane-hard-delete:%s:%d", next.Format(time.RFC3339), attempt)))
				if err != nil {
					return struct{}{}, err
				}
				if _, err = child.Get(ctx); err == nil {
					break
				}
				if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("hard-delete-retry:%s:%d", next.Format(time.RFC3339), attempt))); err != nil {
					return struct{}{}, err
				}
			}
		}
	})
	selvedge.RegisterWorkflow(app, schedule)
	return &DeletionDispatcher{app: app, soft: soft, hard: hard, schedule: schedule, policy: store.Policy()}, nil
}
func deletionInstanceID(in DeletionInput) string {
	return "plane-soft-delete:" + in.EventID + ":" + in.App + ":" + in.Model + ":" + in.ID
}
func (d *DeletionDispatcher) Admit(ctx context.Context, in DeletionInput) (string, error) {
	if d == nil || d.app == nil {
		return "", errors.New("deletion dispatcher is not configured")
	}
	if _, err := uuid.Parse(in.EventID); err != nil {
		return "", err
	}
	model, err := d.policy.Lookup(in.App, in.Model)
	if err != nil {
		return "", err
	}
	if _, err := model.ParseID(in.ID); err != nil {
		return "", err
	}
	return admitOrReconcile(ctx, d.app, d.soft, deletionInstanceID(in), in)
}
func (d *DeletionDispatcher) AdmitHardDelete(ctx context.Context, at time.Time) (string, error) {
	if d == nil || d.app == nil || at.IsZero() {
		return "", errors.New("hard deletion dispatcher and time are required")
	}
	return admitOrReconcile(ctx, d.app, d.hard, "plane-hard-delete:"+at.UTC().Format(time.RFC3339Nano), at.UTC())
}
func (d *DeletionDispatcher) StartBeatSchedule(ctx context.Context) (string, error) {
	if d == nil || d.app == nil {
		return "", errors.New("deletion dispatcher is not configured")
	}
	return admitOrReconcile(ctx, d.app, d.schedule, "plane-beat:hard-delete", struct{}{})
}

type DeletionTaskPublisher struct {
	fallback   TaskFallback
	dispatcher *DeletionDispatcher
}

func NewDeletionTaskPublisher(fallback TaskFallback, dispatcher *DeletionDispatcher) (*DeletionTaskPublisher, error) {
	if fallback == nil || dispatcher == nil {
		return nil, errors.New("deletion fallback and dispatcher are required")
	}
	return &DeletionTaskPublisher{fallback, dispatcher}, nil
}
func (p *DeletionTaskPublisher) SoftDelete(ctx context.Context, app, model, id string) error {
	_, err := p.dispatcher.Admit(ctx, DeletionInput{EventID: uuid.NewString(), App: app, Model: model, ID: id})
	return err
}
func (p *DeletionTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name != deletionTaskName {
		return p.fallback.Task(ctx, name, args, kwargs)
	}
	if len(args) != 0 && len(args) != 3 {
		return errors.New("soft deletion requires app, model and ID")
	}
	app, model, id := stringFromTask(kwargs["app_label"]), stringFromTask(kwargs["model_name"]), stringFromTask(kwargs["instance_pk"])
	if len(args) == 3 {
		app, model, id = stringFromTask(args[0]), stringFromTask(args[1]), stringFromTask(args[2])
	}
	return p.SoftDelete(ctx, app, model, id)
}
