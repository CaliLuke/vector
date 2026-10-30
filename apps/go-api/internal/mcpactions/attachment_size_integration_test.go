//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// MCP sizes are integers; native legacy storage also supports fractional values.
func TestMCPAttachmentListKeepsSizeNumeric(t *testing.T) {
	f := newMilestoneFixture(t)
	repository, err := assets.NewPostgresRepository(f.pool)
	require.NoError(t, err)
	// Listing is database-only; no object storage client is configured.
	app, err := assets.NewAttachmentApplication(repository, &assets.Storage{}, nil, "https://vector.example", assets.AttachmentUploadLimit)
	require.NoError(t, err)
	apps := databaseApplications(t, f.pool)
	apps.Attachments = app
	f.service, err = mcpactions.New(apps, mcpactions.WithOrigin("https://vector.example"))
	require.NoError(t, err)
	runCatalogueFixtureMatrix(t, f, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		item, id := f.item(t), uuid.NewString()
		_, err := f.pool.Exec(t.Context(), `INSERT INTO file_assets(id,created_at,updated_at,workspace_id,project_id,issue_id,attributes,asset,entity_type,size,is_uploaded,is_deleted,is_archived,storage_metadata)
 VALUES($1,now(),now(),$2,$3,$4,'{"name":"notes.txt","type":"text/plain"}','notes.txt','ISSUE_ATTACHMENT',38,true,false,false,'{}')`, id, testsupport.LabelWorkspace, testsupport.LabelProject, item)
		require.NoError(t, err)
		call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
		for _, test := range []struct {
			name, metadata string
			want           float64
		}{
			{"decimal integer", "12.0", 12}, {"negative integer", "-1.0", -1}, {"zero", "0", 38}, {"decimal zero", "0.0", 38},
			{"null", "null", 38}, {"string", "\"12.5\"", 38}, {"infinity string", "\"Infinity\"", 38}, {"object", "{\"value\":12.5}", 38}, {"array", "[12.5]", 38}, {"boolean", "true", 38},
		} {
			t.Run(test.name, func(t *testing.T) {
				_, err := f.pool.Exec(t.Context(), `UPDATE file_assets SET attributes=jsonb_set(attributes,'{size}',$2::jsonb) WHERE id=$1`, id, test.metadata)
				require.NoError(t, err)
				var before string
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(a)::text FROM file_assets a WHERE id=$1`, id).Scan(&before))
				rows := catalogueArray(t, call("workitem_attachment", map[string]any{"action": "list", "include_revision": true}))
				require.Len(t, rows, 1)
				require.Equal(t, id, rows[0]["id"])
				require.Equal(t, test.want, rows[0]["size"])
				require.Equal(t, "notes.txt", rows[0]["name"])
				var after string
				require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT to_jsonb(a)::text FROM file_assets a WHERE id=$1`, id).Scan(&after))
				require.Equal(t, before, after, "normalization must not rewrite stored metadata")
			})
		}
	})
}
