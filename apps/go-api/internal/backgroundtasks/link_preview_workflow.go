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

const LinkPreviewWorkflowName = "plane.links.preview"
const linkPreviewTask = "plane.bgtasks.work_item_link_task.crawl_work_item_link_title"

type linkPreviewStore interface {
	Crawl(context.Context, LinkPreviewInput) (struct{}, error)
}
type LinkPreviewDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[LinkPreviewInput, struct{}]
}

func RegisterLinkPreviewWorkflow(app *selvedge.App, store linkPreviewStore, policy *retry.Policy) (*LinkPreviewDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("link preview app and store required")
	}
	if policy == nil {
		policy = retry.Exponential(5, time.Second, time.Minute, 2)
	}
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[LinkPreviewInput, struct{}]{Name: "plane.links.crawl"}
	if err = executionSelvedge.Register(activities, activity, store.Crawl, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 2 * time.Minute, RetryPolicy: policy}); err != nil {
		return nil, err
	}
	workflow := selvedge.DefineWorkflow(LinkPreviewWorkflowName, func(ctx *selvedge.WorkflowContext, in LinkPreviewInput) (struct{}, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), in, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &LinkPreviewDispatcher{app, workflow}, nil
}
func (d *LinkPreviewDispatcher) Admit(ctx context.Context, in LinkPreviewInput) (string, error) {
	return d.AdmitWithID(ctx, uuid.NewString(), in)
}

// AdmitWithID reuses the committed outbox identity after an uncertain admission.
func (d *LinkPreviewDispatcher) AdmitWithID(ctx context.Context, eventID string, in LinkPreviewInput) (string, error) {
	if d == nil || d.app == nil {
		return "", errors.New("link preview dispatcher required")
	}
	if _, err := uuid.Parse(in.ID); err != nil {
		return "", err
	}
	if in.URL == "" {
		return "", errors.New("link preview URL required")
	}
	if id, err := uuid.Parse(eventID); err != nil || id == uuid.Nil {
		return "", errors.New("invalid link preview event identifier")
	}
	return admitOrReconcile(ctx, d.app, d.workflow, "plane-link-preview:"+eventID, in)
}

type LinkPreviewTaskPublisher struct {
	fallback   TaskFallback
	dispatcher *LinkPreviewDispatcher
}

func NewLinkPreviewTaskPublisher(fallback TaskFallback, dispatcher *LinkPreviewDispatcher) (*LinkPreviewTaskPublisher, error) {
	if fallback == nil || dispatcher == nil {
		return nil, errors.New("link preview publisher dependencies required")
	}
	return &LinkPreviewTaskPublisher{fallback, dispatcher}, nil
}
func (p *LinkPreviewTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name != linkPreviewTask {
		return p.fallback.Task(ctx, name, args, kwargs)
	}
	id, rawURL := stringFromTask(kwargs["id"]), stringFromTask(kwargs["url"])
	if len(args) > 2 {
		return errors.New("link preview requires ID and URL")
	}
	if len(args) > 0 {
		id = stringFromTask(args[0])
	}
	if len(args) > 1 {
		rawURL = stringFromTask(args[1])
	}
	_, err := p.dispatcher.Admit(ctx, LinkPreviewInput{ID: id, URL: rawURL})
	return err
}

func (p *LinkPreviewTaskPublisher) SoftDelete(ctx context.Context, app, model, id string) error {
	return p.fallback.SoftDelete(ctx, app, model, id)
}
