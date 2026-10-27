package activityreads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type activityRow struct {
	scan func(...any) error
}

func (row activityRow) Scan(destinations ...any) error { return row.scan(destinations...) }

type activityDatabase struct {
	allowed   bool
	body      any
	missing   bool
	queries   []string
	arguments [][]any
}

func (database *activityDatabase) QueryRow(_ context.Context, query string, arguments ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.arguments = append(database.arguments, arguments)
	if strings.HasPrefix(query, "SELECT EXISTS") {
		return activityRow{scan: func(destinations ...any) error {
			*destinations[0].(*bool) = database.allowed
			return nil
		}}
	}
	if database.missing {
		return activityRow{scan: func(...any) error { return pgx.ErrNoRows }}
	}
	return activityRow{scan: func(destinations ...any) error {
		encoded, err := json.Marshal(database.body)
		if err != nil {
			return err
		}
		*destinations[0].(*[]byte) = encoded
		return nil
	}}
}

func TestAPIActivitiesPreserveDjangoListAndDetailContracts(t *testing.T) {
	projectID := uuid.New()
	issueID := uuid.New()
	activityID := uuid.New()
	principal := Principal{UserID: uuid.New(), Authenticated: true, Timezone: "UTC"}

	t.Run("list paginates and keeps source membership duplicates", func(t *testing.T) {
		database := &activityDatabase{allowed: true, body: map[string]any{
			"total_count": 2,
			"results":     []any{map[string]any{"id": activityID.String()}},
		}}
		repository, err := NewRepository(database)
		if err != nil {
			t.Fatal(err)
		}
		request := Request{
			Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities",
			Params: map[string]string{
				"slug": "demo", "project_id": projectID.String(), "issue_id": issueID.String(), "per_page": "1",
			},
			Present: map[string]bool{"per_page": true},
		}
		if err := repository.Authorize(context.Background(), principal, request); err != nil {
			t.Fatal(err)
		}
		result, err := repository.Read(context.Background(), principal, request)
		if err != nil {
			t.Fatal(err)
		}
		page := result.Body.(map[string]any)
		if page["total_count"] != 2 || page["count"] != 1 || page["next_page_results"] != true {
			t.Fatalf("page=%#v", page)
		}
		if len(database.queries) != 2 || !strings.Contains(database.queries[1], "membership.is_active=true") || strings.Contains(database.queries[1], "membership.deleted_at") || !strings.Contains(database.queries[1], "LIMIT $5 OFFSET $6") || !strings.Contains(database.queries[1], "count(*) FROM matching") {
			t.Fatalf("queries=%q", database.queries)
		}
	})

	t.Run("detail missing has endpoint-specific error", func(t *testing.T) {
		database := &activityDatabase{allowed: true, missing: true}
		repository, _ := NewRepository(database)
		request := Request{
			Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities_by_pk",
			Params: map[string]string{
				"slug": "demo", "project_id": projectID.String(), "issue_id": issueID.String(), "pk": activityID.String(),
			},
		}
		if err := repository.Authorize(context.Background(), principal, request); err != nil {
			t.Fatal(err)
		}
		_, err := repository.Read(context.Background(), principal, request)
		var httpError *HTTPError
		if !errors.As(err, &httpError) || httpError.Status != 404 || httpError.Body["message"] != "Activity not found." || httpError.Body["code"] != "NOT_FOUND" {
			t.Fatalf("error=%T %#v", err, err)
		}
	})
}
func TestActivityProtocolAuthorizationGatesProjectMemberRole(t *testing.T) {
	database := &activityDatabase{allowed: true}
	repository, _ := NewRepository(database)
	projectID := uuid.New()
	request := Request{
		Operation:                "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities",
		Params:                   map[string]string{"slug": "demo", "project_id": projectID.String()},
		RequireProjectMemberRole: true,
	}
	if err := repository.Authorize(context.Background(), Principal{UserID: uuid.New(), Authenticated: true}, request); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.queries[0], "NOT $4::boolean OR pm.role IN (15, 20)") {
		t.Fatalf("unsafe protocol authorization lacks member-role gate: %s", database.queries[0])
	}
	if requireMemberRole, ok := database.arguments[0][3].(bool); !ok || !requireMemberRole {
		t.Fatalf("unsafe protocol authorization role requirement=%#v", database.arguments[0][3])
	}
}

