//go:build integration

package workspaceprojectreads

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/requestinput"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestNativeProjectJSONKeepsPrecisionThroughPostgresAndActivityEncoding(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, `CREATE TABLE native_json_properties(value jsonb NOT NULL)`)
	input, err := requestinput.Parse(strings.NewReader(`{"icon_prop":{"number":9007199254740993.0,"large":1e300,"label":"Zoë"}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	properties, err := normalizeProjectJSONField(input.Fields["icon_prop"])
	if err != nil {
		t.Fatal(err)
	}
	if _, err = connection.Exec(ctx, `INSERT INTO native_json_properties(value) VALUES($1)`, properties); err != nil {
		t.Fatal(err)
	}
	var number string
	var raw []byte
	if err = connection.QueryRow(ctx, `SELECT value->>'number',value FROM native_json_properties`).Scan(&number, &raw); err != nil {
		t.Fatal(err)
	}
	if number != "9007199254740993.0" {
		t.Fatalf("stored decimal changed: %s", number)
	}
	snapshot, err := projectCurrentInstance(raw)
	if err != nil || !json.Valid([]byte(snapshot)) {
		t.Fatalf("activity snapshot=%q error=%v", snapshot, err)
	}
	if !strings.Contains(snapshot, "9007199254740993.0") || !strings.Contains(snapshot, "Zoë") {
		t.Fatalf("activity snapshot changed stored values: %s", snapshot)
	}
}
