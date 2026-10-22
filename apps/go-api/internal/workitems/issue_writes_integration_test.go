//go:build integration

package workitems

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestAPICreateDefaultIssueTypeMatchesDjangoFirstPrimaryKey(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer conn.Close(ctx)
		_, err = conn.Exec(ctx, `
CREATE TABLE issue_types(id uuid PRIMARY KEY, is_default boolean NOT NULL, deleted_at timestamptz, created_at timestamptz NOT NULL);
CREATE TABLE project_issue_types(project_id uuid NOT NULL, issue_type_id uuid NOT NULL);
INSERT INTO issue_types(id,is_default,created_at) VALUES
 ('00000000-0000-4000-8000-000000000001',true,'2024-01-01T00:00:00Z'),
 ('00000000-0000-4000-8000-000000000002',true,'2025-01-01T00:00:00Z');
INSERT INTO project_issue_types(project_id,issue_type_id) VALUES
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001'),
 ('00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000002');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()

	conn, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	var selected string
	if err := conn.QueryRow(ctx, apiDefaultIssueTypeSQL, "00000000-0000-4000-8000-000000000010").Scan(&selected); err != nil {
		t.Fatal(err)
	}
	if selected != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("selected default issue type %s; Django first() selects the lowest primary key", selected)
	}
}
