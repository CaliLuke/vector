package projectmembershipreads

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestWorkspaceInvitesUseNarrowSerializerAndNewestFirst(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`[]`)
		return nil
	})}
	repository, _ := NewRepository(database)
	data, err := repository.WorkspaceInvites(context.Background(), "demo")
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{"workspace.slug = $1", "invite.deleted_at IS NULL", "ORDER BY invite.created_at DESC", "'accepted'", "'responded_at'"} {
		if !strings.Contains(database.query, fragment) {
			t.Errorf("invite list query missing %q", fragment)
		}
	}
	if strings.Contains(database.query, "'token'") || strings.Contains(database.query, "'message'") {
		t.Fatal("API v1 invite projection exposed private fields")
	}
}

func TestWorkspaceInviteScopesBySlugAndMapsMissing(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}
	repository, _ := NewRepository(database)
	_, err := repository.WorkspaceInvite(context.Background(), "demo", uuid.Nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(database.query, "workspace.slug = $1 AND invite.id = $2") || !strings.Contains(database.query, "invite.deleted_at IS NULL") {
		t.Fatalf("invite lookup is not tenant and tombstone scoped: %s", database.query)
	}
}

func TestAuthorizeWorkspaceRoleRequiresActiveWorkspaceMembershipAndExactRole(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = true
		return nil
	})}
	repository, _ := NewRepository(database)
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	if err := repository.AuthorizeWorkspaceRole(context.Background(), userID, "demo", 15, 20); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"workspace.deleted_at IS NULL", "member.is_active = true", "member.deleted_at IS NULL", "member.role = ANY($3::smallint[])"} {
		if fragment == "workspace.deleted_at IS NULL" {
			if strings.Contains(database.query, fragment) {
				t.Errorf("permission query added workspace tombstone exclusion absent from source")
			}
			continue
		}
		if !strings.Contains(database.query, fragment) {
			t.Errorf("role authorization query missing %q", fragment)
		}
	}
	if got, ok := database.args[2].([]int16); !ok || len(got) != 2 || got[0] != 15 || got[1] != 20 {
		t.Fatalf("role values=%#v", database.args[2])
	}
}

func TestWorkspaceExistsExcludesTombstonedWorkspace(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = false
		return nil
	})}
	repository, _ := NewRepository(database)
	exists, err := repository.WorkspaceExists(context.Background(), "demo")
	if err != nil || exists {
		t.Fatalf("exists=%t err=%v", exists, err)
	}
	if !strings.Contains(database.query, "deleted_at IS NULL") {
		t.Fatalf("workspace lookup does not exclude tombstones: %s", database.query)
	}
}

func TestWorkspaceMemberReadQueriesRetainInactiveRowsAndSerializerFields(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = []byte(`[]`)
		if len(destinations) > 1 {
			*destinations[1].(*int) = 3
		}
		return nil
	})}
	repository, _ := NewRepository(database)
	if _, err := repository.WorkspaceMemberUsers(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(database.query, "is_active = true") || !strings.Contains(database.query, "'role'") || !strings.Contains(database.query, "ORDER BY workspace_member.created_at DESC") {
		t.Fatalf("full roster filters inactive rows, omits role, or has wrong order: %s", database.query)
	}

}

type invitationTestDB struct {
	databaseStub
	beginCount int
	execCount  int
	tx         pgx.Tx
	onExec     func(string, []any) error
}

