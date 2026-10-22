package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/issuequery"
)

func TestIssueListDefaultOrderMatchesNewestFirstSource(t *testing.T) {
	args := []any{"actor", "workspace", "project"}
	order, err := issuequery.Order(queryOptions(Request{}), "", "", "", &args)
	if err != nil || order != "issue.created_at DESC" {
		t.Fatalf("default order=%s err=%v", order, err)
	}
}

func TestSearchInvalidProjectMatchesValidationException(t *testing.T) {
	for _, api := range []bool{false, true} {
		q := Request{API: api, ProjectID: "bad", Params: map[string]string{"search": "Needle", "project_id": "bad"}}
		_, err := (&Repository{}).search(context.Background(), q)
		var response *djangoapi.HTTPError
		if !errors.As(err, &response) || response.Status != 400 || response.Body["error"] != "Please provide valid detail" {
			t.Fatalf("API=%v: invalid project response=%v", api, err)
		}
	}
}

func TestPUTValidatesRichFilterBeforeLookingUpOrReadingBody(t *testing.T) {
	q := Request{Method: "PUT", Params: map[string]string{"filters": "{"}}
	_, err := (&Repository{}).writeIssue(context.Background(), q, strings.NewReader("invalid JSON"), "application/json")
	var filter *issuequery.FilterError
	if !errors.As(err, &filter) {
		t.Fatalf("expected filter failure before DB/body, got %v", err)
	}
}

func TestSubissueGroupingPreservesMissingFieldAndUnassignedContract(t *testing.T) {
	rows := []any{map[string]any{"id": "child", "assignee_ids": []any{}}}
	_, err := groupSubissues(rows, "missing_field")
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 || response.Body["error"] != "The required key does not exist." {
		t.Fatalf("invalid grouping response: %v", err)
	}
	grouped, err := groupSubissues(rows, "assignees__ids")
	if err != nil || len(grouped["None"].([]any)) != 1 {
		t.Fatalf("unassigned group=%#v err=%v", grouped, err)
	}
}

func TestSubissueOrderingDoesNotRequireModuleContext(t *testing.T) {
	for _, order := range []string{"issue_module__module__name", "-issue_module__module__name"} {
		query := subissueOrder(order)
		if strings.Contains(query, "$") || !strings.Contains(query, "mi.issue_id=i.id") {
			t.Fatalf("ordering requires a module parameter: %s", query)
		}
	}
	if got := subissueOrder("sequence_id"); got != "i.sequence_id ASC, i.created_at DESC" {
		t.Fatalf("sequence ordering=%s", got)
	}
	if got := subissueOrder("unknown,(select pg_sleep(5))"); got != "i.created_at DESC" {
		t.Fatalf("unsafe order=%s", got)
	}
}

func TestIdentifierOperationPreservesCanonicalManifestName(t *testing.T) {
	// Reverse generation retains the source operation ID, including the adjacent
	// placeholder separators. Routing must use the decoded composite resource.
	q := Request{Operation: "get_api_workspaces_by_slug_work_items__project_identifier___issue_identifier", Method: "GET", Params: map[string]string{"identifier": "TEST-invalid"}}
	_, err := (&Repository{}).Execute(context.Background(), q, nil, "")
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || failure.Body["error"] != "Invalid issue identifier" {
		t.Fatalf("identifier dispatch: %#v", err)
	}
}

func TestDraftNameUsesNullableOptionalModelContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fields map[string]any
		want   any
	}{{"omitted", map[string]any{}, nil}, {"null", map[string]any{"name": nil}, nil}, {"blank", map[string]any{"name": ""}, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := (&Repository{}).validateIssue(context.Background(), djangoinput.Input{Fields: tc.fields}, issueValidation{Draft: true})
			if err != nil {
				t.Fatal(err)
			}
			if got := changes.values["name"]; got != tc.want {
				t.Fatalf("name=%#v want=%#v", got, tc.want)
			}
		})
	}
}

func TestGlobalCursorNegativePageKeepsSourceCursorAndFirstWindow(t *testing.T) {
	rows := []map[string]any{{"id": "one", "link_count": json.Number("0")}, {"id": "two"}}
	result, err := (&Repository{}).v2Page(Request{Params: map[string]string{"cursor": "1:-1:0"}, Present: map[string]bool{"cursor": true}}, rows)
	if err != nil {
		t.Fatal(err)
	}
	value := result.Body.(map[string]any)
	items := value["results"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != "one" || value["cursor"] != "1:-1:0" || value["prev_cursor"] != "1:-2:0" || value["prev_page_results"] != false || items[0].(map[string]any)["link_count"] != nil {
		t.Fatalf("page=%#v", value)
	}
}

func TestOptionsInvalidFilterOmitsPutActionBeforeDatabaseAccess(t *testing.T) {
	available, err := (&Repository{}).ProtocolPutAction(context.Background(), Request{Params: map[string]string{"filters": "{"}})
	if available || err != nil {
		t.Fatalf("invalid filter metadata: available=%v err=%v", available, err)
	}
}

func TestAPIIdentifierUnsafeMethodsDenyBeforeProjectLookup(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "TRACE"} {
		t.Run(method, func(t *testing.T) {
			// ProjectEntityPermission uses project_identifier only for safe
			// methods. This view has no project_id for its unsafe branch.
			err := (&Repository{}).AuthorizeInitial(context.Background(), Request{API: true, Method: method, Params: map[string]string{"identifier": "IC-999999"}})
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 403 || response.Body["detail"] != "You do not have permission to perform this action." {
				t.Fatalf("unsafe identifier authorization=%v", err)
			}
		})
	}
}
