//go:build integration

package workitems

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"
	"time"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type modelPublisher struct{}

func (modelPublisher) Task(context.Context, string, []any, map[string]any) error { return nil }
func (modelPublisher) SoftDelete(context.Context, string, string, string) error  { return nil }

// The SQL schema is deliberately limited to Issue.save's model dependencies.
// Full Django-schema response/side-effect parity is covered by the HTTP replay.
func TestIssueModelPostgresInvariants(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer conn.Close(ctx)
		_, err = conn.Exec(ctx, `
CREATE FUNCTION work_item_schema_lock(workspace uuid) RETURNS void LANGUAGE sql AS $$
 SELECT pg_advisory_xact_lock(hashtextextended(workspace::text || ':work-item-schema',0));
$$;
CREATE TABLE projects(id uuid PRIMARY KEY, workspace_id uuid NOT NULL, deleted_at timestamptz);
CREATE TABLE states(id uuid PRIMARY KEY,project_id uuid,"group" text,is_triage boolean NOT NULL DEFAULT false,"default" boolean NOT NULL DEFAULT false,sequence double precision,deleted_at timestamptz);
CREATE TABLE issues(
 id uuid PRIMARY KEY, project_id uuid NOT NULL, workspace_id uuid NOT NULL, name text NOT NULL,
 description_html text,description_json jsonb,description_stripped text,description_binary bytea,custom_properties jsonb NOT NULL DEFAULT '{}',
 parent_id uuid,state_id uuid,estimate_point_id uuid,type_id uuid,point integer,priority text,
 start_date date,target_date date,sequence_id integer NOT NULL,sort_order double precision NOT NULL,
 completed_at timestamptz,archived_at date,is_draft boolean,external_id text,external_source text,
 created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,created_by_id uuid,updated_by_id uuid,deleted_at timestamptz
);
-- Issue.save refreshes the current rich-text projection even for untyped items.
CREATE TABLE work_item_properties(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,definition jsonb NOT NULL);
CREATE TABLE work_item_property_search(
 issue_id uuid NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
 property_id uuid NOT NULL REFERENCES work_item_properties(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL,source_hash text NOT NULL CHECK(length(source_hash)=64),search_text text NOT NULL,
 PRIMARY KEY(issue_id,property_id)
);
CREATE TABLE issue_sequences(id uuid PRIMARY KEY,issue_id uuid,project_id uuid,workspace_id uuid,sequence bigint,deleted boolean,created_at timestamptz,updated_at timestamptz,created_by_id uuid,updated_by_id uuid,deleted_at timestamptz);
CREATE TABLE issue_assignees(issue_id uuid,assignee_id uuid,created_at timestamptz,deleted_at timestamptz);
CREATE TABLE labels(id uuid PRIMARY KEY,project_id uuid,workspace_id uuid,deleted_at timestamptz);
CREATE TABLE issue_labels(issue_id uuid,label_id uuid,created_at timestamptz,deleted_at timestamptz);
INSERT INTO projects(id,workspace_id) VALUES
('00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000010'),
('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000010');
INSERT INTO states(id,project_id,"group",is_triage,"default",sequence) VALUES
('00000000-0000-4000-8000-000000000020','00000000-0000-4000-8000-000000000001','triage',true,true,0),
('00000000-0000-4000-8000-000000000021','00000000-0000-4000-8000-000000000001','started',false,false,1),
('00000000-0000-4000-8000-000000000022','00000000-0000-4000-8000-000000000001','backlog',false,true,2),
('00000000-0000-4000-8000-000000000023','00000000-0000-4000-8000-000000000001','completed',false,false,3);
INSERT INTO issue_sequences(id,project_id,sequence,deleted,deleted_at) VALUES
('00000000-0000-4000-8000-000000000030','00000000-0000-4000-8000-000000000001',80,true,NULL),
('00000000-0000-4000-8000-000000000031','00000000-0000-4000-8000-000000000001',999,false,'2020-01-01');
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NewRepository(pool, modelPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	repo.now = func() time.Time { return now }
	const project = "00000000-0000-4000-8000-000000000001"
	const actor = "00000000-0000-4000-8000-000000000011"
	const backlog = "00000000-0000-4000-8000-000000000022"
	const completed = "00000000-0000-4000-8000-000000000023"
	t.Run("concurrent allocation keeps ledger and sort order", func(t *testing.T) {
		const count = 16
		start := make(chan struct{})
		failures := make(chan error, count)
		var writers sync.WaitGroup
		for n := range count {
			writers.Go(func() {
				<-start
				row := defaultIssue(project)
				row["name"] = fmt.Sprintf("Concurrent %d", n)
				row["description_html"] = "<p>A &amp; B</p>"
				_, err := repo.saveIssue(ctx, actor, row, nil)
				failures <- err
			})
		}
		close(start)
		writers.Wait()
		close(failures)
		for err := range failures {
			if err != nil {
				t.Fatal(err)
			}
		}
		var rows, distinct, minSeq, maxSeq, mismatches int
		err := pool.QueryRow(ctx, `SELECT count(*),count(DISTINCT i.sequence_id),min(i.sequence_id),max(i.sequence_id),count(*) FILTER(WHERE s.sequence<>i.sequence_id OR s.workspace_id<>i.workspace_id OR i.sort_order<>65535+(i.sequence_id-81)*10000 OR i.state_id<>$2 OR i.created_by_id<>$3 OR i.updated_by_id IS NOT NULL OR i.description_stripped<>'A & B') FROM issues i JOIN issue_sequences s ON s.issue_id=i.id WHERE i.project_id=$1`, project, backlog, actor).Scan(&rows, &distinct, &minSeq, &maxSeq, &mismatches)
		if err != nil {
			t.Fatal(err)
		}
		if rows != count || distinct != count || minSeq != 81 || maxSeq != 96 || mismatches != 0 {
			t.Fatalf("rows=%d distinct=%d sequences=%d..%d mismatches=%d", rows, distinct, minSeq, maxSeq, mismatches)
		}
	})
	t.Run("completion only changes on state transition", func(t *testing.T) {
		row := defaultIssue(project)
		row["name"] = "Completed"
		row["state_id"] = completed
		saved, err := repo.saveIssue(ctx, actor, row, nil)
		if err != nil {
			t.Fatal(err)
		}
		var completion time.Time
		if err := pool.QueryRow(ctx, `SELECT completed_at FROM issues WHERE id=$1`, saved["id"]).Scan(&completion); err != nil {
			t.Fatal(err)
		}
		if !completion.Equal(now) {
			t.Fatalf("completion=%s; want %s", completion, now)
		}
		now = now.Add(time.Hour)
		changed := maps.Clone(saved)
		changed["name"] = "Name only"
		saved, err = repo.saveIssue(ctx, actor, changed, saved)
		if err != nil {
			t.Fatal(err)
		}
		var touched time.Time
		var updater string
		if err := pool.QueryRow(ctx, `SELECT completed_at,updated_at,updated_by_id FROM issues WHERE id=$1`, saved["id"]).Scan(&completion, &touched, &updater); err != nil {
			t.Fatal(err)
		}
		if !completion.Equal(now.Add(-time.Hour)) || !touched.Equal(now) || updater != actor {
			t.Fatalf("completion=%s touched=%s updater=%s", completion, touched, updater)
		}
		changed = maps.Clone(saved)
		changed["state_id"] = backlog
		saved, err = repo.saveIssue(ctx, actor, changed, saved)
		if err != nil {
			t.Fatal(err)
		}
		if saved["completed_at"] != nil {
			t.Fatalf("reopened completion=%v", saved["completed_at"])
		}
	})
	t.Run("ledger failure rolls back issue and does not consume sequence", func(t *testing.T) {
		var next int64
		if err := pool.QueryRow(ctx, `SELECT max(sequence)+1 FROM issue_sequences WHERE project_id=$1 AND deleted_at IS NULL`, project).Scan(&next); err != nil {
			t.Fatal(err)
		}
		// Fail the second insert inside Issue.save's transaction, after the
		// issue itself has already been inserted.
		if _, err := pool.Exec(ctx, fmt.Sprintf(`ALTER TABLE issue_sequences ADD CONSTRAINT reject_test_sequence CHECK(sequence<>%d)`, next)); err != nil {
			t.Fatal(err)
		}
		row := defaultIssue(project)
		row["name"] = "Ledger failure"
		_, saveErr := repo.saveIssue(ctx, actor, row, nil)
		if _, err := pool.Exec(ctx, `ALTER TABLE issue_sequences DROP CONSTRAINT reject_test_sequence`); err != nil {
			t.Fatal(err)
		}
		if saveErr == nil {
			t.Fatal("expected ledger insert failure")
		}
		var persisted bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issues WHERE id=$1)`, row["id"]).Scan(&persisted); err != nil {
			t.Fatal(err)
		}
		if persisted {
			t.Fatal("issue survived failed ledger insert")
		}
		saved, err := repo.saveIssue(ctx, actor, row, nil)
		if err != nil {
			t.Fatal(err)
		}
		var allocated int64
		if err := pool.QueryRow(ctx, `SELECT sequence_id FROM issues WHERE id=$1`, saved["id"]).Scan(&allocated); err != nil {
			t.Fatal(err)
		}
		if allocated != next {
			t.Fatalf("allocated=%d after rollback; want %d", allocated, next)
		}
	})
	t.Run("no usable state keeps nullable model state", func(t *testing.T) {
		row := defaultIssue("00000000-0000-4000-8000-000000000002")
		row["name"] = "No state"
		row["description_html"] = ""
		saved, err := repo.saveIssue(ctx, actor, row, nil)
		if err != nil {
			t.Fatal(err)
		}
		if saved["state_id"] != nil || saved["completed_at"] != nil || saved["description_stripped"] != nil {
			t.Fatalf("nullable model values=%v", saved)
		}
	})
	t.Run("rejected model update rolls back relationship edits", func(t *testing.T) {
		row := defaultIssue(project)
		row["name"] = "Keep this item"
		saved, saveErr := repo.saveIssue(ctx, actor, row, nil)
		if saveErr != nil {
			t.Fatal(saveErr)
		}
		if _, seedErr := pool.Exec(ctx, `INSERT INTO issue_assignees(issue_id,assignee_id) VALUES($1,$2)`, saved["id"], actor); seedErr != nil {
			t.Fatal(seedErr)
		}
		changed := maps.Clone(saved)
		changed["name"] = nil // The real NOT NULL constraint rejects the model write.
		_, updateErr := repo.updateIssueWithRelations(ctx, actor, changed, saved, issueChanges{assigneesSet: true})
		var constraint *pgconn.PgError
		if !errors.As(updateErr, &constraint) || constraint.Code != "23502" {
			t.Fatalf("want model constraint rejection, got %v", updateErr)
		}
		var name string
		var assignees int
		if readErr := pool.QueryRow(ctx, `SELECT name,(SELECT count(*) FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL) FROM issues WHERE id=$1`, saved["id"]).Scan(&name, &assignees); readErr != nil {
			t.Fatal(readErr)
		}
		if name != "Keep this item" || assignees != 1 {
			t.Fatalf("rejected edit persisted: name=%s assignees=%d", name, assignees)
		}
		changed["name"] = "Retry succeeds"
		if _, retryErr := repo.updateIssueWithRelations(ctx, actor, changed, saved, issueChanges{assigneesSet: true}); retryErr != nil {
			t.Fatal(retryErr)
		}
		if readErr := pool.QueryRow(ctx, `SELECT name,(SELECT count(*) FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL) FROM issues WHERE id=$1`, saved["id"]).Scan(&name, &assignees); readErr != nil {
			t.Fatal(readErr)
		}
		if name != "Retry succeeds" || assignees != 0 {
			t.Fatalf("successful retry not committed: name=%s assignees=%d", name, assignees)
		}
	})
}