func (db *invitationTestDB) Begin(context.Context) (pgx.Tx, error) {
	db.beginCount++
	return db.tx, nil
}
func (db *invitationTestDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected row collection")
}
func (db *invitationTestDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.execCount++
	if db.onExec != nil {
		if err := db.onExec(query, args); err != nil {
			return pgconn.CommandTag{}, err
		}
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

type invitationTaskSpy struct {
	tasks []string
	args  [][]any
}

func (spy *invitationTaskSpy) Task(_ context.Context, name string, args []any, _ map[string]any) error {
	spy.args = append(spy.args, args)
	spy.tasks = append(spy.tasks, name)
	return nil
}

func TestProjectInviteBatchRejectsInvalidRoleBeforeAnyInsertOrTask(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
		*dest[0].(*string) = "00000000-0000-4000-8000-000000000001"
		*dest[1].(*string) = "owner@example.com"
		*dest[2].(*int) = 20
		return nil
	})}}
	publisher := &invitationTaskSpy{}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "project-create", Slug: "workspace", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"emails":[{"email":"valid@example.com","role":15},{"email":"other@example.com","role":true}]}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != "Invalid role" {
		t.Fatalf("error=%v", err)
	}
	if db.beginCount != 0 || db.execCount != 0 || len(publisher.tasks) != 0 {
		t.Fatalf("invalid batch mutated database or queue: begin=%d exec=%d tasks=%v", db.beginCount, db.execCount, publisher.tasks)
	}
}
func invitationMemberDB() *invitationTestDB {
	return &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
		*dest[0].(*string) = "membership"
		*dest[1].(*string) = "target"
		*dest[2].(*string) = "workspace"
		*dest[3].(*int) = 15
		return nil
	})}}
}
func TestWorkspaceGuestDemotionValidatesWholePayloadBeforeCascade(t *testing.T) {
	db := invitationMemberDB()
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", Slug: "workspace", ID: "membership", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"role":5,"is_active":"not-a-boolean"}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["is_active"] == nil {
		t.Fatalf("error=%v", err)
	}
	if db.beginCount != 0 || db.execCount != 0 {
		t.Fatalf("invalid patch reached cascade: begin=%d exec=%d", db.beginCount, db.execCount)
	}
}

type invitationRollbackTx struct {
	pgx.Tx
	pendingRole int
	projectRole int
	committed   bool
	rolledBack  bool
}

func (tx *invitationRollbackTx) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	if strings.HasPrefix(query, "UPDATE project_members") {
		tx.pendingRole = 5
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	return pgconn.CommandTag{}, errors.New("injected member save failure")
}
func (tx *invitationRollbackTx) Commit(context.Context) error {
	tx.committed = true
	tx.projectRole = tx.pendingRole
	return nil
}
func (tx *invitationRollbackTx) Rollback(context.Context) error {
	tx.rolledBack = true
	tx.pendingRole = tx.projectRole
	return nil
}
func TestWorkspaceGuestDemotionRollsBackCascadeIfMemberSaveFails(t *testing.T) {
	tx := &invitationRollbackTx{projectRole: 15, pendingRole: 15}
	db := invitationMemberDB()
	db.tx = tx
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", Slug: "workspace", ID: "membership", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"role":5}`)})
	if err == nil || !strings.Contains(err.Error(), "injected member save failure") {
		t.Fatalf("error=%v", err)
	}
	if tx.committed || !tx.rolledBack || tx.pendingRole != 15 || tx.projectRole != 15 {
		t.Fatalf("cascade did not roll back: %+v", tx)
	}
}
func TestPublicWorkspaceJoinChecksTokenBeforeAuthentication(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
		*dest[0].(*string) = "correct"
		*dest[1].(*string) = "invitee@example.com"
		*dest[2].(*string) = "workspace"
		*dest[3].(*int) = 15
		*dest[4].(**time.Time) = nil
		return nil
	})}}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-join", ContentType: "application/json", Body: strings.NewReader(`{"token":"wrong","accepted":true}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 403 || failure.Body["error"] != "You do not have permission to join the workspace" {
		t.Fatalf("error=%v", err)
	}
	if db.execCount != 0 {
		t.Fatal("invalid token modified invitation")
	}
}

