package workspaceexports

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type exportRowFunc func(...any) error

func (row exportRowFunc) Scan(dest ...any) error { return row(dest...) }

type exportDatabase struct {
	queries []string
	args    [][]any
}

func (database *exportDatabase) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, append([]any(nil), args...))
	switch {
	case len(database.queries) == 1:
		return exportRowFunc(func(dest ...any) error {
			*dest[0].(*string) = "00000000-0000-4000-8000-000000000001"
			return nil
		})
	case len(database.queries) == 2:
		return exportRowFunc(func(dest ...any) error {
			*dest[0].(*[]byte) = []byte(`{"id":"export-id"}`)
			return nil
		})
	default:
		return exportRowFunc(func(...any) error { return errors.New("unexpected query") })
	}
}

func TestDjangoUUIDArrayValues(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   any
		want    pgtype.UUID
		wantErr bool
	}{
		{name: "integer becomes UUID integer", value: json.Number("123"), want: pgtype.UUID{Bytes: [16]byte{15: 123}, Valid: true}},
		{name: "UUID string", value: "123e4567-e89b-12d3-a456-426614174000", want: pgtype.UUID{Bytes: [16]byte{0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00}, Valid: true}},
		{name: "null array member", value: nil, want: pgtype.UUID{}},
		{name: "boolean follows Python integer semantics", value: true, want: pgtype.UUID{Bytes: [16]byte{15: 1}, Valid: true}},
		{name: "negative integer rejected", value: json.Number("-1"), wantErr: true},
		{name: "integer overflow rejected", value: json.Number("340282366920938463463374607431768211456"), wantErr: true},
		{name: "fraction rejected", value: json.Number("1.0"), wantErr: true},
		{name: "malformed UUID rejected", value: "not-a-uuid", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := djangoUUIDArrayValue(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr=%v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("UUID = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCreateIssueExportStoresCanonicalUUIDsAndReturnsOriginalTaskValues(t *testing.T) {
	database := &exportDatabase{}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	original := []any{json.Number("123"), "123e4567-e89b-12d3-a456-426614174000", nil}
	_, _, _, queuedProjects, err := repository.CreateIssueExport(context.Background(), "00000000-0000-4000-8000-000000000002", "demo", "csv", original)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(queuedProjects, original) {
		t.Fatalf("task values = %#v, want original %#v", queuedProjects, original)
	}
	if len(database.args) != 2 {
		t.Fatalf("queries=%d, want workspace lookup and insert", len(database.args))
	}
	stored, ok := database.args[1][2].([]pgtype.UUID)
	if !ok {
		t.Fatalf("stored array arg type=%T, want []pgtype.UUID", database.args[1][2])
	}
	want := []pgtype.UUID{
		{Bytes: [16]byte{15: 123}, Valid: true},
		{Bytes: [16]byte{0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3, 0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00}, Valid: true},
		{},
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("database UUID values = %#v, want %#v", stored, want)
	}
}

func TestCreateIssueExportRejectsMalformedUUIDBeforeInsert(t *testing.T) {
	database := &exportDatabase{}
	repository, err := NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := repository.CreateIssueExport(context.Background(), "user", "demo", "csv", []any{"not-a-uuid"}); err == nil {
		t.Fatal("malformed UUID unexpectedly accepted")
	}
	if len(database.queries) != 1 {
		t.Fatalf("queries=%d, want only workspace lookup before validation failure", len(database.queries))
	}
}
