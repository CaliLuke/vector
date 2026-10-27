//go:build integration

package workspacefavorites

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"uuid"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	favoriteScopeUser      = "00000000-0000-4000-8000-000000000001"
	favoriteScopeOtherUser = "00000000-0000-4000-8000-000000000002"
	favoriteScopeWorkspace = "00000000-0000-4000-8000-000000000010"
	favoriteScopeOtherWS   = "00000000-0000-4000-8000-000000000011"
	favoriteScopeProject   = "00000000-0000-4000-8000-000000000020"
	favoriteScopeForeign   = "00000000-0000-4000-8000-000000000021"
	favoriteScopeUnjoined  = "00000000-0000-4000-8000-000000000022"
	favoriteScopeFolder    = "00000000-0000-4000-8000-000000000030"
)

func TestFavoriteWritesAndReadsEnforceEntityAndFolderScope(t *testing.T) {
	ctx := context.Background()
	pair, fixtureErr := postgresfixture.StartUncloned(ctx, seedFavoriteScopeSchema)
	if fixtureErr != nil {
		t.Fatal(fixtureErr)
	}
	t.Cleanup(func() { _ = pair.Close() })
	connection, connectionErr := pgx.Connect(ctx, pair.SeedURL())
	if connectionErr != nil {
		t.Fatal(connectionErr)
	}
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(ctx)) })
	repository, repositoryErr := NewRepository(connection)
	if repositoryErr != nil {
		t.Fatal(repositoryErr)
	}
	user := uuid.MustParse(favoriteScopeUser)
	project := uuid.MustParse(favoriteScopeProject)
	foreignProject := uuid.MustParse(favoriteScopeForeign)
	unjoinedProject := uuid.MustParse(favoriteScopeUnjoined)
	page := uuid.MustParse("00000000-0000-4000-8000-000000000040")
	foreignPage := uuid.MustParse("00000000-0000-4000-8000-000000000041")
	globalPage := uuid.MustParse("00000000-0000-4000-8000-000000000044")
	privatePage := uuid.MustParse("00000000-0000-4000-8000-000000000045")
	cycle := uuid.MustParse("00000000-0000-4000-8000-000000000050")
	foreignCycle := uuid.MustParse("00000000-0000-4000-8000-000000000051")
	if authorized, err := repository.Authorized(ctx, user, "demo"); err != nil || !authorized {
		t.Fatalf("workspace administrator authorization = %t, %v", authorized, err)
	}
	validProject, projectCreateErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "project", EntityIdentifier: &project, ProjectID: &project})
	if projectCreateErr != nil || !strings.Contains(string(validProject), "Project One") {
		t.Fatalf("create accessible project favorite = %s, %v", validProject, projectCreateErr)
	}
	assertInvalidCreate := func(name string, input CreateInput) {
		t.Helper()
		if _, err := repository.Create(ctx, user, "demo", input); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("%s create error = %v, want ErrInvalidScope", name, err)
		}
	}
	assertInvalidCreate("cross-workspace project", CreateInput{EntityType: "project", EntityIdentifier: &foreignProject, ProjectID: &foreignProject})
	assertInvalidCreate("unjoined project", CreateInput{EntityType: "project", EntityIdentifier: &unjoinedProject, ProjectID: &unjoinedProject})
	assertInvalidCreate("mismatched project identifier", CreateInput{EntityType: "project", EntityIdentifier: &foreignProject, ProjectID: &project})
	assertInvalidCreate("unscoped project entity", CreateInput{EntityType: "project", EntityIdentifier: &foreignProject})
	assertInvalidCreate("cycle from another project", CreateInput{EntityType: "cycle", EntityIdentifier: &foreignCycle, ProjectID: &project})
	assertInvalidCreate("module from another project", CreateInput{EntityType: "module", EntityIdentifier: ptrUUID("00000000-0000-4000-8000-000000000062"), ProjectID: &project})
	assertInvalidCreate("view from another project", CreateInput{EntityType: "view", EntityIdentifier: ptrUUID("00000000-0000-4000-8000-000000000072"), ProjectID: &project})
	assertInvalidCreate("page attached to another project", CreateInput{EntityType: "page", EntityIdentifier: &foreignPage, ProjectID: &project})
	assertInvalidCreate("private page owned by another user", CreateInput{EntityType: "page", EntityIdentifier: &privatePage, ProjectID: &project})
	assertInvalidCreate("unsupported entity kind", CreateInput{EntityType: "issue", EntityIdentifier: &cycle, ProjectID: &project})
	unsupportedRoleUser := uuid.MustParse("00000000-0000-4000-8000-000000000004")
	if _, err := connection.Exec(ctx, `INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000014',$1,$2,15,true)`, favoriteScopeWorkspace, unsupportedRoleUser); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000026',$1,$3,$2,0,true)`, favoriteScopeWorkspace, unsupportedRoleUser, project); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Create(ctx, unsupportedRoleUser, "demo", CreateInput{EntityType: "project", EntityIdentifier: &project, ProjectID: &project}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("create with unsupported active project role = %v, want ErrInvalidScope", err)
	}

	validPage, pageCreateErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "page", EntityIdentifier: &page, ProjectID: &project})
	if pageCreateErr != nil || !strings.Contains(string(validPage), "Page One") {
		t.Fatalf("create page favorite in its linked project = %s, %v", validPage, pageCreateErr)
	}
	globalPageFavorite, globalPageErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "page", EntityIdentifier: &globalPage})
	if globalPageErr != nil || !strings.Contains(string(globalPageFavorite), "Page Global") {
		t.Fatalf("create workspace-global page favorite = %s, %v", globalPageFavorite, globalPageErr)
	}
	guest := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	if _, err := repository.Create(ctx, guest, "demo", CreateInput{EntityType: "page", EntityIdentifier: &page, ProjectID: &project}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("guest favorite on feature-restricted page = %v, want ErrInvalidScope", err)
	}
	validCycle, cycleCreateErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "cycle", EntityIdentifier: &cycle, ProjectID: &project})
	if cycleCreateErr != nil || !strings.Contains(string(validCycle), "Cycle One") {
		t.Fatalf("create cycle favorite = %s, %v", validCycle, cycleCreateErr)
	}
	if _, err := repository.Create(ctx, user, "demo", CreateInput{EntityType: "module", EntityIdentifier: ptrUUID("00000000-0000-4000-8000-000000000060"), ProjectID: &project}); err != nil {
		t.Fatalf("create module favorite: %v", err)
	}
	if _, err := repository.Create(ctx, user, "demo", CreateInput{EntityType: "view", EntityIdentifier: ptrUUID("00000000-0000-4000-8000-000000000070"), ProjectID: &project}); err != nil {
		t.Fatalf("create view favorite: %v", err)
	}

	folderID := uuid.MustParse(favoriteScopeFolder)
	if _, err := repository.Create(ctx, user, "demo", CreateInput{EntityType: "folder", IsFolder: true}); err != nil {
		t.Fatalf("create root folder: %v", err)
	}
	var createdFolder uuid.UUID
	if err := connection.QueryRow(ctx, `SELECT id FROM user_favorites WHERE user_id=$1 AND entity_type='folder'`, user).Scan(&createdFolder); err != nil {
		t.Fatal(err)
	}
	crossWorkspaceFolder := uuid.MustParse("00000000-0000-4000-8000-000000000031")
	_, crossWorkspaceFolderInsertErr := connection.Exec(ctx, `INSERT INTO user_favorites(id,workspace_id,user_id,entity_type,entity_identifier,is_folder,sequence,created_at,updated_at) VALUES ($1,$2,$3,'folder',NULL,true,1,NOW(),NOW())`, crossWorkspaceFolder, favoriteScopeOtherWS, user)
	if crossWorkspaceFolderInsertErr != nil {
		t.Fatal(crossWorkspaceFolderInsertErr)
	}
	nestedFolder, nestedFolderErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "folder", IsFolder: true, Parent: &createdFolder})
	if nestedFolderErr != nil {
		t.Fatalf("create nested folder: %v", nestedFolderErr)
	}
	var nestedFolderID uuid.UUID
	if err := connection.QueryRow(ctx, `SELECT id FROM user_favorites WHERE user_id=$1 AND entity_type='folder' AND parent_id=$2`, user, createdFolder).Scan(&nestedFolderID); err != nil {
		t.Fatal(err)
	}
	if len(nestedFolder) == 0 {
		t.Fatal("nested folder response is empty")
	}
	childModule := uuid.MustParse("00000000-0000-4000-8000-000000000061")
	if _, err := repository.Create(ctx, user, "demo", CreateInput{EntityType: "module", EntityIdentifier: &childModule, ProjectID: &project, Parent: &createdFolder}); err != nil {
		t.Fatalf("create child under own folder: %v", err)
	}
	var childFavoriteID uuid.UUID
	if err := connection.QueryRow(ctx, `SELECT id FROM user_favorites WHERE user_id=$1 AND entity_type='module' AND entity_identifier=$2`, user, childModule).Scan(&childFavoriteID); err != nil {
		t.Fatal(err)
	}
	missingParent := uuid.MustParse("00000000-0000-4000-8000-000000000099")
	assertInvalidCreate("missing parent folder", CreateInput{EntityType: "project", EntityIdentifier: &project, ProjectID: &project, Parent: &missingParent})
	assertInvalidCreate("another workspace's parent folder", CreateInput{EntityType: "project", EntityIdentifier: &project, ProjectID: &project, Parent: &crossWorkspaceFolder})
	_, otherUserFolderInsertErr := connection.Exec(ctx, `INSERT INTO user_favorites(id,workspace_id,user_id,entity_type,entity_identifier,is_folder,sequence,created_at,updated_at) VALUES ($1,$2,$3,'folder',NULL,true,1,NOW(),NOW())`, folderID, favoriteScopeWorkspace, favoriteScopeOtherUser)
	if otherUserFolderInsertErr != nil {
		t.Fatal(otherUserFolderInsertErr)
	}
	assertInvalidCreate("another user's parent folder", CreateInput{EntityType: "project", EntityIdentifier: &project, ProjectID: &project, Parent: &folderID})
	if _, err := repository.Update(ctx, user, "demo", createdFolder, UpdateInput{ParentPresent: true, Parent: &createdFolder}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("self-parent update = %v, want ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, user, "demo", createdFolder, UpdateInput{ParentPresent: true, Parent: &nestedFolderID}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("reparent folder under descendant = %v, want ErrNotFound", err)
	}
	var rootHasNoParent bool
	if err := connection.QueryRow(ctx, `SELECT parent_id IS NULL FROM user_favorites WHERE id=$1`, createdFolder).Scan(&rootHasNoParent); err != nil || !rootHasNoParent {
		t.Fatalf("rejected parent changes left root parent null = %t, err=%v", rootHasNoParent, err)
	}
	siblingFolderResponse, siblingFolderErr := repository.Create(ctx, user, "demo", CreateInput{EntityType: "folder", IsFolder: true})
	if siblingFolderErr != nil || len(siblingFolderResponse) == 0 {
		t.Fatalf("create sibling folder: response=%s err=%v", siblingFolderResponse, siblingFolderErr)
	}
	var siblingFolderID uuid.UUID
	if err := connection.QueryRow(ctx, `SELECT id FROM user_favorites WHERE user_id=$1 AND workspace_id=$3 AND entity_type='folder' AND parent_id IS NULL AND id<>$2`, user, createdFolder, favoriteScopeWorkspace).Scan(&siblingFolderID); err != nil {
		t.Fatal(err)
	}
	secondConnection, secondConnectionErr := pgx.Connect(ctx, pair.SeedURL())
	if secondConnectionErr != nil {
		t.Fatal(secondConnectionErr)
	}
	t.Cleanup(func() { _ = secondConnection.Close(context.WithoutCancel(ctx)) })
	secondRepository, secondRepositoryErr := NewRepository(secondConnection)
	if secondRepositoryErr != nil {
		t.Fatal(secondRepositoryErr)
	}
	lockerConnection, lockerConnectionErr := pgx.Connect(ctx, pair.SeedURL())
	if lockerConnectionErr != nil {
		t.Fatal(lockerConnectionErr)
	}
	t.Cleanup(func() { _ = lockerConnection.Close(context.WithoutCancel(ctx)) })
	lockerTransaction, lockerTransactionErr := lockerConnection.Begin(ctx)
	if lockerTransactionErr != nil {
		t.Fatal(lockerTransactionErr)
	}
	if _, err := lockerTransaction.Exec(ctx, `SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2 FOR UPDATE`, favoriteScopeWorkspace, user); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	go func() {
		ready.Done()
		<-start
		_, updateErr := repository.Update(ctx, user, "demo", createdFolder, UpdateInput{ParentPresent: true, Parent: &siblingFolderID})
		results <- updateErr
	}()
	go func() {
		ready.Done()
		<-start
		_, updateErr := secondRepository.Update(ctx, user, "demo", siblingFolderID, UpdateInput{ParentPresent: true, Parent: &createdFolder})
		results <- updateErr
	}()
	ready.Wait()
	close(start)
	deadline := time.Now().Add(5 * time.Second)
	for {
		// Refresh statistics while the controller transaction retains its lock.
		// Otherwise the first poll can hide a writer that blocks a moment later.
		if _, err := lockerTransaction.Exec(ctx, `SELECT pg_stat_clear_snapshot()`); err != nil {
			_ = lockerTransaction.Rollback(context.WithoutCancel(ctx))
			t.Fatal(err)
		}
		var blockedUpdates int
		if err := lockerConnection.QueryRow(ctx, `SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT wm.id FROM workspace_members wm%'`).Scan(&blockedUpdates); err != nil {
			_ = lockerTransaction.Rollback(context.WithoutCancel(ctx))
			t.Fatal(err)
		}
		if blockedUpdates == 2 {
			break
		}
		if time.Now().After(deadline) {
			_ = lockerTransaction.Rollback(context.WithoutCancel(ctx))
			t.Fatalf("both reciprocal updates did not reach the shared membership lock; blocked=%d", blockedUpdates)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := lockerTransaction.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	firstUpdate, secondUpdate := <-results, <-results
	if (firstUpdate != nil || !errors.Is(secondUpdate, ErrNotFound)) && (secondUpdate != nil || !errors.Is(firstUpdate, ErrNotFound)) {
		t.Fatalf("concurrent reciprocal folder moves = %v and %v, want exactly one accepted and one rejected", firstUpdate, secondUpdate)
	}
	var rootUnderSibling, siblingUnderRoot bool
	if err := connection.QueryRow(ctx, `SELECT COALESCE((SELECT parent_id=$2 FROM user_favorites WHERE id=$1),false),COALESCE((SELECT parent_id=$1 FROM user_favorites WHERE id=$2),false)`, createdFolder, siblingFolderID).Scan(&rootUnderSibling, &siblingUnderRoot); err != nil {
		t.Fatal(err)
	}
	if rootUnderSibling == siblingUnderRoot {
		t.Fatalf("concurrent reciprocal moves created invalid hierarchy: root_under_sibling=%t sibling_under_root=%t", rootUnderSibling, siblingUnderRoot)
	}

	var validFavoriteID uuid.UUID
	if err := connection.QueryRow(ctx, `SELECT id FROM user_favorites WHERE user_id=$1 AND entity_type='project' AND entity_identifier=$2`, user, project).Scan(&validFavoriteID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Update(ctx, user, "demo", validFavoriteID, UpdateInput{ProjectIDPresent: true, ProjectID: &foreignProject}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update to cross-workspace project = %v, want ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, user, "demo", validFavoriteID, UpdateInput{EntityIdentifierPresent: true, EntityIdentifier: &foreignProject}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update to mismatched entity = %v, want ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, user, "demo", validFavoriteID, UpdateInput{ParentPresent: true, Parent: &folderID}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update to another user's parent folder = %v, want ErrNotFound", err)
	}
	if _, err := repository.Get(ctx, user, "demo", validFavoriteID); err != nil {
		t.Fatalf("rejected updates changed the valid favorite: %v", err)
	}

	unsupportedFavoriteID := "00000000-0000-4000-8000-000000000083"
	if _, err := connection.Exec(ctx, `INSERT INTO user_favorites(id,workspace_id,user_id,entity_type,entity_identifier,is_folder,sequence,project_id,created_at,updated_at) VALUES ($1,$2,$3,'project',$4,false,30,$4,NOW(),NOW())`, unsupportedFavoriteID, favoriteScopeWorkspace, unsupportedRoleUser, project); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(ctx, unsupportedRoleUser, "demo", uuid.MustParse(unsupportedFavoriteID)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get favorite with unsupported project role = %v, want ErrNotFound", err)
	}
	if _, err := repository.Update(ctx, unsupportedRoleUser, "demo", uuid.MustParse(unsupportedFavoriteID), UpdateInput{NamePresent: true, Name: ptrString("still denied")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update favorite with unsupported project role = %v, want ErrNotFound", err)
	}
	unsupportedList, unsupportedListErr := repository.List(ctx, unsupportedRoleUser, "demo", true)
	if unsupportedListErr != nil || strings.Contains(string(unsupportedList), unsupportedFavoriteID) {
		t.Fatalf("List exposed unsupported project role favorite: %s, err=%v", unsupportedList, unsupportedListErr)
	}

	malformedIDs := []string{
		"00000000-0000-4000-8000-000000000080",
		"00000000-0000-4000-8000-000000000081",
		"00000000-0000-4000-8000-000000000082",
	}
	_, malformedFavoritesInsertErr := connection.Exec(ctx, `INSERT INTO user_favorites(id,workspace_id,user_id,entity_type,entity_identifier,is_folder,sequence,parent_id,project_id,created_at,updated_at) VALUES
($1,$2,$3,'project',$4,false,20,NULL,$5,NOW(),NOW()),
($6,$2,$3,'project',$4,false,21,NULL,NULL,NOW(),NOW()),
($7,$2,$3,'page',$8,false,22,$9,$5,NOW(),NOW())`, malformedIDs[0], favoriteScopeWorkspace, user, foreignProject, project, malformedIDs[1], malformedIDs[2], foreignPage, createdFolder)
	if malformedFavoritesInsertErr != nil {
		t.Fatal(malformedFavoritesInsertErr)
	}
	listed, listedErr := repository.List(ctx, user, "demo", true)
	if listedErr != nil {
		t.Fatal(listedErr)
	}
	var listRows []map[string]json.RawMessage
	if err := json.Unmarshal(listed, &listRows); err != nil {
		t.Fatalf("decode favorites list: %v", err)
	}
	for _, row := range listRows {
		var rowID string
		if err := json.Unmarshal(row["id"], &rowID); err != nil {
			t.Fatal(err)
		}
		for _, malformedID := range malformedIDs {
			if rowID == malformedID {
				t.Fatalf("list exposed malformed favorite %s: %s", rowID, listed)
			}
		}
	}
	if strings.Contains(string(listed), "Project Foreign") {
		t.Fatalf("list exposed entity metadata from a different project: %s", listed)
	}
	if _, err := repository.Get(ctx, user, "demo", uuid.MustParse(malformedIDs[0])); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get malformed favorite = %v, want ErrNotFound", err)
	}
	grouped, groupedErr := repository.Group(ctx, user, "demo", createdFolder)
	if groupedErr != nil {
		t.Fatal(groupedErr)
	}
	if strings.Contains(string(grouped), "Project Foreign") || strings.Contains(string(grouped), malformedIDs[0]) {
		t.Fatalf("group exposed malformed child or metadata: %s", grouped)
	}
	if !strings.Contains(string(grouped), childFavoriteID.String()) {
		t.Fatalf("group omitted valid child favorite: %s", grouped)
	}
}

func ptrString(value string) *string {
	return &value
}

func ptrUUID(value string) *uuid.UUID {
	parsed := uuid.MustParse(value)
	return &parsed
}

func seedFavoriteScopeSchema(ctx context.Context, databaseURL string) (seedErr error) {
	connection, connectErr := pgx.Connect(ctx, databaseURL)
	if connectErr != nil {
		return connectErr
	}
	defer func() {
		if closeErr := connection.Close(context.WithoutCancel(ctx)); closeErr != nil && seedErr == nil {
			seedErr = closeErr
		}
	}()
	_, seedErr = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE workspace_members (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, member_id uuid NOT NULL, role smallint NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, name text NOT NULL, logo_props jsonb NOT NULL DEFAULT '{}', guest_view_all_features boolean NOT NULL DEFAULT false, deleted_at timestamptz);
CREATE TABLE project_members (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, member_id uuid NOT NULL, role smallint NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
CREATE TABLE pages (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, name text NOT NULL, owned_by_id uuid NOT NULL, access smallint NOT NULL DEFAULT 0, logo_props jsonb NOT NULL DEFAULT '{}', is_global boolean NOT NULL DEFAULT false, deleted_at timestamptz);
CREATE TABLE project_pages (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, page_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE cycles (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, name text NOT NULL, logo_props jsonb NOT NULL DEFAULT '{}', deleted_at timestamptz);
CREATE TABLE modules (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid NOT NULL, name text NOT NULL, logo_props jsonb NOT NULL DEFAULT '{}', deleted_at timestamptz);
CREATE TABLE issue_views (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, project_id uuid, name text NOT NULL, logo_props jsonb NOT NULL DEFAULT '{}', deleted_at timestamptz);
CREATE TABLE user_favorites (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, user_id uuid NOT NULL, entity_type text NOT NULL, entity_identifier uuid, name text, is_folder boolean NOT NULL, sequence double precision NOT NULL, parent_id uuid, project_id uuid, created_by_id uuid, updated_by_id uuid, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL, deleted_at timestamptz);
INSERT INTO workspaces(id,slug) VALUES
 ('00000000-0000-4000-8000-000000000010','demo'),
 ('00000000-0000-4000-8000-000000000011','foreign');
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000012','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',20,true);
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000013','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000003',15,true);
INSERT INTO projects(id,workspace_id,name) VALUES
 ('00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000010','Project One'),
 ('00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000011','Project Foreign'),
 ('00000000-0000-4000-8000-000000000022','00000000-0000-4000-8000-000000000010','Project Unjoined');
INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active) VALUES
 ('00000000-0000-4000-8000-000000000023','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000001',20,true),
 ('00000000-0000-4000-8000-000000000024','00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000001',20,true),
 ('00000000-0000-4000-8000-000000000025','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000003',5,true);
INSERT INTO pages(id,workspace_id,name,owned_by_id,access) VALUES
 ('00000000-0000-4000-8000-000000000040','00000000-0000-4000-8000-000000000010','Page One','00000000-0000-4000-8000-000000000001',0),
 ('00000000-0000-4000-8000-000000000041','00000000-0000-4000-8000-000000000010','Page Foreign','00000000-0000-4000-8000-000000000001',0),
 ('00000000-0000-4000-8000-000000000045','00000000-0000-4000-8000-000000000010','Private Page','00000000-0000-4000-8000-000000000002',1);
INSERT INTO pages(id,workspace_id,name,owned_by_id,is_global) VALUES ('00000000-0000-4000-8000-000000000044','00000000-0000-4000-8000-000000000010','Page Global','00000000-0000-4000-8000-000000000001',true);
INSERT INTO project_pages(id,workspace_id,project_id,page_id) VALUES
 ('00000000-0000-4000-8000-000000000042','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000040'),
 ('00000000-0000-4000-8000-000000000043','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000041'),
 ('00000000-0000-4000-8000-000000000046','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000045');
INSERT INTO cycles(id,workspace_id,project_id,name) VALUES
 ('00000000-0000-4000-8000-000000000050','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','Cycle One'),
 ('00000000-0000-4000-8000-000000000051','00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000021','Cycle Foreign');
INSERT INTO modules(id,workspace_id,project_id,name) VALUES
 ('00000000-0000-4000-8000-000000000060','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','Module One'),
 ('00000000-0000-4000-8000-000000000061','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','Module Child'),
 ('00000000-0000-4000-8000-000000000062','00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000021','Module Foreign');
INSERT INTO issue_views(id,workspace_id,project_id,name) VALUES
 ('00000000-0000-4000-8000-000000000070','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000020','View One'),
 ('00000000-0000-4000-8000-000000000072','00000000-0000-4000-8000-000000000011','00000000-0000-4000-8000-000000000021','View Foreign');
`)
	return seedErr
}
