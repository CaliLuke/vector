package stickies

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

func TestRepositoryRejectsNilDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) error = nil")
	}
}

func TestRepositoryRejectsInvalidInputsBeforeDatabaseAccess(t *testing.T) {
	repository, err := NewRepository(repositoryDatabaseFunc{})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	validWorkspace := WorkspaceID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701"))
	validUser := UserID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702"))
	if _, err := repository.ResolveWorkspace(context.Background(), "", validUser); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("ResolveWorkspace() error = %v", err)
	}
	if _, err := repository.List(context.Background(), Scope{}, ListOptions{}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := repository.Retrieve(context.Background(), StickyID{}, Scope{Operation: OperationRetrieve, WorkspaceID: validWorkspace, UserID: validUser, Ownership: RequireOwner}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if err := repository.Delete(context.Background(), StickyID{}, Scope{Operation: OperationDelete, WorkspaceID: validWorkspace, UserID: validUser, Ownership: RequireOwner, RequireCreator: true}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestEscapeStickySearchTreatsSQLPatternCharactersLiterally(t *testing.T) {
	got := escapeStickySearch(`need\le_%`)
	want := `need\\le\_\%`
	if got != want {
		t.Fatalf("escapeStickySearch() = %q, want %q", got, want)
	}
}

type repositoryDatabaseFunc struct{}

func (repositoryDatabaseFunc) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected begin")
}
func (repositoryDatabaseFunc) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (repositoryDatabaseFunc) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}
