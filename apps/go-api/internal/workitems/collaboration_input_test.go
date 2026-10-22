package workitems

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
)

type collaborationAuthorizationDB struct {
	Database
	queries int
	args    []any
}

func (db *collaborationAuthorizationDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	db.queries++
	db.args = append([]any(nil), args...)
	return collaborationAuthorizationRow{}
}

type collaborationAuthorizationRow struct{}

func (collaborationAuthorizationRow) Scan(dest ...any) error {
	*dest[0].(*bool) = true
	return nil
}

func TestCollaborationFiltersRejectMalformedUUIDWrapper(t *testing.T) {
	q := Request{
		Params:  map[string]string{"issue__id": "x123e4567-e89b-12d3-a456-426614174000y"},
		Present: map[string]bool{"issue__id": true},
	}
	args := []any{}
	if _, err := collaborationFilters(q, &args); err == nil {
		t.Fatal("malformed UUID wrapper was accepted as a collaboration filter")
	}
	if len(args) != 0 {
		t.Fatalf("invalid UUID reached SQL arguments: %#v", args)
	}
}

func TestCollaborationFiltersAcceptDjangoUUIDForms(t *testing.T) {
	for _, value := range []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"{123e4567-e89b-12d3-a456-426614174000}",
		"urn:uuid:123e4567-e89b-12d3-a456-426614174000",
	} {
		q := Request{Params: map[string]string{"issue__id": value}, Present: map[string]bool{"issue__id": true}}
		args := []any{}
		if _, err := collaborationFilters(q, &args); err != nil {
			t.Fatalf("Django UUID form %q rejected: %v", value, err)
		}
		if len(args) != 1 || args[0] != "123e4567-e89b-12d3-a456-426614174000" {
			t.Fatalf("Django UUID form %q produced SQL args %#v", value, args)
		}
	}
}

func TestCollaborationAuthorizationRejectsMalformedUUIDBeforeDatabase(t *testing.T) {
	db := &collaborationAuthorizationDB{}
	repository := &Repository{db: db}
	err := repository.AuthorizeCollaborationInitial(context.Background(), Request{
		API: true, Method: "GET", Operation: "get_api_v1_workspaces_projects_issues_comments",
		ProjectID: "x123e4567-e89b-12d3-a456-426614174000y", Slug: "demo", ActorID: "user",
	})
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("malformed UUID authorization error = %v, want validation response", err)
	}
	if db.queries != 0 {
		t.Fatalf("malformed UUID reached authorization database query (%d queries)", db.queries)
	}
}

func TestCollaborationAuthorizationCanonicalizesDjangoUUIDForms(t *testing.T) {
	for _, value := range []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"{123e4567-e89b-12d3-a456-426614174000}",
		"urn:uuid:123e4567-e89b-12d3-a456-426614174000",
	} {
		db := &collaborationAuthorizationDB{}
		repository := &Repository{db: db}
		err := repository.AuthorizeCollaborationInitial(context.Background(), Request{
			API: true, Method: "GET", Operation: "get_api_v1_workspaces_projects_issues_comments",
			ProjectID: value, Slug: "demo", ActorID: "user",
		})
		if err != nil {
			t.Fatalf("Django UUID form %q authorization failed: %v", value, err)
		}
		if db.queries != 1 || len(db.args) < 2 || db.args[1] != "123e4567-e89b-12d3-a456-426614174000" {
			t.Fatalf("Django UUID form %q reached authorization with %#v", value, db.args)
		}
	}
}

func TestCollaborationURLValidation(t *testing.T) {
	for _, tc := range []struct {
		url   string
		valid bool
	}{
		{"https://example.com/" + strings.Repeat("a", 220), true},
		{"https://example.com/" + strings.Repeat("a", 2028), true},
		{"https://example.com/" + strings.Repeat("a", 2029), false},
		{"http://localhost:8000/path", true},
		{"HTTPS://EXAMPLE.COM/path", true},
		{"http://example.com:99999/path", true},
		{"http://example.com:999999/path", false},
		{"http://bad..example.com/path", false},
		{"http://-bad.example.com/path", false},
		{"http://bad-.example.com/path", false},
		{"http://" + strings.Repeat("a", 64) + ".com/path", false},
		{"http://example.c/path", false},
		{"http://example.123/path", false},
		{"http://example.xn--p1ai/path", true},
		{"http://例子.中国/path", true},
		{"http://example.com./path", true},
		{"http://127.0.0.1/path", true},
		{"http://127.00.0.1/path", false},
		{"http://[2001:db8::1]:8000/path", true},
		{"http://[2001::db8::1]/path", false},
		{"http://example.com/a\u00a0b", false},
		{"http://example.com/a\x1cb", false},
		{"http://example.com/%", true},
	} {
		t.Run(tc.url, func(t *testing.T) {
			_, err := (&Repository{}).validateCollaboration(context.Background(), Request{API: true, Method: "PATCH"}, djangoinput.Input{Fields: map[string]any{"url": tc.url}}, "issue_links")
			if got := err == nil; got != tc.valid {
				t.Fatalf("API link PATCH valid=%v, want %v: %v", got, tc.valid, err)
			}
		})
	}
}

func TestCollaborationTextLimitsFollowModelFields(t *testing.T) {
	r := &Repository{}
	for _, table := range []string{"issue_reactions", "comment_reactions"} {
		value := strings.Repeat("a", 256)
		values, err := r.validateCollaboration(context.Background(), Request{Method: "POST"}, djangoinput.Input{Fields: map[string]any{"reaction": value}}, table)
		if err != nil || values["reaction"] != value {
			t.Fatalf("%s rejected a TextField value: %v", table, err)
		}
	}
	for _, api := range []bool{false, true} {
		for _, method := range []string{"POST", "PATCH"} {
			value := "https://example.com/" + strings.Repeat("a", 220)
			values, err := r.validateCollaboration(context.Background(), Request{API: api, Method: method}, djangoinput.Input{Fields: map[string]any{"url": value}}, "issue_links")
			if err != nil || values["url"] != value {
				t.Fatalf("API=%v method=%s rejected a valid long URL: %v", api, method, err)
			}
		}
	}
	_, err := r.validateCollaboration(context.Background(), Request{Method: "PATCH"}, djangoinput.Input{Fields: map[string]any{"title": strings.Repeat("a", 256)}}, "issue_links")
	if err == nil {
		t.Fatal("link title CharField length limit was lost")
	}
	_, err = r.validateCollaboration(context.Background(), Request{Method: "PATCH"}, djangoinput.Input{Fields: map[string]any{"attachments": []any{"https://example.com/" + strings.Repeat("a", 182)}}}, "issue_comments")
	if err == nil {
		t.Fatal("attachment URLField length limit was lost")
	}
}
