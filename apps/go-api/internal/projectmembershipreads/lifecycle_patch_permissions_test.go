package projectmembershipreads

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyPatchKeepsCurrentActorAsUpdatedBy(t *testing.T) {
	db := &lifecycleReviewDB{}
	_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(t.Context(), LifecycleCommand{
		MemberID: "membership", UserID: "actor", ContentType: "application/json",
		Body: strings.NewReader(`{"updated_by":"` + lifecycleReviewUser + `"}`),
	})
	require.NoError(t, err)
	require.Len(t, db.writes, 1)
	require.Equal(t, []any{"membership", "actor"}, db.writes[0])
	require.Contains(t, db.writeQueries[0], "updated_by_id=$2")
}

func TestLegacyPatchPermissionAndParsingPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, body, message                                                  string
		requesterProject, requesterWorkspace, targetProject, targetWorkspace int
		self                                                                 bool
		status                                                               int
	}{
		{
			name: "own membership guard precedes malformed JSON", body: "{", self: true,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 15, targetWorkspace: 15,
			status: 400, message: "You cannot update your own role",
		},
		{
			name: "role permission precedes role coercion and status permission", body: `{"role":"invalid","is_active":false}`,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 403, message: "You do not have permission to update roles",
		},
		{
			name: "equal target role precedes invalid submitted role", body: `{"role":"invalid"}`,
			requesterProject: 20, requesterWorkspace: 15, targetProject: 20, targetWorkspace: 15,
			status: 403, message: "You cannot update the role of a member with a role equal to or higher than your own",
		},
		{
			name: "submitted role cannot equal requester", body: `{"role":20}`,
			requesterProject: 20, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 403, message: "You cannot assign a role equal to or higher than your own",
		},
		{
			name: "workspace guest cap still applies to administrators", body: `{"role":15}`,
			requesterProject: 20, requesterWorkspace: 20, targetProject: 5, targetWorkspace: 5,
			status: 400, message: "You cannot add a user with role higher than the workspace role",
		},
		{
			name: "status permission precedes invalid status value", body: `{"is_active":{}}`,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 403, message: "You do not have permission to update member status",
		},
		{
			name: "equal target status precedes invalid status value", body: `{"is_active":{}}`,
			requesterProject: 20, requesterWorkspace: 15, targetProject: 20, targetWorkspace: 15,
			status: 403, message: "You cannot update the status of a member with a role equal to or higher than your own",
		},
		{
			name: "nonobject role permission precedes shape error", body: `["role","is_active"]`,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 403, message: "You do not have permission to update roles",
		},
		{
			name: "nonobject status permission precedes shape error", body: `["is_active"]`,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 403, message: "You do not have permission to update member status",
		},
		{
			name: "permitted nonobject role retains view error", body: `["role"]`,
			requesterProject: 20, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
			status: 500, message: "Something went wrong please try again later",
		},
		{
			name: "workspace administrator can change own role", body: `{"role":5}`, self: true,
			requesterProject: 20, requesterWorkspace: 20, targetProject: 20, targetWorkspace: 20,
		},
		{
			name: "empty patch retains ordinary member access", body: `{}`,
			requesterProject: 15, requesterWorkspace: 15, targetProject: 5, targetWorkspace: 15,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requester := "requester"
			if tc.self {
				requester = lifecycleReviewUser
			}
			db := &lifecycleReviewDB{
				projectRoles:   map[string]int{lifecycleReviewUser: tc.targetProject},
				workspaceRoles: map[string]int{lifecycleReviewUser: tc.targetWorkspace},
			}
			db.projectRoles[requester] = tc.requesterProject
			db.workspaceRoles[requester] = tc.requesterWorkspace
			_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(t.Context(), LifecycleCommand{
				UserID: requester, Body: strings.NewReader(tc.body), ContentType: "application/json",
			})
			if tc.status != 0 {
				lifecycleReviewError(t, err, tc.status, map[string]any{"error": tc.message})
				require.Empty(t, db.writes)
				return
			}
			require.NoError(t, err)
			require.Len(t, db.writes, 1)
			if tc.self {
				require.Len(t, db.writes[0], 3)
				require.Equal(t, 5, db.writes[0][2])
			} else {
				require.Len(t, db.writes[0], 2)
			}
		})
	}
}