func TestProjectInvitationRejectsEveryNoncanonicalChoiceBeforeWrites(t *testing.T) {
	for _, raw := range []string{`15.0`, `15e0`, `true`, `null`, `" 15 "`, `"+15"`} {
		t.Run(raw, func(t *testing.T) {
			db := &invitationTestDB{}
			publisher := &invitationTaskSpy{}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
			_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "project-create", ContentType: "application/json", Body: strings.NewReader(`{"emails":[{"email":"first@example.com","role":15},{"email":"second@example.com","role":` + raw + `}]}`)})
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != "Invalid role" {
				t.Fatalf("error=%v", err)
			}
			if db.beginCount != 0 || db.execCount != 0 || len(publisher.tasks) != 0 {
				t.Fatal("invalid batch produced a durable effect")
			}
		})
	}
}

type invitationCreateTx struct {
	pgx.Tx
	insertCount int
	committed   bool
}

func (tx *invitationCreateTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.insertCount++
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (tx *invitationCreateTx) Commit(context.Context) error   { tx.committed = true; return nil }
func (tx *invitationCreateTx) Rollback(context.Context) error { return nil }
func TestProjectAdminCanCreateInviteWithoutWorkspaceMembership(t *testing.T) {
	tx := &invitationCreateTx{}
	db := &invitationTestDB{tx: tx}
	publisher := &invitationTaskSpy{}
	db.row = rowFunc(func(dest ...any) error {
		if strings.Contains(db.query, "wm.member_id=$2") {
			return pgx.ErrNoRows
		}
		if strings.Contains(db.query, "SELECT wm.role") {
			return pgx.ErrNoRows
		}
		*dest[0].(*string) = "00000000-0000-4000-8000-000000000001"
		*dest[1].(*string) = "project-admin@example.com"
		return nil
	})
	repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher, baseHost: "http://localhost:8000", secretKey: "secret"}
	data, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "project-create", Slug: "workspace", ProjectID: "project", UserID: "admin", ContentType: "application/json", Body: strings.NewReader(`{"emails":[{"email":"recipient@example.com","role":15}]}`)})
	if err != nil || string(data) != `{"message":"Email sent successfully"}` {
		t.Fatalf("data=%s error=%v", data, err)
	}
	if publisher.args[0][3] != "http://localhost:8000" {
		t.Fatalf("task host=%v", publisher.args[0][3])
	}
	if !tx.committed || tx.insertCount != 1 || len(publisher.tasks) != 1 || publisher.tasks[0] != "plane.bgtasks.project_invitation_task.project_invitation" {
		t.Fatalf("insert=%d committed=%t tasks=%v", tx.insertCount, tx.committed, publisher.tasks)
	}
}

