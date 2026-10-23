package backgroundtasks

import (
	"context"
	"errors"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
)

const WorkspaceSeedWorkflowName = "plane.workspace.seed"
const workspaceSeedTask = "plane.bgtasks.workspace_seed_task.workspace_seed"

type workspaceSeedStore interface {
	Seed(context.Context, WorkspaceSeedInput) (struct{}, error)
}
type WorkspaceSeedDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[WorkspaceSeedInput, struct{}]
}

func RegisterWorkspaceSeedWorkflow(app *selvedge.App, store workspaceSeedStore, policy *retry.Policy) (*WorkspaceSeedDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("workspace seed app and store required")
	}
	if policy == nil {
		policy = retry.Exponential(5, time.Second, time.Minute, 2)
	}
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[WorkspaceSeedInput, struct{}]{Name: "plane.workspace.seed_rows"}
	if err = executionSelvedge.Register(activities, activity, store.Seed, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: policy}); err != nil {
		return nil, err
	}
	workflow := selvedge.DefineWorkflow(WorkspaceSeedWorkflowName, func(ctx *selvedge.WorkflowContext, in WorkspaceSeedInput) (struct{}, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), in, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &WorkspaceSeedDispatcher{app, workflow}, nil
}
func (d *WorkspaceSeedDispatcher) Admit(ctx context.Context, in WorkspaceSeedInput) (string, error) {
	if d == nil || d.app == nil {
		return "", errors.New("workspace seed dispatcher required")
	}
	if _, err := uuid.Parse(in.WorkspaceID); err != nil {
		return "", err
	}
	if in.At.IsZero() {
		return "", errors.New("workspace seed timestamp required")
	}
	return admitOrReconcile(ctx, d.app, d.workflow, "plane-workspace-seed:"+in.WorkspaceID, in)
}

type WorkspaceSeedTaskPublisher struct {
	fallback   TaskFallback
	dispatcher *WorkspaceSeedDispatcher
}

func NewWorkspaceSeedTaskPublisher(fallback TaskFallback, dispatcher *WorkspaceSeedDispatcher) (*WorkspaceSeedTaskPublisher, error) {
	if fallback == nil || dispatcher == nil {
		return nil, errors.New("workspace seed publisher dependencies required")
	}
	return &WorkspaceSeedTaskPublisher{fallback, dispatcher}, nil
}
func (p *WorkspaceSeedTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name != workspaceSeedTask {
		return p.fallback.Task(ctx, name, args, kwargs)
	}
	id := stringFromTask(kwargs["workspace_id"])
	if len(args) > 1 {
		return errors.New("workspace seed requires one argument")
	}
	if len(args) == 1 {
		id = stringFromTask(args[0])
	}
	_, err := p.dispatcher.Admit(ctx, WorkspaceSeedInput{WorkspaceID: id, At: time.Now().UTC()})
	return err
}

func (p *WorkspaceSeedTaskPublisher) SoftDelete(ctx context.Context, app, model, id string) error {
	return p.fallback.SoftDelete(ctx, app, model, id)
}
