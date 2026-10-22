package systemapi

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	stickyapi "github.com/CaliLuke/go-plane/gen/stickies"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

var (
	testUserID      = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290701")
	testWorkspaceID = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290702")
	testStickyID    = uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290703")
)

type stubSessions struct{ err error }

func (sessions stubSessions) Validate(context.Context, string) (djangosession.Principal, error) {
	if sessions.err != nil {
		return djangosession.Principal{}, sessions.err
	}
	return djangosession.Principal{UserID: testUserID.String()}, nil
}

type stubStickyRepository struct {
	resolveErr error
	list       stickystore.StickyPage
	created    stickystore.CreateStickyInput
	patched    stickystore.PatchStickyInput
	sticky     stickystore.Sticky
	err        error
	deleted    bool
}

func (repository *stubStickyRepository) ResolveWorkspace(context.Context, string, stickystore.UserID) (stickystore.WorkspaceID, error) {
	return stickystore.WorkspaceID(testWorkspaceID), repository.resolveErr
}
func (repository *stubStickyRepository) List(context.Context, stickystore.Scope, stickystore.ListOptions) (stickystore.StickyPage, error) {
	return repository.list, repository.err
}
func (repository *stubStickyRepository) Create(_ context.Context, _ stickystore.StickyID, _ stickystore.Scope, input stickystore.CreateStickyInput) (stickystore.Sticky, error) {
	repository.created = input
	return repository.sticky, repository.err
}
func (repository *stubStickyRepository) Retrieve(context.Context, stickystore.StickyID, stickystore.Scope) (stickystore.Sticky, error) {
	return repository.sticky, repository.err
}
func (repository *stubStickyRepository) Patch(_ context.Context, _ stickystore.StickyID, _ stickystore.Scope, input stickystore.PatchStickyInput) (stickystore.Sticky, error) {
	repository.patched = input
	return repository.sticky, repository.err
}
func (repository *stubStickyRepository) Delete(context.Context, stickystore.StickyID, stickystore.Scope) error {
	repository.deleted = true
	return repository.err
}

func TestStickyServiceSessionAuthenticationAndMembershipDifference(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, err := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatalf("APIKeyAuth() error = %v", err)
	}
	repository.resolveErr = stickystore.ErrWorkspaceAccessDenied
	_, err = service.Retrieve(ctx, &stickyapi.RetrievePayload{Slug: "test", Pk: testStickyID.String()})
	var missing *stickyapi.StickyNotFoundError
	if !errors.As(err, &missing) || missing.Detail != notFoundDetail {
		t.Fatalf("Retrieve() error = %#v, want M1-001 404", err)
	}
	_, err = service.List(ctx, &stickyapi.ListPayload{Slug: "test"})
	var forbidden *stickyapi.StickyPermissionError
	if !errors.As(err, &forbidden) || forbidden.Message != forbiddenDetail {
		t.Fatalf("List() error = %#v, want source-compatible 403", err)
	}
	_, err = service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal("valid stub unexpectedly rejected session")
	}
	invalid := newTestStickyService(t, repository, stubSessions{err: djangosession.ErrSessionNotFound})
	_, err = invalid.APIKeyAuth(context.Background(), "not-secret", &security.APIKeyScheme{})
	var unauthenticated *stickyapi.StickyUnauthenticatedError
	if !errors.As(err, &unauthenticated) || unauthenticated.Detail != unauthenticatedDetail {
		t.Fatalf("APIKeyAuth() error = %#v", err)
	}
}

