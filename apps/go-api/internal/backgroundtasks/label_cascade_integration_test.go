//go:build integration

package backgroundtasks

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLabelDeletionWorkerCleansUpValidNestedTree(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		_, schemaErr := connection.Exec(ctx, `CREATE TABLE labels (
 id uuid PRIMARY KEY, parent_id uuid, deleted_at timestamptz, updated_at timestamptz
);`)
		closeErr := connection.Close(context.WithoutCancel(ctx))
		if schemaErr != nil {
			return schemaErr
		}
		return closeErr
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close postgres fixture: %v", closeErr)
		}
	})
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	rootID, childID, grandchildID := uuid.New(), uuid.New(), uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO labels(id,parent_id) VALUES
 ($1,NULL),($2,$1),($3,$2)`, rootID, childID, grandchildID); err != nil {
		t.Fatal(err)
	}
	store, err := NewPostgresDeletionStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	labelModel, err := store.policy.Lookup("db", "label")
	if err != nil {
		t.Fatal(err)
	}
	relationIndex := -1
	for index, relation := range deletionRelations(store.policy, labelModel) {
		if relation.model.Table == labelModel.Table && relation.relation.Column == "parent_id" && relation.relation.OnDelete == "CASCADE" {
			relationIndex = index
			break
		}
	}
	if relationIndex < 0 {
		t.Fatal("label parent cascade relation is missing from deletion policy")
	}
	eventID := uuid.NewString()
	pending := []DeletionInput{{EventID: eventID, App: "db", Model: "label", ID: rootID.String()}}
	for len(pending) != 0 {
		current := pending[0]
		pending = pending[1:]
		if _, err = store.MarkRoot(ctx, current); err != nil {
			t.Fatal(err)
		}
		plan, planErr := store.Plan(ctx, deletionPlanInput{Root: current, Relation: relationIndex})
		if planErr != nil {
			t.Fatal(planErr)
		}
		children, applyErr := store.Apply(ctx, plan)
		if applyErr != nil {
			t.Fatal(applyErr)
		}
		pending = append(pending, children...)
	}
	var live int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM labels WHERE deleted_at IS NULL`).Scan(&live); err != nil {
		t.Fatal(err)
	}
	if live != 0 {
		t.Fatalf("worker left %d live labels in valid nested tree", live)
	}
}
