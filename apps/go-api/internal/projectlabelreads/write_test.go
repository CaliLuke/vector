package projectlabelreads

import (
	"context"
	"errors"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	writeProject = "11111111-1111-4111-8111-111111111111"
	writeID      = "22222222-2222-4222-8222-222222222222"
	writeParent  = "33333333-3333-4333-8333-333333333333"
)

type parentWriteDatabase struct {
	label   Label
	queries []string
	args    [][]any
	row     func(string, []any) pgx.Row
	saved   bool
}

type writeTransaction struct {
	pgx.Tx
	database *parentWriteDatabase
}

func (db *parentWriteDatabase) Begin(context.Context) (pgx.Tx, error) {
	return &writeTransaction{database: db}, nil
}

func (tx *writeTransaction) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return tx.database.QueryRow(ctx, query, args...)
}

func (*writeTransaction) Commit(context.Context) error   { return nil }
func (*writeTransaction) Rollback(context.Context) error { return nil }

func (db *parentWriteDatabase) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queries = append(db.queries, query)
	db.args = append(db.args, args)
	if db.row != nil {
		return db.row(query, args)
	}
	switch {
	case strings.Contains(query, "FOR UPDATE OF project"):
		return stringRow("workspace")
	case strings.Contains(query, "FOR KEY SHARE"):
		if args[0] == writeParent {
			return stringRow(writeParent)
		}
		return rowFunc(func(...any) error { return pgx.ErrNoRows })
	case strings.Contains(query, "WITH RECURSIVE ancestry"):
		return boolPairRow(false, false)
	case strings.Contains(query, "SELECT EXISTS"):
		return boolRow(false)
	case strings.Contains(query, "UPDATE labels SET"):
		db.saved = true
		if strings.Contains(query, "name=$2") {
			db.label.Name = args[1].(string)
			db.label.Color = args[2].(string)
			db.label.Parent = args[3].(*string)
			db.label.SortOrder = args[4].(float64)
		}
		return writeLabelRow(db.label)
	case strings.Contains(query, "INSERT INTO labels"):
		db.saved = true
		project := writeProject
		db.label = Label{ID: args[0].(string), Name: args[1].(string), Color: args[2].(string), Parent: args[3].(*string), ProjectID: &project, WorkspaceID: "workspace", SortOrder: 40010}
		return writeLabelRow(db.label)
	default:
		return writeLabelRow(db.label)
	}
}

func stringRow(value string) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = value
		return nil
	})
}

func boolPairRow(first, second bool) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = first
		*destinations[1].(*bool) = second
		return nil
	})
}

func writeLabelRow(label Label) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = label.ID
		*destinations[1].(*string) = label.Name
		*destinations[2].(*string) = label.Color
		*destinations[3].(**string) = label.Parent
		*destinations[4].(**string) = label.ProjectID
		*destinations[5].(*string) = label.WorkspaceID
		*destinations[6].(*float64) = label.SortOrder
		return nil
	})
}

func existingWriteLabel() Label {
	project := writeProject
	return Label{ID: writeID, Name: "Alpha", Color: "#111111", ProjectID: &project, WorkspaceID: "workspace", SortOrder: 10}
}

func newWriteRepository(t *testing.T, db *parentWriteDatabase) *WriteRepository {
	t.Helper()
	repository, err := NewWriteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func assertWriteHTTPError(t *testing.T, err error, status int, body map[string]any) {
	t.Helper()
	var httpError *HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("expected HTTPError, got %v", err)
	}
	if httpError.Status != status || !reflect.DeepEqual(httpError.Body, body) {
		t.Fatalf("status/body = %d %#v, want %d %#v", httpError.Status, httpError.Body, status, body)
	}
}

func TestPutMissingTargetPrecedesMalformedBody(t *testing.T) {
	db := &parentWriteDatabase{row: func(string, []any) pgx.Row { return rowFunc(func(...any) error { return pgx.ErrNoRows }) }}
	repository := newWriteRepository(t, db)

	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPut, strings.NewReader(`{"name":`), "application/json")

	assertWriteHTTPError(t, err, 404, map[string]any{"detail": "No Label matches the given query."})
	if len(db.queries) != 1 {
		t.Fatalf("queries = %v", db.queries)
	}
}

