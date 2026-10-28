//go:build integration

package systemapi

import (
	"context"
	"encoding/json"
	"testing"

	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	store "github.com/CaliLuke/go-plane/internal/workspaceanalytics"
	"github.com/jackc/pgx/v5"
)

type allowedWorkspaceAnalyticsRepository struct{ *store.Repository }

func (allowedWorkspaceAnalyticsRepository) Authorized(context.Context, uuid.UUID, string, string) (bool, error) {
	return true, nil
}

func TestWorkspaceAnalyticsUsesIssueAndLabelManagersForProjection(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text NOT NULL);
CREATE TABLE projects(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,deleted_at timestamptz,archived_at timestamptz);
CREATE TABLE states(id uuid PRIMARY KEY,"group" text,name text,color text);
CREATE TABLE estimate_points(id uuid PRIMARY KEY,value text);
CREATE TABLE issues(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,project_id uuid NOT NULL,deleted_at timestamptz,archived_at timestamptz,is_draft boolean,state_id uuid,estimate_point_id uuid,priority text);
CREATE TABLE issue_labels(issue_id uuid NOT NULL,label_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE labels(id uuid PRIMARY KEY,color text,name text);
CREATE TABLE issue_assignees(issue_id uuid NOT NULL,assignee_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE users(id uuid PRIMARY KEY,display_name text,first_name text,last_name text,avatar_asset_id uuid,avatar text);
CREATE TABLE cycle_issues(issue_id uuid NOT NULL,cycle_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE cycles(id uuid PRIMARY KEY,name text);
CREATE TABLE module_issues(issue_id uuid NOT NULL,module_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE modules(id uuid PRIMARY KEY,name text);
INSERT INTO workspaces VALUES ('00000000-0000-4000-8000-000000000101','demo');
INSERT INTO projects(id,workspace_id) VALUES ('00000000-0000-4000-8000-000000000102','00000000-0000-4000-8000-000000000101');
INSERT INTO issues(id,workspace_id,project_id,priority) VALUES
 ('00000000-0000-4000-8000-000000000111','00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000102','high'),
 ('00000000-0000-4000-8000-000000000112','00000000-0000-4000-8000-000000000101','00000000-0000-4000-8000-000000000102','low');
UPDATE issues SET archived_at=now() WHERE id='00000000-0000-4000-8000-000000000112';
INSERT INTO labels VALUES
 ('00000000-0000-4000-8000-000000000121','red','Active issue label'),
 ('00000000-0000-4000-8000-000000000122','blue','Archived issue label');
INSERT INTO issue_labels VALUES
 ('00000000-0000-4000-8000-000000000111','00000000-0000-4000-8000-000000000121',NULL),
 ('00000000-0000-4000-8000-000000000112','00000000-0000-4000-8000-000000000122',NULL);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL testcontainer: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	repository, err := store.NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	service := &workspaceAnalyticsService{repository: allowedWorkspaceAnalyticsRepository{repository}}
	ctx = context.WithValue(ctx, workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{id: uuid.MustParse("00000000-0000-4000-8000-000000000131")})
	result, body, err := service.analytics(ctx, "demo", "labels__id", "issue_count", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	var response map[string]any
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if result.ContentType != "application/json" || response["total"] != float64(1) {
		t.Fatalf("analytics content type/total = %q/%v, want application/json/1", result.ContentType, response["total"])
	}
	extras, ok := response["extras"].(map[string]any)
	if !ok {
		t.Fatalf("extras = %T, want object", response["extras"])
	}
	if extras["state_details"] == nil || len(extras["state_details"].(map[string]any)) != 0 {
		t.Fatalf("unrequested state_details = %#v, want empty object", extras["state_details"])
	}
	labels, ok := extras["label_details"].([]any)
	if !ok || len(labels) != 2 {
		t.Fatalf("label_details = %#v, want both labels from the non-deleted Issue.objects manager", extras["label_details"])
	}
}
