package projectmembershipreads

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestV1InvitationSupportedMediaAndCompleteDocumentParsing(t *testing.T) {
	for _, operation := range []string{"create", "patch"} {
		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		_ = writer.WriteField("role", "15")
		if operation == "create" {
			_ = writer.WriteField("email", "fresh@example.com")
		}
		testsupport.Close(t, writer)
		for _, test := range []struct {
			name, contentType, body string
			status                  int
		}{
			{name: "form", contentType: "application/x-www-form-urlencoded", body: map[string]string{"create": "email=fresh%40example.com&role=15", "patch": "role=15"}[operation]},
			{name: "multipart", contentType: writer.FormDataContentType(), body: multipartBody.String()},
			{name: "trailing-document", contentType: "application/json", body: `{"email":"fresh@example.com","role":15} {}`, status: 400},
			{name: "trailing-garbage", contentType: "application/json", body: `{"role":15} garbage`, status: 400},
			{name: "unsupported", contentType: "text/plain", body: `{}`, status: 415},
		} {
			t.Run(operation+"/"+test.name, func(t *testing.T) {
				db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
					switch value := dest[0].(type) {
					case *string:
						*value = "workspace"
					case *bool:
						*value = false
					case *[]byte:
						*value = []byte(`{"email":"existing@example.com"}`)
					}
					return nil
				})}}
				repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
				var err error
				if operation == "create" {
					_, err = repository.CreateAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(test.body), test.contentType)
				} else {
					_, err = repository.PatchAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(test.body), test.contentType)
				}
				if test.status != 0 {
					var failure *djangoapi.HTTPError
					if !errors.As(err, &failure) || failure.Status != test.status || strings.Contains(db.query, "INSERT") || strings.Contains(db.query, "UPDATE") {
						t.Fatalf("error=%v final query=%s", err, db.query)
					}
					return
				}
				if err != nil || !(strings.Contains(db.query, "INSERT") || strings.Contains(db.query, "UPDATE")) {
					t.Fatalf("error=%v final query=%s", err, db.query)
				}
				if operation == "create" && (db.args[1] != "fresh@example.com" || db.args[2] != 15) {
					t.Fatalf("created values=%v", db.args)
				}
				if operation == "patch" && (db.args[3].(*int) == nil || *db.args[3].(*int) != 15) {
					t.Fatalf("updated values=%v", db.args)
				}
			})
		}
	}
}

type unreadInvitationBody struct{ t *testing.T }

func (body unreadInvitationBody) Read([]byte) (int, error) {
	body.t.Error("operation read a body the source never accesses")
	return 0, errors.New("unused body was read")
}

func TestInvitationReadsAndDeletesIgnoreUnusedRequestBodies(t *testing.T) {
	for _, operation := range []string{"workspace-list", "workspace-detail", "workspace-public", "project-list", "project-detail", "project-public", "user-workspace-list", "user-project-list", "workspace-delete", "project-delete"} {
		t.Run(operation, func(t *testing.T) {
			db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
				switch value := dest[0].(type) {
				case *string:
					*value = uuid.NewString()
				case *[]byte:
					*value = []byte(`[]`)
				}
				return nil
			})}}
			repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
			if _, err := repository.Invitation(context.Background(), InvitationCommand{Operation: operation, Body: unreadInvitationBody{t}, ContentType: "text/plain"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvitationLeaveIgnoresUnusedRequestBody(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
		if len(dest) == 4 {
			*dest[0].(*string) = "membership"
			*dest[1].(*string) = "actor"
			*dest[2].(*string) = "workspace"
			*dest[3].(*int) = 15
		} else {
			*dest[0].(*bool) = false
		}
		return nil
	})}}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
	if _, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-leave", UserID: "actor", Body: unreadInvitationBody{t}, ContentType: "text/plain"}); err != nil {
		t.Fatal(err)
	}
	if db.execCount != 2 {
		t.Fatalf("leave did not deactivate both memberships: writes=%d", db.execCount)
	}
}

func TestInvitationTargetLookupPrecedesRequestDataParsing(t *testing.T) {
	for _, operation := range []string{"workspace-patch", "workspace-join", "project-join", "member-patch", "v1-put"} {
		for _, media := range []string{"application/json", "text/plain"} {
			t.Run(operation+"/"+media, func(t *testing.T) {
				db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })}}
				repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
				_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: operation, Slug: "workspace", ID: uuid.NewString(), UserID: "actor", ContentType: media, Body: strings.NewReader("{")})
				var failure *djangoapi.HTTPError
				if !errors.Is(err, ErrNotFound) && (!errors.As(err, &failure) || failure.Status != 404) {
					t.Fatalf("missing target error=%v", err)
				}
				if db.query == "" || db.execCount != 0 || db.beginCount != 0 {
					t.Fatalf("lookup/effects: query=%q writes=%d transactions=%d", db.query, db.execCount, db.beginCount)
				}
			})
		}
	}
}

