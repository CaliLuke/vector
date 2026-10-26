//go:build integration

package assets

import (
	"context"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type draftAssetFixture struct {
	t                                       *testing.T
	ctx                                     context.Context
	db                                      *pgx.Conn
	repository                              *PostgresRepository
	workspace, actor, other, draft, project uuid.UUID
}

func newDraftAssetFixture(t *testing.T) draftAssetFixture {
	t.Helper()
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, url string) error {
		db, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
		_, err = db.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY, slug text DEFAULT 'drafts', deleted_at timestamptz);
CREATE TABLE workspace_members(workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
CREATE TABLE project_members(workspace_id uuid,project_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,guest_view_all_features boolean DEFAULT false);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,created_by_id uuid,deleted_at timestamptz,is_draft boolean DEFAULT false);
CREATE TABLE issue_comments(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,issue_id uuid,created_by_id uuid,deleted_at timestamptz);
CREATE TABLE pages(id uuid PRIMARY KEY,workspace_id uuid,owned_by_id uuid,access integer,deleted_at timestamptz);
CREATE TABLE project_pages(page_id uuid,project_id uuid,deleted_at timestamptz);
CREATE TABLE draft_issues(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,created_by_id uuid,deleted_at timestamptz);
CREATE TABLE file_assets(
 id uuid PRIMARY KEY,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),
 attributes jsonb DEFAULT '{}',asset text DEFAULT 'image.png',workspace_id uuid,project_id uuid,
 user_id uuid,issue_id uuid,comment_id uuid,page_id uuid,draft_issue_id uuid REFERENCES draft_issues(id),
 entity_type text,entity_identifier text,created_by_id uuid,updated_by_id uuid,
 size double precision DEFAULT 10,is_uploaded boolean DEFAULT true,is_deleted boolean DEFAULT false,
 is_archived boolean DEFAULT false,storage_metadata jsonb DEFAULT '{}',deleted_at timestamptz,external_id text,external_source text
);
CREATE TABLE work_item_effects(id uuid PRIMARY KEY,kind text NOT NULL,payload jsonb NOT NULL);`)
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
	t.Cleanup(func() { _ = db.Close(ctx) })
	repository, err := NewPostgresRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	f := draftAssetFixture{t: t, ctx: ctx, db: db, repository: repository, workspace: uuid.New(), actor: uuid.New(), other: uuid.New(), draft: uuid.New(), project: uuid.New()}
	f.exec(`INSERT INTO workspaces(id) VALUES($1)`, f.workspace)
	f.exec(`INSERT INTO workspace_members VALUES($1,$2,15,true,NULL),($1,$3,20,true,NULL)`, f.workspace, f.actor, f.other)
	f.exec(`INSERT INTO project_members VALUES($1,$2,$3,15,true,NULL),($1,$2,$4,20,true,NULL)`, f.workspace, f.project, f.actor, f.other)
	f.exec(`INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)`, f.draft, f.workspace, f.actor)
	return f
}
func (f draftAssetFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(f.ctx, query, args...); err != nil {
		f.t.Fatal(err)
	}
}
func (f draftAssetFixture) asset(kind string, actor uuid.UUID) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO file_assets(id,workspace_id,created_by_id,entity_type) VALUES($1,$2,$3,$4)`, id, f.workspace, actor, kind)
	return id
}
func (f draftAssetFixture) assertBound(id uuid.UUID, want bool) {
	f.t.Helper()
	var bound bool
	err := f.db.QueryRow(f.ctx, `SELECT draft_issue_id=$2 AND entity_identifier=$2::uuid::text AND entity_type='DRAFT_ISSUE_DESCRIPTION' AND project_id IS NOT DISTINCT FROM (SELECT project_id FROM draft_issues WHERE id=$2) FROM file_assets WHERE id=$1`, id, f.draft).Scan(&bound)
	if !want {
		err = f.db.QueryRow(f.ctx, `SELECT draft_issue_id IS NOT NULL FROM file_assets WHERE id=$1`, id).Scan(&bound)
	}
	if err != nil || bound != want {
		f.t.Fatalf("asset bound=%v want=%v: %v", bound, want, err)
	}
}