func TestUserActivitiesPaginateBeforeJSONAggregation(t *testing.T) {
	database := &activityDatabase{body: map[string]any{
		"total_count": 2,
		"results":     []any{map[string]any{"id": uuid.New().String()}},
	}}
	repository, _ := NewRepository(database)
	result, err := repository.Read(context.Background(), Principal{UserID: uuid.New(), Authenticated: true}, Request{
		Operation: "get_api_users_me_activities",
		Params:    map[string]string{"per_page": "1", "cursor": "1:1:0"},
		Present:   map[string]bool{"per_page": true, "cursor": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	page := result.Body.(map[string]any)
	if page["count"] != 1 || page["prev_page_results"] != true || page["next_page_results"] != false {
		t.Fatalf("page=%#v", page)
	}
	if len(database.queries) != 1 || !strings.Contains(database.queries[0], "LIMIT $2 OFFSET $3") || !strings.Contains(database.queries[0], "count(*) FROM matching") {
		t.Fatalf("queries=%q", database.queries)
	}
}

func TestUserActivitiesPreserveNegativeOnePageFailure(t *testing.T) {
	database := &activityDatabase{body: map[string]any{"total_count": 2, "results": []any{}}}
	repository, _ := NewRepository(database)
	_, err := repository.Read(context.Background(), Principal{UserID: uuid.New(), Authenticated: true}, Request{
		Operation: "get_api_users_me_activities",
		Params:    map[string]string{"per_page": "-1"},
		Present:   map[string]bool{"per_page": true},
	})
	var httpError *HTTPError
	if !errors.As(err, &httpError) || httpError.Status != http.StatusInternalServerError || httpError.Body["error"] != "Something went wrong please try again later" {
		t.Fatalf("error=%T %#v", err, err)
	}
	if len(database.queries) != 0 {
		t.Fatalf("queries=%q", database.queries)
	}
}
func TestNegativePageSizePreservesDjangoFailureOrder(t *testing.T) {
	tests := []struct {
		name       string
		perPage    string
		cursor     string
		wantStatus int
	}{
		{name: "positive page", perPage: "-1", cursor: "1:1:0", wantStatus: http.StatusBadRequest},
		{name: "backward cursor", perPage: "-2", cursor: "1:1:1", wantStatus: http.StatusBadRequest},
		{name: "negative page", perPage: "-2", cursor: "-2:-1:0", wantStatus: http.StatusInternalServerError},
		{name: "outside int64", perPage: "-18446744073709551615", wantStatus: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := map[string]string{"per_page": test.perPage}
			present := map[string]bool{"per_page": true}
			if test.cursor != "" {
				params["cursor"] = test.cursor
				present["cursor"] = true
			}
			_, _, _, err := pageWindow(Request{Params: params, Present: present})
			var httpError *HTTPError
			if !errors.As(err, &httpError) || httpError.Status != test.wantStatus {
				t.Fatalf("error=%T %#v, want status %d", err, err, test.wantStatus)
			}
		})
	}
}

func TestUserActivitiesRejectInvalidPagination(t *testing.T) {
	database := &activityDatabase{body: []any{}}
	repository, _ := NewRepository(database)
	_, err := repository.Read(context.Background(), Principal{UserID: uuid.New(), Authenticated: true}, Request{
		Operation: "get_api_users_me_activities",
		Params:    map[string]string{"per_page": "1001"},
		Present:   map[string]bool{"per_page": true},
	})
	var httpError *HTTPError
	if !errors.As(err, &httpError) || httpError.Status != 400 || httpError.Body["detail"] != "Invalid per_page value. Cannot exceed 1000." {
		t.Fatalf("error=%T %#v", err, err)
	}
}
func TestActivityPaginationMatchesCursorContract(t *testing.T) {
	for cursor, expected := range map[string]int{
		"2:1:1":      1,
		"1:1:1":      1,
		"2.5:1:1":    1,
		"1.e999:0:0": 0,
	} {
		t.Run("valid_"+strings.ReplaceAll(cursor, ":", "_"), func(t *testing.T) {
			page, err := cursorPage(Request{Params: map[string]string{"cursor": cursor}, Present: map[string]bool{"cursor": true}}, 2)
			if err != nil || page != expected {
				t.Fatalf("page=%d error=%v", page, err)
			}
		})
	}
	t.Run("cursor size does not set page size", func(t *testing.T) {
		size, err := pageSize(Request{Params: map[string]string{"cursor": "2:1:0"}, Present: map[string]bool{"cursor": true}})
		if err != nil || size != 1000 {
			t.Fatalf("size=%d error=%v", size, err)
		}
	})
	for _, cursor := range []string{"garbage:0:0", "%ZZ", "", "2:0", "2:page:0", "2:0:previous", "NaN:0:0", "Inf:0:0", "1_.0:0:0", "1e999:0:0"} {
		t.Run("invalid_"+strings.ReplaceAll(cursor, ":", "_"), func(t *testing.T) {
			_, err := cursorPage(Request{Params: map[string]string{"cursor": cursor}, Present: map[string]bool{"cursor": true}}, 1000)
			var httpError *HTTPError
			if !errors.As(err, &httpError) || httpError.Status != 400 {
				t.Fatalf("cursor=%q error=%T %#v", cursor, err, err)
			}
		})
	}
	for cursor, status := range map[string]int{
		"1:9223372036854775808:0":  http.StatusInternalServerError,
		"1:-9223372036854775809:0": http.StatusBadRequest,
	} {
		_, err := cursorPage(Request{Params: map[string]string{"cursor": cursor}, Present: map[string]bool{"cursor": true}}, 1000)
		var httpError *HTTPError
		if !errors.As(err, &httpError) || httpError.Status != status {
			t.Fatalf("cursor=%q error=%T %#v", cursor, err, err)
		}
	}
	limit, page, offset, err := pageWindow(Request{Params: map[string]string{"cursor": "1:1:1"}, Present: map[string]bool{"cursor": true}})
	if err != nil || limit != 1000 || page != 1 || offset != 1000 {
		t.Fatalf("backward default-limit window=(%d,%d,%d), error=%v", limit, page, offset, err)
	}
	for raw, expected := range map[string]int{" 1_0 ": 10, "١٠": 10} {
		size, err := pageSize(Request{Params: map[string]string{"per_page": raw}, Present: map[string]bool{"per_page": true}})
		if err != nil || size != expected {
			t.Fatalf("per_page=%q size=%d error=%v", raw, size, err)
		}
	}
	for _, raw := range []string{"", "%ZZ"} {
		_, err = pageSize(Request{Params: map[string]string{"per_page": raw}, Present: map[string]bool{"per_page": true}})
		var httpError *HTTPError
		if !errors.As(err, &httpError) || httpError.Status != 400 {
			t.Fatalf("per_page=%q error=%T %#v", raw, err, err)
		}
	}
}

func TestActivitySerializerExpandsOnlyExactSelectedFields(t *testing.T) {
	serializer := activityAPISerializer(Request{Params: map[string]string{"expand": "actor, workspace,unknown"}})
	if !strings.Contains(serializer, activityAPIJSON) || !strings.Contains(serializer, "jsonb_build_object('actor',") || strings.Contains(serializer, "jsonb_build_object('workspace',") {
		t.Fatalf("serializer=%s", serializer)
	}
	collision := activityAPISerializer(Request{Params: map[string]string{"expand": "attachments,issue_comment"}})
	if !strings.Contains(collision, "jsonb_build_object('attachments',NULL)") || !strings.Contains(collision, "jsonb_build_object('issue_comment',a.issue_comment_id)") {
		t.Fatalf("colliding expansion serializer=%s", collision)
	}
	legacyAvatarFallback := "THEN NULLIF(u.avatar,'') ELSE"
	if !strings.Contains(activityActorExpansion, legacyAvatarFallback) || !strings.Contains(activityJSON, legacyAvatarFallback) {
		t.Fatalf("missing legacy actor avatar fallback: api=%s user=%s", activityActorExpansion, activityJSON)
	}
	for name, query := range map[string]string{"actor": activityActorExpansion, "project": activityProjectExpansion, "user": activityJSON} {
		if !strings.Contains(query, "asset.entity_type='ISSUE_ATTACHMENT'") || !strings.Contains(query, "'ISSUE_DESCRIPTION','COMMENT_DESCRIPTION','PAGE_DESCRIPTION','DRAFT_ISSUE_DESCRIPTION'") || !strings.Contains(query, "ELSE NULL END FROM file_assets asset") {
			t.Fatalf("%s projection does not use FileAsset entity URL rules: %s", name, query)
		}
	}
	if !strings.Contains(activityIssueExpansion, "replace(encode(i.description_binary,'base64'),E'\\n','')") {
		t.Fatalf("wrapped binary description encoding: %s", activityIssueExpansion)
	}
}

func TestActivityOrderingUsesUniqueStableTieBreak(t *testing.T) {
	for value, expected := range map[string]string{
		"created_at":  "a.created_at ASC, a.id DESC",
		"updated_at":  "a.updated_at ASC NULLS LAST, a.created_at DESC, a.id DESC",
		"-updated_at": "a.updated_at DESC NULLS LAST, a.created_at DESC, a.id DESC",
	} {
		if actual := activityOrder(Request{Params: map[string]string{"order_by": value}}, "a.created_at DESC"); actual != expected {
			t.Fatalf("order %q=%q", value, actual)
		}
	}
}
