package pagereads

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"uuid"
)

var pageLifecycleForbiddenBody = map[string]any{
	"detail": "You do not have permission to perform this action.",
}

func TestAuthorizeLifecycleRejectsPublicGuestMutations(t *testing.T) {
	for _, operation := range []string{"update", "description_update", "duplicate", "access"} {
		t.Run(operation, func(t *testing.T) {
			userID := uuid.New()
			repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 5, 0, false, nil)

			err := repository.AuthorizeLifecycle(context.Background(), operation, userID, "demo", uuid.New(), uuid.New())

			assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
			assertLifecycleStateReadOnce(t, database)
		})
	}
}

func TestAuthorizeLifecycleAllowsPageOwnerWithGuestProjectRole(t *testing.T) {
	userID := uuid.New()
	for _, operation := range []string{"update", "description_update", "duplicate", "access", "archive", "unarchive"} {
		t.Run(operation, func(t *testing.T) {
			repository, database := pageLifecycleAuthorizationRepository(userID, 5, 1, false, nil)

			err := repository.AuthorizeLifecycle(context.Background(), operation, userID, "demo", uuid.New(), uuid.New())

			if err != nil {
				t.Fatalf("owner with guest project role: err=%v, want authorization success", err)
			}
			assertLifecycleStateReadOnce(t, database)
		})
	}
}

func TestAuthorizeLifecycleMapsMissingPageToRecorded403(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)

	err := repository.AuthorizeLifecycle(context.Background(), "update", uuid.New(), "demo", uuid.New(), uuid.New())

	assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
	if len(database.queries) != 1 {
		t.Fatalf("queries=%d, want one scoped page lookup", len(database.queries))
	}
}

func TestAuthorizeLifecycleDeniesPublicMemberDeleteBeforeArchivedValidation(t *testing.T) {
	userID := uuid.New()
	repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 15, 0, false, nil)

	err := repository.AuthorizeLifecycle(context.Background(), "delete", userID, "demo", uuid.New(), uuid.New())

	// The source permission class rejects a non-admin DELETE before the view's
	// archived-page validation can return its 400 response.
	assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
	assertLifecycleStateReadOnce(t, database)
}

func TestLifecycleAuthorizationMatchesArchiveAndUnarchivePermissionPrecedence(t *testing.T) {
	userID := uuid.New()
	t.Run("archive member passes permission class and is handled later", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 15, 0, false, nil)
		err := repository.AuthorizeLifecycle(context.Background(), "archive", userID, "demo", uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("archive member permission: %v", err)
		}
		assertLifecycleStateReadOnce(t, database)
	})
	t.Run("unarchive member is denied by method permission first", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 15, 0, true, nil)
		err := repository.AuthorizeLifecycle(context.Background(), "unarchive", userID, "demo", uuid.New(), uuid.New())
		assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
		assertLifecycleStateReadOnce(t, database)
	})
}

func TestAuthorizeLifecycleRejectsPrivatePageForNonOwner(t *testing.T) {
	userID := uuid.New()
	repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 15, 1, false, nil)

	err := repository.AuthorizeLifecycle(context.Background(), "duplicate", userID, "demo", uuid.New(), uuid.New())

	assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
	assertLifecycleStateReadOnce(t, database)
}

func TestAuthorizeLifecycleUnknownMethodsRequirePageOwner(t *testing.T) {
	userID := uuid.New()
	t.Run("owner may continue to method dispatch", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(userID, 5, 1, false, nil)
		if err := repository.AuthorizeLifecycle(context.Background(), "unsupported", userID, "demo", uuid.New(), uuid.New()); err != nil {
			t.Fatalf("owner permission=%v, want method dispatch allowed", err)
		}
		assertLifecycleStateReadOnce(t, database)
	})
	t.Run("non-owner is denied before method dispatch", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 20, 0, false, nil)
		err := repository.AuthorizeLifecycle(context.Background(), "unsupported", userID, "demo", uuid.New(), uuid.New())
		assertPageLifecycleFixtureError(t, err, 403, pageLifecycleForbiddenBody)
		assertLifecycleStateReadOnce(t, database)
	})
}

