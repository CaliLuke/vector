package systemapi

import (
	"context"
	"errors"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type milestoneApplication interface {
	ResolveScope(context.Context, uuid.UUID, uuid.UUID, string, string) (store.Scope, error)
	List(context.Context, store.Scope, djangoapi.Query) (store.Page[store.Record], error)
	Retrieve(context.Context, store.Scope, uuid.UUID, bool) (store.Record, error)
	ListWorkItems(context.Context, store.Scope, uuid.UUID, djangoapi.Query) (store.Page[store.Link], error)
	Create(context.Context, store.WriteOptions, store.Fields) (store.MutationResult, error)
	Update(context.Context, store.WriteOptions, uuid.UUID, store.Fields) (store.MutationResult, error)
	Delete(context.Context, store.WriteOptions, uuid.UUID) (store.MutationResult, error)
	ManageWorkItems(context.Context, store.WriteOptions, uuid.UUID, []uuid.UUID, []uuid.UUID) (store.MutationResult, error)
}

type milestoneService struct {
	application milestoneApplication
	keys        activityReadsAPIKeys
	throttle    activityReadsThrottle
	origin      string
}
type milestoneActorKey struct{}
type milestoneActor struct {
	apikey.Principal
	credential string
}

func NewMilestones(application milestoneApplication, keys activityReadsAPIKeys, throttle activityReadsThrottle, origin string) (api.Service, error) {
	if application == nil || keys == nil || throttle == nil {
		return nil, errors.New("milestone application, primary API-key validator and throttle are required")
	}
	return &milestoneService{application, keys, throttle, origin}, nil
}

func (s *milestoneService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.keys.Validate(ctx, key)
	if err != nil {
		return ctx, milestoneAuthError(key, err)
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Authenticated = true
	}
	return context.WithValue(ctx, milestoneActorKey{}, milestoneActor{principal, key}), nil
}
func milestoneAuthError(key string, err error) error {
	if key == "" {
		return milestoneFailure("unauthorized", "Authentication credentials were not provided.")
	}
	if errors.Is(err, apikey.ErrInvalid) {
		return milestoneFailure("forbidden", "Given API token is not valid.")
	}
	return milestoneFailure("internal_error", "Authentication could not be completed.")
}
func milestoneFailure(name, message string) error {
	return &api.MilestoneError{Name: name, Message: message}
}

func (s *milestoneService) scope(ctx context.Context, slug, project string) (store.Scope, error) {
	actor, ok := ctx.Value(milestoneActorKey{}).(milestoneActor)
	if !ok {
		return store.Scope{}, milestoneFailure("unauthorized", "Authentication credentials were not provided.")
	}
	rate, err := s.throttle.Allow(ctx, actor.credential)
	if err != nil {
		return store.Scope{}, milestoneError(err)
	}
	if effects := apikey.EffectsFromContext(ctx); effects != nil {
		effects.Rate = &rate
	}
	if !rate.Allowed {
		return store.Scope{}, milestoneFailure("rate_limited", "Request was throttled.")
	}
	if err = djangosession.ValidateUserTimezone(actor.Timezone); err != nil {
		return store.Scope{}, milestoneFailure("invalid_input", "Invalid account timezone.")
	}
	id, err := milestoneUUID(project)
	if err != nil {
		return store.Scope{}, err
	}
	scope, err := s.application.ResolveScope(ctx, actor.UserID, id, slug, actor.Timezone)
	return scope, milestoneError(err)
}

func milestoneUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, milestoneFailure("invalid_input", "Invalid resource identifier.")
	}
	return id, nil
}

func (s *milestoneService) writeOptions(ctx context.Context, slug, project string, request, revision *string) (store.WriteOptions, error) {
	scope, err := s.scope(ctx, slug, project)
	if err != nil {
		return store.WriteOptions{}, err
	}
	opts := store.WriteOptions{Scope: scope, ExpectedRevision: revision, Origin: s.origin}
	if request != nil {
		id, parseErr := milestoneUUID(*request)
		if parseErr != nil {
			return store.WriteOptions{}, parseErr
		}
		opts.RequestID = &id
	}
	actor := ctx.Value(milestoneActorKey{}).(milestoneActor)
	opts.Revalidate = func(ctx context.Context) error {
		current, authErr := s.keys.Validate(ctx, actor.credential)
		if authErr != nil {
			return milestoneAuthError(actor.credential, authErr)
		}
		if current.UserID != scope.ActorID {
			return milestoneFailure("forbidden", "Authorized identity changed.")
		}
		return nil
	}
	return opts, nil
}

func milestoneError(err error) error {
	if err == nil {
		return nil
	}
	var safe *api.MilestoneError
	if errors.As(err, &safe) {
		return safe
	}
	for _, failure := range []struct {
		target        error
		name, message string
	}{
		{store.ErrForbidden, "forbidden", "Milestone access denied."},
		{store.ErrNotFound, "not_found", "Milestone resource not found."},
		{store.ErrInvalidInput, "invalid_input", "Invalid milestone input."},
		{mutationreceipts.ErrInvalidRequest, "invalid_input", "Invalid mutation request."},
		{store.ErrConflict, "conflict", "Milestone conflicts with an existing resource."},
		{store.ErrRevisionConflict, "revision_conflict", "Milestone revision changed."},
		{mutationreceipts.ErrConflict, "request_conflict", "Request input changed."},
		{mutationreceipts.ErrIndeterminate, "indeterminate", "Commit outcome is unknown; reconcile before retrying."},
	} {
		if errors.Is(err, failure.target) {
			return milestoneFailure(failure.name, failure.message)
		}
	}
	return milestoneFailure("internal_error", "Milestone operation could not be completed.")
}