func TestWorkspaceJoinSavesUserWithoutInventingProfileColumn(t *testing.T) {
	db := &invitationTestDB{}
	publisher := &invitationTaskSpy{}
	userSaved := false
	db.row = rowFunc(func(dest ...any) error {
		switch {
		case strings.Contains(db.query, "SELECT i.token"):
			*dest[0].(*string) = "token"
			*dest[1].(*string) = "invitee@example.com"
			*dest[2].(*string) = "workspace"
			*dest[3].(*int) = 15
			*dest[4].(**time.Time) = nil
		case strings.Contains(db.query, "SELECT email FROM users"):
			*dest[0].(*string) = "invitee@example.com"
		case strings.Contains(db.query, "SELECT id::text FROM users"):
			*dest[0].(*string) = "invitee"
		case strings.Contains(db.query, "SELECT id::text FROM workspace_members"):
			*dest[0].(*string) = "membership"
		case strings.Contains(db.query, "SELECT i.id::text"):
			*dest[0].(*string) = "00000000-0000-4000-8000-000000000001"
		default:
			return errors.New("unexpected lookup: " + db.query)
		}
		return nil
	})
	db.onExec = func(query string, args []any) error {
		if strings.HasPrefix(query, "UPDATE users") {
			if strings.Contains(query, "last_workspace_id") {
				return errors.New("column last_workspace_id does not exist")
			}
			userSaved = true
		}
		return nil
	}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
	data, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-join", ID: "invite", Slug: "workspace", UserID: "invitee", ContentType: "application/json", Body: strings.NewReader(`{"token":"token","accepted":true}`)})
	if err != nil || string(data) != `{"message":"Workspace Invitation Accepted"}` {
		t.Fatalf("data=%s error=%v", data, err)
	}
	if !userSaved || len(publisher.tasks) != 2 {
		t.Fatalf("userSaved=%t tasks=%v", userSaved, publisher.tasks)
	}
}
func TestProjectJoinCreatesTargetMembershipRatherThanReusingForeignProject(t *testing.T) {
	db := &invitationTestDB{}
	publisher := &invitationTaskSpy{}
	targetCreated, foreignActivated := false, false
	db.row = rowFunc(func(dest ...any) error {
		switch {
		case strings.Contains(db.query, "SELECT i.token"):
			*dest[0].(*string) = "token"
			*dest[1].(*string) = "invitee@example.com"
			*dest[2].(*string) = "workspace"
			*dest[3].(*int) = 15
			*dest[4].(**time.Time) = nil
		case strings.Contains(db.query, "SELECT email FROM users"):
			*dest[0].(*string) = "invitee@example.com"
		case strings.Contains(db.query, "SELECT id::text FROM workspace_members"):
			*dest[0].(*string) = "workspace-membership"
		case strings.Contains(db.query, "SELECT id::text FROM project_members"):
			if strings.Contains(db.query, "project_id=$2") {
				return pgx.ErrNoRows
			}
			*dest[0].(*string) = "foreign-membership"
		default:
			return errors.New("unexpected lookup: " + db.query)
		}
		return nil
	})
	db.onExec = func(query string, args []any) error {
		if strings.HasPrefix(query, "INSERT INTO project_members") {
			targetCreated = args[1] == "target-project"
		}
		if strings.HasPrefix(query, "UPDATE project_members") {
			foreignActivated = true
		}
		return nil
	}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
	data, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "project-join", ID: "invite", ProjectID: "target-project", Slug: "workspace", UserID: "invitee", ContentType: "application/json", Body: strings.NewReader(`{"token":"token","accepted":true}`)})
	if err != nil || string(data) != `{"message":"Project Invitation Accepted"}` {
		t.Fatalf("data=%s error=%v", data, err)
	}
	if !targetCreated || foreignActivated || len(publisher.tasks) != 0 {
		t.Fatalf("targetCreated=%t foreignActivated=%t tasks=%v", targetCreated, foreignActivated, publisher.tasks)
	}
}

func TestWorkspaceInvitePatchReturnsUpdatedTombstonedInstance(t *testing.T) {
	db := &invitationTestDB{}
	deleted := false
	db.row = rowFunc(func(dest ...any) error {
		if deleted && strings.Contains(db.query, "i.deleted_at IS NULL") {
			return pgx.ErrNoRows
		}
		payload := `{"email":"invitee@example.com","deleted_at":null}`
		if deleted {
			payload = `{"email":"invitee@example.com","deleted_at":"2026-01-01T00:00:00Z"}`
		}
		*dest[0].(*[]byte) = []byte(payload)
		return nil
	})
	db.onExec = func(query string, args []any) error {
		if strings.Contains(query, "deleted_at=") {
			deleted = true
		}
		return nil
	}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	data, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-patch", Slug: "workspace", ID: "invite", UserID: "actor", ContentType: "application/json", Body: strings.NewReader(`{"deleted_at":"2026-01-01T00:00:00Z"}`)})
	if err != nil || !deleted || !strings.Contains(string(data), `"deleted_at":"2026-01-01T00:00:00Z"`) {
		t.Fatalf("data=%s error=%v deleted=%t", data, err, deleted)
	}
}

