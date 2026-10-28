//go:build integration

package stickies

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/database"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sortProofConcurrentCreates = 8

func TestConcurrentStickyCreateAllocatesDistinctIncreasingSortOrders(t *testing.T) {
	if os.Getenv("PLANE_RUN_STICKY_SORT_PROOF") != "1" {
		t.Skip("set PLANE_RUN_STICKY_SORT_PROOF=1 to run the Sticky sort-order proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pair, err := postgresfixture.Start(ctx, createSortTestSchema)
	if err != nil {
		t.Fatalf("start PostgreSQL fixture: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL fixture: %v", closeErr)
		}
	})

	configuration, err := pgxpool.ParseConfig(pair.SourceURL)
	if err != nil {
		t.Fatalf("parse PostgreSQL configuration: %v", err)
	}
	configuration.MaxConns = 16
	if configurePoolErr := database.ConfigurePool(configuration); configurePoolErr != nil {
		t.Fatalf("configure PostgreSQL pool: %v", configurePoolErr)
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	workspaceID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290411")
	otherWorkspaceID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290412")
	userID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290413")
	inactiveUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290414")
	seedSortProof(t, ctx, pool, workspaceID, otherWorkspaceID, userID, inactiveUserID)

	repository, err := NewSortRepository(pool)
	if err != nil {
		t.Fatalf("NewSortRepository() error = %v", err)
	}
	memberships, err := NewPostgresMembershipReader(pool)
	if err != nil {
		t.Fatalf("NewPostgresMembershipReader() error = %v", err)
	}
	authorizer, err := NewAuthorizer(memberships)
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	createPolicyScope, err := authorizer.ScopeFor(
		ctx,
		OperationCreate,
		WorkspaceID(workspaceID),
		UserID(userID),
	)
	if err != nil {
		t.Fatalf("ScopeFor(create) error = %v", err)
	}

	start := make(chan struct{})
	results := make(chan CreatedSticky, sortProofConcurrentCreates)
	errorsChannel := make(chan error, sortProofConcurrentCreates)
	var group sync.WaitGroup
	for index := range sortProofConcurrentCreates {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			<-start
			stickyID := StickyID(uuid.MustParse(fmt.Sprintf("0198a28d-d227-7752-82ac-bc4c1c29%04d", 500+index)))
			created, createErr := repository.CreateWithAllocatedSort(
				ctx,
				stickyID,
				createPolicyScope,
			)
			if createErr != nil {
				errorsChannel <- createErr
				return
			}
			results <- created
		}(index)
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsChannel)
	for createErr := range errorsChannel {
		t.Errorf("concurrent CreateWithAllocatedSort() error = %v", createErr)
	}
	if t.Failed() {
		return
	}

	orders := make([]float64, 0, sortProofConcurrentCreates)
	for created := range results {
		orders = append(orders, created.SortOrder)
	}
	if len(orders) != sortProofConcurrentCreates {
		t.Fatalf("created Stickies = %d, want %d", len(orders), sortProofConcurrentCreates)
	}
	sort.Float64s(orders)
	for index, order := range orders {
		want := float64(75535 + index*sortOrderStep)
		if order != want {
			t.Fatalf("sorted order[%d] = %v, want %v", index, order, want)
		}
		if index > 0 && orders[index-1] >= order {
			t.Fatalf("sort orders are not distinct and increasing: %v", orders)
		}
	}
	assertStoredSortOrders(t, ctx, pool, workspaceID, orders)

	otherCreateScope, err := authorizer.ScopeFor(
		ctx,
		OperationCreate,
		WorkspaceID(otherWorkspaceID),
		UserID(userID),
	)
	if err != nil {
		t.Fatalf("ScopeFor(other workspace create) error = %v", err)
	}
	otherCreated, err := repository.CreateWithAllocatedSort(
		ctx,
		StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290601")),
		otherCreateScope,
	)
	if err != nil {
		t.Fatalf("other-workspace CreateWithAllocatedSort() error = %v", err)
	}
	if otherCreated.SortOrder != initialSortOrder {
		t.Fatalf("other-workspace sort order = %v, want %v", otherCreated.SortOrder, initialSortOrder)
	}

	_, err = repository.CreateWithAllocatedSort(
		ctx,
		StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290602")),
		createScope(workspaceID, inactiveUserID),
	)
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("inactive-member CreateWithAllocatedSort() error = %v, want access denied", err)
	}

	assertSortTransactionRollback(t, ctx, pool, repository, workspaceID, userID)
}

