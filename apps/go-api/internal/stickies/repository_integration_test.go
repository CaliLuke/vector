//go:build integration

package stickies

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/database"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryPostgresOperationsAndIsolation(t *testing.T) {
	if os.Getenv("PLANE_RUN_STICKY_REPOSITORY_PROOF") != "1" {
		t.Skip("set PLANE_RUN_STICKY_REPOSITORY_PROOF=1 to run the Sticky repository proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pair, err := postgresfixture.Start(ctx, createRepositoryTestSchema)
	if err != nil {
		t.Fatalf("start PostgreSQL fixture: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL fixture: %v", closeErr)
		}
	})
	config, err := pgxpool.ParseConfig(pair.SourceURL)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.ConfigurePool(config); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	workspace := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290711")
	otherWorkspace := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290712")
	deletedWorkspace := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290719")
	owner := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290713")
	other := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290714")
	_, err = pool.Exec(ctx, `INSERT INTO workspaces (id,slug) VALUES ($1,'one'),($2,'two')`, workspace, otherWorkspace)
	if err == nil {
		_, err = pool.Exec(ctx, `INSERT INTO workspace_members (id,workspace_id,member_id,is_active) VALUES ('0198a28d-d227-7752-82ac-bc4c1c290715',$1,$3,true),('0198a28d-d227-7752-82ac-bc4c1c290716',$2,$4,true)`, workspace, otherWorkspace, owner, other)
	}
	if err == nil {
		_, err = pool.Exec(ctx, `INSERT INTO workspaces (id,slug,deleted_at) VALUES ($1,'deleted','2026-01-02T03:04:05Z')`, deletedWorkspace)
	}
	if err == nil {
		_, err = pool.Exec(ctx, `INSERT INTO workspace_members (id,workspace_id,member_id,is_active) VALUES ('0198a28d-d227-7752-82ac-bc4c1c290720',$1,$2,true)`, deletedWorkspace, owner)
	}
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewRepository(pool)
	if err != nil {
		t.Fatal(err)
	}
	if got, resolveWorkspaceErr := repository.ResolveWorkspace(ctx, "one", UserID(owner)); resolveWorkspaceErr != nil || uuid.UUID(got) != workspace {
		t.Fatalf("ResolveWorkspace() = %s, %v", uuid.UUID(got), resolveWorkspaceErr)
	}
	if got, resolveAPIWorkspaceErr := repository.ResolveAPIWorkspace(ctx, "deleted", UserID(owner)); !errors.Is(resolveAPIWorkspaceErr, ErrStickyWorkspaceDeleted) || uuid.UUID(got) != deletedWorkspace {
		t.Fatalf("ResolveAPIWorkspace(deleted) = %s, %v, want workspace ID and ErrStickyWorkspaceDeleted", uuid.UUID(got), resolveAPIWorkspaceErr)
	}
	if _, resolveWorkspaceErr := repository.ResolveWorkspace(ctx, "one", UserID(other)); !errors.Is(resolveWorkspaceErr, ErrWorkspaceAccessDenied) {
		t.Fatalf("cross tenant resolution error = %v", resolveWorkspaceErr)
	}
	create := Scope{Operation: OperationCreate, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: AssignOwner, AssignCreator: true}
	id := StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290717"))
	name := "first"
	html := "<p>first</p>"
	stripped := "first"
	created, err := repository.Create(ctx, id, create, CreateStickyInput{Name: &name, DescriptionHTML: &html, DescriptionStripped: &stripped})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.SortOrder != initialSortOrder || created.OwnerID != UserID(owner) || created.CreatedByID == nil || *created.CreatedByID != UserID(owner) {
		t.Fatalf("created Sticky = %+v", created)
	}
	listScope := Scope{Operation: OperationList, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: RequireOwner}
	page, err := repository.List(ctx, listScope, ListOptions{Query: "first", Limit: 1})
	if err != nil || len(page.Stickies) != 1 {
		t.Fatalf("List() = %+v, %v", page, err)
	}
	update := Scope{Operation: OperationUpdate, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: RequireOwner, RequireCreator: true}
	changed := "changed"
	changedPointer := &changed
	patched, err := repository.Patch(ctx, id, update, PatchStickyInput{Name: &changedPointer})
	if err != nil || patched.Name == nil || *patched.Name != "changed" || patched.UpdatedByID == nil {
		t.Fatalf("Patch() = %+v, %v", patched, err)
	}
	ownedName := "API v1 owner update"
	ownedNamePointer := &ownedName
	ownedStripped := "first"
	ownedStrippedPointer := &ownedStripped
	owned, err := repository.PatchOwned(ctx, id, Scope{Operation: OperationUpdate, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: RequireOwner}, PatchStickyInput{Name: &ownedNamePointer, DescriptionStripped: &ownedStrippedPointer})
	if err != nil || owned.Name == nil || *owned.Name != ownedName {
		t.Fatalf("PatchOwned(name-only) = %+v, %v", owned, err)
	}
	if _, retrieveErr := repository.Retrieve(ctx, id, Scope{Operation: OperationRetrieve, WorkspaceID: WorkspaceID(workspace), UserID: UserID(other), Ownership: RequireOwner}); !errors.Is(retrieveErr, ErrStickyNotFound) {
		t.Fatalf("cross tenant Retrieve() error = %v", retrieveErr)
	}
	deleteScope := Scope{Operation: OperationDelete, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: RequireOwner, RequireCreator: true}
	if deleteErr := repository.Delete(ctx, id, deleteScope); deleteErr != nil {
		t.Fatalf("Delete() error = %v", deleteErr)
	}
	if _, retrieveErr := repository.Retrieve(ctx, id, Scope{Operation: OperationRetrieve, WorkspaceID: WorkspaceID(workspace), UserID: UserID(owner), Ownership: RequireOwner}); !errors.Is(retrieveErr, ErrStickyNotFound) {
		t.Fatalf("soft deleted Retrieve() error = %v", retrieveErr)
	}
	empty := ""
	emptyID := StickyID(uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290718"))
	emptySticky, err := repository.Create(ctx, emptyID, create, CreateStickyInput{DescriptionHTML: &empty})
	if err != nil || emptySticky.DescriptionStripped != nil {
		t.Fatalf("Create(empty HTML) = %+v, %v, want null stripped text", emptySticky, err)
	}
	nullStripped := (*string)(nil)
	emptySticky, err = repository.Patch(ctx, emptyID, update, PatchStickyInput{DescriptionStripped: &nullStripped})
	if err != nil || emptySticky.DescriptionStripped != nil {
		t.Fatalf("Patch(null stripped text) = %+v, %v", emptySticky, err)
	}
	if _, err := repository.Create(ctx, id, create, CreateStickyInput{}); !errors.Is(err, ErrStickyCreateFailed) {
		t.Fatalf("duplicate Create() error = %v", err)
	}
}

func createRepositoryTestSchema(ctx context.Context, databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}
	if err = database.ConfigurePool(config); err != nil {
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	defer pool.Close()
	_, err = pool.Exec(ctx, `CREATE TABLE workspaces (id uuid PRIMARY KEY,slug text NOT NULL,deleted_at timestamptz); CREATE TABLE workspace_members (id uuid PRIMARY KEY,workspace_id uuid NOT NULL,member_id uuid NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz); CREATE TABLE stickies (created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,deleted_at timestamptz,id uuid PRIMARY KEY,name text,description jsonb NOT NULL DEFAULT '{}'::jsonb,description_html text NOT NULL DEFAULT '<p></p>',description_stripped text,description_binary bytea,logo_props jsonb NOT NULL DEFAULT '{}'::jsonb,color varchar(255),background_color varchar(255),created_by_id uuid,owner_id uuid NOT NULL,updated_by_id uuid,workspace_id uuid NOT NULL,sort_order double precision NOT NULL)`)
	return err
}
