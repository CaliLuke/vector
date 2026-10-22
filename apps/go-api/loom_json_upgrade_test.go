package systemapi

import (
	"context"
	"encoding/json"
	"testing"

	analyticapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	stickyapi "github.com/CaliLuke/go-plane/gen/stickies"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

func TestStickyJSONPreservesEmptyValuesAndLargeNumbers(t *testing.T) {
	for _, value := range []string{"{}", "[]", "null", `{"id":9007199254740993,"empty":[]}`, `{"id":123456789012345678901234567890}`} {
		t.Run(value, func(t *testing.T) {
			repository := &stubStickyRepository{sticky: testSticky()}
			service := newTestStickyService(t, repository, stubSessions{})
			ctx, err := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Update(ctx, &stickyapi.UpdatePayload{
				Slug: "test", Pk: testStickyID.String(), Description: loom.JSONValue(value), LogoProps: loom.JSONValue(value),
			})
			if err != nil {
				t.Fatal(err)
			}
			if repository.patched.Description == nil || string(*repository.patched.Description) != value || repository.patched.LogoProps == nil || string(*repository.patched.LogoProps) != value {
				t.Fatalf("stored JSON = description %v logo %v, want %s", repository.patched.Description, repository.patched.LogoProps, value)
			}
			repository.sticky.Description = []byte(value)
			repository.sticky.LogoProps = []byte(value)
			result, err := service.Retrieve(ctx, &stickyapi.RetrievePayload{Slug: "test", Pk: testStickyID.String()})
			if err != nil {
				t.Fatal(err)
			}
			if string(result.Description) != value || string(result.LogoProps) != value {
				t.Fatalf("returned JSON = %s, %s, want %s", result.Description, result.LogoProps, value)
			}
		})
	}
}

func TestAnalyticQueryPreservesJSONNumbersAndZeroTruthiness(t *testing.T) {
	repository := &analyticRepoTest{authorized: true, raw: analyticRaw("Numbers")}
	service, ctx := analyticService(t, repository)
	const input = `{"state":[9007199254740993],"priority":0,"assignees":0.0,"labels":1e-999,"project":123456789012345678901234567890}`
	_, err := service.Create(ctx, &analyticapi.CreatePayload{Slug: "demo", Name: "Numbers", QueryDict: loom.JSONValue(input)})
	if err != nil {
		t.Fatal(err)
	}
	const normalized = `{"state":[9007199254740993],"priority":0,"assignees":0.0,"labels":0.0,"project":123456789012345678901234567890}`
	if string(repository.queryDict) != normalized {
		t.Fatalf("stored query_dict = %s, want %s", repository.queryDict, normalized)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(repository.query, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["state__in"]) != "[9007199254740993]" || string(fields["project__in"]) != "123456789012345678901234567890" {
		t.Fatalf("derived numeric query = %s", repository.query)
	}
	if fields["priority__in"] != nil || fields["assignees__in"] != nil || fields["labels__in"] != nil {
		t.Fatalf("zero-valued filters were retained: %s", repository.query)
	}
}