func TestInvitationSelfEditPrecedesRequestDataParsing(t *testing.T) {
	for _, media := range []string{"application/json", "text/plain"} {
		t.Run(media, func(t *testing.T) {
			db := invitationMemberDB()
			repository := &InvitationsRepository{reader: db, writer: db, publisher: &invitationTaskSpy{}}
			_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "member-patch", ID: "membership", UserID: "target", ContentType: media, Body: strings.NewReader("{")})
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != "You cannot update your own role" || db.beginCount != 0 || db.execCount != 0 {
				t.Fatalf("error=%v transactions=%d writes=%d", err, db.beginCount, db.execCount)
			}
		})
	}
}

func TestInvitationEmailAcceptsDjangoQuotedLocalPart(t *testing.T) {
	if !validInvitationEmail(`"quoted"@example.com`) {
		t.Fatal("Django quoted local part was rejected")
	}
	for _, invalid := range []string{"bad..local@example.com", "local@example..com", "a b@example.com", "local@-example.com", "local@example-.com"} {
		if validInvitationEmail(invalid) {
			t.Errorf("invalid email accepted: %q", invalid)
		}
	}
}

func TestInvitationEmailFollowsPinnedDjangoGrammar(t *testing.T) {
	for _, test := range []struct {
		email string
		valid bool
	}{
		{`"quoted"@example.com`, true},
		{`"escaped\"quote"@example.com`, true},
		{`"quoted space"@example.com`, false},
		{`"escaped\ space"@example.com`, true},
		{`""@example.com`, true},
		{"a@[127.0.0.1]", true},
		{"a@[2001:db8::1]", true},
		{"a@[999.0.0.1]", false},
		{"a@[127.00.0.1]", false},
		{"a@[fe80::1%zone]", false},
		{"a@bücher.de", true},
		{"a@xn--bcher-kva.de", true},
		{"a@example.xn--12", true},
		{"a@例子.测试", true},
		{"a@localhost", true},
		{"a@LOCALHOST", false},
		{"İıſK@example.com", true},
		{"a@example.c", false},
		{"a..b@example.com", false},
		{"a@example..com", false},
		{"a@-bad.example", false},
		{"a@bad-.example", false},
		{"a@example.123", false},
		{"a@😀.example", false},
		{"a@" + strings.Repeat("b", 64) + ".example", false},
		{strings.Repeat("a", 308) + "@example.com", true},
		{strings.Repeat("a", 309) + "@example.com", false},
		{"a@" + strings.Repeat("ü", 63) + "." + strings.Repeat("ü", 63) + ".de", true},
	} {
		if got := validInvitationEmail(test.email); got != test.valid {
			t.Errorf("validInvitationEmail(%q)=%t, want %t", test.email, got, test.valid)
		}
	}
}

func TestInvitationEmailNormalizationFollowsPythonBeforeProjectValidation(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{"İ@example.com", "i\u0307@example.com"},
		{"a@ΟΣ.example", "a@οσ.example"},
		{"a@ΟΣ-EXAMPLE.COM", "a@ος-example.com"},
		{"a@ΣΟ.example", "a@σο.example"},
		{"\u001c\u001dA@EXAMPLE.COM\u001e\u001f", "a@example.com"},
		{"\u0085 A@EXAMPLE.COM\u00a0", "a@example.com"},
	} {
		if got := normalizeInvitationEmail(test.raw); got != test.want {
			t.Errorf("normalizeInvitationEmail(%q)=%q, want %q", test.raw, got, test.want)
		}
	}
	if !validInvitationEmail("İ@example.com") || validInvitationEmail(normalizeInvitationEmail("İ@example.com")) {
		t.Fatal("workspace validates before normalization; project validates afterward")
	}
}

func TestWorkspaceInvitationExistingMemberPrecedesEmailValidation(t *testing.T) {
	db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
		if len(dest) == 3 {
			*dest[0].(*string), *dest[1].(*string), *dest[2].(*int) = "workspace", "owner@example.com", 20
		} else if value, ok := dest[0].(*string); ok {
			*value = "workspace"
		} else {
			*dest[0].(*[]byte) = []byte(`[{"id":"existing-member"}]`)
		}
		return nil
	})}}
	publisher := &invitationTaskSpy{}
	repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
	_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-create", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(`{"emails":[{"email":"existing@example.com","role":5},{"email":"bad","role":5}]}`)})
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != "Some users are already member of workspace" || failure.Body["workspace_users"] == nil {
		t.Fatalf("error=%v", err)
	}
	if db.execCount != 0 || db.beginCount != 0 || len(publisher.tasks) != 0 {
		t.Fatal("rejected batch mutated membership, invitations, or queue")
	}
}

