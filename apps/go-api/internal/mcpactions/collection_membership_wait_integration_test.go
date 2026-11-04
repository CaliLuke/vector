//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestCollectionEditorPermissionIsFreshAfterMembershipRowWait(t *testing.T) {
	f := templateFixture(t)
	owner, _ := customerReferenceTarget(t, f, "USER")
	repo := pagereads.NewPageAPIRepository(f.pool)
	cmd := pagereads.PageAPICommand{PageAPIQuery: pagereads.PageAPIQuery{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one", Tool: "collection", Action: "create"}, Fields: map[string]any{"name": "Original", "access": 1}}
	created, err := repo.MutatePageAPI(t.Context(), cmd)
	require.NoError(t, err)
	cmd.CollectionID = uuid.MustParse(created.ID)
	cmd.Action = "update"
	cmd.Fields = map[string]any{"name": "Accepted edit"}
	_, err = f.pool.Exec(t.Context(), `UPDATE page_collections SET owned_by_id=$2 WHERE id=$1`, cmd.CollectionID, owner)
	require.NoError(t, err)
	request := uuid.New()
	cmd.RequestID = &request
	for _, phase := range []string{"new write", "receipt replay"} {
		t.Run(phase, func(t *testing.T) {
			if phase == "receipt replay" {
				accepted, acceptErr := repo.MutatePageAPI(t.Context(), cmd)
				require.NoError(t, acceptErr)
				require.False(t, *accepted.Replayed)
			}
			var beforeName string
			var beforeEvents, beforeReceipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name,(SELECT count(*) FROM page_api_events),(SELECT count(*) FROM mutation_receipts) FROM page_collections WHERE id=$1`, cmd.CollectionID).Scan(&beforeName, &beforeEvents, &beforeReceipts))
			trigger := `CREATE OR REPLACE FUNCTION hold_collection_member_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(720010);RETURN NEW;END $$;
CREATE TRIGGER hold_collection_member_change AFTER UPDATE ON page_collection_members FOR EACH ROW EXECUTE FUNCTION hold_collection_member_change()`
			native, write := contendWithNativeCatalogueWriter(t, f, trigger,
				func(ctx context.Context) (pgconn.CommandTag, error) {
					return f.pool.Exec(ctx, `UPDATE page_collection_members SET access=0 WHERE collection_id=$1 AND member_id=$2`, cmd.CollectionID, cmd.ActorID)
				},
				func(ctx context.Context) (projectmutations.Result, error) { return repo.MutatePageAPI(ctx, cmd) },
			)
			_, err = f.pool.Exec(t.Context(), `DROP TRIGGER hold_collection_member_change ON page_collection_members`)
			require.NoError(t, err)
			_, err = f.pool.Exec(t.Context(), `UPDATE page_collection_members SET access=2 WHERE collection_id=$1 AND member_id=$2`, cmd.CollectionID, cmd.ActorID)
			require.NoError(t, err)
			require.NoError(t, native.err)
			require.EqualValues(t, 1, native.value.RowsAffected())
			require.ErrorIs(t, write.err, projectmutations.ErrForbidden)
			var name string
			var events, receipts int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name,(SELECT count(*) FROM page_api_events),(SELECT count(*) FROM mutation_receipts) FROM page_collections WHERE id=$1`, cmd.CollectionID).Scan(&name, &events, &receipts))
			require.Equal(t, beforeName, name)
			require.Equal(t, beforeEvents, events)
			require.Equal(t, beforeReceipts, receipts)
		})
	}
}
