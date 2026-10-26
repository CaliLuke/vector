//go:build integration

package projectconfiguration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

// This fixture isolates the estimate writer's storage and authorization contract.
// The workitemtypes suite also exercises it against the full migrated schema.
func TestEstimateDeletionRejectsForeignReplacementAndCommitsEffectsWithValues(t *testing.T) {
	ctx := t.Context()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
		_, err = conn.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid);
CREATE TABLE project_members(member_id uuid,project_id uuid,workspace_id uuid,is_active boolean,deleted_at timestamptz,role integer);
CREATE TABLE estimate_points(id uuid PRIMARY KEY,key bigint,value text,description text,estimate_id uuid,project_id uuid,workspace_id uuid,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),created_by_id uuid,updated_by_id uuid,deleted_at timestamptz);
CREATE TABLE issues(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,estimate_point_id uuid,updated_at timestamptz,deleted_at timestamptz);
CREATE TABLE work_item_effects(id uuid PRIMARY KEY,kind text,payload jsonb);
CREATE FUNCTION work_item_schema_lock(workspace uuid) RETURNS void LANGUAGE sql AS $$ SELECT pg_advisory_xact_lock(hashtextextended(workspace::text || ':work-item-schema',0)); $$;
INSERT INTO workspaces VALUES('00000000-0000-4000-8000-000000000001','estimates');
INSERT INTO projects VALUES('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001'),('00000000-0000-4000-8000-000000000008','00000000-0000-4000-8000-000000000001');
INSERT INTO project_members VALUES('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001',true,NULL,20);
INSERT INTO estimate_points(id,key,value,description,estimate_id,project_id,workspace_id) VALUES
('00000000-0000-4000-8000-000000000004',0,'1','','00000000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001'),
('00000000-0000-4000-8000-000000000005',1,'2','','00000000-0000-4000-8000-000000000006','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001'),
('00000000-0000-4000-8000-000000000009',0,'9','','00000000-0000-4000-8000-000000000008','00000000-0000-4000-8000-000000000008','00000000-0000-4000-8000-000000000001');
INSERT INTO issues VALUES('00000000-0000-4000-8000-000000000007','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000004',now(),NULL);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pair.Close() })
	conn, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })
	repo, err := NewRepository(conn)
	if err != nil {
		t.Fatal(err)
	}
	const actor = "00000000-0000-4000-8000-000000000003"
	const project = "00000000-0000-4000-8000-000000000002"
	const estimate = "00000000-0000-4000-8000-000000000006"
	const original = "00000000-0000-4000-8000-000000000004"
	const replacement = "00000000-0000-4000-8000-000000000005"
	t.Run("foreign replacement leaves the current value and effect queue untouched", func(t *testing.T) {
		_, err := repo.DeleteEstimatePoint(ctx, actor, "estimates", project, estimate, original, strings.NewReader(`{"new_estimate_id":"00000000-0000-4000-8000-000000000009"}`), "application/json")
		var failure *HTTPError
		if !errors.As(err, &failure) || failure.Status != 404 {
			t.Fatalf("foreign replacement: %v", err)
		}
		var point string
		var count int
		if err = conn.QueryRow(ctx, `SELECT estimate_point_id::text,(SELECT count(*) FROM work_item_effects) FROM issues`).Scan(&point, &count); err != nil {
			t.Fatal(err)
		}
		if point != original || count != 0 {
			t.Fatalf("rejected replacement changed item/effects: %s/%d", point, count)
		}
	})
	t.Run("accepted replacement commits the new value and recoverable effects", func(t *testing.T) {
		raw, err := repo.DeleteEstimatePoint(ctx, actor, "estimates", project, estimate, original, strings.NewReader(`{"new_estimate_id":"`+replacement+`"}`), "application/json")
		if err != nil {
			t.Fatal(err)
		}
		var points []struct {
			ID  string `json:"id"`
			Key int    `json:"key"`
		}
		if err = json.Unmarshal(raw, &points); err != nil || len(points) != 1 || points[0].ID != replacement || points[0].Key != 0 {
			t.Fatalf("remaining points: %s, %v", raw, err)
		}
		var point string
		var deleted bool
		var effects []byte
		var effectCount int
		if err = conn.QueryRow(ctx, `SELECT estimate_point_id::text,(SELECT deleted_at IS NOT NULL FROM estimate_points WHERE id=$1),(SELECT jsonb_object_agg(kind,payload) FROM work_item_effects),(SELECT count(*) FROM work_item_effects) FROM issues`, original).Scan(&point, &deleted, &effects, &effectCount); err != nil {
			t.Fatal(err)
		}
		var payloads map[string]map[string]any
		if err = json.Unmarshal(effects, &payloads); err != nil {
			t.Fatal(err)
		}
		activity := payloads["activity"]
		if point != replacement || !deleted || len(payloads) != 2 || effectCount != 2 || payloads["soft_delete"]["id"] != original || activity["actor_id"] != actor || activity["project_id"] != project || activity["issue_id"] != "00000000-0000-4000-8000-000000000007" || activity["type"] != "issue.activity.updated" {
			t.Fatalf("incomplete committed change: point=%s deleted=%v effects=%s", point, deleted, effects)
		}
		for field, expected := range map[string]string{"requested_data": replacement, "current_instance": original} {
			text, ok := activity[field].(string)
			if !ok {
				t.Fatalf("activity %s is not serialized JSON: %s", field, effects)
			}
			var value map[string]string
			if err = json.Unmarshal([]byte(text), &value); err != nil || len(value) != 1 || value["estimate_point"] != expected {
				t.Fatalf("activity %s lost former/new value: %s, %v", field, text, err)
			}
		}
	})
}