func TestWorkspaceInvitationTombstoneLookupFollowsRoleCap(t *testing.T) {
	for _, test := range []struct {
		role    int
		status  int
		message string
	}{
		{15, 404, "The required object does not exist."},
		{30, 400, "You cannot invite a user with higher role"},
	} {
		var db *invitationTestDB
		db = &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
			if strings.Contains(db.query, "w.deleted_at IS NULL") || strings.Contains(db.query, "FROM workspaces WHERE") {
				return pgx.ErrNoRows
			}
			*dest[0].(*string), *dest[1].(*string), *dest[2].(*int) = "workspace", "owner@example.com", 20
			return nil
		})}}
		publisher := &invitationTaskSpy{}
		repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
		body := fmt.Sprintf(`{"emails":[{"email":"fresh@example.com","role":%d}]}`, test.role)
		_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-create", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(body)})
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != test.status || failure.Body["error"] != test.message {
			t.Fatalf("role=%d error=%v", test.role, err)
		}
		if db.execCount != 0 || db.beginCount != 0 || len(publisher.tasks) != 0 {
			t.Fatal("rejected tombstoned-workspace invitation produced effects")
		}
	}
}

func TestWorkspaceInvitationInvalidEmailPreservesPythonRepresentation(t *testing.T) {
	for _, test := range []struct{ body, entry string }{
		{`{"emails":[{"email":"\"quoted space\"@example.com","role":5}]}`, `{'email': '"quoted space"@example.com', 'role': 5}`},
		{`{"emails":[{"email":"o'bad","role":5}]}`, `{'email': "o'bad", 'role': 5}`},
		{`{"emails":[{"role":5,"email":"bad: , email"}]}`, `{'role': 5, 'email': 'bad: , email'}`},
		{`{"emails":[{"email":"bad","role":5,"extra":[true,false,null,1.0,1e2,{"z":"\n","a":"x"}]}]}`, `{'email': 'bad', 'role': 5, 'extra': [True, False, None, 1.0, 100.0, {'z': '\n', 'a': 'x'}]}`},
		{`{"emails":[{"email":"bad\u001cemail","role":5}]}`, `{'email': 'bad\x1cemail', 'role': 5}`},
	} {
		db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
			if len(dest) == 3 {
				*dest[0].(*string), *dest[1].(*string), *dest[2].(*int) = "workspace", "owner@example.com", 20
			} else if value, ok := dest[0].(*string); ok {
				*value = "workspace"
			} else {
				*dest[0].(*[]byte) = []byte(`[]`)
			}
			return nil
		})}}
		publisher := &invitationTaskSpy{}
		repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
		_, err := repository.Invitation(context.Background(), InvitationCommand{Operation: "workspace-create", UserID: "owner", ContentType: "application/json", Body: strings.NewReader(test.body)})
		var failure *djangoapi.HTTPError
		want := "Invalid email - " + test.entry + " provided a valid email address is required to send the invite"
		if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != want {
			t.Fatalf("body=%s error=%v expected=%s", test.body, failure, want)
		}
		if db.execCount != 0 || db.beginCount != 0 || len(publisher.tasks) != 0 {
			t.Fatal("invalid email produced durable effects")
		}
	}
}

func TestInvitationNonObjectJSONFollowsOperationSourceAccess(t *testing.T) {
	for _, operation := range []string{"v1-create", "v1-patch", "v1-put", "workspace-patch", "member-patch"} {
		for _, body := range []string{`null`, `[]`, `"other"`, `"role"`, `["role"]`, `17`, `17.5`, `true`} {
			t.Run(operation+"/"+body, func(t *testing.T) {
				want := 500
				if operation == "v1-create" || operation == "workspace-patch" || operation == "member-patch" && (body == `[]` || body == `"other"`) {
					want = 400
				}
				db := &invitationTestDB{databaseStub: databaseStub{row: rowFunc(func(dest ...any) error {
					if len(dest) == 4 {
						*dest[0].(*string), *dest[1].(*string), *dest[2].(*string), *dest[3].(*int) = "membership", "target", "workspace", 15
					} else if value, ok := dest[0].(*string); ok {
						*value = "workspace"
					} else {
						*dest[0].(*[]byte) = []byte(`{"email":"existing@example.com"}`)
					}
					return nil
				})}}
				publisher := &invitationTaskSpy{}
				repository := &InvitationsRepository{reader: db, writer: db, publisher: publisher}
				var err error
				switch operation {
				case "v1-create":
					_, err = repository.CreateAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(body), "application/json")
				case "v1-patch":
					_, err = repository.PatchAPIWorkspaceInvite(context.Background(), "workspace", uuid.New(), strings.NewReader(body), "application/json")
				default:
					_, err = repository.Invitation(context.Background(), InvitationCommand{Operation: operation, Slug: "workspace", ID: uuid.NewString(), UserID: "actor", ContentType: "application/json", Body: strings.NewReader(body)})
				}
				if err == nil {
					t.Fatal("non-object JSON reached save")
				}
				got := 500
				var failure *djangoapi.HTTPError
				if errors.As(err, &failure) {
					got = failure.Status
				}
				if got != want || got == 400 && failure.Body["non_field_errors"] == nil {
					t.Fatalf("status=%d error=%v, want=%d", got, err, want)
				}
				if db.execCount != 0 || db.beginCount != 0 || len(publisher.tasks) != 0 || strings.Contains(db.query, "INSERT") || strings.Contains(db.query, "UPDATE") {
					t.Fatal("invalid body produced durable effects")
				}
			})
		}
	}
}
