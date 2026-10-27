package systemapi

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	api "github.com/CaliLuke/go-plane/milestones/gen/milestones"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type milestoneFailureApplication struct {
	*store.Repository
	failure error
}

func (a milestoneFailureApplication) ResolveScope(_ context.Context, actor, project uuid.UUID, slug, timezone string) (store.Scope, error) {
	return store.Scope{ActorID: actor, ProjectID: project, WorkspaceID: uuid.New(), WorkspaceSlug: slug, Timezone: timezone}, nil
}
func (a milestoneFailureApplication) Create(context.Context, store.WriteOptions, store.Fields) (store.MutationResult, error) {
	return store.MutationResult{}, a.failure
}

func TestMilestoneHTTPServiceMapsDomainFailuresWithoutPrivateDetails(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
	}{
		{"forbidden", store.ErrForbidden}, {"not_found", store.ErrNotFound}, {"invalid_input", store.ErrInvalidInput},
		{"invalid_input", mutationreceipts.ErrInvalidRequest}, {"conflict", store.ErrConflict},
		{"request_conflict", mutationreceipts.ErrConflict}, {"revision_conflict", store.ErrRevisionConflict},
		{"indeterminate", mutationreceipts.ErrIndeterminate}, {"internal_error", errors.New("database unavailable")},
	} {
		t.Run(tc.name+"/"+tc.failure.Error(), func(t *testing.T) {
			app := milestoneFailureApplication{failure: fmt.Errorf("private database details: %w", tc.failure)}
			keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
			service, err := NewMilestones(app, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}}, "https://vector.example.test")
			require.NoError(t, err)
			ctx, err := service.(api.Authorizer).APIKeyAuth(t.Context(), "test-key", nil)
			require.NoError(t, err)
			result, err := service.Create(ctx, &api.CreatePayload{Slug: "demo", ProjectID: uuid.NewString(), Body: &api.MilestoneCreateFields{Title: loom.NullableValue("Release")}})
			require.Nil(t, result)
			var failure *api.MilestoneError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, tc.name, failure.Name)
			require.NotContains(t, failure.Message, "private")
			require.NotContains(t, failure.Message, "database")
		})
	}
}

func TestMilestoneHTTPServiceRejectsMissingDependenciesAndInvalidIdentity(t *testing.T) {
	app := milestoneFailureApplication{failure: errors.New("unexpected application call")}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: true}}
	_, err := NewMilestones(nil, keys, throttle, "")
	require.Error(t, err)
	_, err = NewMilestones(app, nil, throttle, "")
	require.Error(t, err)
	_, err = NewMilestones(app, keys, nil, "")
	require.Error(t, err)
	service, err := NewMilestones(app, keys, throttle, "")
	require.NoError(t, err)
	_, err = service.List(t.Context(), &api.ListPayload{})
	var failure *api.MilestoneError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "unauthorized", failure.Name)
	keys.err = errors.New("private database failure")
	_, err = service.(api.Authorizer).APIKeyAuth(t.Context(), "test-key", nil)
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "internal_error", failure.Name)
	require.NotContains(t, failure.Message, "private")
	keys.err = nil
	ctx, err := service.(api.Authorizer).APIKeyAuth(t.Context(), "test-key", nil)
	require.NoError(t, err)
	_, err = service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: uuid.Nil.String()})
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "invalid_input", failure.Name)
	keys.principal.Timezone = "Invalid/Timezone"
	ctx, err = service.(api.Authorizer).APIKeyAuth(t.Context(), "test-key", nil)
	require.NoError(t, err)
	_, err = service.List(ctx, &api.ListPayload{Slug: "demo", ProjectID: uuid.NewString()})
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "invalid_input", failure.Name)
}
