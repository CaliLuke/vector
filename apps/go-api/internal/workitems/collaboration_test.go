package workitems

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
)

type collaborationQueueSpy struct {
	name   string
	args   []any
	kwargs map[string]any
}

func (p *collaborationQueueSpy) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	p.name, p.args, p.kwargs = name, args, kwargs
	return nil
}
func (*collaborationQueueSpy) SoftDelete(context.Context, string, string, string) error {
	panic("unexpected cleanup")
}
func TestCollaborationPublicCommentReactionRetainsQueuedNoopPayload(t *testing.T) {
	publisher := &collaborationQueueSpy{}
	r := &Repository{publisher: publisher, now: func() time.Time { return time.Unix(123, 0) }}
	q := Request{Operation: "post_api_public_anchor_comments_comment_id_reactions", ProjectID: "board-project", ActorID: "user"}
	err := r.collaborationActivity(context.Background(), q, "commentReaction", "created", `{"reaction": "heart"}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"type": "comment_reaction.activity.created", "requested_data": `{"reaction": "heart"}`, "current_instance": nil, "actor_id": "user", "issue_id": nil, "project_id": "None", "epoch": int64(123)}
	if publisher.name != "plane.bgtasks.issue_activities_task.issue_activity" || !reflect.DeepEqual(publisher.args, []any{}) || !reflect.DeepEqual(publisher.kwargs, want) {
		t.Fatalf("queued activity %#v, want %#v", publisher, want)
	}
	if err := r.collaborationActivity(context.Background(), q, "commentReaction", "deleted", nil, `{"reaction": "heart"}`); err != nil {
		t.Fatal(err)
	}
	if publisher.kwargs["project_id"] != "board-project" {
		t.Fatalf("delete must retain actual board project: %#v", publisher.kwargs)
	}
}
func TestCollaborationCommentSanitizerAlignsAPIAndApp(t *testing.T) {
	for _, api := range []bool{false, true} {
		r := &Repository{}
		q := Request{API: api, Method: "POST"}
		input := djangoinput.Input{Fields: map[string]any{"comment_html": "<p>Safe</p><script>alert(1)</script>"}}
		values, err := r.validateCollaboration(context.Background(), q, input, "issue_comments")
		if err != nil {
			t.Fatal(err)
		}
		if values["comment_html"] != "<p>Safe</p>" {
			t.Fatalf("API=%v persisted HTML=%q", api, values["comment_html"])
		}
		if input.Fields["comment_html"] != "<p>Safe</p><script>alert(1)</script>" {
			t.Fatal("raw webhook payload was sanitized")
		}
	}
}

type collaborationReferenceRow struct{ project, workspace string }

func (row collaborationReferenceRow) Scan(values ...any) error {
	*values[0].(*string) = row.project
	*values[1].(*string) = row.workspace
	return nil
}

// Embedding the database interface makes accidental writes fail immediately.
type collaborationReferenceDB struct {
	Database
	project, workspace string
}

func (db collaborationReferenceDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return collaborationReferenceRow{db.project, db.workspace}
}
func TestCollaborationReferenceRequiresBothProjectAndWorkspace(t *testing.T) {
	for _, tc := range []struct {
		project, workspace string
		allowed            bool
	}{{"project", "workspace", true}, {"foreign-project", "workspace", false}, {"project", "foreign-workspace", false}, {"foreign-project", "foreign-workspace", false}, {"", "workspace", false}} {
		r := &Repository{db: collaborationReferenceDB{project: tc.project, workspace: tc.workspace}}
		for _, table := range []string{"issues", "issue_comments", "descriptions"} {
			err := r.collaborationReference(context.Background(), Request{ProjectID: "project", Slug: "workspace"}, table, "reference-id")
			if tc.allowed {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var denied *djangoapi.HTTPError
				if !errors.As(err, &denied) || denied.Status != 404 {
					t.Fatalf("%s ownership %s/%s allowed: %v", table, tc.project, tc.workspace, err)
				}
			}
		}
	}
}

func TestCollaborationMissingLinkCreatorStaysNullInActivitySnapshot(t *testing.T) {
	r := &Repository{}
	got, err := r.collaborationDump(context.Background(), Request{Method: "PATCH", Timezone: "UTC"}, map[string]any{"created_by_id": nil}, "link", false)
	if err != nil || !strings.Contains(got, `"created_by_detail": null`) {
		t.Fatalf("missing creator activity snapshot=%s error=%v", got, err)
	}
}
func TestCollaborationAPICommentNonObjectFailsBeforeDomainEffects(t *testing.T) {
	r := &Repository{}
	for _, input := range []djangoinput.Input{{Null: true}, {Kind: "list"}} {
		err := r.collaborationExternalConflict(context.Background(), Request{API: true, Operation: "post_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_comments"}, input, nil)
		if err == nil {
			t.Fatal("non-object request reached serializer/domain processing")
		}
		var response *djangoapi.HTTPError
		if errors.As(err, &response) {
			t.Fatalf("source server failure became serializer response: %v", err)
		}
	}
}

func TestCollaborationUpdateRejectsPersistedForeignDescriptionBeforeWrites(t *testing.T) {
	for _, api := range []bool{false, true} {
		repository := &Repository{db: collaborationReferenceDB{project: "foreign-project", workspace: "workspace"}}
		// An omitted issue reference isolates the effective description check;
		// any project lookup, transaction, or effect panics through the embedded DB.
		_, err := repository.writeCollaboration(context.Background(), Request{API: api, Method: "PATCH", ProjectID: "project", Slug: "workspace"}, "comment", "issue_comments", djangoinput.Input{}, map[string]any{"comment_html": "<p>changed</p>"}, map[string]any{"description_id": "foreign-description"})
		var denied *djangoapi.HTTPError
		if !errors.As(err, &denied) || denied.Status != 404 {
			t.Fatalf("API=%v persisted foreign description allowed: %v", api, err)
		}
	}
}