func TestWorkspaceMemberPatchAggregatesFieldErrorsWithoutCascade(t *testing.T) {
	db := invitationMemberDB()
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", ID: "membership", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"role":6,"is_active":"invalid","default_props":null}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("error=%v", err)
	}
	if len(failure.Body) != 3 || failure.Body["role"] == nil || failure.Body["is_active"] == nil || failure.Body["default_props"] == nil {
		t.Fatalf("errors=%v", failure.Body)
	}
	if db.beginCount != 0 || db.execCount != 0 {
		t.Fatalf("invalid payload wrote data: begin=%d exec=%d", db.beginCount, db.execCount)
	}
}
func TestWorkspaceMemberPatchValidatesRelatedFieldsBeforeSaving(t *testing.T) {
	db := invitationMemberDB()
	memberRow := db.row
	db.row = rowFunc(func(dest ...any) error {
		if len(dest) == 1 {
			*dest[0].(*bool) = false
			return nil
		}
		return memberRow.Scan(dest...)
	})
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", ID: "membership", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"role":5,"created_by":"00000000-0000-4000-8000-000000000001","updated_by":"00000000-0000-4000-8000-000000000002","workspace":"00000000-0000-4000-8000-000000000003"}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("error=%v", err)
	}
	if len(failure.Body) != 3 || failure.Body["created_by"] == nil || failure.Body["updated_by"] == nil || failure.Body["workspace"] == nil {
		t.Fatalf("errors=%v", failure.Body)
	}
	if db.beginCount != 0 || db.execCount != 0 {
		t.Fatal("invalid relation reached cascade or save")
	}
}
func TestWorkspaceInvitePatchValidatesUpdaterThenUsesAuthenticatedActor(t *testing.T) {
	db := &invitationTestDB{}
	actor := "00000000-0000-4000-8000-000000000001"
	savedActor := ""
	db.row = rowFunc(func(dest ...any) error {
		switch target := dest[0].(type) {
		case *bool:
			*target = true
		case *[]byte:
			*target = []byte(`{"email":"invitee@example.com"}`)
		}
		return nil
	})
	db.onExec = func(query string, args []any) error {
		if strings.Count(query, "updated_by_id=") != 1 {
			return errors.New("submitted updater overrode audit actor")
		}
		savedActor = args[1].(string)
		return nil
	}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-patch", ID: "invite", UserID: actor, ContentType: "application/json", Body: strings.NewReader(`{"updated_by":"00000000-0000-4000-8000-000000000002"}`)})
	if err != nil || savedActor != actor {
		t.Fatalf("error=%v savedActor=%q", err, savedActor)
	}
}
func TestV1WorkspaceLookupPrecedesMalformedCreateBody(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.CreateAPIWorkspaceInvite(context.Background(), "deleted-workspace", uuid.New(), strings.NewReader(`not-json`), "application/json")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(db.query, "FROM workspaces WHERE slug=$1 AND deleted_at IS NULL") || db.execCount != 0 {
		t.Fatalf("query=%s writes=%d", db.query, db.execCount)
	}
}

func TestWorkspaceMemberPatchReturnsSourceMalformedUUIDFieldError(t *testing.T) {
	db := invitationMemberDB()
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", ID: "membership", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"created_by":"not-a-uuid"}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("error=%v", err)
	}
	messages, ok := failure.Body["created_by"].([]string)
	if !ok || len(messages) != 1 || messages[0] != "“not-a-uuid” is not a valid UUID." {
		t.Fatalf("errors=%v", failure.Body)
	}
	if db.beginCount != 0 || db.execCount != 0 {
		t.Fatal("invalid related UUID wrote data")
	}
}
func TestV1InviteCreateAggregatesSerializerErrors(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*string) = "workspace"; return nil })}}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.CreateAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(`{"email":"","role":true}`), "application/json")
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || len(failure.Body) != 2 || failure.Body["email"] == nil || failure.Body["role"] == nil {
		t.Fatalf("error=%v", err)
	}
	if db.execCount != 0 {
		t.Fatal("invalid invite inserted a row")
	}
}

