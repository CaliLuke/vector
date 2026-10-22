package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
)

func TestLifecycleRoleRetainsDRFChoiceIdentity(t *testing.T) {
	for _, input := range []any{5, 15, 20, "5", "15", "20", json.Number("15")} {
		if _, failure := lifecycleRole(input); failure != "" {
			t.Errorf("role %v rejected: %s", input, failure)
		}
	}
	for _, input := range []any{json.Number("15.0"), " 15 ", true, 42, []any{15}} {
		if _, failure := lifecycleRole(input); failure == "" {
			t.Errorf("invalid choice %#v accepted", input)
		}
	}
	if _, failure := lifecycleRole(nil); failure != "This field may not be null." {
		t.Fatalf("null role error=%q", failure)
	}
}

type lifecycleTaskSpy struct{ calls int }

func (s *lifecycleTaskSpy) Task(context.Context, string, []any, map[string]any) error {
	s.calls++
	return nil
}

func TestLifecyclePermissionsPrecedeRateAndNeverRequireWorkspaceMembershipForAPIKeyReads(t *testing.T) {
	db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*bool) = false; return nil })}
	repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	command := LifecycleCommand{Operation: "v1-list-members", Method: "GET", UserID: "user", Slug: "demo", ProjectID: "project"}
	err = repo.AuthorizeLifecycle(context.Background(), command)
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("denied=%v", err)
	}
	// The source permission accepts a project member even when their workspace
	// membership is inactive; passing false above solely represents project denial.
	if len(db.args) != 4 || db.args[3] != true {
		t.Fatalf("read permission args=%#v", db.args)
	}
	command.Method = "HEAD"
	_ = repo.AuthorizeLifecycle(context.Background(), command)
	if db.args[3] != false {
		t.Fatalf("collection HEAD must use source admin permission: %#v", db.args)
	}
	command.Operation = "v1-list-members-lite"
	_ = repo.AuthorizeLifecycle(context.Background(), command)
	if db.args[3] != true {
		t.Fatalf("lite HEAD must retain SAFE_METHOD membership permission: %#v", db.args)
	}
}

func TestLegacyAuthenticationBoundaryDefersActionPermissionUntilAfterTimezone(t *testing.T) {
	db := &databaseStub{}
	repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.AuthorizeLifecycle(context.Background(), LifecycleCommand{Operation: "legacy-create-members"}); err != nil {
		t.Fatal(err)
	}
	if db.query != "" {
		t.Fatalf("legacy initial permission unexpectedly queried domain: %s", db.query)
	}
}

func TestLifecycleReplicaRoutingKeepsWritesOnPrimary(t *testing.T) {
	row := func(dest ...any) error { *dest[0].(*bool) = true; return nil }
	reader := &databaseStub{row: rowFunc(row)}
	writer := &databaseStub{row: rowFunc(row)}
	repository, err := NewLifecycleRepositoryWithReaderWriter(reader, writer, &lifecycleTaskSpy{}, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	command := LifecycleCommand{Operation: "v1-list-members", Method: "GET", UserID: "user", Slug: "demo", ProjectID: "project"}
	if err = repository.AuthorizeLifecycle(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if reader.query == "" || writer.query != "" {
		t.Fatalf("GET reader=%q writer=%q", reader.query, writer.query)
	}
	reader.query = ""
	command.Method = "POST"
	command.Operation = "v1-create-members"
	if err = repository.AuthorizeLifecycle(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if writer.query == "" || reader.query != "" {
		t.Fatalf("POST reader=%q writer=%q", reader.query, writer.query)
	}
}

func TestLegacyPatchNonObjectParsingRetainsViewGuardPrecedence(t *testing.T) {
	for _, test := range []struct {
		body, kind   string
		role, status bool
	}{{`[]`, "list", false, false}, {`["role"]`, "list", true, false}, {`["is_active"]`, "list", false, true}, {`"role"`, "str", true, false}} {
		input, err := lifecycleLegacyPatchInput(LifecycleCommand{Body: strings.NewReader(test.body), ContentType: "application/json"})
		if err != nil {
			t.Fatal(err)
		}
		_, role := input.Fields["role"]
		_, status := input.Fields["is_active"]
		if input.Kind != test.kind || role != test.role || status != test.status {
			t.Fatalf("body=%s input=%#v", test.body, input)
		}
	}
	for _, body := range []string{`null`, `true`, `5`} {
		_, err := lifecycleLegacyPatchInput(LifecycleCommand{Body: strings.NewReader(body), ContentType: "application/json"})
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != 500 {
			t.Fatalf("body=%s error=%v", body, err)
		}
	}
}

func TestLegacyMemberRenderingRejectsPersistedNonfiniteSortOrder(t *testing.T) {
	for _, order := range []string{"NaN", "Infinity", "-Infinity"} {
		body, _ := json.Marshal(map[string]any{"sort_order": order})
		db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*[]byte) = body; return nil })}
		repository, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
		if err != nil {
			t.Fatal(err)
		}
		_, err = repository.lifecycleLegacyRepresentation(context.Background(), LifecycleCommand{MemberID: "member"})
		if !errors.Is(err, ErrLifecycleRendering) {
			t.Fatalf("sort_order=%s error=%v", order, err)
		}
	}
}

func TestLegacyStatusBooleanRetainsDRFNumericAndTextValues(t *testing.T) {
	for _, test := range []struct {
		input any
		want  bool
	}{{json.Number("1.0"), true}, {json.Number("0.0"), false}, {"t", true}, {"y", true}, {"n", false}, {"f", false}, {true, true}, {false, false}} {
		got, failure := lifecycleBoolean(test.input)
		if failure != "" || got != test.want {
			t.Fatalf("input=%#v got=%v failure=%s", test.input, got, failure)
		}
	}
	for _, input := range []any{" false ", json.Number("0.5"), []any{false}, map[string]any{}} {
		if _, failure := lifecycleBoolean(input); failure == "" {
			t.Fatalf("invalid input accepted: %#v", input)
		}
	}
}

func TestLegacyBulkNonObjectBodyFailsBeforeProjectLookup(t *testing.T) {
	for _, body := range []string{`null`, `[]`} {
		db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*bool) = true; return nil })}
		repository, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
		if err != nil {
			t.Fatal(err)
		}
		_, err = repository.ExecuteLifecycle(context.Background(), LifecycleCommand{Operation: "legacy-create-members", Method: "POST", Body: strings.NewReader(body), ContentType: "application/json"})
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != 500 {
			t.Fatalf("body=%s error=%v", body, err)
		}
		if !strings.Contains(db.query, "FROM project_members pm") {
			t.Fatalf("body=%s performed a project lookup before request.data.get failed: %s", body, db.query)
		}
	}
}
