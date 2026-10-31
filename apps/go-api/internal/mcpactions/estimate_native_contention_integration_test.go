//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const nativePointPauseTrigger = `CREATE FUNCTION pause_native_point_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.value='native' THEN PERFORM pg_advisory_xact_lock(720010); END IF; RETURN NEW; END $$;
 CREATE TRIGGER pause_native_point_write BEFORE UPDATE ON estimate_points FOR EACH ROW EXECUTE FUNCTION pause_native_point_write()`

func TestEstimateNativePointEditSerializesWithMCP(t *testing.T) {
	f, repository, command := estimatePointEditFixture(t)
	command.ExpectedRevision = nil
	native, mcp := contendWithNativeCatalogueWriter(t, f, nativePointPauseTrigger,
		func(ctx context.Context) (json.RawMessage, error) {
			return repository.PatchEstimatePoint(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), command.ParentID.String(), strings.NewReader(`{"value":"native"}`), "application/json")
		}, func(ctx context.Context) (projectmutations.Result, error) {
			return repository.MutateEstimate(ctx, command)
		})
	require.NoError(t, native.err, "native and MCP point writers must use the same lock order")
	require.NoError(t, mcp.err, "native and MCP point writers must use the same lock order")
	var nativePoint, mcpPoint struct{ Value string }
	require.NoError(t, json.Unmarshal(native.value, &nativePoint))
	require.NoError(t, json.Unmarshal(mcp.value.Record, &mcpPoint))
	require.Equal(t, "native", nativePoint.Value, "the native response must describe its own accepted write")
	require.Equal(t, "mcp", mcpPoint.Value)
	var saved string
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT value FROM estimate_points WHERE id=$1`, command.ParentID).Scan(&saved))
	require.Equal(t, "mcp", saved)
}

func TestEstimateNativeCreationInvalidatesCompetingMCPRevision(t *testing.T) {
	f, repository, command := estimatePointWriteFixture(t)
	command.Action, command.ResourceID = "create", uuid.Nil
	command.Fields = map[string]any{"name": "Native contender", "description": "mcp"}
	native, mcp := contendWithNativeCatalogueWriter(t, f, `CREATE FUNCTION pause_native_estimate_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM pg_advisory_xact_lock(720010); RETURN NEW; END $$;
 CREATE TRIGGER a_pause_native_estimate_write AFTER INSERT ON estimates FOR EACH ROW WHEN (NEW.name='Native contender' AND NEW.description='') EXECUTE FUNCTION pause_native_estimate_write()`,
		func(ctx context.Context) (json.RawMessage, error) {
			return repository.CreateEstimate(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, strings.NewReader(`{"estimate":{"name":"Native contender"}}`), "application/json")
		}, func(ctx context.Context) (projectmutations.Result, error) {
			return repository.MutateEstimate(ctx, command)
		})
	require.NoError(t, native.err)
	require.ErrorIs(t, mcp.err, projectmutations.ErrRevision)
	var count, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM estimates WHERE name='Native contender'),(SELECT count(*) FROM mutation_receipts WHERE request_id=$1)`, command.RequestID).Scan(&count, &receipts))
	require.Equal(t, 1, count)
	require.Zero(t, receipts)
}

func TestEstimateNativeEditCompletesBeforeWorkspaceDeletion(t *testing.T) {
	f, repository, command := estimatePointEditFixture(t)
	cipher, err := instancecipher.New("estimate-workspace-deletion-fixture")
	require.NoError(t, err)
	workspaces, err := instancemanagement.NewRepository(f.pool, cipher)
	require.NoError(t, err)
	native, deletion := contendWithNativeCatalogueWriter(t, f, nativePointPauseTrigger,
		func(ctx context.Context) (json.RawMessage, error) {
			return repository.PatchEstimatePoint(ctx, testsupport.LabelActor, "workspace-one", testsupport.LabelProject, command.ResourceID.String(), command.ParentID.String(), strings.NewReader(`{"value":"native"}`), "application/json")
		}, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, workspaces.SoftDeleteWorkspace(ctx, stduuid.MustParse(command.ActorID.String()), instancemanagement.WorkspaceDeletion{ID: testsupport.LabelWorkspace, Slug: "workspace-one"}, time.Now())
		})
	require.NoError(t, native.err)
	require.NoError(t, deletion.err)
	var point struct{ Value string }
	require.NoError(t, json.Unmarshal(native.value, &point))
	require.Equal(t, "native", point.Value)
	var deleted bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM workspaces WHERE id=$1`, testsupport.LabelWorkspace).Scan(&deleted))
	require.True(t, deleted)
}
