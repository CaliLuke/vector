//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentPublicationCreationAndDraftEditSerializeWithoutResurrection(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, draft, _ := seedPublicationDraft(t, f)
	otherTarget := publicationTarget(t, f)
	pool, err := pgxpool.New(f.ctx, f.db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := workitems.NewRepository(pool, typeTestPublisher{}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Rollback(context.WithoutCancel(f.ctx)) }()
	if _, err = lock.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		kind   string
		result workitems.Result
		err    error
	}
	results := make(chan outcome, 4)
	go func() {
		result, publicationErr := publishDraftForTest(f, repo, draft, f.member, `{}`)
		results <- outcome{"publish", result, publicationErr}
	}()
	// Publication queues first. The later edit must observe its consumed draft,
	// never restore the pre-publication row from a stale read.
	waitForSchemaWaiters(t, f, 1)
	go func() {
		result, publicationErr := publishDraftForTest(f, repo, draft, f.member, `{"project_id":"`+otherTarget+`","label_ids":[],"assignee_ids":[]}`)
		results <- outcome{"publish", result, publicationErr}
	}()
	go func() {
		result, editErr := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues_by_pk", Method: "PATCH", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{"pk": draft}}, strings.NewReader(`{"name":"Stale draft edit"}`), "application/json")
		results <- outcome{"edit", result, editErr}
	}()
	go func() {
		result, createErr := repo.Execute(f.ctx, workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, API: true, Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"name":"Ordinary creation"}`), "application/json")
		results <- outcome{"create", result, createErr}
	}()
	waitForSchemaWaiters(t, f, 4)
	if err = lock.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	var published, ordinary string
	for range 4 {
		select {
		case result := <-results:
			if result.kind == "edit" {
				var failure *djangoapi.HTTPError
				if !errors.As(result.err, &failure) || failure.Status != 404 {
					t.Fatalf("consumed draft edit was not rejected: %v", result.err)
				}
				continue
			}
			if result.err != nil {
				t.Fatalf("%s failed under concurrency: %v", result.kind, result.err)
			}
			id := resultIssueID(t, result.result)
			if result.kind == "create" {
				ordinary = id
			} else if published == "" {
				published = id
			} else if published != id {
				t.Fatal("concurrent publication created duplicate work items")
			}
		case <-f.ctx.Done():
			t.Fatal("concurrent creation/publication deadlocked")
		}
	}
	if ordinary == "" || ordinary == published {
		t.Fatal("ordinary creation did not obtain its own identity")
	}
	var issues int
	var deleted bool
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues),(SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$1)`, draft).Scan(&issues, &deleted); err != nil || issues != 2 || !deleted {
		t.Fatalf("draft resurrected or publication duplicated: issues=%d deleted=%v %v", issues, deleted, err)
	}
}

func waitForSchemaWaiters(t *testing.T, f projectTypeFixture, expected int) {
	t.Helper()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		var count int
		// PostgreSQL caches statistics for a transaction. Refresh observation
		// while retaining the advisory lock that coordinates the writers.
		if _, err := f.db.Exec(f.ctx, `SELECT pg_stat_clear_snapshot()`); err != nil {
			t.Fatal(err)
		}
		if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory' AND pid<>pg_backend_pid()`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == expected {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("wanted %d schema waiters, got %d", expected, count)
		case <-tick.C:
		}
	}
}
