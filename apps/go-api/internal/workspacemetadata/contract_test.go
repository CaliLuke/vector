package workspacemetadata

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"uuid"
)

func TestWorkspaceMetadataJSONContractsPreserveSerializerFields(t *testing.T) {
	tests := []struct {
		name   string
		list   func(*Repository) ([]byte, error)
		body   string
		fields []string
	}{
		{
			name: "labels",
			list: func(r *Repository) ([]byte, error) {
				return r.ListLabels(context.Background(), uuid.Nil(), "demo")
			},
			body:   `[{"parent":null,"name":"Bug","color":"#f00","id":"00000000-0000-4000-8000-000000000001","project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","sort_order":10}]`,
			fields: []string{"color", "id", "name", "parent", "project_id", "sort_order", "workspace_id"},
		},
		{
			name: "states",
			list: func(r *Repository) ([]byte, error) {
				return r.ListStates(context.Background(), uuid.Nil(), "demo")
			},
			body:   `[{"id":"00000000-0000-4000-8000-000000000001","project_id":"00000000-0000-4000-8000-000000000002","workspace_id":"00000000-0000-4000-8000-000000000003","name":"Started","color":"#0f0","group":"started","default":false,"description":null,"sequence":1,"order":0.5}]`,
			fields: []string{"color", "default", "description", "group", "id", "name", "order", "project_id", "sequence", "workspace_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &metadataDB{rows: []pgx.Row{metadataJSON(tt.body)}}
			r, err := NewRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			body, err := tt.list(r)
			if err != nil {
				t.Fatal(err)
			}
			var values []map[string]any
			if err := json.Unmarshal(body, &values); err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(values[0]))
			for field := range values[0] {
				got = append(got, field)
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.fields) {
				t.Fatalf("response fields = %v, want %v", got, tt.fields)
			}
		})
	}
}

func TestWorkspaceMetadataAuthorizationRequiresActiveWorkspaceMembership(t *testing.T) {
	db := &metadataDB{rows: []pgx.Row{metadataBool(false)}}
	r, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := r.Authorized(context.Background(), uuid.Nil(), "demo"); err != nil || ok {
		t.Fatalf("Authorized() = %v, %v; denied memberships must not be authorized", ok, err)
	}
	for _, fragment := range []string{
		"workspace_members",
		"wm.member_id=$1",
		"w.slug=$2",
		"wm.is_active=true",
		"wm.deleted_at IS NULL",
		"w.deleted_at IS NULL",
	} {
		if len(db.queries) == 0 || !strings.Contains(db.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}
}