func TestInvitationEmptyBodiesReachOperationValidation(t *testing.T) {
	for _, body := range [][]byte{nil, {}, []byte(`{}`)} {
		t.Run(fmt.Sprintf("create-%q", body), func(t *testing.T) {
			db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*string) = "workspace"; return nil })}}
			publisher := &invitationTaskSpy{}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
			_, err := repository.CreateAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(string(body)), "application/json")
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 || !reflect.DeepEqual(failure.Body, map[string]any{"email": []string{"This field is required."}}) {
				t.Fatalf("error=%v", err)
			}
			if db.execCount != 0 || len(publisher.tasks) != 0 {
				t.Fatal("empty create mutated database or queue")
			}
		})
		t.Run(fmt.Sprintf("patch-%q", body), func(t *testing.T) {
			db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*[]byte) = []byte(`{"email":"existing@example.com"}`); return nil })}}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
			if _, err := repository.PatchAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(string(body)), "application/json"); err != nil {
				t.Fatalf("partial update rejected empty fields: %v", err)
			}
			if !strings.Contains(db.query, "UPDATE workspace_member_invites") {
				t.Fatalf("empty partial update did not reach save: %s", db.query)
			}
		})
		t.Run(fmt.Sprintf("put-%q", body), func(t *testing.T) {
			db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*[]byte) = []byte(`{"email":"existing@example.com"}`); return nil })}}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
			_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "v1-put", Slug: "workspace", ID: uuid.NewString(), ContentType: "application/json", Body: strings.NewReader(string(body))})
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 || !reflect.DeepEqual(failure.Body, map[string]any{"email": []string{"This field is required."}}) {
				t.Fatalf("error=%v", err)
			}
			if db.execCount != 0 || strings.Contains(db.query, "UPDATE workspace_member_invites") {
				t.Fatal("empty PUT reached save")
			}
		})
	}
	for _, body := range []string{" ", "not-json"} {
		if _, err := decodeInvitationSerializerPayload(strings.NewReader(body), "application/json"); err == nil {
			t.Fatalf("nonempty malformed JSON %q was accepted", body)
		}
	}
}
func TestV1InvitePatchAggregatesBlankEmailAndNullRoleErrors(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*[]byte) = []byte(`{"email":"existing@example.com"}`); return nil })}}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	_, err := repository.PatchAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(`{"email":"","role":null}`), "application/json")
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || len(failure.Body) != 2 || failure.Body["email"] == nil || failure.Body["role"] == nil {
		t.Fatalf("error=%v", err)
	}
	if db.execCount != 0 {
		t.Fatal("invalid invite patch modified a row")
	}
}

func TestInvitationMissingObjectBodiesFollowSourceLookupKinds(t *testing.T) {
	for _, test := range []struct{ operation, key, message string }{
		{"project-detail", "detail", "No ProjectMemberInvite matches the given query."},
		{"project-delete", "detail", "No ProjectMemberInvite matches the given query."},
		{"workspace-detail", "detail", "No WorkspaceMemberInvite matches the given query."},
		{"workspace-patch", "detail", "No WorkspaceMemberInvite matches the given query."},
		{"workspace-delete", "error", "The required object does not exist."},
		{"workspace-public", "error", "The required object does not exist."},
		{"project-public", "error", "The required object does not exist."},
		{"workspace-join", "error", "The required object does not exist."},
		{"project-join", "error", "The required object does not exist."},
	} {
		t.Run(test.operation, func(t *testing.T) {
			db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}}
			publisher := &invitationTaskSpy{}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
			_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: test.operation, Slug: "workspace", ProjectID: "project", ID: "missing", UserID: "actor", ContentType: "application/json"})
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 404 || len(failure.Body) != 1 || failure.Body[test.key] != test.message {
				t.Fatalf("error=%v", err)
			}
			if db.execCount != 0 || len(publisher.tasks) != 0 {
				t.Fatal("missing object produced durable effects")
			}
		})
	}
}