func assertStoredSortOrders(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	workspaceID uuid.UUID,
	createdOrders []float64,
) {
	t.Helper()
	rows, err := pool.Query(ctx, `
SELECT sort_order
FROM stickies
WHERE workspace_id = $1
  AND deleted_at IS NULL
ORDER BY sort_order
`, workspaceID)
	if err != nil {
		t.Fatalf("query stored sort orders: %v", err)
	}
	defer rows.Close()

	stored := make([]float64, 0, len(createdOrders)+1)
	for rows.Next() {
		var order float64
		if err := rows.Scan(&order); err != nil {
			t.Fatalf("scan stored sort order: %v", err)
		}
		stored = append(stored, order)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read stored sort orders: %v", err)
	}
	want := append([]float64{initialSortOrder}, createdOrders...)
	if len(stored) != len(want) {
		t.Fatalf("stored sort orders = %v, want %v", stored, want)
	}
	for index := range want {
		if stored[index] != want[index] {
			t.Fatalf("stored sort orders = %v, want %v", stored, want)
		}
	}
}

func seedSortProof(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	workspaceID uuid.UUID,
	otherWorkspaceID uuid.UUID,
	userID uuid.UUID,
	inactiveUserID uuid.UUID,
) {
	t.Helper()
	if _, err := pool.Exec(
		ctx,
		"INSERT INTO workspaces (id, deleted_at) VALUES ($1, NULL), ($2, NULL)",
		workspaceID,
		otherWorkspaceID,
	); err != nil {
		t.Fatalf("seed workspaces: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO workspace_members (id, workspace_id, member_id, is_active, deleted_at)
VALUES
    ('0198a28d-d227-7752-82ac-bc4c1c290421', $1, $3, TRUE, NULL),
    ('0198a28d-d227-7752-82ac-bc4c1c290422', $2, $3, TRUE, NULL),
	    ('0198a28d-d227-7752-82ac-bc4c1c290423', $1, $4, FALSE, NULL)
`, workspaceID, otherWorkspaceID, userID, inactiveUserID); err != nil {
		t.Fatalf("seed workspace memberships: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO stickies (id, workspace_id, owner_id, created_by_id, sort_order, deleted_at)
VALUES
	    ('0198a28d-d227-7752-82ac-bc4c1c290431', $1, $2, $2, 65535, NULL),
	    ('0198a28d-d227-7752-82ac-bc4c1c290432', $1, $2, $2, 999999, CURRENT_TIMESTAMP)
`, workspaceID, userID); err != nil {
		t.Fatalf("seed Stickies: %v", err)
	}
}

func assertSortTransactionRollback(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	repository *SortRepository,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) {
	t.Helper()
	duplicateID := StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290431"))
	_, err := repository.CreateWithAllocatedSort(
		ctx,
		duplicateID,
		createScope(workspaceID, userID),
	)
	if !errors.Is(err, ErrStickyCreateFailed) {
		t.Fatalf("duplicate CreateWithAllocatedSort() error = %v, want create failure", err)
	}

	var beforeCount int
	if scanErr := pool.QueryRow(ctx, "SELECT count(*) FROM stickies WHERE workspace_id = $1", workspaceID).Scan(&beforeCount); scanErr != nil {
		t.Fatalf("count Stickies after rollback: %v", scanErr)
	}
	created, err := repository.CreateWithAllocatedSort(
		ctx,
		StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290603")),
		createScope(workspaceID, userID),
	)
	if err != nil {
		t.Fatalf("post-rollback CreateWithAllocatedSort() error = %v", err)
	}
	want := float64(75535 + sortProofConcurrentCreates*sortOrderStep)
	if created.SortOrder != want {
		t.Fatalf("post-rollback sort order = %v, want %v", created.SortOrder, want)
	}
	var afterCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM stickies WHERE workspace_id = $1", workspaceID).Scan(&afterCount); err != nil {
		t.Fatalf("count Stickies after successful retry: %v", err)
	}
	if afterCount != beforeCount+1 {
		t.Fatalf("Sticky count after rollback and retry = %d, want %d", afterCount, beforeCount+1)
	}
}

func createScope(workspaceID uuid.UUID, userID uuid.UUID) Scope {
	return Scope{
		Operation:     OperationCreate,
		WorkspaceID:   WorkspaceID(workspaceID),
		UserID:        UserID(userID),
		Ownership:     AssignOwner,
		AssignCreator: true,
	}
}

func createSortTestSchema(ctx context.Context, databaseURL string) error {
	configuration, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}
	if configurePoolErr := database.ConfigurePool(configuration); configurePoolErr != nil {
		return configurePoolErr
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `
CREATE TABLE workspaces (
    id uuid PRIMARY KEY,
    deleted_at timestamptz NULL
);
CREATE TABLE workspace_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    is_active boolean NOT NULL,
    deleted_at timestamptz NULL
);
CREATE TABLE stickies (
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamptz NULL,
    id uuid PRIMARY KEY,
    name text NULL,
    description jsonb NOT NULL DEFAULT '{}'::jsonb,
    description_html text NOT NULL DEFAULT '<p></p>',
    description_stripped text NULL,
    description_binary bytea NULL,
    logo_props jsonb NOT NULL DEFAULT '{}'::jsonb,
    color varchar(255) NULL,
    background_color varchar(255) NULL,
    created_by_id uuid NULL,
    owner_id uuid NOT NULL,
    updated_by_id uuid NULL,
    workspace_id uuid NOT NULL,
    sort_order double precision NOT NULL
)
`)
	return err
}
