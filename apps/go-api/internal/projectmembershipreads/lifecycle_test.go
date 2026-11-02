package projectmembershipreads

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
)

type lifecycleTaskSpy struct{ calls int }

func (s *lifecycleTaskSpy) Task(context.Context, string, []any, map[string]any) error {
	s.calls++
	return nil
}

func TestLifecyclePermissionsPrecedeRateAndNeverRequireWorkspaceMembershipForAPIKeyReads(t *testing.T) {
	db := &databaseStub{row: rowFunc(func(dest ...any) error { *dest[0].(*bool) = false; return nil })}
	repo := &LifecycleRepository{Repository: &Repository{database: db}, reader: db}
	command := LifecycleCommand{Operation: "v1-list-members", Method: "GET", UserID: "user", Slug: "demo", ProjectID: "project"}
	err := repo.AuthorizeLifecycle(context.Background(), command)
	var denied *apicontract.HTTPError
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("denied=%v", err)
	}
	// The source permission accepts a project member even when their workspace
	// membership is inactive; passing false above solely represents project denial.
	if len(db.args) != 4 || db.args[3] != true {
		t.Fatalf("read permission args=%#v", db.args)
	}
	command.Method = "HEAD"
	err = repo.AuthorizeLifecycle(context.Background(), command)
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("HEAD permission denial=%v", err)
	}
	if db.args[3] != false {
		t.Fatalf("collection HEAD must use source admin permission: %#v", db.args)
	}
	command.Operation = "v1-list-members-lite"
	err = repo.AuthorizeLifecycle(context.Background(), command)
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("lite HEAD permission denial=%v", err)
	}
	if db.args[3] != true {
		t.Fatalf("lite HEAD must retain SAFE_METHOD membership permission: %#v", db.args)
	}
}

func TestLifecycleReplicaRoutingKeepsWritesOnPrimary(t *testing.T) {
	row := func(dest ...any) error { *dest[0].(*bool) = true; return nil }
	reader := &databaseStub{row: rowFunc(row)}
	writer := &databaseStub{row: rowFunc(row)}
	repository := &LifecycleRepository{Repository: &Repository{database: writer}, reader: reader}
	command := LifecycleCommand{Operation: "v1-list-members", Method: "GET", UserID: "user", Slug: "demo", ProjectID: "project"}
	if err := repository.AuthorizeLifecycle(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if reader.query == "" || writer.query != "" {
		t.Fatalf("GET reader=%q writer=%q", reader.query, writer.query)
	}
	reader.query = ""
	command.Method = "POST"
	command.Operation = "v1-create-members"
	if err := repository.AuthorizeLifecycle(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if writer.query == "" || reader.query != "" {
		t.Fatalf("POST reader=%q writer=%q", reader.query, writer.query)
	}
}
