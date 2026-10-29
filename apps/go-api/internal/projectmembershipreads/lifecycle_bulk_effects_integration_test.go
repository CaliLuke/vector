//go:build integration

package projectmembershipreads

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/stretchr/testify/require"
)

var errBulkNotification = errors.New("notification queue unavailable")

type bulkNotificationRecorder struct {
	fail      bool
	memberIDs []string
}

func (p *bulkNotificationRecorder) Task(_ context.Context, _ string, args []any, _ map[string]any) error {
	if p.fail {
		return errBulkNotification
	}
	p.memberIDs = append(p.memberIDs, args[1].(string))
	return nil
}

func TestLifecycleBulkWritesKeepIndependentCommitBoundaries(t *testing.T) {
	ctx, db := postgresfixture.OpenSQL(t, "")
	for index, tc := range []struct {
		name, constraint                                         string
		failNotification                                         bool
		wantRole, wantMembers, wantProperties, wantNotifications int
	}{
		{"update failure", "ALTER TABLE project_members ADD CHECK(role <> 15) NOT VALID", false, 5, 1, 0, 0},
		{"insert failure", "ALTER TABLE project_members ADD CHECK(member_id <> '00000000-0000-4000-8000-00000000000b')", false, 15, 1, 0, 0},
		{"property failure", "ALTER TABLE project_user_properties ADD CHECK(user_id <> '00000000-0000-4000-8000-00000000000b')", false, 15, 3, 0, 0},
		{"notification failure", "", true, 15, 3, 3, 0},
		{"success", "", false, 15, 3, 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(ctx, fmt.Sprintf("CREATE SCHEMA bulk_%d; SET search_path TO bulk_%d", index, index))
			require.NoError(t, err)
			_, err = db.Exec(ctx, bulkLifecycleSchema)
			require.NoError(t, err)
			if tc.constraint != "" {
				_, err = db.Exec(ctx, tc.constraint)
				require.NoError(t, err)
			}
			publisher := &bulkNotificationRecorder{fail: tc.failNotification}
			repository, err := NewLifecycleRepository(db, publisher, "http://localhost:8000")
			require.NoError(t, err)
			value, err := repository.lifecycleBulkCreate(ctx, LifecycleCommand{
				Slug: "bulk", ProjectID: "00000000-0000-4000-8000-000000000100",
				UserID: "00000000-0000-4000-8000-00000000000a", ContentType: "application/json",
				Body: strings.NewReader(`{"members":[{"member_id":"00000000-0000-4000-8000-00000000000a","role":15},{"member_id":"00000000-0000-4000-8000-00000000000c","role":15},{"member_id":"00000000-0000-4000-8000-00000000000b","role":15}]}`),
			})
			switch {
			case tc.constraint != "":
				lifecycleReviewError(t, err, 400, map[string]any{"error": "The payload is not valid"})
			case tc.failNotification:
				require.ErrorIs(t, err, errBulkNotification)
			default:
				require.NoError(t, err)
				rows := value.([]any)
				require.Len(t, rows, 3)
				for _, row := range rows {
					require.Contains(t, publisher.memberIDs, row.(map[string]any)["id"])
				}
			}
			var role, members, properties int
			var active bool
			require.NoError(t, db.QueryRow(ctx, "SELECT role,is_active FROM project_members WHERE member_id=$1", bulkIdentityUser).Scan(&role, &active))
			require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM project_members").Scan(&members))
			require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM project_user_properties").Scan(&properties))
			require.Equal(t, tc.wantRole, role, "earlier role update must retain its commit boundary")
			require.Equal(t, tc.wantRole == 15, active, "successful update reactivates the existing membership")
			require.Equal(t, tc.wantMembers, members, "membership insert must remain atomic")
			require.Equal(t, tc.wantProperties, properties, "property insert must remain atomic")
			require.Len(t, publisher.memberIDs, tc.wantNotifications)
		})
	}
}

const bulkLifecycleSchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY, slug text);
CREATE TABLE projects(id uuid PRIMARY KEY, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid, member_id uuid, role smallint, is_active boolean, deleted_at timestamptz);
CREATE TABLE project_members(
 id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, member_id uuid, role smallint,
 comment text, view_props jsonb, default_props jsonb, preferences jsonb, sort_order double precision,
 is_active boolean, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, UNIQUE(project_id,member_id)
);
CREATE TABLE project_user_properties(
 id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, user_id uuid,
 filters jsonb, display_filters jsonb, display_properties jsonb, rich_filters jsonb, preferences jsonb,
 sort_order double precision, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, UNIQUE(project_id,user_id)
);
INSERT INTO workspaces VALUES('00000000-0000-4000-8000-000000000010','bulk');
INSERT INTO projects VALUES('00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000010',NULL);
INSERT INTO workspace_members VALUES
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-00000000000a',15,true,NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-00000000000c',15,true,NULL),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-00000000000b',15,true,NULL);
INSERT INTO project_members(id,project_id,workspace_id,member_id,role,is_active,created_at) VALUES
 ('00000000-0000-4000-8000-000000000200','00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-00000000000a',5,false,now());
`
