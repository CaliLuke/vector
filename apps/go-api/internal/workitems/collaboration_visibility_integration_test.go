//go:build integration

package workitems

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const childVisibilitySchema = `
CREATE TABLE workspaces(id uuid PRIMARY KEY, slug text, deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY, workspace_id uuid, guest_view_all_features boolean DEFAULT false, deleted_at timestamptz, archived_at timestamptz);
CREATE TABLE workspace_members(id uuid PRIMARY KEY, workspace_id uuid, member_id uuid, role integer, is_active boolean DEFAULT true, deleted_at timestamptz);
CREATE TABLE project_members(id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, member_id uuid, role integer, is_active boolean DEFAULT true, deleted_at timestamptz, created_at timestamptz DEFAULT now());
CREATE TABLE issues(id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, created_by_id uuid, state_id uuid, is_draft boolean DEFAULT false, archived_at timestamptz, deleted_at timestamptz);
CREATE TABLE states(id uuid PRIMARY KEY, "group" text);
CREATE TABLE issue_comments(id uuid PRIMARY KEY, created_at timestamptz DEFAULT now(), updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, comment_stripped text, comment_json jsonb,
 comment_html text, attachments jsonb DEFAULT '[]', access text DEFAULT 'INTERNAL', external_source text, external_id text,
 edited_at timestamptz, description_id uuid, issue_id uuid, actor_id uuid, parent_id uuid);
CREATE TABLE descriptions(id uuid PRIMARY KEY, created_at timestamptz DEFAULT now(), updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, description_html text, description_json jsonb,
 description_stripped text, description_binary bytea);
CREATE TABLE issue_links(id uuid PRIMARY KEY, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, title text, url text, metadata jsonb, issue_id uuid);
CREATE TABLE issue_reactions(id uuid PRIMARY KEY, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, actor_id uuid, issue_id uuid, reaction text);
CREATE TABLE comment_reactions(id uuid PRIMARY KEY, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, actor_id uuid, comment_id uuid, reaction text);
CREATE TABLE issue_relations(id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, issue_id uuid, related_issue_id uuid, relation_type text, created_at timestamptz DEFAULT now(), deleted_at timestamptz);
`

type childVisibilityPublisher struct{ activities, cleanup int }

func (p *childVisibilityPublisher) Task(context.Context, string, []any, map[string]any) error {
	p.activities++
	return nil
}
func (p *childVisibilityPublisher) SoftDelete(context.Context, string, string, string) error {
	p.cleanup++
	return nil
}

type revokeBeforeChildWrite struct{ Database }

func (db revokeBeforeChildWrite) Begin(ctx context.Context) (pgx.Tx, error) {
	if _, err := db.Exec(ctx, `UPDATE project_members SET is_active=false`); err != nil {
		return nil, err
	}
	return db.Database.Begin(ctx)
}

func requireChildHidden(t *testing.T, err error) {
	t.Helper()
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 404 {
		t.Fatalf("want hidden child 404, got %v", err)
	}
}

func rollbackChildVisibilityTx(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		t.Error(rollbackErr)
	}
}

