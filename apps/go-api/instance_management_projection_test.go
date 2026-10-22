package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/instance_management"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
)

func TestNormalizeAdminBasePath(t *testing.T) {
	for _, testCase := range []struct{ input, want string }{
		{"", "/"}, {"god-mode", "/god-mode/"}, {"/admin", "/admin/"}, {"/admin/", "/admin/"},
	} {
		if got := normalizeAdminBasePath(testCase.input); got != testCase.want {
			t.Errorf("normalizeAdminBasePath(%q)=%q want %q", testCase.input, got, testCase.want)
		}
	}
}

type disableEmailRepositoryStub struct {
	instanceManagementRepository
	allowed bool
	calls   *[]string
}

func (repository disableEmailRepositoryStub) IsAdmin(context.Context, uuid.UUID) (bool, error) {
	return repository.allowed, nil
}

func (repository disableEmailRepositoryStub) DisableEmail(context.Context) error {
	*repository.calls = append(*repository.calls, "repository")
	return nil
}

type disableEmailCacheStub struct {
	calls *[]string
}

func (cache disableEmailCacheStub) Invalidate(_ context.Context, path, _ string, _ bool) error {
	*cache.calls = append(*cache.calls, path)
	return nil
}

func TestFormatInstanceManagementDateTimesMatchesDRF(t *testing.T) {
	raw := []byte(`{"instance":{"created_at":"2026-10-22T00:29:52.126475+00:00","last_checked_at":"2026-01-02T00:00:00+00:00"},"configuration":{"value":"2026-01-02T00:00:00+00:00","updated_at":null}}`)
	got, err := formatInstanceManagementDateTimes(raw, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(got, &value); err != nil {
		t.Fatal(err)
	}
	instance := value["instance"].(map[string]any)
	configuration := value["configuration"].(map[string]any)
	if instance["created_at"] != "2026-10-22T00:29:52.126475Z" || instance["last_checked_at"] != "2026-01-02T00:00:00Z" {
		t.Fatalf("DRF timestamp projection mismatch: %s", got)
	}
	if configuration["value"] != "2026-01-02T00:00:00+00:00" || configuration["updated_at"] != nil {
		t.Fatalf("non-timestamp values changed: %s", got)
	}
}

func TestNonObjectBodyKindUsesDRFValidationEnvelope(t *testing.T) {
	err := instanceManagementInputError(instanceManagementBodyKindError{kind: "list"})
	var response *api.InstanceManagementError
	if !errors.As(err, &response) {
		t.Fatalf("error type = %T, want *InstanceManagementError", err)
	}
	if response.Kind != "bad_request" || string(response.Body) != `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}` {
		t.Fatalf("root PATCH error = %#v", response)
	}
}

func TestWorkspaceValidationErrorsUseDRFFieldBodies(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "name field", err: &instancemanagement.WorkspaceValidationError{Field: "name", Message: "Name must not contain URLs"}, want: `{"name":["Name must not contain URLs"]}`},
		{name: "slug field", err: &instancemanagement.WorkspaceValidationError{Field: "slug", Message: "Slug is not valid"}, want: `{"slug":["Slug is not valid"]}`},
		{name: "view-level length error", err: &instancemanagement.WorkspaceValidationError{Message: "The maximum length for name is 80 and for slug is 48"}, want: `{"error":"The maximum length for name is 80 and for slug is 48"}`},
		{name: "unrelated error", err: errors.New("database unavailable"), want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := instanceManagementWorkspaceValidationError(test.err)
			if test.want == "" {
				if err != nil {
					t.Fatalf("mapped error=%v, want nil", err)
				}
				return
			}
			var response *api.InstanceManagementError
			if !errors.As(err, &response) || response.Kind != "bad_request" || string(response.Body) != test.want {
				t.Fatalf("mapped error=%#v, want bad_request body %s", err, test.want)
			}
		})
	}
}

func TestDisableEmailInvalidatesConfigurationCacheBeforeWrite(t *testing.T) {
	order := []string{}
	service := instanceManagementService{
		repository: disableEmailRepositoryStub{allowed: true, calls: &order},
		cache:      disableEmailCacheStub{calls: &order},
	}
	actor := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	ctx := context.WithValue(context.Background(), instanceManagementPrincipalKey{}, actor)
	if err := service.DisableEmail(ctx, &api.DisableEmailPayload{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"/api/instances/", "/api/instances/configurations/", "repository"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("DisableEmail order=%v, want %v", order, want)
	}
}