func TestDraftAssetAssociationIsPrivateAtomicAndIdempotent(t *testing.T) {
	f := newDraftAssetFixture(t)
	ordinary, draft := f.asset("ISSUE_DESCRIPTION", f.actor), f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
	foreign := f.asset("DRAFT_ISSUE_DESCRIPTION", f.other)
	assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{ordinary, foreign}), "mixed-owner uploads")
	f.assertBound(ordinary, false)
	assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.other, f.draft, []uuid.UUID{foreign}), "administrator attaching to another user's draft")
	if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{ordinary, draft, ordinary}); err != nil {
		t.Fatal(err)
	}
	f.assertBound(ordinary, true)
	f.assertBound(draft, true)
	if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{ordinary, draft}); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE draft_issues SET deleted_at=now() WHERE id=$1`, f.draft)
	assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{ordinary}), "consumed draft")
}

func TestDraftAssetAssociationAllowsOwnerGuestButRejectsRetargetingAndRevocation(t *testing.T) {
	f := newDraftAssetFixture(t)
	f.exec(`UPDATE workspace_members SET role=5 WHERE member_id=$1`, f.actor)
	id := f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
	if err := f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id}); err != nil {
		t.Fatal(err)
	}
	second := uuid.New()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)`, second, f.workspace, f.actor)
	assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, second, []uuid.UUID{id}), "retargeting owned asset")
	f.exec(`UPDATE workspace_members SET is_active=false WHERE member_id=$1`, f.actor)
	assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{id}), "revoked workspace")
	f.assertBound(id, true)
}

func TestDraftAssetAssociationRejectsUnusableUploadsWithoutPartialWrites(t *testing.T) {
	f := newDraftAssetFixture(t)
	for _, tc := range []struct{ name, mutation string }{
		{"missing", "DELETE FROM file_assets WHERE id=$1"},
		{"pending", "UPDATE file_assets SET is_uploaded=false WHERE id=$1"},
		{"deleted", "UPDATE file_assets SET deleted_at=now() WHERE id=$1"},
		{"soft deleted", "UPDATE file_assets SET is_deleted=true WHERE id=$1"},
		{"archived", "UPDATE file_assets SET is_archived=true WHERE id=$1"},
		{"foreign workspace", "UPDATE file_assets SET workspace_id=gen_random_uuid() WHERE id=$1"},
		{"bound issue", "UPDATE file_assets SET issue_id=gen_random_uuid() WHERE id=$1"},
		{"bound comment", "UPDATE file_assets SET comment_id=gen_random_uuid() WHERE id=$1"},
		{"bound page", "UPDATE file_assets SET page_id=gen_random_uuid() WHERE id=$1"},
		{"bound user", "UPDATE file_assets SET user_id=gen_random_uuid() WHERE id=$1"},
		{"conflicting identifier", "UPDATE file_assets SET entity_identifier=gen_random_uuid()::text WHERE id=$1"},
		{"wrong type", "UPDATE file_assets SET entity_type='WORKSPACE_LOGO' WHERE id=$1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			good, bad := f.asset("ISSUE_DESCRIPTION", f.actor), f.asset("DRAFT_ISSUE_DESCRIPTION", f.actor)
			f.exec(tc.mutation, bad)
			assertScopeNotFound(t, f.repository.BulkAssociateDraftAssets(f.ctx, f.workspace, f.actor, f.draft, []uuid.UUID{good, bad}), tc.name)
			f.assertBound(good, false)
		})
	}
}

func TestWorkspaceDraftUploadBindsOwnerWithoutSourceProjectAccess(t *testing.T) {
	f := newDraftAssetFixture(t)
	for _, project := range []*uuid.UUID{nil, &f.project} {
		f.exec(`UPDATE draft_issues SET project_id=$2 WHERE id=$1`, f.draft, project)
		id := uuid.New()
		err := f.repository.CreateWorkspaceEntityUpload(f.ctx, id, f.actor, f.workspace, uuid.Nil, "image.png", []byte(`{}`), 10, "DRAFT_ISSUE_DESCRIPTION", f.draft.String(), false)
		if err != nil {
			t.Fatal(err)
		}
		f.assertBound(id, true)
	}
	assertScopeNotFound(t, f.repository.CreateWorkspaceEntityUpload(f.ctx, uuid.New(), f.other, f.workspace, uuid.Nil, "image.png", []byte(`{}`), 10, "DRAFT_ISSUE_DESCRIPTION", f.draft.String(), false), "foreign draft upload")
	f.exec(`UPDATE workspace_members SET is_active=false WHERE member_id=$1`, f.actor)
	assertScopeNotFound(t, f.repository.CreateWorkspaceEntityUpload(f.ctx, uuid.New(), f.actor, f.workspace, uuid.Nil, "image.png", []byte(`{}`), 10, "DRAFT_ISSUE_DESCRIPTION", f.draft.String(), false), "revoked workspace upload")
}

func TestProjectBulkCannotBypassDraftOwnership(t *testing.T) {
	f := newDraftAssetFixture(t)
	draft, ordinary := f.asset("DRAFT_ISSUE_DESCRIPTION", f.other), f.asset("ISSUE_DESCRIPTION", f.other)
	assertScopeNotFound(t, f.repository.BulkAssociateProjectAssets(f.ctx, f.workspace, f.project, f.other, f.draft, []uuid.UUID{draft, ordinary}), "old project bulk draft destination")
	f.assertBound(draft, false)
	f.assertBound(ordinary, false)
}
