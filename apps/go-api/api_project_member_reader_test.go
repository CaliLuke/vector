package systemapi

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type projectRosterThrottle struct{ beforeRead func() }

func (t projectRosterThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	if t.beforeRead != nil {
		t.beforeRead()
	}
	return apikey.Rate{Allowed: true}, nil
}

func TestProjectRosterHTTPUsesTheSharedAuthorizedApplication(t *testing.T) {
	actor, project := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name   string
		revoke bool
	}{{"roster", false}, {"revoked before application read", true}} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &projectMembershipLifecycleRepositoryStub{result: []any{map[string]any{"id": actor.String(), "display_name": "Alice"}}}
			throttle := projectRosterThrottle{beforeRead: func() {
				if tc.revoke {
					repository.authorizeErr = &djangoapi.HTTPError{Status: http.StatusForbidden, Body: map[string]any{"detail": "forbidden"}}
				}
			}}
			service, err := NewProjectMembershipLifecycle(repository,
				&activityAPIKeyStub{principal: apikey.Principal{UserID: actor, Timezone: "UTC"}}, throttle, &activitySessionStub{})
			require.NoError(t, err)
			ctx, err := service.(*projectMembershipLifecycleService).APIKeyAuth(t.Context(), "test-token", &security.APIKeyScheme{Name: "ApiKeyAuth"})
			require.NoError(t, err)
			metadata, body, err := service.GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembers(ctx,
				&api.GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersPayload{Slug: "workspace", ProjectID: project.String()})
			if tc.revoke {
				membershipLifecycleStatus(t, err, "403")
				assert.Nil(t, body)
				assert.Zero(t, repository.executeCalls)
				return
			}
			require.NoError(t, err)
			defer func() { require.NoError(t, body.Close()) }()
			data, err := io.ReadAll(body)
			require.NoError(t, err)
			assert.Equal(t, "application/json", metadata.ContentType)
			assert.JSONEq(t, `[{"id":"`+actor.String()+`","display_name":"Alice"}]`, string(data))
			assert.Equal(t, actor.String(), repository.command.UserID)
			assert.Equal(t, project.String(), repository.command.ProjectID)
			assert.Equal(t, "workspace", repository.command.Slug)
		})
	}
}
