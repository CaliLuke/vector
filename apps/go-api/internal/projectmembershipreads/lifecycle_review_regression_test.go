package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const lifecycleReviewUser = "00000000-0000-4000-8000-000000000001"

type lifecycleReviewDB struct {
	queries                []string
	arguments              [][]any
	writes                 [][]any
	writeQueries           []string
	projectRoles           map[string]int
	workspaceRoles         map[string]int
	nullTarget             bool
	missingWorkspaceMember bool
}

func (db *lifecycleReviewDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.arguments = append(db.arguments, args)
	return rowFunc(func(dest ...any) error {
		switch {
		case strings.Contains(query, "SELECT pm.member_id::text,pm.role"):
			if target, ok := dest[0].(**string); ok {
				if !db.nullTarget {
					value := lifecycleReviewUser
					*target = &value
				}
			} else if db.nullTarget {
				return errors.New("cannot scan NULL into string")
			} else {
				*dest[0].(*string) = lifecycleReviewUser
			}
			*dest[1].(*int) = lifecycleReviewRole(db.projectRoles, lifecycleReviewUser, 15)
		case strings.Contains(query, "SELECT wm.role"):
			if db.missingWorkspaceMember {
				return pgx.ErrNoRows
			}
			if value, nullable := args[1].(*string); nullable && value == nil {
				return pgx.ErrNoRows
			}
			member, _ := args[1].(string)
			if pointer, ok := args[1].(*string); ok {
				member = *pointer
			}
			*dest[0].(*int) = lifecycleReviewRole(db.workspaceRoles, member, 20)
		case strings.HasPrefix(query, "SELECT role FROM project_members"):
			*dest[0].(*int) = lifecycleReviewRole(db.projectRoles, args[1].(string), 20)
		case strings.Contains(query, "to_jsonb(pm)"):
			*dest[0].(*[]byte) = []byte(`{"sort_order":65535}`)
		default:
			for _, target := range dest {
				*target.(*bool) = true
			}
		}
		return nil
	})
}

func lifecycleReviewRole(roles map[string]int, member string, fallback int) int {
	if role, ok := roles[member]; ok {
		return role
	}
	return fallback
}

