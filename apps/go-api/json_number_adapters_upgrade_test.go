package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"uuid"

	analyticapi "github.com/CaliLuke/go-plane/gen/analytic_views"
	standardapi "github.com/CaliLuke/go-plane/gen/standard_views"
	stickyapi "github.com/CaliLuke/go-plane/gen/stickies"
	identityapi "github.com/CaliLuke/go-plane/gen/user_identity"
	propertiesapi "github.com/CaliLuke/go-plane/gen/user_properties"
	quickapi "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	themeapi "github.com/CaliLuke/go-plane/gen/workspace_themes"
	standardstore "github.com/CaliLuke/go-plane/internal/standardviews"
	identitystore "github.com/CaliLuke/go-plane/internal/useridentity"
	propertiesstore "github.com/CaliLuke/go-plane/internal/userproperties"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/jackc/pgx/v5"
)

// Expectations were characterized with Python 3.12 / DRF 3.17.1 JSONParser.
const upgradeNumberInput = `{"integer":9007199254740993,"decimal":9007199254740993.0,"tiny":1e-999,"nested":[-1e-999,0.100000000000000005],"zero":-0}`
const upgradeNumberExpected = `{"integer":9007199254740993,"decimal":9007199254740992.0,"tiny":0.0,"nested":[-0.0,0.1],"zero":0}`
const upgradeProjectID = "00000000-0000-4000-8000-000000000002"

func assertUpgradeJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	decode := func(data []byte) any {
		t.Helper()
		var value any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(got), decode([]byte(want))) {
		t.Fatalf("persisted JSON = %s, want %s", got, want)
	}
}

func assertUpgradeFieldError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected JSON field validation error")
	}
	body, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	assertUpgradeJSON(t, body, `{"`+field+`":["Value must be valid JSON."]}`)
}

type upgradePropertiesRepository struct {
	userPropertiesRepositoryStub
	writes         int
	workspaceReads int
	projectReads   int
	lookupErr      error
	scopedChecks   []string
}

func (r *upgradePropertiesRepository) GetWorkspace(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	r.workspaceReads++
	return r.body, r.lookupErr
}
func (r *upgradePropertiesRepository) GetProject(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error) {
	r.projectReads++
	return r.body, r.lookupErr
}

func (r *upgradePropertiesRepository) CheckScopedPatch(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID, kind string) error {
	r.scopedChecks = append(r.scopedChecks, kind)
	return r.lookupErr
}