func TestPrivateItemChildrenRequireCurrentParentVisibility(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		db, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := db.Close(context.WithoutCancel(ctx)); closeErr != nil {
				t.Error(closeErr)
			}
		}()
		_, err = db.Exec(ctx, childVisibilitySchema)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	db, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, db, ctx) })
	actor, creator, workspace, project, issue, comment, link, target := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	exec := func(db Database, sql string, args ...any) {
		t.Helper()
		if _, err := db.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(db, `INSERT INTO workspaces(id,slug) VALUES($1,'test')`, workspace)
	exec(db, `INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, project, workspace)
	exec(db, `INSERT INTO workspace_members(id,workspace_id,member_id,role) VALUES($1,$2,$3,15)`, uuid.NewString(), workspace, actor)
	exec(db, `INSERT INTO project_members(id,workspace_id,project_id,member_id,role) VALUES($1,$2,$3,$4,15)`, uuid.NewString(), workspace, project, actor)
	exec(db, `INSERT INTO issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4),($5,$2,$3,$4)`, issue, workspace, project, creator, target)
	exec(db, `INSERT INTO issue_comments(id,workspace_id,project_id,issue_id) VALUES($1,$2,$3,$4)`, comment, workspace, project, issue)
	exec(db, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,metadata) VALUES($1,$2,$3,$4,'https://example.test','{}')`, link, workspace, project, issue)
	exec(db, `INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type) VALUES($1,$2,$3,$4,$5,'relates_to')`, uuid.NewString(), workspace, project, issue, target)
	request := Request{Slug: "test", ProjectID: project, IssueID: issue, ActorID: actor, Method: "GET", Params: map[string]string{}}

	for _, tc := range []struct {
		name, change string
		allowed      bool
	}{
		{"member", `SELECT 1`, true},
		{"restricted guest", `UPDATE project_members SET role=5`, false},
		{"guest creator", `UPDATE project_members SET role=5; UPDATE issues SET created_by_id=(SELECT member_id FROM project_members)`, true},
		{"guest view all", `UPDATE project_members SET role=5; UPDATE projects SET guest_view_all_features=true`, true},
		{"workspace admin with restricted raw guest role", `UPDATE workspace_members SET role=20; UPDATE project_members SET role=5`, false},
		{"revoked project member", `UPDATE project_members SET is_active=false`, false},
		{"revoked workspace member", `UPDATE workspace_members SET is_active=false`, false},
		{"deleted item", `UPDATE issues SET deleted_at=now()`, false},
		{"draft item", `UPDATE issues SET is_draft=true`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer rollbackChildVisibilityTx(t, ctx, tx)
			exec(tx, tc.change)
			repository := &Repository{db: tx}
			result, err := repository.readCollaboration(ctx, request, "link", "issue_links")
			if tc.allowed {
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Body.([]any)) != 1 {
					t.Fatalf("visible links = %#v", result.Body)
				}
			} else {
				requireChildHidden(t, err)
			}
			for _, table := range []string{"issue_reactions", "comment_reactions"} {
				q := request
				q.Method = "POST"
				q.Params = map[string]string{"comment_id": comment}
				row := collaborationDefault(table, q, workspace)
				row["reaction"] = "heart"
				err := writeCollaborationRecord(ctx, tx, q, table, row, true)
				if tc.allowed {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					requireChildHidden(t, err)
				}
				var count int
				if err := tx.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE id=$1`, row["id"]).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if (count == 1) != tc.allowed {
					t.Fatalf("%s persisted=%d allowed=%v", table, count, tc.allowed)
				}
			}
		})
	}
	t.Run("a child moved after lookup cannot be overwritten through its former item", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		exec(tx, `UPDATE issue_links SET issue_id=$1 WHERE id=$2`, target, link)
		row := collaborationDefault("issue_links", request, workspace)
		row["id"], row["issue_id"], row["url"] = link, issue, "https://changed.test"
		requireChildHidden(t, writeCollaborationRecord(ctx, tx, request, "issue_links", row, false))
		var savedParent string
		if err := tx.QueryRow(ctx, `SELECT issue_id::text FROM issue_links WHERE id=$1`, link).Scan(&savedParent); err != nil {
			t.Fatal(err)
		}
		if savedParent != target {
			t.Fatalf("child moved to %s; want %s", savedParent, target)
		}
	})
	t.Run("visible empty collection differs from denied parent", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		exec(tx, `DELETE FROM issue_links`)
		repository := &Repository{db: tx}
		result, err := repository.readCollaboration(ctx, request, "link", "issue_links")
		if err != nil || len(result.Body.([]any)) != 0 {
			t.Fatalf("empty visible collection: %#v, %v", result, err)
		}
		exec(tx, `UPDATE project_members SET role=5`)
		_, err = repository.readCollaboration(ctx, request, "link", "issue_links")
		requireChildHidden(t, err)
	})
	t.Run("deleted parent comment rejects reactions", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		exec(tx, `UPDATE issue_comments SET deleted_at=now()`)
		q := request
		q.Params = map[string]string{"comment_id": comment}
		row := collaborationDefault("comment_reactions", q, workspace)
		requireChildHidden(t, writeCollaborationRecord(ctx, tx, q, "comment_reactions", row, true))
	})
	t.Run("late membership revocation prevents deletion and durable effects", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		publisher := &childVisibilityPublisher{}
		repository := &Repository{db: revokeBeforeChildWrite{tx}, publisher: publisher, now: time.Now}
		q := request
		q.Method = "DELETE"
		q.Params = map[string]string{"pk": link}
		_, err = repository.destroyCollaboration(ctx, q, "link", "issue_links")
		requireChildHidden(t, err)
		var deleted bool
		if err := tx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM issue_links WHERE id=$1`, link).Scan(&deleted); err != nil {
			t.Fatal(err)
		}
		if deleted || publisher.activities != 0 || publisher.cleanup != 0 {
			t.Fatalf("unauthorized effects: deleted=%v publisher=%+v", deleted, publisher)
		}
	})
	t.Run("relations hide unauthorized targets and reject hidden sources", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		exec(tx, `UPDATE project_members SET role=5`)
		exec(tx, `UPDATE issues SET created_by_id=$1 WHERE id=$2`, actor, issue)
		q := request
		q.API = true
		repository := &Repository{db: tx}
		result, err := repository.listRelations(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		for key, rows := range result.Body.(map[string]any) {
			if len(rows.([]any)) != 0 {
				t.Fatalf("hidden relation target leaked: %s=%v", key, rows)
			}
		}
		q.IssueID = target
		_, err = repository.listRelations(ctx, q)
		requireChildHidden(t, err)
	})
	t.Run("comment parent references are live same-issue and visible", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		sameIssueParent, otherIssueParent, deletedParent := uuid.NewString(), uuid.NewString(), uuid.NewString()
		exec(tx, `INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,deleted_at) VALUES($1,$2,$3,$4,NULL),($5,$2,$3,$6,NULL),($7,$2,$3,$4,now())`, sameIssueParent, workspace, project, issue, otherIssueParent, target, deletedParent)
		exec(tx, `UPDATE project_members SET role=5`)
		exec(tx, `UPDATE issues SET created_by_id=$1 WHERE id=$2`, actor, issue)
		repository := &Repository{db: tx}
		if validationErr := repository.validateCollaborationCommentReferences(ctx, request, map[string]any{"parent_id": sameIssueParent}, nil); validationErr != nil {
			t.Fatalf("same-issue visible parent rejected: %v", validationErr)
		}
		for _, parentID := range []string{otherIssueParent, deletedParent} {
			validationErr := repository.validateCollaborationCommentReferences(ctx, request, map[string]any{"parent_id": parentID}, nil)
			requireChildHidden(t, validationErr)
		}
		err = repository.validateCollaborationCommentReferences(ctx, request, map[string]any{"parent_id": sameIssueParent}, map[string]any{"id": sameIssueParent, "description_id": nil})
		requireChildHidden(t, err)
		row := collaborationDefault("issue_comments", request, workspace)
		row["parent_id"] = sameIssueParent
		if err := writeCollaborationRecord(ctx, tx, request, "issue_comments", row, true); err != nil {
			t.Fatalf("atomic valid parent write rejected: %v", err)
		}
		row["parent_id"] = row["id"]
		requireChildHidden(t, writeCollaborationRecord(ctx, tx, request, "issue_comments", row, false))
		row = collaborationDefault("issue_comments", request, workspace)
		row["parent_id"] = otherIssueParent
		requireChildHidden(t, writeCollaborationRecord(ctx, tx, request, "issue_comments", row, true))
	})
	t.Run("comment parent projection masks mismatched deleted and internal public parents", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		sameIssueParent, otherIssueParent, deletedParent, internalParent := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
		exec(tx, `INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,access,deleted_at) VALUES($1,$2,$3,$4,'EXTERNAL',NULL),($5,$2,$3,$6,'EXTERNAL',NULL),($7,$2,$3,$4,'EXTERNAL',now()),($8,$2,$3,$4,'INTERNAL',NULL)`, sameIssueParent, workspace, project, issue, otherIssueParent, target, deletedParent, internalParent)
		repository := &Repository{db: tx}
		row := map[string]any{"parent_id": sameIssueParent}
		parent, err := repository.collaborationCommentParent(ctx, request, row)
		if err != nil || parent["id"] != sameIssueParent {
			t.Fatalf("same-issue parent projection=%v, %v", parent, err)
		}
		for _, parentID := range []string{otherIssueParent, deletedParent} {
			projectedParent, parentErr := repository.collaborationCommentParent(ctx, request, map[string]any{"parent_id": parentID})
			if parentErr != nil || projectedParent != nil {
				t.Fatalf("invalid parent %s projected as %v, %v", parentID, projectedParent, parentErr)
			}
		}
		publicRequest := request
		publicRequest.Params = map[string]string{"anchor": "board"}
		publicRequest.ActorID = ""
		parent, err = repository.collaborationCommentParent(ctx, publicRequest, map[string]any{"parent_id": internalParent})
		if err != nil || parent != nil {
			t.Fatalf("internal public parent projected as %v, %v", parent, err)
		}
		parent, err = repository.collaborationCommentParent(ctx, publicRequest, row)
		if err != nil || parent["id"] != sameIssueParent {
			t.Fatalf("external public parent projection=%v, %v", parent, err)
		}
		if err := repository.validateCollaborationCommentReferences(ctx, publicRequest, row, nil); err != nil {
			t.Fatalf("external public parent write rejected: %v", err)
		}
		requireChildHidden(t, repository.validateCollaborationCommentReferences(ctx, publicRequest, map[string]any{"parent_id": internalParent}, nil))
	})
	t.Run("comment description references remain within live comment scope", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		ownedDescription, deletedDescription, foreignDescription := uuid.NewString(), uuid.NewString(), uuid.NewString()
		foreignProject, foreignWorkspace := uuid.NewString(), uuid.NewString()
		exec(tx, `INSERT INTO workspaces(id,slug) VALUES($1,'foreign')`, foreignWorkspace)
		exec(tx, `INSERT INTO projects(id,workspace_id) VALUES($1,$2)`, foreignProject, foreignWorkspace)
		exec(tx, `INSERT INTO descriptions(id,workspace_id,project_id,deleted_at) VALUES($1,$2,$3,NULL),($4,$2,$3,now()),($5,$6,$7,NULL)`, ownedDescription, workspace, project, deletedDescription, foreignDescription, foreignWorkspace, foreignProject)
		repository := &Repository{db: tx}
		for _, tc := range []struct {
			name string
			id   string
			want bool
		}{{"owned live", ownedDescription, true}, {"deleted", deletedDescription, false}, {"foreign", foreignDescription, false}} {
			t.Run(tc.name, func(t *testing.T) {
				owned, err := repository.collaborationDescriptionOwned(ctx, request, map[string]any{"description_id": tc.id, "project_id": project, "workspace_id": workspace})
				if err != nil || owned != tc.want {
					t.Fatalf("description owned=%v, error=%v; want %v", owned, err, tc.want)
				}
			})
		}
	})
	t.Run("hidden external ID collision returns generic conflict without hidden identity", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer rollbackChildVisibilityTx(t, ctx, tx)
		hiddenComment := uuid.NewString()
		exec(tx, `UPDATE project_members SET role=5`)
		exec(tx, `UPDATE issues SET created_by_id=$1 WHERE id=$2`, actor, issue)
		exec(tx, `INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,external_source,external_id) VALUES($1,$2,$3,$4,'sync','external-1')`, hiddenComment, workspace, project, target)
		query := Request{API: true, Operation: "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments", Slug: "test", ProjectID: project, IssueID: issue, ActorID: actor}
		input := djangoinput.Input{Fields: map[string]any{"external_source": "sync", "external_id": "external-1"}}
		err = (&Repository{db: tx}).collaborationExternalConflict(ctx, query, input, nil)
		var conflict *djangoapi.HTTPError
		if !errors.As(err, &conflict) || conflict.Status != 409 {
			t.Fatalf("expected duplicate conflict, got %v", err)
		}
		if _, leaksID := conflict.Body["id"]; leaksID {
			t.Fatalf("hidden comment identity disclosed: %#v", conflict.Body)
		}
		exec(tx, `UPDATE issues SET created_by_id=$1 WHERE id=$2`, actor, target)
		err = (&Repository{db: tx}).collaborationExternalConflict(ctx, query, input, nil)
		if !errors.As(err, &conflict) {
			t.Fatalf("visible duplicate identity missing: %#v, %v", conflict, err)
		}
		visibleID, ok := conflict.Body["id"].(string)
		if !ok || visibleID != hiddenComment {
			t.Fatalf("visible duplicate identity missing: %#v, %v", conflict, err)
		}
	})
}
