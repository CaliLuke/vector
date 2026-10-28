//go:build integration

package projectstatereads

import (
	"context"
	"encoding/json"
	"testing"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

const writeTestWorkspaceID = "00000000-0000-4000-8000-000000000001"
const writeTestProjectID = "00000000-0000-4000-8000-000000000002"
const writeTestUserID = "00000000-0000-4000-8000-000000000003"

func TestWriteAPIV1PreservesEmptyAndNullExternalFields(t *testing.T) {
	ctx := context.Background()
	connection := projectStateWriteDatabase(t, ctx)
	repository, err := NewWriteRepository(connection)
	if err != nil {
		t.Fatal(err)
	}

	input := StateWriteInput{
		UserID: writeTestUserID, Slug: "demo", ProjectID: writeTestProjectID,
		Method: "POST", Fields: map[string]any{
			"name": "External", "color": "#60646C", "external_id": "", "external_source": "",
		},
	}
	created, err := repository.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	stateID := stateWriteResponseID(t, created)
	assertStateWriteExternalValues(t, created, "", "")
	assertPersistedStateWriteExternalValues(t, ctx, connection, stateID, "", "")

	input.Method = "PATCH"
	input.StateID = stateID
	input.Fields = map[string]any{"external_id": nil, "external_source": nil}
	nulled, err := repository.Update(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	assertStateWriteExternalValues(t, nulled, nil, nil)
	assertPersistedStateWriteExternalValues(t, ctx, connection, stateID, "<NULL>", "<NULL>")

	input.Fields = map[string]any{"external_id": "", "external_source": ""}
	emptied, err := repository.Update(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	assertStateWriteExternalValues(t, emptied, "", "")
	assertPersistedStateWriteExternalValues(t, ctx, connection, stateID, "", "")
}

func TestMarkDefaultAllowsOrdinaryGroupStateWithTriageFlag(t *testing.T) {
	ctx := context.Background()
	connection := projectStateWriteDatabase(t, ctx)
	_, err := connection.Exec(ctx, `INSERT INTO states
 (id,project_id,workspace_id,name,color,slug,"group","default",description,sequence,is_triage,created_at,updated_at)
 VALUES
 ('00000000-0000-4000-8000-000000000010',$1,$2,'Backlog','#60646C','backlog','backlog',true,'',100,false,now(),now()),
 ('00000000-0000-4000-8000-000000000011',$1,$2,'Triaged started','#60646C','triaged-started','started',false,'',200,true,now(),now()),
 ('00000000-0000-4000-8000-000000000012',$1,$2,'Triage group','#60646C','triage-group','triage',true,'',300,true,now(),now())`, writeTestProjectID, writeTestWorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := NewWriteRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	if markDefaultErr := repository.MarkDefault(ctx, StateWriteInput{
		UserID: writeTestUserID, Slug: "demo", ProjectID: writeTestProjectID,
		StateID: "00000000-0000-4000-8000-000000000011", Method: "POST", App: true,
	}); markDefaultErr != nil {
		t.Fatalf("MarkDefault returned %v", markDefaultErr)
	}
	rows, err := connection.Query(ctx, `SELECT id::text,"default" FROM states ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	defaults := map[string]bool{}
	for rows.Next() {
		var id string
		var isDefault bool
		if err := rows.Scan(&id, &isDefault); err != nil {
			t.Fatal(err)
		}
		defaults[id] = isDefault
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if defaults["00000000-0000-4000-8000-000000000010"] || !defaults["00000000-0000-4000-8000-000000000011"] || !defaults["00000000-0000-4000-8000-000000000012"] {
		t.Fatalf("defaults after mark-default=%v; ordinary-group target should be enabled while triage-group row remains untouched", defaults)
	}
}

func projectStateWriteDatabase(t *testing.T, ctx context.Context) *pgx.Conn {
	t.Helper()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL UNIQUE);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid NOT NULL, archived_at timestamptz, default_state_id uuid);
CREATE TABLE states (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), project_id uuid NOT NULL, workspace_id uuid NOT NULL,
 name text NOT NULL, color text NOT NULL, slug text NOT NULL, "group" text NOT NULL,
 "default" boolean NOT NULL, description text NOT NULL, sequence double precision NOT NULL,
 is_triage boolean NOT NULL, external_source text NULL, external_id text NULL,
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 created_by_id uuid NULL, updated_by_id uuid NULL, deleted_at timestamptz NULL
);
INSERT INTO workspaces (id,slug) VALUES ('`+writeTestWorkspaceID+`','demo');
INSERT INTO projects (id,workspace_id) VALUES ('`+writeTestProjectID+`','`+writeTestWorkspaceID+`');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	return connection
}

func stateWriteResponseID(t *testing.T, response json.RawMessage) string {
	t.Helper()
	var row map[string]any
	if err := json.Unmarshal(response, &row); err != nil {
		t.Fatal(err)
	}
	id, ok := row["id"].(string)
	if !ok || id == "" {
		t.Fatalf("response has no state id: %s", response)
	}
	return id
}

func assertStateWriteExternalValues(t *testing.T, response json.RawMessage, externalID, externalSource any) {
	t.Helper()
	var row map[string]any
	if err := json.Unmarshal(response, &row); err != nil {
		t.Fatal(err)
	}
	if row["external_id"] != externalID || row["external_source"] != externalSource {
		t.Fatalf("external response values=(%#v,%#v), want (%#v,%#v): %s", row["external_id"], row["external_source"], externalID, externalSource, response)
	}
}

func assertPersistedStateWriteExternalValues(t *testing.T, ctx context.Context, connection *pgx.Conn, stateID, externalID, externalSource string) {
	t.Helper()
	var persistedID, persistedSource string
	if err := connection.QueryRow(ctx, `SELECT COALESCE(external_id,'<NULL>'),COALESCE(external_source,'<NULL>') FROM states WHERE id=$1`, stateID).Scan(&persistedID, &persistedSource); err != nil {
		t.Fatal(err)
	}
	if persistedID != externalID || persistedSource != externalSource {
		t.Fatalf("persisted external values=(%q,%q), want (%q,%q)", persistedID, persistedSource, externalID, externalSource)
	}
}
