//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMCPPlanningDeleteReplayRequiresCurrentDeleteAuthority(t *testing.T) {
	runPlanningMatrix(t, func(t *testing.T, f milestoneFixture, call catalogueCall, kind string) {
		_, created := createPlanningReceipt(t, call, kind)
		id := created[kind+"_id"].(string)
		ownerColumn := "created_by_id"
		if kind == "cycle" {
			ownerColumn = "owned_by_id"
		}
		otherOwner := uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,
 to_jsonb(u)||jsonb_build_object('id',$1::uuid,'username',$1::text,'email',$1::text||'@example.test','token',$1::text))).*
 FROM users u WHERE id=$2`, otherOwner, testsupport.LabelActor)
		require.NoError(t, err)
		_, err = f.pool.Exec(t.Context(), "UPDATE "+pgx.Identifier{kind + "s"}.Sanitize()+" SET "+pgx.Identifier{ownerColumn}.Sanitize()+"=$2 WHERE id=$1", id, otherOwner)
		require.NoError(t, err)
		input := map[string]any{"action": "delete", kind + "_id": id, "request_id": uuid.NewString()}
		deleted := milestoneObject(t, call(kind, input))
		require.Equal(t, false, deleted["replayed"])
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=15 WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, testsupport.LabelProject)
		require.NoError(t, err)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
			defer cancel()
			_, restoreErr := f.pool.Exec(ctx, `UPDATE project_members SET role=20 WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, testsupport.LabelProject)
			require.NoError(t, restoreErr)
		})
		before := planningMutationSnapshot(t, f)
		milestoneFailure(t, call(kind, input), "forbidden")
		require.Equal(t, before, planningMutationSnapshot(t, f))
		_, err = f.pool.Exec(t.Context(), `UPDATE project_members SET role=20 WHERE member_id=$1 AND project_id=$2`, testsupport.LabelActor, testsupport.LabelProject)
		require.NoError(t, err)
		require.Equal(t, true, milestoneObject(t, call(kind, input))["replayed"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
		// Removing a cycle owner physically cascades its cycle even while the
		// project and deleting administrator remain active.
		_, err = f.pool.Exec(t.Context(), `DELETE FROM users WHERE id=$1`, otherOwner)
		require.NoError(t, err)
		before = planningMutationSnapshot(t, f)
		require.Equal(t, true, milestoneObject(t, call(kind, input))["replayed"])
		require.Equal(t, before, planningMutationSnapshot(t, f))
	})
}
