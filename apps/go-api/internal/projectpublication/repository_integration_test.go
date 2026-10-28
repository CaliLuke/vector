//go:build integration

package projectpublication

import (
	"context"
	"encoding/json"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

type testPublisher struct {
	called    bool
	model, id string
}

func (p *testPublisher) SoftDelete(_ context.Context, _ string, model, id string) error {
	p.called = true
	p.model = model
	p.id = id
	return nil
}

func TestRepositoryPublicationUpsertSerializesAndSoftDeletesBeforeTask(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text UNIQUE,name text,logo text,deleted_at timestamptz);
CREATE TABLE users(id uuid PRIMARY KEY,user_timezone text);
CREATE TABLE file_assets(id uuid PRIMARY KEY,entity_type text,workspace_id uuid,project_id uuid,issue_id uuid);
ALTER TABLE workspaces ADD COLUMN logo_asset_id uuid;
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,identifier text,name text,cover_image text,cover_image_asset_id uuid,logo_props jsonb,description text,deleted_at timestamptz);
CREATE TABLE workspace_members(id uuid PRIMARY KEY,workspace_id uuid,member_id uuid,is_active boolean,role integer,deleted_at timestamptz);
CREATE TABLE project_members(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,member_id uuid,is_active boolean,role integer,deleted_at timestamptz);
CREATE TABLE deploy_boards(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,entity_name text,entity_identifier uuid,anchor text,is_comments_enabled boolean,is_reactions_enabled boolean,intake_id uuid,is_votes_enabled boolean,view_props jsonb,is_activity_enabled boolean,is_disabled boolean,created_at timestamptz,updated_at timestamptz,created_by_id uuid,updated_by_id uuid,deleted_at timestamptz);
CREATE UNIQUE INDEX deploy_board_active_unique ON deploy_boards(entity_name,entity_identifier) WHERE deleted_at IS NULL;
INSERT INTO users VALUES('00000000-0000-4000-8000-000000000001','UTC');
INSERT INTO workspaces(id,slug,name,logo,deleted_at) VALUES('00000000-0000-4000-8000-000000000010','demo','Demo','',NULL);
INSERT INTO projects(id,workspace_id,identifier,name,cover_image,logo_props,description,deleted_at) VALUES('00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000010','DEMO','Project','cover.png','{}','',NULL);
INSERT INTO workspace_members VALUES('00000000-0000-4000-8000-000000000200','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',true,15,NULL);
INSERT INTO project_members VALUES('00000000-0000-4000-8000-000000000201','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000100','00000000-0000-4000-8000-000000000001',true,20,NULL);`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	publisher := &testPublisher{}
	repository, err := NewRepository(connection, publisher)
	if err != nil {
		t.Fatal(err)
	}
	user, project := "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000100"
	if authorizeErr := repository.Authorize(ctx, user, "POST", "demo", project); authorizeErr != nil {
		t.Fatalf("workspace member create permission: %v", authorizeErr)
	}
	board, err := repository.Create(ctx, user, "demo", project, CreateInput{Comments: true, Views: map[string]any{"list": true}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(board, &decoded); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if decoded["entity_name"] != "project" || decoded["is_comments_enabled"] != true || decoded["view_props"].(map[string]any)["list"] != true {
		t.Fatalf("serialized board = %s", board)
	}
	id := decoded["id"].(string)
	board, err = repository.Create(ctx, user, "demo", project, CreateInput{Votes: true, Views: map[string]any{"kanban": true}})
	if err != nil {
		t.Fatal(err)
	}
	if unmarshalErr := json.Unmarshal(board, &decoded); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if decoded["id"] != id {
		t.Fatalf("get_or_create allocated a second board: %s", board)
	}
	board, err = repository.Update(ctx, user, "demo", project, id, Patch{Fields: map[string]any{"is_comments_enabled": false, "view_props": map[string]any{"calendar": true}}})
	if err != nil {
		t.Fatalf("update board: %v", err)
	}
	if err := json.Unmarshal(board, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["is_comments_enabled"] != false || decoded["view_props"].(map[string]any)["calendar"] != true {
		t.Fatalf("updated board = %s", board)
	}
	if err := repository.Delete(ctx, user, "demo", project, id); err != nil {
		t.Fatal(err)
	}
	if !publisher.called || publisher.model != "deployboard" || publisher.id != id {
		t.Fatalf("delete effect = %#v", publisher)
	}
	var deleted bool
	if err := connection.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM deploy_boards WHERE id=$1::uuid`, id).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("database row was not soft-deleted before task publication")
	}
}
