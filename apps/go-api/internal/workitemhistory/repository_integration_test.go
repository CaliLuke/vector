//go:build integration

package workitemhistory

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	historyUserID    = "00000000-0000-4000-8000-000000000001"
	historyWorkspace = "00000000-0000-4000-8000-000000000010"
	historyProject   = "00000000-0000-4000-8000-000000000100"
	historyIssue     = "00000000-0000-4000-8000-000000000101"
	historyVersion   = "00000000-0000-4000-8000-000000000201"
	historyDescVer   = "00000000-0000-4000-8000-000000000202"
	historyActivity  = "00000000-0000-4000-8000-000000000301"
	historyComment   = "00000000-0000-4000-8000-000000000401"
)

func TestRepositoryReadsAllHistoryRoutesAgainstPostgres(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text, name text, logo text, logo_asset_id uuid, deleted_at timestamptz);
CREATE TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, identifier text, name text, cover_image text, cover_image_asset_id uuid, logo_props jsonb, description text, archived_at timestamptz, deleted_at timestamptz, guest_view_all_features boolean);
CREATE TABLE users (id uuid PRIMARY KEY, is_active boolean NOT NULL DEFAULT true, first_name text, last_name text, avatar text, avatar_asset_id uuid, is_bot boolean, display_name text, user_timezone text);
CREATE TABLE file_assets (id uuid PRIMARY KEY, entity_type text, workspace_id uuid, project_id uuid, issue_id uuid);
CREATE TABLE workspace_members (id uuid PRIMARY KEY, workspace_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE project_members (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, member_id uuid, role integer, is_active boolean, deleted_at timestamptz);
CREATE TABLE states (id uuid PRIMARY KEY, "group" text);
CREATE TABLE issues (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, state_id uuid, sequence_id integer, name text, description_json jsonb, description_html text, priority text, start_date date, target_date date, sort_order double precision, is_draft boolean, archived_at timestamptz, deleted_at timestamptz, created_by_id uuid, description_binary bytea, description_stripped text);
CREATE TABLE issue_versions (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, issue_id uuid, parent uuid, state uuid, estimate_point uuid, name text, priority text, start_date date, target_date date, assignees uuid[], sequence_id integer, labels uuid[], sort_order double precision, completed_at timestamptz, archived_at date, is_draft boolean, external_source text, external_id text, type uuid, cycle uuid, modules uuid[], properties jsonb, meta jsonb, last_saved_at timestamptz, owned_by_id uuid, created_at timestamptz, updated_at timestamptz, created_by_id uuid, updated_by_id uuid, deleted_at timestamptz);
CREATE TABLE issue_description_versions (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, issue_id uuid, description_binary bytea, description_html text, description_stripped text, description_json jsonb, last_saved_at timestamptz, owned_by_id uuid, created_at timestamptz, updated_at timestamptz, created_by_id uuid, updated_by_id uuid, deleted_at timestamptz);
CREATE TABLE issue_activities (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, issue_id uuid, issue_comment_id uuid, actor_id uuid, created_by_id uuid, updated_by_id uuid, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz, verb text, field text, old_value text, new_value text, comment text, attachments text[], old_identifier uuid, new_identifier uuid, epoch double precision);
CREATE TABLE issue_comments (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, issue_id uuid, description_id uuid, actor_id uuid, parent_id uuid, created_by_id uuid, updated_by_id uuid, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz, comment_stripped text, comment_json jsonb, comment_html text, attachments text[], access text, external_source text, external_id text, edited_at timestamptz);
CREATE TABLE comment_reactions (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, comment_id uuid, actor_id uuid, created_by_id uuid, updated_by_id uuid, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz, reaction text);
CREATE TABLE intake_issues (id uuid PRIMARY KEY, issue_id uuid, source text, source_email text, extra jsonb, created_at timestamptz, deleted_at timestamptz);

INSERT INTO workspaces(id,slug,name,logo) VALUES ('`+historyWorkspace+`','demo','Demo','');
INSERT INTO projects(id,workspace_id,identifier,name,cover_image,logo_props,description,guest_view_all_features) VALUES ('`+historyProject+`','`+historyWorkspace+`','DEMO','Project','','{}','','false');
INSERT INTO users(id,first_name,last_name,avatar,is_bot,display_name,user_timezone) VALUES ('`+historyUserID+`','Ada','Lovelace','',false,'Ada Lovelace','UTC');
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000502','`+historyWorkspace+`','`+historyUserID+`',15,true);
INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active) VALUES ('00000000-0000-4000-8000-000000000501','`+historyWorkspace+`','`+historyProject+`','`+historyUserID+`',15,true);
INSERT INTO issues(id,workspace_id,project_id,sequence_id,name,description_json,description_html,priority,sort_order,is_draft,created_by_id,description_binary,description_stripped) VALUES ('`+historyIssue+`','`+historyWorkspace+`','`+historyProject+`',7,'Demo issue','{}','<p>Demo</p>','medium',0,false,'`+historyUserID+`',decode('616263','hex'),'Demo');
INSERT INTO issue_versions(id,workspace_id,project_id,issue_id,name,priority,assignees,sequence_id,labels,sort_order,is_draft,modules,properties,meta,last_saved_at,owned_by_id,created_at,updated_at,created_by_id) VALUES ('`+historyVersion+`','`+historyWorkspace+`','`+historyProject+`','`+historyIssue+`','Demo issue','medium','{}',7,'{}',0,false,'{}','{}','{}','2026-10-21T12:00:00Z','`+historyUserID+`','2026-10-21T11:00:00Z','2026-10-21T11:00:00Z','`+historyUserID+`');
INSERT INTO issue_description_versions(id,workspace_id,project_id,issue_id,description_binary,description_html,description_stripped,description_json,last_saved_at,owned_by_id,created_at,updated_at,created_by_id) VALUES ('`+historyDescVer+`','`+historyWorkspace+`','`+historyProject+`','`+historyIssue+`',decode('616263','hex'),'<p>Demo</p>','Demo','{}','2026-10-21T12:00:00Z','`+historyUserID+`','2026-10-21T11:00:00Z','2026-10-21T11:00:00Z','`+historyUserID+`');
INSERT INTO issue_activities(id,workspace_id,project_id,issue_id,actor_id,created_at,updated_at,verb,field,old_value,new_value,comment,attachments,epoch) VALUES ('`+historyActivity+`','`+historyWorkspace+`','`+historyProject+`','`+historyIssue+`','`+historyUserID+`','2026-10-21T10:00:00Z','2026-10-21T10:00:00Z','updated','priority','low','medium','',ARRAY[]::text[],0);
INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id,created_at,updated_at,comment_stripped,comment_json,comment_html,attachments,access) VALUES ('`+historyComment+`','`+historyWorkspace+`','`+historyProject+`','`+historyIssue+`','`+historyUserID+`','2026-10-21T10:01:00Z','2026-10-21T10:01:00Z','Hello','{}','<p>Hello</p>',ARRAY[]::text[],'INTERNAL');
INSERT INTO comment_reactions(id,workspace_id,project_id,comment_id,actor_id,created_at,updated_at,reaction) VALUES ('00000000-0000-4000-8000-000000000402','`+historyWorkspace+`','`+historyProject+`','`+historyComment+`','`+historyUserID+`','2026-10-21T10:02:00Z','2026-10-21T10:02:00Z','👍');`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	repository, err := NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{
		Slug:       "demo",
		ProjectID:  uuid.MustParse(historyProject),
		ResourceID: uuid.MustParse(historyIssue),
		UserID:     uuid.MustParse(historyUserID),
		Timezone:   "America/Los_Angeles",
	}

	testRetainedPropertyReads(t, ctx, connection, request)

	for _, operation := range []string{"meta", "versions", "version_detail", "description_versions", "description_version_detail", "history"} {
		t.Run(operation, func(t *testing.T) {
			request.Operation = operation
			switch operation {
			case "version_detail":
				request.VersionID = uuid.MustParse(historyVersion)
			case "description_version_detail":
				request.VersionID = uuid.MustParse(historyDescVer)
			}
			data, readErr := repository.Read(ctx, request)
			if readErr != nil {
				t.Fatalf("Read(%s): %v", operation, readErr)
			}
			if !json.Valid(data) {
				t.Fatalf("Read(%s) returned invalid JSON: %s", operation, data)
			}
			text := string(data)
			switch operation {
			case "meta":
				if !strings.Contains(text, `"sequence_id": 7`) || !strings.Contains(text, `"project_identifier": "DEMO"`) {
					t.Fatalf("metadata shape: %s", data)
				}
			case "versions", "description_versions":
				for _, key := range []string{`"prev_cursor"`, `"next_cursor"`, `"total_results"`, `"total_pages"`, `"results"`} {
					if !strings.Contains(text, key) {
						t.Fatalf("missing paginator key %s: %s", key, data)
					}
				}
				if strings.Contains(text, `"total_count"`) || strings.Contains(text, `"grouped_by"`) {
					t.Fatalf("non-source paginator keys: %s", data)
				}
			case "version_detail":
				if !strings.Contains(text, `"parent"`) || !strings.Contains(text, `"last_saved_at":"2026-10-21T05:00:00-07:00"`) {
					t.Fatalf("version detail shape/timezone: %s", data)
				}
			case "description_version_detail":
				if !strings.Contains(text, `"description_binary":"YWJj"`) || !strings.Contains(text, `"last_saved_at":"2026-10-21T05:00:00-07:00"`) {
					t.Fatalf("description version detail shape/timezone: %s", data)
				}
			case "history":
				if !strings.Contains(text, `"source_data":null`) || !strings.Contains(text, `"display_name":"Ada Lovelace"`) {
					t.Fatalf("history relation shape: %s", data)
				}
			}
		})
	}

	request.Operation = "history"
	request.ActivityType = stringAddress("issue-comment")
	data, err := repository.Read(ctx, request)
	if err != nil || !strings.Contains(string(data), "Hello") || !strings.Contains(string(data), "👍") {
		t.Fatalf("comment-only history = %s, err = %v", data, err)
	}

	testVisibleGuestHistory(t, ctx, connection, request)
	testHistoryRevocationAtRead(t, ctx, connection, request)
	testHiddenItemHistory(t, ctx, connection, request)
}

// Interpose only after the repository's permission checks, immediately before
// its result statement, so a revoked membership cannot use a stale preflight.
type historyReadBoundary struct {
	database
	beforeResult func()
	called       bool
}

func (b *historyReadBoundary) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if !b.called && strings.HasPrefix(query, "SELECT history_result.data") {
		b.called = true
		b.beforeResult()
	}
	return b.database.QueryRow(ctx, query, args...)
}

func stringAddress(value string) *string { return &value }

func testVisibleGuestHistory(t *testing.T, ctx context.Context, connection *pgx.Conn, request Request) {
	t.Run("authorized guests can read their visible item history", func(t *testing.T) {
		for _, viewAll := range []bool{false, true} {
			tx, err := connection.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, "UPDATE project_members SET role=5"); err != nil {
				t.Fatal(err)
			}
			if viewAll {
				if _, err = tx.Exec(ctx, "UPDATE projects SET guest_view_all_features=true; UPDATE issues SET created_by_id=NULL"); err != nil {
					t.Fatal(err)
				}
			}
			scoped, err := NewRepository(tx)
			if err != nil {
				t.Fatal(err)
			}
			local := request
			local.Operation, local.ActivityType = "history", nil
			result, err := scoped.Read(ctx, local)
			if err != nil || !strings.Contains(string(result), "Hello") {
				t.Errorf("visible guest history = %s, %v", result, err)
			}
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func testHistoryRevocationAtRead(t *testing.T, ctx context.Context, connection *pgx.Conn, request Request) {
	t.Run("membership revoked after preflight hides history", func(t *testing.T) {
		tx, err := connection.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		guarded := &historyReadBoundary{database: tx, beforeResult: func() {
			if _, err = tx.Exec(ctx, "UPDATE workspace_members SET is_active=false"); err != nil {
				t.Fatal(err)
			}
		}}
		scoped, err := NewRepository(guarded)
		if err != nil {
			t.Fatal(err)
		}
		local := request
		local.Operation, local.ActivityType = "history", nil
		result, err := scoped.Read(ctx, local)
		if !guarded.called || !errors.Is(err, ErrNotFound) || result != nil {
			t.Fatalf("revoked read = %s, %v; boundary reached %v", result, err, guarded.called)
		}
	})
}

func testHiddenItemHistory(t *testing.T, ctx context.Context, connection *pgx.Conn, request Request) {
	t.Run("history respects current item visibility", func(t *testing.T) {
		scenarios := []struct{ name, change string }{
			{"restricted guest", "UPDATE project_members SET role=5; UPDATE issues SET created_by_id=NULL"},
			{"inactive workspace member", "UPDATE workspace_members SET is_active=false"},
			{"deleted workspace member", "UPDATE workspace_members SET deleted_at=now()"},
			{"deleted workspace", "UPDATE workspaces SET deleted_at=now()"},
			{"deleted project", "UPDATE projects SET deleted_at=now()"},
			{"deleted item", "UPDATE issues SET deleted_at=now()"},
			{"private draft", "UPDATE issues SET is_draft=true"},
		}
		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				tx, err := connection.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = tx.Rollback(ctx) }()
				if _, err = tx.Exec(ctx, scenario.change); err != nil {
					t.Fatal(err)
				}
				scoped, err := NewRepository(tx)
				if err != nil {
					t.Fatal(err)
				}
				for _, activityType := range []string{"issue-comment", "issue-property"} {
					local := request
					local.Operation, local.ActivityType = "history", &activityType
					result, err := scoped.Read(ctx, local)
					if !errors.Is(err, ErrNotFound) || result != nil {
						t.Errorf("%s exposed %s, error %v", activityType, result, err)
					}
				}
				for _, operation := range []string{"meta", "history", "versions", "version_detail", "description_versions", "description_version_detail"} {
					local := request
					local.Operation = operation
					local.ActivityType = nil
					local.VersionID = uuid.MustParse(historyVersion)
					if operation == "description_version_detail" {
						local.VersionID = uuid.MustParse(historyDescVer)
					}
					result, err := scoped.Read(ctx, local)
					if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrGuestRestricted) {
						t.Errorf("%s exposed %s, error %v", operation, result, err)
					}
				}
			})
		}
	})
}
