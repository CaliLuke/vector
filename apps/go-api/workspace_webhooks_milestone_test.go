package systemapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	store "github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	api "github.com/CaliLuke/go-plane/workspace_webhooks/gen/workspace_webhooks"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type milestoneWebhookRepository struct {
	existingWebhookRepository
	patch store.PatchParams
}

func (r *milestoneWebhookRepository) Update(_ context.Context, p store.PatchParams) (json.RawMessage, error) {
	r.patch = p
	return json.Marshal(map[string]any{"milestone": p.Milestone})
}

func TestWorkspaceWebhookMilestonePatchDistinguishesOmittedAndFalse(t *testing.T) {
	for _, tc := range []struct {
		name, body     string
		present, value bool
	}{
		{"subscribe", `{"milestone":true}`, true, true},
		{"unsubscribe", `{"milestone":false}`, true, false},
		{"unchanged", `{}`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &milestoneWebhookRepository{}
			service, err := NewWorkspaceWebhooks(repo, unusedWebhookSessions{}, unusedWebhookPublisher{})
			require.NoError(t, err)
			ctx := context.WithValue(t.Context(), workspaceWebhooksActor{}, uuid.MustParse("00000000-0000-4000-8000-000000034201"))
			metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Content-Type"}, nil)
			require.NoError(t, err)
			var result *api.Webhook
			handler := loomhttp.RequestMetadataMiddleware(metadata)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				result, err = service.(*workspaceWebhooksService).PatchDetail(r.Context(), &api.PatchDetailPayload{Slug: "milestones", Pk: "00000000-0000-4000-8000-000000034202"}, r.Body)
			}))
			request := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(tc.body)).WithContext(ctx)
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(httptest.NewRecorder(), request)
			require.NoError(t, err)
			require.Equal(t, tc.present, repo.patch.MilestonePresent)
			require.Equal(t, tc.value, repo.patch.Milestone)
			require.Equal(t, tc.value, result.Milestone)
		})
	}
}