func (db *lifecycleReviewDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.writes = append(db.writes, args)
	db.writeQueries = append(db.writeQueries, query)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func lifecycleReviewRepository(t *testing.T, db *lifecycleReviewDB) *LifecycleRepository {
	t.Helper()
	repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func lifecycleReviewError(t *testing.T, err error, status int, body map[string]any) {
	t.Helper()
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != status || !reflect.DeepEqual(failure.Body, body) {
		t.Fatalf("error = %#v, want %d %#v", err, status, body)
	}
}

func lifecycleReviewForm(field, value, media string) LifecycleCommand {
	body := url.Values{field: []string{value}}.Encode()
	if media == "multipart/form-data; boundary=plane" {
		body = "--plane\r\nContent-Disposition: form-data; name=\"" + field + "\"\r\n\r\n" + value + "\r\n--plane--\r\n"
	}
	return LifecycleCommand{UserID: "requester", Slug: "demo", ProjectID: "project", MemberID: "membership", Method: "PATCH", Body: strings.NewReader(body), ContentType: media}
}

func TestLifecycleLegacyHTMLJSONFieldsDecodeAndRejectInvalidJSON(t *testing.T) {
	for _, media := range []string{"application/x-www-form-urlencoded", "multipart/form-data; boundary=plane"} {
		for _, field := range []string{"preferences", "view_props", "default_props"} {
			for _, value := range []string{`{"pages":{"block_display":false}}`, `not-json`, `{} trailing`} {
				t.Run(media+"/"+field+"/"+value, func(t *testing.T) {
					db := &lifecycleReviewDB{}
					_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), lifecycleReviewForm(field, value, media))
					if value != `{"pages":{"block_display":false}}` {
						lifecycleReviewError(t, err, 400, map[string]any{field: []string{"Value must be valid JSON."}})
						if len(db.writes) > 0 {
							t.Fatal("invalid JSON saved")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if got := db.writes[0][2]; got != value {
						t.Fatalf("stored JSON = %#v, want object %s", got, value)
					}
				})
			}
		}
	}
}

func TestLifecycleOptionalHTMLValuesPreserveSerializerSemantics(t *testing.T) {
	for _, media := range []string{"application/x-www-form-urlencoded", "multipart/form-data; boundary=plane"} {
		for _, field := range []string{"deleted_at", "sort_order", "created_by", "updated_by"} {
			t.Run(media+"/"+field, func(t *testing.T) {
				db := &lifecycleReviewDB{}
				_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), lifecycleReviewForm(field, "", media))
				if err != nil {
					t.Fatal(err)
				}
				want := 2
				if field == "deleted_at" || field == "created_by" {
					want = 3
				}
				if len(db.writes) != 1 || len(db.writes[0]) != want {
					t.Fatalf("writes=%#v", db.writes)
				}
				if want == 3 && db.writes[0][2] != nil {
					t.Fatalf("nullable field retained empty string: %#v", db.writes)
				}
			})
		}
		for _, create := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/v1-create=%v", media, create), func(t *testing.T) {
				db := &lifecycleReviewDB{}
				command := lifecycleReviewForm("role", "", media)
				if create {
					if strings.HasPrefix(media, "multipart") {
						command.Body = strings.NewReader("--plane\r\nContent-Disposition: form-data; name=\"role\"\r\n\r\n\r\n--plane\r\nContent-Disposition: form-data; name=\"member\"\r\n\r\n" + lifecycleReviewUser + "\r\n--plane--\r\n")
					} else {
						command.Body = strings.NewReader("role=&member=" + lifecycleReviewUser)
					}
				}
				value, err := lifecycleReviewRepository(t, db).lifecycleV1Write(context.Background(), command, create)
				if err != nil {
					t.Fatal(err)
				}
				want := 15
				if create {
					want = 5
				}
				if value.(map[string]any)["role"] != want {
					t.Fatalf("result=%#v", value)
				}
			})
		}
	}
}

func TestLifecycleLegacyAuditBooleanPKErrorsPrecedeWrites(t *testing.T) {
	for _, field := range []string{"created_by", "updated_by"} {
		db := &lifecycleReviewDB{}
		command := LifecycleCommand{UserID: "requester", Body: strings.NewReader(`{"` + field + `":true}`), ContentType: "application/json"}
		_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), command)
		lifecycleReviewError(t, err, 400, map[string]any{field: []string{"Incorrect type. Expected pk value, received bool."}})
		if len(db.writes) > 0 {
			t.Fatal("invalid audit FK saved")
		}
	}
}

func TestLifecycleLegacyNullablePatchTargetIsMissing(t *testing.T) {
	db := &lifecycleReviewDB{nullTarget: true}
	_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), LifecycleCommand{Body: strings.NewReader(`{}`), ContentType: "application/json"})
	lifecycleReviewError(t, err, 404, map[string]any{"error": "The required object does not exist."})
	if len(db.writes) > 0 {
		t.Fatal("missing workspace target saved")
	}
}

func TestLifecycleBulkMalformedUUIDIsDistinctFromMissingMember(t *testing.T) {
	for _, test := range []struct {
		identity string
		status   int
		message  string
	}{
		{`"not-a-uuid"`, 400, "Please provide valid detail"},
		{`1.5`, 400, "Please provide valid detail"},
		{`null`, 404, "The required object does not exist."},
		{`true`, 404, "The required object does not exist."},
		{`123`, 404, "The required object does not exist."},
		{`{}`, 500, "Something went wrong please try again later"},
		{`[]`, 500, "Something went wrong please try again later"},
	} {
		db := &lifecycleReviewDB{missingWorkspaceMember: true}
		_, err := lifecycleReviewRepository(t, db).lifecycleBulkCreate(context.Background(), LifecycleCommand{Body: strings.NewReader(`{"members":[{"member_id":` + test.identity + `,"role":15}]}`), ContentType: "application/json"})
		lifecycleReviewError(t, err, test.status, map[string]any{"error": test.message})
		if len(db.writes) > 0 {
			t.Fatal("invalid UUID saved")
		}
		if test.identity == "true" || test.identity == "123" {
			want := "00000000-0000-0000-0000-000000000001"
			if test.identity == "123" {
				want = "00000000-0000-0000-0000-00000000007b"
			}
			if db.arguments[len(db.arguments)-1][1] != want {
				t.Fatalf("ORM lookup lost integer identity: %#v", db.arguments)
			}
		}
	}
}