func TestStickyServiceListMatchesDjangoCursorWrapper(t *testing.T) {
	repository := &stubStickyRepository{list: stickystore.StickyPage{
		Stickies: []stickystore.Sticky{testSticky()}, TotalCount: 21, Page: 0, Limit: 20, HasNext: true,
	}}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	page, err := service.List(ctx, &stickyapi.ListPayload{Slug: "test"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.NextCursor != "20:1:0" || page.PrevCursor != "20:-1:1" || !page.NextPageResults || page.PrevPageResults || page.TotalPages != 2 || page.Count != 1 || len(page.Results) != 1 {
		t.Fatalf("List() = %+v", page)
	}
	bad := "broken"
	_, err = service.List(ctx, &stickyapi.ListPayload{Slug: "test", Cursor: &bad})
	var invalid *stickyapi.StickyInvalidPaginationError
	if !errors.As(err, &invalid) || invalid.Detail != "Invalid cursor parameter." {
		t.Fatalf("List(invalid cursor) error = %#v", err)
	}
}

func TestStickyServiceSanitizesCreateAndUpdate(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	html := `<p onclick="bad">safe<script>bad</script></p>`
	name := "safe"
	if _, err := service.Create(ctx, &stickyapi.CreatePayload{Slug: "test", Name: loom.NullableValue(name), DescriptionHTML: &html, Description: loom.JSONValue(`{"type":"doc"}`)}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.created.DescriptionHTML == nil || *repository.created.DescriptionHTML != "<p>safe</p>" || repository.created.DescriptionStripped == nil || *repository.created.DescriptionStripped != "safe" {
		t.Fatalf("Create() input = %+v", repository.created)
	}
	sortOrder := 12345.5
	updated := "<p>after</p>"
	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String(), DescriptionHTML: &updated, SortOrder: &sortOrder}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repository.patched.SortOrder == nil || *repository.patched.SortOrder != sortOrder || repository.patched.DescriptionStripped == nil || *repository.patched.DescriptionStripped == nil || **repository.patched.DescriptionStripped != "after" {
		t.Fatalf("Update() input = %+v", repository.patched)
	}
}

func TestStickyServicePreservesEmptyHTMLStrippedNull(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	empty := ""

	if _, err := service.Create(ctx, &stickyapi.CreatePayload{Slug: "test", DescriptionHTML: &empty}); err != nil {
		t.Fatalf("Create(empty HTML) error = %v", err)
	}
	if repository.created.DescriptionStripped != nil {
		t.Fatalf("Create(empty HTML) stripped = %q, want null", *repository.created.DescriptionStripped)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String(), DescriptionHTML: &empty}); err != nil {
		t.Fatalf("Update(empty HTML) error = %v", err)
	}
	if repository.patched.DescriptionStripped == nil || *repository.patched.DescriptionStripped != nil {
		t.Fatalf("Update(empty HTML) stripped = %#v, want explicit null", repository.patched.DescriptionStripped)
	}
}

func TestStickyServicePreservesSourceAuditSortAndDeletionSemantics(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	createdBy := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290704").String()
	updatedBy := uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290705").String()
	deletedAt := "2026-08-13T02:03:04Z"
	sortOrder := 12345.5
	if _, err := service.Create(ctx, &stickyapi.CreatePayload{
		Slug: "test", CreatedBy: loom.NullableValue(createdBy), UpdatedBy: loom.NullableValue(updatedBy),
		DeletedAt: loom.NullableValue(deletedAt), SortOrder: &sortOrder,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.created.DeletedAt == nil || !repository.created.DeletedAt.Equal(time.Date(2026, 8, 13, 2, 3, 4, 0, time.UTC)) || repository.created.SortOrder == nil || *repository.created.SortOrder != sortOrder {
		t.Fatalf("Create() input = %+v", repository.created)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{
		Slug: "test", Pk: testStickyID.String(), UpdatedBy: loom.NullableValue(updatedBy),
		CreatedBy: loom.NullValue[string](), DeletedAt: loom.NullValue[string](),
	}); err != nil {
		t.Fatalf("Update(explicit nulls) error = %v", err)
	}
	if repository.patched.CreatedByID == nil || *repository.patched.CreatedByID != nil || repository.patched.DeletedAt == nil || *repository.patched.DeletedAt != nil {
		t.Fatalf("Update(explicit nulls) input = %+v", repository.patched)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{
		Slug: "test", Pk: testStickyID.String(), CreatedBy: loom.NullableValue(createdBy), DeletedAt: loom.NullableValue(deletedAt),
	}); err != nil {
		t.Fatalf("Update(values) error = %v", err)
	}
	if repository.patched.CreatedByID == nil || *repository.patched.CreatedByID == nil || uuid.UUID(**repository.patched.CreatedByID).String() != createdBy || repository.patched.DeletedAt == nil || *repository.patched.DeletedAt == nil || !(*repository.patched.DeletedAt).Equal(time.Date(2026, 8, 13, 2, 3, 4, 0, time.UTC)) {
		t.Fatalf("Update(values) input = %+v", repository.patched)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String()}); err != nil {
		t.Fatalf("Update(omitted) error = %v", err)
	}
	if repository.patched.CreatedByID != nil || repository.patched.DeletedAt != nil {
		t.Fatalf("Update(omitted) input = %+v", repository.patched)
	}
}

func TestStickyServicePreservesNullableJSONPresence(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})

	if _, err := service.Create(ctx, &stickyapi.CreatePayload{Slug: "test"}); err != nil {
		t.Fatalf("Create(absent) error = %v", err)
	}
	if string(repository.created.Description) != "{}" || string(repository.created.LogoProps) != "{}" {
		t.Fatalf("Create(absent) input = %+v", repository.created)
	}

	if _, err := service.Create(ctx, &stickyapi.CreatePayload{
		Slug: "test", Description: loom.JSONValue(`null`), LogoProps: loom.JSONValue(`null`),
	}); err != nil {
		t.Fatalf("Create(null) error = %v", err)
	}
	if string(repository.created.Description) != "null" || string(repository.created.LogoProps) != "null" {
		t.Fatalf("Create(null) input = %+v", repository.created)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String()}); err != nil {
		t.Fatalf("Update(absent) error = %v", err)
	}
	if repository.patched.Description != nil || repository.patched.LogoProps != nil {
		t.Fatalf("Update(absent) input = %+v", repository.patched)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{
		Slug: "test", Pk: testStickyID.String(), Description: loom.JSONValue(`null`), LogoProps: loom.JSONValue(`{"mark":true}`),
	}); err != nil {
		t.Fatalf("Update(values) error = %v", err)
	}
	if repository.patched.Description == nil || string(*repository.patched.Description) != "null" || repository.patched.LogoProps == nil || string(*repository.patched.LogoProps) != `{"mark":true}` {
		t.Fatalf("Update(values) input = %+v", repository.patched)
	}

	repository.sticky.Description = []byte("null")
	repository.sticky.LogoProps = []byte("null")
	response, err := service.Retrieve(ctx, &stickyapi.RetrievePayload{Slug: "test", Pk: testStickyID.String()})
	if err != nil {
		t.Fatalf("Retrieve(null) error = %v", err)
	}
	if string(response.Description) != "null" || string(response.LogoProps) != "null" {
		t.Fatalf("Retrieve(null) response = %+v", response)
	}
}

