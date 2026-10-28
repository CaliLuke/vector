//go:build integration

package stickies

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/database"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresMembershipScopeForEveryStickyOperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pair, err := postgresfixture.Start(ctx, createMembershipTestSchema)
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
	if configurePoolErr := database.ConfigurePool(configuration); configurePoolErr != nil {
		t.Fatalf("configure PostgreSQL pool: %v", configurePoolErr)
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		t.Fatalf("create PostgreSQL pool: %v", err)
	}
	defer pool.Close()

	workspaceID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290361")
	otherWorkspaceID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290362")
	activeUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290363")
	inactiveUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290364")
	deletedUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290365")
	otherOnlyUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290366")
	ownerWithoutMembershipUserID := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290367")
	if _, execErr := pool.Exec(ctx, `
INSERT INTO workspace_members (id, workspace_id, member_id, is_active, deleted_at)
VALUES
    ($1, $2, $3, TRUE, NULL),
    ($4, $2, $5, FALSE, NULL),
    ($6, $2, $7, TRUE, CURRENT_TIMESTAMP),
    ($8, $9, $3, TRUE, NULL),
    ($10, $9, $11, TRUE, NULL)
`,
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290371"), workspaceID, activeUserID,
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290372"), inactiveUserID,
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290373"), deletedUserID,
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290374"), otherWorkspaceID,
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290375"), otherOnlyUserID,
	); execErr != nil {
		t.Fatalf("seed workspace memberships: %v", execErr)
	}

	reader, err := NewPostgresMembershipReader(pool)
	if err != nil {
		t.Fatalf("NewPostgresMembershipReader() error = %v", err)
	}
	authorizer, err := NewAuthorizer(reader)
	if err != nil {
		t.Fatalf("NewAuthorizer() error = %v", err)
	}
	for _, operation := range allOperations {
		if _, err := authorizer.ScopeFor(
			ctx,
			operation,
			WorkspaceID(workspaceID),
			UserID(activeUserID),
		); err != nil {
			t.Fatalf("ScopeFor(%s) active membership error = %v", operation, err)
		}
	}

	denied := []struct {
		name        string
		workspaceID uuid.UUID
		userID      uuid.UUID
	}{
		{name: "inactive", workspaceID: workspaceID, userID: inactiveUserID},
		{name: "soft deleted", workspaceID: workspaceID, userID: deletedUserID},
		{name: "owner without membership", workspaceID: workspaceID, userID: ownerWithoutMembershipUserID},
		{name: "cross workspace", workspaceID: workspaceID, userID: otherOnlyUserID},
	}
	for _, test := range denied {
		t.Run(test.name, func(t *testing.T) {
			for _, operation := range allOperations {
				_, scopeErr := authorizer.ScopeFor(
					ctx,
					operation,
					WorkspaceID(test.workspaceID),
					UserID(test.userID),
				)
				if !errors.Is(scopeErr, ErrWorkspaceAccessDenied) {
					t.Fatalf("ScopeFor(%s) error = %v, want access denied", operation, scopeErr)
				}
			}
		})
	}
}

func createMembershipTestSchema(ctx context.Context, databaseURL string) error {
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
CREATE TABLE workspace_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL,
    member_id uuid NOT NULL,
    is_active boolean NOT NULL,
    deleted_at timestamptz NULL
)
`)
	return err
}
