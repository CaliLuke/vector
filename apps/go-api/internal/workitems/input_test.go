package workitems

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

const issueSourceFixtureDir = "../../../../apps/go-api/internal/parity/testdata/fixtures/work_item_core"

func TestValidateIssueMatchesSourceScalarValidationFixtures(t *testing.T) {
	// These fixtures are source-recorded for IssueViewSet.create/update and
	// exercise DRF field validation before IssueCreateSerializer.validate.
	for _, tc := range []struct {
		name     string
		fixture  string
		wantName string
	}{
		{"missing name", "issue-core-app-list-name-missing.json", ""},
		{"empty name", "issue-core-app-list-name-empty.json", ""},
		{"null name", "issue-core-app-list-name-null.json", ""},
		{"boolean name", "issue-core-app-list-name-boolean.json", ""},
		{"long name", "issue-core-app-list-name-long.json", ""},
		{"required name on PUT", "issue-core-app-put-name-required.json", ""},
		{"invalid priority choice", "issue-core-app-list-priority-invalid.json", ""},
		{"reversed dates", "issue-core-app-list-dates-reversed.json", ""},
		{"control whitespace is trimmed", "issue-core-app-list-name-control-trim.json", "Trimmed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := readIssueValidationFixture(t, tc.fixture)
			input, err := djangoinput.Parse(bytes.NewReader(fixture.Request.Body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			changes, err := (&Repository{}).validateIssue(context.Background(), input, issueValidation{})
			if tc.wantName != "" {
				if err != nil {
					t.Fatalf("validateIssue() error = %v", err)
				}
				if got := changes.values["name"]; got != tc.wantName {
					t.Fatalf("validated name = %#v, want %q", got, tc.wantName)
				}
				return
			}
			var got *djangoapi.HTTPError
			if !errors.As(err, &got) {
				t.Fatalf("validateIssue() error = %v, want source HTTP 400", err)
			}
			if got.Status != fixture.Response.Status {
				t.Fatalf("validation status = %d, source status = %d", got.Status, fixture.Response.Status)
			}
			if !sameJSON(t, got.Body, fixture.Response.Body) {
				t.Fatalf("validation body = %s, source body = %s", marshalJSON(t, got.Body), fixture.Response.Body)
			}
		})
	}
}

func TestValidateIssueRunsObjectValidationOnlyAfterFieldValidation(t *testing.T) {
	// The source name and priority cases are recorded in
	// issue-core-app-list-name-boolean.json and
	// issue-core-app-list-priority-invalid.json. DRF runs object-level
	// IssueCreateSerializer.validate only after all fields validate, so its
	// reversed-date check must not add non_field_errors to field failures.
	input, err := djangoinput.Parse(bytes.NewBufferString(`{"name":true,"priority":"unknown","start_date":"2020-03-01","target_date":"2020-02-01"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&Repository{}).validateIssue(context.Background(), input, issueValidation{})
	var got *djangoapi.HTTPError
	if !errors.As(err, &got) {
		t.Fatalf("validateIssue() error = %v, want field validation error", err)
	}
	want := map[string]any{
		"name":     []string{"Not a valid string."},
		"priority": []string{`"unknown" is not a valid choice.`},
	}
	if !sameJSON(t, got.Body, want) {
		t.Fatalf("validation ordering body = %s, want %s", marshalJSON(t, got.Body), marshalJSON(t, want))
	}
}

type issueValidationFixture struct {
	Request struct {
		Body json.RawMessage `json:"body"`
	} `json:"request"`
	Response struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"response"`
}

func readIssueValidationFixture(t *testing.T, name string) issueValidationFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(issueSourceFixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	var fixture issueValidationFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func sameJSON(t *testing.T, actual, expected any) bool {
	t.Helper()
	var got, want any
	if err := json.Unmarshal(marshalJSON(t, actual), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(marshalJSON(t, expected), &want); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(got, want)
}

func marshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAPIWritableCreationTimestampValidatesBeforePersistence(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{{"invalid", "{\"created_at\":\"not-a-date\"}", "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}, {"null", "{\"created_at\":null}", "This field may not be null."}} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := djangoinput.Parse(bytes.NewBufferString(tc.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			_, err = (&Repository{}).validateIssue(context.Background(), input, issueValidation{API: true, Partial: true})
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || !sameJSON(t, failure.Body, map[string]any{"created_at": []string{tc.want}}) {
				t.Fatalf("validation=%#v", err)
			}
		})
	}
	input, err := djangoinput.Parse(bytes.NewBufferString(`{"created_at":"2010-01-02T03:04:05"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	changes, err := (&Repository{}).validateIssue(context.Background(), input, issueValidation{API: true, Partial: true, Timezone: "America/Los_Angeles"})
	if err != nil {
		t.Fatal(err)
	}
	if got := changes.values["created_at"]; got != "2010-01-02T03:04:05-08:00" {
		t.Fatalf("created_at=%#v", got)
	}
}