func (r *upgradePropertiesRepository) PatchWorkspace(_ context.Context, _ uuid.UUID, _ string, fields map[string]json.RawMessage) (json.RawMessage, error) {
	r.writes++
	r.patched = fields
	return r.body, r.err
}
func (r *upgradePropertiesRepository) PatchProject(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	r.writes++
	r.patched = fields
	return r.body, r.err
}
func (r *upgradePropertiesRepository) PatchCycle(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	r.writes++
	r.patched = fields
	return r.body, r.err
}
func (r *upgradePropertiesRepository) PatchModule(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	r.writes++
	r.patched = fields
	return r.body, r.err
}
func upgradePropertiesService(t *testing.T, repo *upgradePropertiesRepository) (propertiesapi.Service, context.Context) {
	t.Helper()
	service, err := NewUserProperties(repo, userPropertiesSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(propertiesapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

type upgradePropertyValues map[string]loom.Nullable[loom.JSONValue]

type upgradePropertiesPatch func(propertiesapi.Service, context.Context, upgradePropertyValues) error

var upgradePropertiesPatches = []struct {
	name  string
	patch upgradePropertiesPatch
}{
	{"workspace", func(s propertiesapi.Service, ctx context.Context, v upgradePropertyValues) error {
		_, err := s.WorkspacePatch(ctx, &propertiesapi.WorkspacePatchPayload{Slug: "demo", Filters: v["filters"], DisplayFilters: v["display_filters"], DisplayProperties: v["display_properties"], RichFilters: v["rich_filters"]})
		return err
	}},
	{"project", func(s propertiesapi.Service, ctx context.Context, v upgradePropertyValues) error {
		_, err := s.ProjectPatch(ctx, &propertiesapi.ProjectPatchPayload{Slug: "demo", ProjectID: upgradeProjectID, Filters: v["filters"], DisplayFilters: v["display_filters"], DisplayProperties: v["display_properties"], RichFilters: v["rich_filters"], Preferences: v["preferences"]})
		return err
	}},
	{"cycle", func(s propertiesapi.Service, ctx context.Context, v upgradePropertyValues) error {
		_, err := s.CyclePatch(ctx, &propertiesapi.CyclePatchPayload{Slug: "demo", ProjectID: upgradeProjectID, CycleID: analyticTestID, Filters: v["filters"], DisplayFilters: v["display_filters"], DisplayProperties: v["display_properties"], RichFilters: v["rich_filters"]})
		return err
	}},
	{"module", func(s propertiesapi.Service, ctx context.Context, v upgradePropertyValues) error {
		_, err := s.ModulePatch(ctx, &propertiesapi.ModulePatchPayload{Slug: "demo", ProjectID: upgradeProjectID, ModuleID: analyticTestID, Filters: v["filters"], DisplayFilters: v["display_filters"], DisplayProperties: v["display_properties"], RichFilters: v["rich_filters"]})
		return err
	}},
}

func TestJSONNumberUpgradeUserPropertiesNormalizeEveryScope(t *testing.T) {
	for _, scope := range upgradePropertiesPatches {
		t.Run(scope.name, func(t *testing.T) {
			repo := &upgradePropertiesRepository{userPropertiesRepositoryStub: userPropertiesRepositoryStub{authorized: true, projectAuthorized: true, body: userPropertiesJSON()}}
			service, ctx := upgradePropertiesService(t, repo)
			if err := scope.patch(service, ctx, upgradePropertyValues{
				"filters":            loom.NullableValue(loom.JSONValue(upgradeNumberInput)),
				"display_filters":    loom.NullableValue(loom.JSONValue(`{"display":1e-999}`)),
				"display_properties": loom.NullableValue(loom.JSONValue(`{"precision":9007199254740993.0}`)),
				"rich_filters":       loom.NullableValue(loom.JSONValue(`[-1e-999]`)),
				"preferences":        loom.NullableValue(loom.JSONValue(`{"ratio":0.100000000000000005}`)),
			}); err != nil {
				t.Fatal(err)
			}
			if repo.writes != 1 {
				t.Fatalf("writes=%d", repo.writes)
			}
			assertUpgradeJSON(t, repo.patched["filters"], upgradeNumberExpected)
			assertUpgradeJSON(t, repo.patched["display_filters"], `{"display":0.0}`)
			assertUpgradeJSON(t, repo.patched["display_properties"], `{"precision":9007199254740992.0}`)
			assertUpgradeJSON(t, repo.patched["rich_filters"], `[-0.0]`)
			if scope.name == "project" {
				assertUpgradeJSON(t, repo.patched["preferences"], `{"ratio":0.1}`)
			}
		})
	}
}

func TestJSONNumberUpgradeUserPropertiesRetainPresence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value loom.Nullable[loom.JSONValue]
		want  string
	}{
		{"absent", loom.Nullable[loom.JSONValue]{}, ""},
		{"null", loom.NullValue[loom.JSONValue](), "null"},
		{"empty object", loom.NullableValue(loom.JSONValue(`{}`)), `{}`},
		{"empty array", loom.NullableValue(loom.JSONValue(`[]`)), `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &upgradePropertiesRepository{userPropertiesRepositoryStub: userPropertiesRepositoryStub{projectAuthorized: true, body: userPropertiesJSON()}}
			service, ctx := upgradePropertiesService(t, repo)
			_, err := service.ProjectPatch(ctx, &propertiesapi.ProjectPatchPayload{Slug: "demo", ProjectID: upgradeProjectID, Preferences: tc.value, SortOrder: loom.NullableValue(7.5)})
			if err != nil {
				t.Fatal(err)
			}
			if string(repo.patched["preferences"]) != tc.want {
				t.Fatalf("preferences=%s, want %s", repo.patched["preferences"], tc.want)
			}
			if string(repo.patched["sort_order"]) != "7.5" {
				t.Fatalf("typed sort_order changed: %s", repo.patched["sort_order"])
			}
			if _, present := repo.patched["filters"]; present {
				t.Fatal("absent filters were supplied")
			}
		})
	}
}

func TestJSONNumberUpgradeUserPropertiesOverflowDoesNotPersist(t *testing.T) {
	for _, scope := range upgradePropertiesPatches {
		t.Run(scope.name, func(t *testing.T) {
			repo := &upgradePropertiesRepository{userPropertiesRepositoryStub: userPropertiesRepositoryStub{authorized: true, projectAuthorized: true, body: userPropertiesJSON()}}
			service, ctx := upgradePropertiesService(t, repo)
			err := scope.patch(service, ctx, upgradePropertyValues{"filters": loom.NullableValue(loom.JSONValue(`{"overflow":1e309}`))})
			if scope.name == "cycle" || scope.name == "module" {
				if !reflect.DeepEqual(repo.scopedChecks, []string{scope.name}) || repo.workspaceReads != 0 || repo.projectReads != 0 {
					t.Fatalf("scoped existence checks=%v workspace reads=%d project reads=%d", repo.scopedChecks, repo.workspaceReads, repo.projectReads)
				}
				var internal *propertiesapi.UserPropertiesInternal
				if !errors.As(err, &internal) {
					t.Fatalf("error=%T %v, want source direct-save internal error", err, err)
				}
			} else {
				var invalid propertiesapi.JSONFieldValidation
				if !errors.As(err, &invalid) || !reflect.DeepEqual(invalid["filters"], []string{"Value must be valid JSON."}) {
					t.Fatalf("error=%T %v", err, err)
				}
			}
			if repo.writes != 0 {
				t.Fatalf("overflow reached persistence: writes=%d", repo.writes)
			}
			if scope.name == "workspace" && (repo.workspaceReads != 1 || repo.projectReads != 0) {
				t.Fatalf("workspace default-row preparation: workspace=%d project=%d", repo.workspaceReads, repo.projectReads)
			}
			if scope.name == "project" && (repo.projectReads != 1 || repo.workspaceReads != 0) {
				t.Fatalf("project default-row preparation: workspace=%d project=%d", repo.workspaceReads, repo.projectReads)
			}
		})
	}
}

func TestJSONNumberUpgradeAuthorizationPrecedesInvalidProperties(t *testing.T) {
	repo := &upgradePropertiesRepository{}
	service, ctx := upgradePropertiesService(t, repo)
	payload := &propertiesapi.ProjectPatchPayload{Slug: "demo", ProjectID: upgradeProjectID, Preferences: loom.NullableValue(loom.JSONValue(`1e309`))}
	_, err := service.ProjectPatch(ctx, payload)
	var forbidden *propertiesapi.UserPropertiesForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("error=%T %v, want forbidden", err, err)
	}
	_, err = service.ProjectPatch(context.Background(), payload)
	var unauth *propertiesapi.UserPropertiesUnauthenticated
	if !errors.As(err, &unauth) {
		t.Fatalf("error=%T %v, want unauthenticated", err, err)
	}
	if repo.writes != 0 || repo.workspaceReads != 0 || repo.projectReads != 0 || len(repo.scopedChecks) != 0 {
		t.Fatal("unauthorized request reached persistence or default-row preparation")
	}
}

type upgradeIdentityDatabase struct {
	writes  int
	reads   int
	readErr error
	patch   json.RawMessage
}

func (d *upgradeIdentityDatabase) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	if len(args) == 2 {
		d.writes++
		d.patch = append(json.RawMessage(nil), args[1].(json.RawMessage)...)
	} else {
		d.reads++
		if d.readErr != nil {
			return identityServiceRow(func(...any) error { return d.readErr })
		}
	}
	return identityJSONRowForService(`{}`)
}
func upgradeIdentityService(t *testing.T, database *upgradeIdentityDatabase) (identityapi.Service, context.Context) {
	t.Helper()
	repository, err := identitystore.NewRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewUserIdentity(repository, identitySessionValidator{principal: themeUser})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(identityapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}
func TestJSONNumberUpgradeProfileNormalizesAllJSONProperties(t *testing.T) {
	database := &upgradeIdentityDatabase{}
	service, ctx := upgradeIdentityService(t, database)
	_, err := service.UpdateProfile(ctx, &identityapi.UpdateProfilePayload{Body: &identityapi.UserProfilePatch{
		Theme:                loom.JSONValue(upgradeNumberInput),
		OnboardingStep:       loom.JSONValue(`{"step":1e-999}`),
		BillingAddress:       loom.JSONValue(`{"precision":9007199254740993.0}`),
		MobileOnboardingStep: loom.JSONValue(`[-1e-999]`),
		Goals:                loom.JSONValue(`{"id":9007199254740993}`),
		ProductTour:          loom.JSONValue(`{"ratio":0.100000000000000005}`),
	}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(database.patch, &fields); err != nil {
		t.Fatal(err)
	}
	if database.writes != 1 || len(fields) != 6 {
		t.Fatalf("writes=%d, fields=%s", database.writes, database.patch)
	}
	assertUpgradeJSON(t, fields["theme"], upgradeNumberExpected)
	assertUpgradeJSON(t, fields["onboarding_step"], `{"step":0.0}`)
	assertUpgradeJSON(t, fields["billing_address"], `{"precision":9007199254740992.0}`)
	assertUpgradeJSON(t, fields["mobile_onboarding_step"], `[-0.0]`)
	assertUpgradeJSON(t, fields["goals"], `{"id":9007199254740993}`)
	assertUpgradeJSON(t, fields["product_tour"], `{"ratio":0.1}`)
}
func TestJSONNumberUpgradeProfileRetainsNullEmptyAndAbsent(t *testing.T) {
	database := &upgradeIdentityDatabase{}
	service, ctx := upgradeIdentityService(t, database)
	_, err := service.UpdateProfile(ctx, &identityapi.UpdateProfilePayload{Body: &identityapi.UserProfilePatch{Theme: loom.JSONValue(`{}`), Goals: loom.JSONValue(`[]`), BillingAddress: loom.JSONValue(`null`)}})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, database.patch, `{"theme":{},"goals":[],"billing_address":null}`)
}
func TestJSONNumberUpgradeProfileOverflowDoesNotPersist(t *testing.T) {
	database := &upgradeIdentityDatabase{}
	service, ctx := upgradeIdentityService(t, database)
	payload := &identityapi.UpdateProfilePayload{Body: &identityapi.UserProfilePatch{Theme: loom.JSONValue(`{"nested":[1e309]}`)}}
	_, err := service.UpdateProfile(ctx, payload)
	assertUpgradeFieldError(t, err, "theme")
	if database.writes != 0 || database.reads != 1 {
		t.Fatalf("overflow must look up the profile without updating it: reads=%d writes=%d", database.reads, database.writes)
	}
	_, err = service.UpdateProfile(context.Background(), payload)
	var unauth *identityapi.UserIdentityUnauthenticated
	if !errors.As(err, &unauth) {
		t.Fatalf("error=%T %v, want unauthenticated", err, err)
	}
	if database.writes != 0 || database.reads != 1 {
		t.Fatal("unauthenticated profile request reached repository")
	}
}

func TestJSONNumberUpgradeAnalyticStoredAndDerivedValuesAgree(t *testing.T) {
	repo := &analyticRepoTest{authorized: true, raw: analyticRaw("Numbers")}
	service, ctx := analyticService(t, repo)
	_, err := service.Create(ctx, &analyticapi.CreatePayload{Slug: "demo", Name: "Numbers", QueryDict: loom.JSONValue(`{"state":[9007199254740993,9007199254740993.0],"labels":1e-999,"assignees":-1e-999}`)})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.queryDict, `{"state":[9007199254740993,9007199254740992.0],"labels":0.0,"assignees":-0.0}`)
	assertUpgradeJSON(t, repo.query, `{"state__in":[9007199254740993,9007199254740992.0],"label_issue__deleted_at__isnull":true,"issue_assignee__deleted_at__isnull":true}`)
}
func TestJSONNumberUpgradeAnalyticRejectsOverflowButIgnoresPatchQueryData(t *testing.T) {
	repo := &analyticRepoTest{authorized: true, raw: analyticRaw("Numbers")}
	service, ctx := analyticService(t, repo)
	_, err := service.Create(ctx, &analyticapi.CreatePayload{Slug: "demo", Name: "Numbers", QueryDict: loom.JSONValue(`{"labels":1e309}`)})
	assertUpgradeFieldError(t, err, "query_dict")
	if repo.queryDict != nil || repo.query != nil {
		t.Fatal("overflow reached analytic create")
	}
	_, err = service.Patch(ctx, &analyticapi.PatchPayload{Slug: "demo", Pk: analyticTestID, QueryData: loom.JSONValue(`{"labels":1e309}`)})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.updated, `{}`)
}

func TestJSONNumberUpgradeStickyNormalizesBothJSONFields(t *testing.T) {
	repo := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repo, stubSessions{})
	ctx, err := service.APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String(), Description: loom.JSONValue(upgradeNumberInput), LogoProps: loom.JSONValue(upgradeNumberInput)})
	if err != nil {
		t.Fatal(err)
	}
	if repo.patched.Description == nil || repo.patched.LogoProps == nil {
		t.Fatal("JSON fields were omitted")
	}
	assertUpgradeJSON(t, *repo.patched.Description, upgradeNumberExpected)
	assertUpgradeJSON(t, *repo.patched.LogoProps, upgradeNumberExpected)
}
func TestJSONNumberUpgradeThemeNormalizesColors(t *testing.T) {
	repo := &themeRepoStub{authorized: true}
	service, ctx := themeService(t, repo)
	_, err := service.Patch(ctx, &themeapi.PatchPayload{Slug: "demo", Pk: themeID, Colors: loom.NullableValue(loom.JSONValue(upgradeNumberInput))})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.colors, upgradeNumberExpected)
}

type upgradeQuickRepository struct {
	quickRepoStub
	metadata json.RawMessage
}

func (r *upgradeQuickRepository) Update(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ bool, _ *string, _ *string, _ bool, metadata json.RawMessage) (json.RawMessage, error) {
	r.metadata = metadata
	return json.RawMessage(quickBody), nil
}
func TestJSONNumberUpgradeQuickLinkNormalizesMetadata(t *testing.T) {
	repo := &upgradeQuickRepository{quickRepoStub: quickRepoStub{authorized: true}}
	service, err := NewWorkspaceQuickLinks(repo, quickSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(quickapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Patch(ctx, &quickapi.PatchPayload{Slug: "demo", Pk: quickID, Metadata: loom.JSONValue(upgradeNumberInput)})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.metadata, upgradeNumberExpected)
}

type upgradeStandardRepository struct {
	standardViewsRepositoryStub
	input standardstore.ViewInput
}

func (r *upgradeStandardRepository) PatchWorkspace(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, input standardstore.ViewInput) (json.RawMessage, error) {
	r.input = input
	return r.body, nil
}
func TestJSONNumberUpgradeStandardViewNormalizesAllJSONFields(t *testing.T) {
	repo := &upgradeStandardRepository{standardViewsRepositoryStub: standardViewsRepositoryStub{allowed: true, body: json.RawMessage(`{}`)}}
	service, err := NewStandardViews(repo, standardViewsSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(standardapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.PatchWorkspace(ctx, &standardapi.PatchWorkspacePayload{
		Slug: "demo", Pk: analyticTestID,
		Filters:           loom.NullableValue(loom.JSONValue(upgradeNumberInput)),
		DisplayFilters:    loom.NullableValue(loom.JSONValue(`{"display":1e-999}`)),
		DisplayProperties: loom.NullableValue(loom.JSONValue(`{"precision":9007199254740993.0}`)),
		RichFilters:       loom.NullableValue(loom.JSONValue(`[-1e-999]`)),
		LogoProps:         loom.NullableValue(loom.JSONValue(`{"ratio":0.100000000000000005}`)),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.input.Filters, upgradeNumberExpected)
	assertUpgradeJSON(t, repo.input.DisplayFilters, `{"display":0.0}`)
	assertUpgradeJSON(t, repo.input.DisplayProperties, `{"precision":9007199254740992.0}`)
	assertUpgradeJSON(t, repo.input.RichFilters, `[-0.0]`)
	assertUpgradeJSON(t, repo.input.LogoProps, `{"ratio":0.1}`)
}

func TestJSONNumberUpgradeStandardViewPreservesOmittedFields(t *testing.T) {
	repo := &upgradeStandardRepository{standardViewsRepositoryStub: standardViewsRepositoryStub{allowed: true, body: json.RawMessage(`{}`)}}
	service, err := NewStandardViews(repo, standardViewsSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(standardapi.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.PatchWorkspace(ctx, &standardapi.PatchWorkspacePayload{
		Slug: "demo", Pk: analyticTestID,
		Filters:           loom.NullableValue(loom.JSONValue(`{"state":[9007199254740993]}`)),
		DisplayProperties: loom.NullValue[loom.JSONValue](),
		LogoProps:         loom.NullableValue(loom.JSONValue(`{}`)),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertUpgradeJSON(t, repo.input.Filters, `{"state":[9007199254740993]}`)
	assertUpgradeJSON(t, repo.input.DisplayProperties, `null`)
	assertUpgradeJSON(t, repo.input.LogoProps, `{}`)
	if repo.input.DisplayFilters != nil || repo.input.RichFilters != nil {
		t.Fatalf("omitted fields were populated: display_filters=%s rich_filters=%s", repo.input.DisplayFilters, repo.input.RichFilters)
	}
}

func TestJSONNumberUpgradeUserPropertiesLookupFailurePrecedesOverflow(t *testing.T) {
	for _, scope := range upgradePropertiesPatches {
		t.Run(scope.name, func(t *testing.T) {
			repo := &upgradePropertiesRepository{userPropertiesRepositoryStub: userPropertiesRepositoryStub{authorized: true, projectAuthorized: true}, lookupErr: propertiesstore.ErrNotFound}
			service, ctx := upgradePropertiesService(t, repo)
			err := scope.patch(service, ctx, upgradePropertyValues{"filters": loom.NullableValue(loom.JSONValue(`1e309`))})
			var missing *propertiesapi.UserPropertiesNotFound
			if !errors.As(err, &missing) {
				t.Fatalf("error=%T %v, want lookup not found", err, err)
			}
			if repo.writes != 0 {
				t.Fatal("invalid request patched user properties")
			}
		})
	}
}
func TestJSONNumberUpgradeMissingProfilePrecedesOverflow(t *testing.T) {
	database := &upgradeIdentityDatabase{readErr: pgx.ErrNoRows}
	service, ctx := upgradeIdentityService(t, database)
	_, err := service.UpdateProfile(ctx, &identityapi.UpdateProfilePayload{Body: &identityapi.UserProfilePatch{Theme: loom.JSONValue(`1e309`)}})
	var missing *identityapi.UserIdentityNotFound
	if !errors.As(err, &missing) {
		t.Fatalf("error=%T %v, want missing profile", err, err)
	}
	if database.writes != 0 || database.reads != 1 {
		t.Fatalf("reads=%d writes=%d", database.reads, database.writes)
	}
}
