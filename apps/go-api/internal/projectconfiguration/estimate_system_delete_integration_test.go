//go:build integration

package projectconfiguration

import (
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
)

func TestEstimateSystemDeletionPreservesOtherPointerAndCommitsCleanup(t *testing.T) {
	ctx, connection := postgresfixture.OpenSQL(t, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,estimate_id uuid,updated_at timestamptz,updated_by_id uuid);
CREATE TABLE project_members(member_id uuid,project_id uuid,workspace_id uuid,is_active bool,deleted_at timestamptz,role int);
CREATE TABLE estimates(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz,updated_at timestamptz,updated_by_id uuid);
CREATE TABLE work_item_effects(id uuid PRIMARY KEY,kind text,payload jsonb);
CREATE FUNCTION work_item_schema_lock(workspace uuid) RETURNS void LANGUAGE sql AS $$ SELECT pg_advisory_xact_lock(hashtextextended(workspace::text || ':work-item-schema',0)); $$;
INSERT INTO workspaces VALUES('00000000-0000-4000-8000-000000000001','estimates');
INSERT INTO projects(id,workspace_id,estimate_id) VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000004');
INSERT INTO project_members VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',true,NULL,20);
INSERT INTO estimates(id,project_id,workspace_id) VALUES
('00000000-0000-4000-8000-000000000004','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001'),
('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001');
`)
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	const actor = "00000000-0000-4000-8000-000000000003"
	const project = "00000000-0000-4000-8000-000000000002"
	const active = "00000000-0000-4000-8000-000000000004"
	const archived = "00000000-0000-4000-8000-000000000005"
	if err = repository.DeleteEstimate(ctx, actor, "estimates", project, archived); err != nil {
		t.Fatal(err)
	}
	var pointer *string
	var deleted bool
	var payload []byte
	if err = connection.QueryRow(ctx, `SELECT estimate_id::text,(SELECT deleted_at IS NOT NULL FROM estimates WHERE id=$1),(SELECT payload FROM work_item_effects WHERE kind='soft_delete') FROM projects`, archived).Scan(&pointer, &deleted, &payload); err != nil {
		t.Fatal(err)
	}
	if pointer == nil || *pointer != active || !deleted {
		t.Fatalf("archived delete pointer=%v deleted=%v", pointer, deleted)
	}
	if len(payload) == 0 {
		t.Fatal("estimate deletion committed without durable cleanup")
	}
	var effect map[string]string
	if err = json.Unmarshal(payload, &effect); err != nil {
		t.Fatal(err)
	}
	if effect["model"] != "estimate" || effect["id"] != archived || effect["app"] != "db" {
		t.Fatalf("cleanup=%v", effect)
	}
	if _, err = connection.Exec(ctx, `ALTER TABLE work_item_effects ADD CONSTRAINT reject_new_effect CHECK(false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if err = repository.DeleteEstimate(ctx, actor, "estimates", project, active); err == nil {
		t.Fatal("cleanup failure committed estimate deletion")
	}
	if err = connection.QueryRow(ctx, `SELECT estimate_id::text,(SELECT deleted_at IS NOT NULL FROM estimates WHERE id=$1) FROM projects`, active).Scan(&pointer, &deleted); err != nil {
		t.Fatal(err)
	}
	if pointer == nil || *pointer != active || deleted {
		t.Fatalf("rollback pointer=%v deleted=%v", pointer, deleted)
	}
	if _, err = connection.Exec(ctx, `ALTER TABLE work_item_effects DROP CONSTRAINT reject_new_effect`); err != nil {
		t.Fatal(err)
	}
	if err = repository.DeleteEstimate(ctx, actor, "estimates", project, active); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = connection.QueryRow(ctx, `SELECT estimate_id::text,(SELECT count(*) FROM work_item_effects) FROM projects`).Scan(&pointer, &count); err != nil {
		t.Fatal(err)
	}
	if pointer != nil || count != 2 {
		t.Fatalf("selected delete pointer=%v effects=%d", pointer, count)
	}
}