func TestPatchExactDuplicatePrecedesMissingTarget(t *testing.T) {
	db := &parentWriteDatabase{row: func(query string, _ []any) pgx.Row {
		if strings.Contains(query, "SELECT EXISTS") {
			return boolRow(true)
		}
		return rowFunc(func(...any) error { return pgx.ErrNoRows })
	}}
	repository := newWriteRepository(t, db)

	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(`{"name":"Beta"}`), "application/json")

	assertWriteHTTPError(t, err, 400, map[string]any{"error": "Label with the same name already exists in the project"})
	if len(db.queries) != 1 || !strings.Contains(db.queries[0], "SELECT EXISTS") {
		t.Fatalf("queries = %v", db.queries)
	}
}

func TestPatchOmittedParentPreservesItAndNullClearsIt(t *testing.T) {
	for _, scenario := range []struct {
		name, body string
		cleared    bool
	}{
		{"omitted", `{"color":"#abcdef"}`, false},
		{"null", `{"parent":null}`, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			parent := writeParent
			db := &parentWriteDatabase{label: existingWriteLabel()}
			db.label.Parent = &parent
			repository := newWriteRepository(t, db)
			label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(scenario.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			if scenario.cleared && label.Parent != nil {
				t.Fatalf("parent = %v, want nil", *label.Parent)
			}
			if !scenario.cleared && (label.Parent == nil || *label.Parent != writeParent) {
				t.Fatalf("parent = %v, want %s", label.Parent, writeParent)
			}
			if !db.saved {
				t.Fatal("expected saved update")
			}
		})
	}
}

func TestPatchRejectsParentOutsideProject(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	db.row = func(query string, _ []any) pgx.Row {
		if strings.Contains(query, "FOR UPDATE OF project") {
			return stringRow("workspace")
		}
		if strings.Contains(query, "FOR KEY SHARE") {
			return rowFunc(func(...any) error { return pgx.ErrNoRows })
		}
		return writeLabelRow(db.label)
	}
	repository := newWriteRepository(t, db)

	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(`{"parent":"`+writeParent+`"}`), "application/json")
	assertWriteHTTPError(t, err, 400, map[string]any{"parent": []string{`Invalid pk "` + writeParent + `" - object does not exist.`}})
	if db.saved {
		t.Fatal("out-of-project parent changed the label")
	}
}

func TestPatchNaNSortOrderIsPassedToPersistence(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	repository := newWriteRepository(t, db)

	label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(`{"sort_order":"NaN"}`), "application/json")

	if err != nil {
		t.Fatal(err)
	}
	if !db.saved || !math.IsNaN(label.SortOrder) {
		t.Fatalf("saved=%v sort_order=%v", db.saved, label.SortOrder)
	}
	if !math.IsNaN(db.args[len(db.args)-1][4].(float64)) {
		t.Fatalf("persisted sort_order = %v", db.args[len(db.args)-1][4])
	}
}

func TestPatchPythonFloatSyntaxAndOverflowArePersisted(t *testing.T) {
	for _, scenario := range []struct {
		name, body string
		want       float64
		infinite   bool
	}{
		{"Unicode decimal digits", `{"sort_order":"١٢"}`, 12, false},
		{"digit separator", `{"sort_order":"1_000"}`, 1000, false},
		{"overflow", `{"sort_order":"1e10000"}`, 0, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)

			label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(scenario.body), "application/json")

			if err != nil {
				t.Fatal(err)
			}
			if !db.saved {
				t.Fatal("sort order was not persisted")
			}
			persisted := db.args[len(db.args)-1][4].(float64)
			if scenario.infinite {
				if !math.IsInf(label.SortOrder, 1) || !math.IsInf(persisted, 1) {
					t.Fatalf("result=%v persisted=%v, want +Inf", label.SortOrder, persisted)
				}
			} else if label.SortOrder != scenario.want || persisted != scenario.want {
				t.Fatalf("result=%v persisted=%v, want %v", label.SortOrder, persisted, scenario.want)
			}
		})
	}
}

func TestPatchParentPrimaryKeyCoercionMatchesRecordedCases(t *testing.T) {
	for _, scenario := range []struct{ name, body, want string }{
		{"numeric", `{"parent":123}`, `Invalid pk "123" - object does not exist.`},
		{"boolean", `{"parent":true}`, "Incorrect type. Expected pk value, received bool."},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)

			_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(scenario.body), "application/json")

			assertWriteHTTPError(t, err, 400, map[string]any{"parent": []string{scenario.want}})
			if db.saved {
				t.Fatal("invalid parent changed label state")
			}
		})
	}
}
