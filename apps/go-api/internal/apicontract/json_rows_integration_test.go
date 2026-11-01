//go:build integration

package apicontract

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReadJSONRowsPreservesOrderingEmptyArraysAndConnectionOwnership(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, "SELECT 1")
	rows, err := ReadJSONRows(ctx, connection, `SELECT json_build_object('id',value) FROM (VALUES ('b'),('a')) source(value) ORDER BY value`)
	if err != nil {
		t.Fatal(err)
	}
	if want := []map[string]any{{"id": "a"}, {"id": "b"}}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("ordered rows=%#v", rows)
	}
	rows, err = ReadJSONRows(ctx, connection, `SELECT '{}'::json WHERE false`)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(rows)
	if err != nil || string(encoded) != "[]" {
		t.Fatalf("empty API rows=%s error=%v", encoded, err)
	}
	if _, err = ReadJSONRows(ctx, connection, `SELECT value FROM (VALUES ('not-json'),('unread')) source(value)`); err == nil {
		t.Fatal("malformed row accepted")
	}
	var available int
	if err = connection.QueryRow(ctx, "SELECT 1").Scan(&available); err != nil || available != 1 {
		t.Fatalf("row decoding failure retained connection ownership: %v", err)
	}
	if _, err = JSON(ctx, connection, "SELECT 'not-json'::text"); err == nil {
		t.Fatal("single-object read accepted malformed JSON")
	}
	_, queryErr := JSON(ctx, connection, "SELECT 1/0")
	var databaseError *pgconn.PgError
	if !errors.As(queryErr, &databaseError) || databaseError.Code != "22012" {
		t.Fatalf("single-object read lost database failure: %v", queryErr)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = ReadJSONRows(canceled, connection, "SELECT '{}'::json"); !errors.Is(err, context.Canceled) {
		t.Fatalf("query cancellation lost: %v", err)
	}
}
