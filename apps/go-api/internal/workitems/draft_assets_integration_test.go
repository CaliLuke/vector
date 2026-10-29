//go:build integration

package workitems

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Full publication/rollback is also exercised through Execute in workitemtypes.
// This database boundary isolates reference parsing and pending-upload scope.
func TestDraftPublicationOnlyWaitsForReferencedOwnedUploads(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		conn, connectErr := pgx.Connect(ctx, url)
		if connectErr != nil {
			return connectErr
		}
		defer func() { testsupport.CloseContext(t, conn, ctx) }()
		_, createErr := conn.Exec(ctx, `CREATE TABLE file_assets(id uuid PRIMARY KEY, draft_issue_id uuid, workspace_id uuid, created_by_id uuid, deleted_at timestamptz, is_deleted boolean DEFAULT false, is_archived boolean DEFAULT false, is_uploaded boolean DEFAULT false);
CREATE TABLE work_item_properties(id uuid PRIMARY KEY, workspace_id uuid, definition jsonb);`)
		return createErr
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Close(t, pair) }()
	conn, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.CloseContext(t, conn, ctx) }()
	draft, workspace, actor, asset := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	richText, plainText := uuid.NewString(), uuid.NewString()
	_, err = conn.Exec(ctx, `INSERT INTO work_item_properties(id,workspace_id,definition) VALUES($1,$2,'{"kind":"Rich text"}'),($3,$2,'{"kind":"Text"}')`, richText, workspace, plainText)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(ctx, `INSERT INTO file_assets(id,draft_issue_id,workspace_id,created_by_id) VALUES($1,$2,$3,$4)`, asset, draft, workspace, actor)
	if err != nil {
		t.Fatal(err)
	}
	repo := &draftOperation{db: conn}
	identity := map[string]any{"id": draft, "workspace_id": workspace, "created_by_id": actor}
	image := fmt.Sprintf(`<image-component src="%s"></image-component>`, asset)
	for _, test := range []struct {
		name    string
		row     map[string]any
		blocked bool
	}{
		{"abandoned upload", map[string]any{"description_html": "<p>Finished text</p>"}, false},
		{"external image", map[string]any{"description_html": `<img src="https://example.test/image.png">`}, false},
		{"description image", map[string]any{"description_html": image}, true},
		{"custom rich text image", map[string]any{"custom_properties": map[string]any{richText: image}}, true},
		{"literal plain text", map[string]any{"custom_properties": map[string]any{plainText: image}}, false},
		{"non image element", map[string]any{"description_html": fmt.Sprintf(`<div src="%s"></div>`, asset)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			validationErr := repo.validateDraftUploads(ctx, identity, test.row)
			if !test.blocked {
				if validationErr != nil {
					t.Fatal(validationErr)
				}
				return
			}
			var conflict *djangoapi.HTTPError
			if !errors.As(validationErr, &conflict) || conflict.Status != 409 {
				t.Fatalf("pending referenced upload must block publication: %v", validationErr)
			}
		})
	}
	for _, test := range []struct{ name, update string }{
		{"finished", `UPDATE file_assets SET is_uploaded=true`},
		{"deleted", `UPDATE file_assets SET is_deleted=true`},
		{"archived", `UPDATE file_assets SET is_archived=true`},
		{"soft deleted", `UPDATE file_assets SET deleted_at=now()`},
		{"another owner", `UPDATE file_assets SET created_by_id=gen_random_uuid()`},
		{"another workspace", `UPDATE file_assets SET workspace_id=gen_random_uuid()`},
		{"another draft", `UPDATE file_assets SET draft_issue_id=gen_random_uuid()`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, resetErr := conn.Exec(ctx, `UPDATE file_assets SET is_uploaded=false,is_deleted=false,is_archived=false,deleted_at=NULL,draft_issue_id=$1,workspace_id=$2,created_by_id=$3`, draft, workspace, actor); resetErr != nil {
				t.Fatal(resetErr)
			}
			if _, updateErr := conn.Exec(ctx, test.update); updateErr != nil {
				t.Fatal(updateErr)
			}
			if validationErr := repo.validateDraftUploads(ctx, identity, map[string]any{"description_html": image}); validationErr != nil {
				t.Fatalf("finished, removed or foreign upload blocked publication: %v", validationErr)
			}
		})
	}
}
