//go:build integration

package standardviews

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestCreateViewsSuppliesDjangoModelDefaults(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, seedMutationValidation)
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	// Django's default=False is supplied by the ORM, not a database default.
	if _, err := connection.Exec(ctx, `ALTER TABLE issue_views ALTER COLUMN is_locked DROP DEFAULT, ALTER COLUMN is_locked SET NOT NULL`); err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	project := uuid.MustParse("00000000-0000-4000-8000-000000000020")
	for _, scope := range []string{"workspace", "project"} {
		t.Run(scope, func(t *testing.T) {
			name := "Default regression " + scope
			input := ViewInput{Name: &name}
			var body json.RawMessage
			if scope == "workspace" {
				body, err = service.CreateWorkspace(ctx, user, "alpha", input)
			} else {
				body, err = service.CreateProject(ctx, user, "alpha", project, input)
			}
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				ID     string `json:"id"`
				Locked *bool  `json:"is_locked"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatal(err)
			}
			if got.Locked == nil || *got.Locked {
				t.Fatalf("created view is not unlocked: %s", body)
			}
			var locked bool
			if err := connection.QueryRow(ctx, `SELECT is_locked FROM issue_views WHERE id=$1::uuid`, got.ID).Scan(&locked); err != nil {
				t.Fatal(err)
			}
			if locked {
				t.Fatal("persisted view is locked")
			}
		})
	}
}