func TestStickyServicePreservesNullableTextWrites(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky()}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})

	if _, err := service.Create(ctx, &stickyapi.CreatePayload{
		Slug: "test", Name: loom.NullValue[string](), Color: loom.NullValue[string](), BackgroundColor: loom.NullValue[string](),
	}); err != nil {
		t.Fatalf("Create(null) error = %v", err)
	}
	if repository.created.Name != nil || repository.created.Color != nil || repository.created.BackgroundColor != nil {
		t.Fatalf("Create(null) input = %+v", repository.created)
	}

	name, color, background := "note", "blue", "white"
	if _, err := service.Create(ctx, &stickyapi.CreatePayload{
		Slug: "test", Name: loom.NullableValue(name), Color: loom.NullableValue(color), BackgroundColor: loom.NullableValue(background),
	}); err != nil {
		t.Fatalf("Create(values) error = %v", err)
	}
	if repository.created.Name == nil || *repository.created.Name != name || repository.created.Color == nil || *repository.created.Color != color || repository.created.BackgroundColor == nil || *repository.created.BackgroundColor != background {
		t.Fatalf("Create(values) input = %+v", repository.created)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String()}); err != nil {
		t.Fatalf("Update(absent) error = %v", err)
	}
	if repository.patched.Name != nil || repository.patched.Color != nil || repository.patched.BackgroundColor != nil {
		t.Fatalf("Update(absent) input = %+v", repository.patched)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{
		Slug: "test", Pk: testStickyID.String(), Name: loom.NullValue[string](), Color: loom.NullValue[string](), BackgroundColor: loom.NullValue[string](),
	}); err != nil {
		t.Fatalf("Update(null) error = %v", err)
	}
	if repository.patched.Name == nil || *repository.patched.Name != nil || repository.patched.Color == nil || *repository.patched.Color != nil || repository.patched.BackgroundColor == nil || *repository.patched.BackgroundColor != nil {
		t.Fatalf("Update(null) input = %+v", repository.patched)
	}

	if _, err := service.Update(ctx, &stickyapi.UpdatePayload{
		Slug: "test", Pk: testStickyID.String(), Name: loom.NullableValue(name), Color: loom.NullableValue(color), BackgroundColor: loom.NullableValue(background),
	}); err != nil {
		t.Fatalf("Update(values) error = %v", err)
	}
	if repository.patched.Name == nil || *repository.patched.Name == nil || **repository.patched.Name != name || repository.patched.Color == nil || *repository.patched.Color == nil || **repository.patched.Color != color || repository.patched.BackgroundColor == nil || *repository.patched.BackgroundColor == nil || **repository.patched.BackgroundColor != background {
		t.Fatalf("Update(values) input = %+v", repository.patched)
	}
}

func TestStickyServiceMapsCreatorPermissionAndDelete(t *testing.T) {
	repository := &stubStickyRepository{sticky: testSticky(), err: stickystore.ErrStickyForbidden}
	service := newTestStickyService(t, repository, stubSessions{})
	ctx, _ := service.APIKeyAuth(context.Background(), "valid-session", &security.APIKeyScheme{})
	_, err := service.Update(ctx, &stickyapi.UpdatePayload{Slug: "test", Pk: testStickyID.String()})
	var forbidden *stickyapi.StickyPermissionError
	if !errors.As(err, &forbidden) {
		t.Fatalf("Update() error = %#v", err)
	}
	repository.err = nil
	if err := service.Delete(ctx, &stickyapi.DeletePayload{Slug: "test", Pk: testStickyID.String()}); err != nil || !repository.deleted {
		t.Fatalf("Delete() = %v, called %t", err, repository.deleted)
	}
}

func newTestStickyService(t *testing.T, repository stickyRepository, sessions stickySessionValidator) *stickyService {
	t.Helper()
	service, err := NewStickies(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	return service.(*stickyService)
}

func testSticky() stickystore.Sticky {
	createdBy := stickystore.UserID(testUserID)
	name, stripped := "fixture", "fixture"
	return stickystore.Sticky{
		ID: stickystore.StickyID(testStickyID), WorkspaceID: stickystore.WorkspaceID(testWorkspaceID), OwnerID: stickystore.UserID(testUserID), CreatedByID: &createdBy,
		CreatedAt: time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC), UpdatedAt: time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC), Name: &name,
		Description: []byte(`{"type":"doc"}`), DescriptionHTML: "<p>fixture</p>", DescriptionStripped: &stripped, LogoProps: []byte(`{}`), SortOrder: 65535,
	}
}
