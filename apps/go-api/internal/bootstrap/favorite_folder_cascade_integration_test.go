//go:build integration

package bootstrap_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestDeletingFavoriteFolderCascadesThroughNestedRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })

	_, err = connection.Exec(ctx, `
INSERT INTO users (
 password,id,username,email,first_name,last_name,avatar,date_joined,created_at,updated_at,last_location,created_location,
 is_superuser,is_managed,is_password_expired,is_active,is_staff,is_email_verified,is_password_autoset,token,user_timezone,
 last_login_ip,last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required
) VALUES (
 '', '00000000-0000-4000-8000-000000000001', 'favorite-cascade', 'favorite-cascade@example.test', 'Favorite', 'Cascade', '', NOW(), NOW(), NOW(), '', '',
 FALSE,FALSE,FALSE,TRUE,FALSE,TRUE,FALSE,'favorite-cascade-token','UTC','', '', '', '',FALSE,'Favorite Cascade',TRUE,FALSE
);
INSERT INTO workspaces (id, name, slug, created_at, updated_at, owner_id, timezone, background_color)
VALUES ('00000000-0000-4000-8000-000000000010', 'Cascade test', 'favorite-cascade-test', NOW(), NOW(), '00000000-0000-4000-8000-000000000001', 'UTC', '');
INSERT INTO user_favorites (id, created_at, updated_at, entity_type, is_folder, sequence, user_id, workspace_id, parent_id)
VALUES
 ('00000000-0000-4000-8000-000000000020', NOW(), NOW(), 'folder', TRUE, 1, '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000010', NULL),
 ('00000000-0000-4000-8000-000000000021', NOW(), NOW(), 'folder', TRUE, 1, '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000020'),
 ('00000000-0000-4000-8000-000000000022', NOW(), NOW(), 'issue', FALSE, 1, '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000010', '00000000-0000-4000-8000-000000000021'),
 ('00000000-0000-4000-8000-000000000023', NOW(), NOW(), 'folder', TRUE, 2, '00000000-0000-4000-8000-000000000001', '00000000-0000-4000-8000-000000000010', NULL);
`)
	if err != nil {
		t.Fatalf("seed nested favorites: %v", err)
	}

	if _, err := connection.Exec(ctx, `DELETE FROM user_favorites WHERE id=$1`, "00000000-0000-4000-8000-000000000020"); err != nil {
		t.Fatalf("delete parent folder: %v", err)
	}
	var remaining int
	if err := connection.QueryRow(ctx, `SELECT count(*) FROM user_favorites`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("favorite rows after folder deletion = %d, want only unrelated root folder", remaining)
	}
	var siblingRemains bool
	if err := connection.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_favorites WHERE id=$1)`, "00000000-0000-4000-8000-000000000023").Scan(&siblingRemains); err != nil || !siblingRemains {
		t.Fatalf("unrelated root favorite remains=%t error=%v", siblingRemains, err)
	}
}
