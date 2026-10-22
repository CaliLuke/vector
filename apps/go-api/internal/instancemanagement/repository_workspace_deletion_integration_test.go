//go:build integration

package instancemanagement

import (
	"context"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

func TestSoftDeleteWorkspacePersistsTimestampAndReleasesSlug(t *testing.T) {
	ctx := t.Context()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY,slug text NOT NULL,deleted_at timestamptz,updated_at timestamptz,updated_by_id uuid);
INSERT INTO workspaces(id,slug) VALUES('00000000-0000-4000-8000-000000000001','workspace');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	workspace := WorkspaceDeletion{ID: "00000000-0000-4000-8000-000000000001", Slug: "workspace"}
	deletedAt := time.Unix(1234567890, 123456000).UTC()
	if err := (&Repository{db: pool}).SoftDeleteWorkspace(ctx, actor, workspace, deletedAt); err != nil {
		t.Fatalf("soft delete workspace: %v", err)
	}
	var slug string
	var deleted, updated time.Time
	var updatedBy uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT slug,deleted_at,updated_at,updated_by_id FROM workspaces WHERE id=$1`, workspace.ID).Scan(&slug, &deleted, &updated, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if slug != "workspace__1234567890" || !deleted.Equal(deletedAt) || !updated.Equal(deletedAt) || updatedBy != actor {
		t.Fatalf("soft-deleted workspace: slug=%q deleted=%s updated=%s actor=%s", slug, deleted, updated, updatedBy)
	}
}
