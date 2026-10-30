//go:build integration

package projectmembershipreads

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

func TestMalformedMemberWritesPreserveStoredRows(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, bulkLifecycleSchema)
	_, err := db.Exec(ctx, `UPDATE project_members SET is_active=true,role=20,sort_order=12.5;
 UPDATE workspace_members SET role=20 WHERE member_id='00000000-0000-4000-8000-00000000000a'`)
	require.NoError(t, err)
	publisher := &bulkNotificationRecorder{}
	repository, err := NewLifecycleRepository(db, publisher, "http://localhost:8000")
	require.NoError(t, err)
	const snapshot = `SELECT jsonb_build_object(
 'members',(SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM project_members m),
 'properties',(SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM project_user_properties p))::text`
	var before string
	require.NoError(t, db.QueryRow(ctx, snapshot).Scan(&before))
	for _, test := range []struct{ method, body string }{
		{"PATCH", `null`}, {"PATCH", `true`}, {"PATCH", `5`}, {"PATCH", `["role"]`},
		{"PATCH", `{"role":"invalid","comment":"must not persist"}`}, {"PATCH", `{"role":null}`},
		{"POST", `null`}, {"POST", `[]`}, {"POST", `{"members":true}`},
		{"POST", `{"members":[true]}`}, {"POST", `{"members":[{"member_id":[]}]}`},
	} {
		t.Run(test.method+test.body, func(t *testing.T) {
			operation := "legacy-create-members"
			if test.method == "PATCH" {
				operation = "legacy-patch-members"
			}
			_, writeErr := repository.ExecuteLifecycle(ctx, LifecycleCommand{
				Operation: operation, Method: test.method, Slug: "bulk", ProjectID: "00000000-0000-4000-8000-000000000100",
				UserID: "00000000-0000-4000-8000-00000000000a", MemberID: "00000000-0000-4000-8000-000000000200",
				ContentType: "application/json", Body: strings.NewReader(test.body),
			})
			var failure *djangoapi.HTTPError
			require.ErrorAs(t, writeErr, &failure)
			require.Equal(t, 400, failure.Status)
			var after string
			require.NoError(t, db.QueryRow(ctx, snapshot).Scan(&after))
			require.Equal(t, before, after)
			require.Empty(t, publisher.memberIDs)
		})
	}
}
