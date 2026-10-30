//go:build integration

package projectmembershipreads

import (
	"net/url"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestLegacyMemberNonfiniteOrderDoesNotChangeMembership(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, bulkLifecycleSchema)
	_, err := db.Exec(ctx, `UPDATE project_members SET is_active=true,role=20,sort_order=12.5;
 UPDATE workspace_members SET role=20 WHERE member_id='00000000-0000-4000-8000-00000000000a'`)
	require.NoError(t, err)
	publisher := &bulkNotificationRecorder{}
	repo, err := NewLifecycleRepository(db, publisher, "http://localhost:8000")
	require.NoError(t, err)
	var before string
	require.NoError(t, db.QueryRow(ctx, `SELECT to_jsonb(pm)::text FROM project_members pm`).Scan(&before))
	for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
		for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e999"} {
			t.Run(contentType+"/"+value, func(t *testing.T) {
				body := `{"sort_order":"` + value + `","comment":"must not persist"}`
				if contentType == "application/x-www-form-urlencoded" {
					body = url.Values{"sort_order": {value}, "comment": {"must not persist"}}.Encode()
				}
				_, patchErr := repo.ExecuteLifecycle(ctx, LifecycleCommand{
					Operation: "legacy-patch-members", Method: "PATCH", Slug: "bulk", ProjectID: "00000000-0000-4000-8000-000000000100",
					UserID: "00000000-0000-4000-8000-00000000000a", MemberID: "00000000-0000-4000-8000-000000000200",
					ContentType: contentType, Body: strings.NewReader(body),
				})
				var failure *djangoapi.HTTPError
				require.ErrorAs(t, patchErr, &failure)
				require.Equal(t, 400, failure.Status)
				require.Equal(t, []string{"A valid number is required."}, failure.Body["sort_order"])
				var after string
				require.NoError(t, db.QueryRow(ctx, `SELECT to_jsonb(pm)::text FROM project_members pm`).Scan(&after))
				require.Equal(t, before, after)
				require.Empty(t, publisher.memberIDs)
			})
		}
	}
}
