package assets

import (
	"context"
	"net/http"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/google/uuid"
)

func TestSessionResourceProtocolUsesSourcePermissionClasses(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	service := &LifecycleService{assets: &Service{repository: invalidMethodRepository{}}}
	authenticated := systemapi.WithReadServiceSessionPrincipal(context.Background(), userID.String(), "UTC")
	for _, testCase := range []struct {
		name     string
		ctx      context.Context
		resource systemapi.SessionProtocolResource
		want     int
	}{
		{
			name:     "static AllowAny includes unsupported method",
			resource: systemapi.SessionProtocolResource{Method: http.MethodPatch, Pattern: "/api/assets/v2/static/{asset_id}/"},
			want:     0,
		},
		{
			name:     "public anchor exact GET is anonymous",
			resource: systemapi.SessionProtocolResource{Method: http.MethodGet, Pattern: "/api/public/assets/v2/anchor/{anchor}/"},
			want:     0,
		},
		{
			name:     "public anchor other methods require authentication",
			resource: systemapi.SessionProtocolResource{Method: http.MethodPatch, Pattern: "/api/public/assets/v2/anchor/{anchor}/"},
			want:     http.StatusUnauthorized,
		},
		{
			name:     "public anchor other methods accept session",
			ctx:      authenticated,
			resource: systemapi.SessionProtocolResource{Method: http.MethodPatch, Pattern: "/api/public/assets/v2/anchor/{anchor}/"},
			want:     0,
		},
		{
			name:     "legacy workspace key requires active workspace membership",
			ctx:      authenticated,
			resource: systemapi.SessionProtocolResource{Method: http.MethodOptions, Pattern: "/api/workspaces/file-assets/{workspace_id}/{asset_key}/", PathParameters: map[string]string{"workspace_id": workspaceID.String()}},
			want:     403,
		},
		{
			name:     "legacy workspace slug permission treats missing workspace as forbidden",
			ctx:      authenticated,
			resource: systemapi.SessionProtocolResource{Method: http.MethodOptions, Pattern: "/api/workspaces/{slug}/file-assets/", Slug: "missing"},
			want:     403,
		},
		{
			name:     "ordinary asset protocol requires session",
			resource: systemapi.SessionProtocolResource{Method: http.MethodOptions, Pattern: "/api/assets/v2/user-assets/"},
			want:     http.StatusUnauthorized,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := testCase.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			status, _ := service.AuthorizeSessionResourceProtocol(ctx, "", testCase.resource)
			if status != testCase.want {
				t.Fatalf("permission status=%d, want %d", status, testCase.want)
			}
		})
	}
}
