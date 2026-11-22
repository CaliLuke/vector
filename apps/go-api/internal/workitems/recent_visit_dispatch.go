package workitems

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	recentVisitConcurrency = 8
	recentVisitTimeout     = 2 * time.Second
	recentVisitLogInterval = 10 * time.Second
)

type recentVisitKey struct {
	slug, project, actor, entity, id string
}

// Discovery readers copy Repository, so the dispatcher is shared with the
// original reader for concurrency and coalescing across read paths.
type recentVisitDispatcher struct {
	mu       sync.Mutex
	active   map[recentVisitKey]struct{}
	warnNext time.Time
}

func newRecentVisitDispatcher() *recentVisitDispatcher {
	return &recentVisitDispatcher{active: make(map[recentVisitKey]struct{})}
}

func (r *Repository) recordRecentVisit(ctx context.Context, q Request, entity, id string) {
	dispatch := r.recentVisitDispatch
	if dispatch == nil {
		return
	}
	key := recentVisitKey{q.Slug, q.ProjectID, q.ActorID, entity, id}
	dispatch.submit(ctx, key, func(ctx context.Context) error {
		return r.recentVisit(ctx, Request{Slug: key.slug, ProjectID: key.project, ActorID: key.actor}, key.entity, key.id)
	})
}

func (d *recentVisitDispatcher) reserve(key recentVisitKey, now time.Time) (accepted, warnCapacity bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, present := d.active[key]; present {
		return false, false
	}
	if len(d.active) >= recentVisitConcurrency {
		warn := !now.Before(d.warnNext)
		if warn {
			d.warnNext = now.Add(recentVisitLogInterval)
		}
		return false, warn
	}
	d.active[key] = struct{}{}
	return true, false
}

func (d *recentVisitDispatcher) submit(ctx context.Context, key recentVisitKey, admit func(context.Context) error) {
	accepted, warnCapacity := d.reserve(key, time.Now())
	if warnCapacity {
		logRecentVisitFailure(ctx, key.entity, "capacity")
	}
	if !accepted {
		return
	}
	go func() {
		admissionContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), recentVisitTimeout)
		defer cancel()
		if err := admit(admissionContext); err != nil {
			logRecentVisitFailure(admissionContext, key.entity, recentVisitFailureKind(err))
		}
		d.mu.Lock()
		delete(d.active, key)
		d.mu.Unlock()
	}()
}

func recentVisitFailureKind(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "admission"
	}
}

func logRecentVisitFailure(ctx context.Context, entity, kind string) {
	slog.WarnContext(ctx, "work item recent visit was not recorded", "visit.entity", entity, "error.kind", kind)
}