func TestAuthorizeLifecycleUnsupportedCollectionMethodsRequireProjectBoundary(t *testing.T) {
	userID := uuid.New()
	t.Run("delete requires an admin project role", func(t *testing.T) {
		database := &databaseStub{rows: []pgx.Row{rowFunc(func(destinations ...any) error {
			*destinations[0].(*int) = 15
			return nil
		})}}
		repository, err := NewRepository(database)
		if err != nil {
			t.Fatal(err)
		}
		assertPageLifecycleFixtureError(t, repository.AuthorizeLifecycle(context.Background(), "delete_collection", userID, "demo", uuid.New(), uuid.Nil()), 403, pageLifecycleForbiddenBody)
	})
	t.Run("unknown method denies every active project role", func(t *testing.T) {
		database := &databaseStub{rows: []pgx.Row{rowFunc(func(destinations ...any) error {
			*destinations[0].(*int) = 20
			return nil
		})}}
		repository, err := NewRepository(database)
		if err != nil {
			t.Fatal(err)
		}
		assertPageLifecycleFixtureError(t, repository.AuthorizeLifecycle(context.Background(), "unsupported_collection", userID, "demo", uuid.New(), uuid.Nil()), 403, pageLifecycleForbiddenBody)
	})
}

func TestValidateLifecycleHandlerPreservesLockedAndArchivedDescriptionBodies(t *testing.T) {
	userID := uuid.New()
	t.Run("page update locked body", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(userID, 5, 0, true, nil)
		err := repository.ValidateLifecycleHandler(context.Background(), "update", userID, "demo", uuid.New(), uuid.New())
		assertPageLifecycleFixtureError(t, err, 400, map[string]any{"error": "Page is locked"})
		assertLifecycleStateReadOnce(t, database)
	})
	t.Run("description locked body", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(userID, 5, 0, true, nil)
		err := repository.ValidateLifecycleHandler(context.Background(), "description_update", userID, "demo", uuid.New(), uuid.New())
		assertPageLifecycleFixtureError(t, err, 400, map[string]any{"error_code": 4701, "error_message": "PAGE_LOCKED"})
		assertLifecycleStateReadOnce(t, database)
	})
	t.Run("description archived body", func(t *testing.T) {
		repository, database := pageLifecycleAuthorizationRepository(userID, 5, 0, false, timePointer(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		err := repository.ValidateLifecycleHandler(context.Background(), "description_update", userID, "demo", uuid.New(), uuid.New())
		assertPageLifecycleFixtureError(t, err, 400, map[string]any{"error_code": 4702, "error_message": "PAGE_ARCHIVED"})
		assertLifecycleStateReadOnce(t, database)
	})
}

func TestValidateLifecycleHandlerReturnsArchiveMemberViewError(t *testing.T) {
	userID := uuid.New()
	repository, database := pageLifecycleAuthorizationRepository(uuid.New(), 15, 0, false, nil)
	err := repository.ValidateLifecycleHandler(context.Background(), "archive", userID, "demo", uuid.New(), uuid.New())
	assertPageLifecycleFixtureError(t, err, 400, map[string]any{"error": "Only the owner or admin can archive the page"})
	assertLifecycleStateReadOnce(t, database)
}

func pageLifecycleAuthorizationRepository(ownerID uuid.UUID, role, access int, locked bool, archivedAt *time.Time) (*Repository, *databaseStub) {
	rows := []pgx.Row{rowFunc(func(destinations ...any) error {
		*destinations[0].(*uuid.UUID) = uuid.New()
		*destinations[1].(*uuid.UUID) = ownerID
		*destinations[2].(**uuid.UUID) = nil
		*destinations[3].(**uuid.UUID) = nil
		*destinations[4].(*int) = role
		*destinations[5].(*int) = access
		*destinations[6].(*bool) = locked
		*destinations[7].(**time.Time) = archivedAt
		return nil
	})}
	database := &databaseStub{rows: rows}
	repository, err := NewRepository(database)
	if err != nil {
		panic(err)
	}
	return repository, database
}

func timePointer(value time.Time) *time.Time { return &value }

func assertLifecycleStateReadOnce(t *testing.T, database *databaseStub) {
	t.Helper()
	if len(database.queries) != 1 {
		t.Fatalf("queries=%d, want one authorization lookup", len(database.queries))
	}
}

func assertPageLifecycleFixtureError(t *testing.T, err error, status int, body map[string]any) {
	t.Helper()
	var httpError *djangoapi.HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("error=%v, want HTTP %d with source body %v", err, status, body)
	}
	if httpError.Status != status || !reflect.DeepEqual(httpError.Body, body) {
		t.Fatalf("error status/body=(%d, %#v), want source fixture=(%d, %#v)", httpError.Status, httpError.Body, status, body)
	}
}