func TestLifecycleLegacyEmptyFormRoleStillFailsAtViewGuard(t *testing.T) {
	db := &lifecycleReviewDB{}
	_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), lifecycleReviewForm("role", "", "application/x-www-form-urlencoded"))
	lifecycleReviewError(t, err, 500, map[string]any{"error": "Something went wrong please try again later"})
	if len(db.writes) > 0 {
		t.Fatal("invalid role saved")
	}
}

func TestLifecycleJSONStringsRemainJSONStrings(t *testing.T) {
	db := &lifecycleReviewDB{}
	_, err := lifecycleReviewRepository(t, db).lifecycleLegacyPatch(context.Background(), LifecycleCommand{UserID: "requester", Body: strings.NewReader(`{"preferences":"literal"}`), ContentType: "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	var stored any
	if err = json.Unmarshal([]byte(db.writes[0][2].(string)), &stored); err != nil || stored != "literal" {
		t.Fatalf("stored=%#v err=%v", stored, err)
	}
}

func TestLifecycleLegacyActionRestrictsDeclaredRolesAndRetainsWorkspaceAdminOverride(t *testing.T) {
	for _, operation := range []string{"legacy-patch-members", "legacy-leave-members", "legacy-create-members", "legacy-delete-members"} {
		db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*bool) = false; return nil })}
		repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
		if err != nil {
			t.Fatal(err)
		}
		err = repo.authorizeLegacyAction(context.Background(), LifecycleCommand{Operation: operation})
		lifecycleReviewError(t, err, 403, map[string]any{"error": "You don't have the required permissions."})
		want := []int{5, 15, 20}
		if operation == "legacy-create-members" || operation == "legacy-delete-members" {
			want = []int{20}
		}
		if len(db.args) != 4 || !reflect.DeepEqual(db.args[3], want) || !strings.Contains(db.query, "pm.role=ANY($4)") || !strings.Contains(db.query, "wm.role=20") || !strings.Contains(db.query, "pm.is_active") {
			t.Fatalf("%s permission query=%s args=%#v", operation, db.query, db.args)
		}
	}
}

func TestLifecycleLitePermissionsFollowSafePostAndOtherUnsafeMethods(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST", "PATCH", "DELETE", "PUT"} {
		db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*bool) = true; return nil })}
		repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
		if err != nil {
			t.Fatal(err)
		}
		if err = repo.AuthorizeLifecycle(context.Background(), LifecycleCommand{Operation: "v1-list-members-lite", Method: method}); err != nil {
			t.Fatal(err)
		}
		switch method {
		case "GET", "HEAD", "OPTIONS":
			if len(db.args) != 4 || db.args[3] != true || !strings.Contains(db.query, "FROM project_members") {
				t.Fatalf("%s query=%s args=%#v", method, db.query, db.args)
			}
		case "POST":
			if !strings.Contains(db.query, "FROM workspace_members") || !strings.Contains(db.query, "wm.role IN (15,20)") || strings.Contains(db.query, "FROM project_members") {
				t.Fatalf("POST must require workspace membership only: %s", db.query)
			}
		default:
			if !strings.Contains(db.query, "pm.role IN (15,20)") || strings.Contains(db.query, "FROM workspace_members") {
				t.Fatalf("%s must require project MEMBER/ADMIN: %s", method, db.query)
			}
		}
	}
}
