package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/deployboardreads"
	"github.com/CaliLuke/loom/security"
)

const deployBoardFixture = `{"anchor":"demo-anchor","created_at":"2026-01-02T03:04:05Z","created_by":null,"deleted_at":null,"entity_identifier":"00000000-0000-4000-8000-000000000002","entity_name":"project","id":"00000000-0000-4000-8000-000000000003","intake":null,"is_activity_enabled":true,"is_comments_enabled":true,"is_disabled":false,"is_reactions_enabled":false,"is_votes_enabled":true,"project":"00000000-0000-4000-8000-000000000002","project_details":{"id":"00000000-0000-4000-8000-000000000002","identifier":"DEMO","name":"Demo","cover_image":null,"cover_image_url":"/api/assets/v2/static/00000000-0000-4000-8000-000000000020/","logo_props":{},"description":"Demo project"},"updated_at":"2026-01-02T03:04:06Z","updated_by":null,"view_props":{"views":["list"]},"workspace":"00000000-0000-4000-8000-000000000001","workspace_detail":{"id":"00000000-0000-4000-8000-000000000001","name":"Demo Workspace","slug":"demo","logo_url":"/api/assets/v2/static/00000000-0000-4000-8000-000000000021/"}}`
const deployBoardUnsavedDefaultsFixture = `{"created_by":null,"deleted_at":null,"entity_identifier":null,"entity_name":"","intake":null,"is_activity_enabled":false,"is_comments_enabled":false,"is_disabled":false,"is_reactions_enabled":false,"is_votes_enabled":false,"updated_by":null,"view_props":null}`

type deployBoardReadsRepositoryStub struct {
	collection      json.RawMessage
	detail          json.RawMessage
	err             error
	collectionCalls int
	detailCalls     int
	userID          uuid.UUID
	slug            string
	projectID       uuid.UUID
	boardID         uuid.UUID
}

func (stub *deployBoardReadsRepositoryStub) Collection(_ context.Context, userID uuid.UUID, slug string, projectID uuid.UUID) (json.RawMessage, error) {
	stub.collectionCalls++
	stub.userID, stub.slug, stub.projectID = userID, slug, projectID
	return stub.collection, stub.err
}

func (stub *deployBoardReadsRepositoryStub) Detail(_ context.Context, userID uuid.UUID, slug string, projectID, boardID uuid.UUID) (json.RawMessage, error) {
	stub.detailCalls++
	stub.userID, stub.slug, stub.projectID, stub.boardID = userID, slug, projectID, boardID
	return stub.detail, stub.err
}

type deployBoardReadsSessionsStub struct {
	principal djangosession.Principal
	err       error
}

func (stub deployBoardReadsSessionsStub) Validate(context.Context, string) (djangosession.Principal, error) {
	if stub.err != nil {
		return djangosession.Principal{}, stub.err
	}
	if stub.principal.UserID == "" {
		return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000010"}, nil
	}
	return stub.principal, nil
}

func TestNewDeployBoardReadsRequiresDependencies(t *testing.T) {
	if _, err := NewDeployBoardReads(nil, deployBoardReadsSessionsStub{}); err == nil {
		t.Fatal("missing repository returned nil error")
	}
	if _, err := NewDeployBoardReads(&deployBoardReadsRepositoryStub{}, nil); err == nil {
		t.Fatal("missing sessions returned nil error")
	}
}

func TestDeployBoardCollectionPreservesUnsavedDefaultsAndSingleObjectResponses(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "unsaved defaults", body: deployBoardUnsavedDefaultsFixture},
		{name: "single", body: deployBoardFixture},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &deployBoardReadsRepositoryStub{collection: json.RawMessage(test.body)}
			serviceValue, _ := NewDeployBoardReads(repository, deployBoardReadsSessionsStub{})
			service := serviceValue.(*deployBoardReadsService)
			ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
			if err != nil {
				t.Fatal(err)
			}
			body := readRawResponse(t, captureRawResponse(service.Collection(ctx, &api.CollectionPayload{
				Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002",
			})))
			if body != test.body {
				t.Fatalf("body=%s, want=%s", body, test.body)
			}
			if repository.collectionCalls != 1 || repository.detailCalls != 0 || repository.slug != "demo" {
				t.Fatalf("repository calls: collection=%d detail=%d slug=%q", repository.collectionCalls, repository.detailCalls, repository.slug)
			}
		})
	}
}

func TestDeployBoardDetailUsesDistinctScopedRepositoryPath(t *testing.T) {
	repository := &deployBoardReadsRepositoryStub{detail: json.RawMessage(deployBoardFixture)}
	serviceValue, _ := NewDeployBoardReads(repository, deployBoardReadsSessionsStub{})
	service := serviceValue.(*deployBoardReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "session", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	body := readRawResponse(t, captureRawResponse(service.Detail(ctx, &api.DetailPayload{
		Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000002", Pk: "00000000-0000-4000-8000-000000000003",
	})))
	if body != deployBoardFixture {
		t.Fatalf("body=%s", body)
	}
	if repository.detailCalls != 1 || repository.collectionCalls != 0 {
		t.Fatalf("repository calls: detail=%d collection=%d", repository.detailCalls, repository.collectionCalls)
	}
	if repository.projectID.String() != "00000000-0000-4000-8000-000000000002" || repository.boardID.String() != "00000000-0000-4000-8000-000000000003" {
		t.Fatalf("repository scope: project=%s board=%s", repository.projectID, repository.boardID)
	}
}

func TestDeployBoardReadsMapAuthenticationAndRepositoryErrors(t *testing.T) {
	service := &deployBoardReadsService{
		repository: &deployBoardReadsRepositoryStub{},
		sessions:   deployBoardReadsSessionsStub{err: errors.New("invalid")},
	}
	_, err := service.APIKeyAuth(context.Background(), "bad", &security.APIKeyScheme{})
	var unauthenticated *api.DeployBoardReadsUnauthenticated
	if !errors.As(err, &unauthenticated) {
		t.Fatalf("authentication err=%T %v", err, err)
	}

	ctx := context.WithValue(context.Background(), deployBoardReadsPrincipal{}, uuid.New())
	service.repository = &deployBoardReadsRepositoryStub{err: store.ErrForbidden}
	_, _, err = service.Collection(ctx, &api.CollectionPayload{ProjectID: uuid.New().String()})
	var forbidden *api.DeployBoardReadsForbidden
	if !errors.As(err, &forbidden) {
		t.Fatalf("collection err=%T %v", err, err)
	}
	service.repository = &deployBoardReadsRepositoryStub{err: store.ErrNotFound}
	_, _, err = service.Detail(ctx, &api.DetailPayload{ProjectID: uuid.New().String(), Pk: uuid.New().String()})
	var notFound *api.DeployBoardReadsNotFound
	if !errors.As(err, &notFound) || notFound.Detail != "No DeployBoard matches the given query." {
		t.Fatalf("detail err=%T %v", err, err)
	}
}

func TestDeployBoardReadsStreamArbitraryRepositoryJSON(t *testing.T) {
	ctx := context.WithValue(context.Background(), deployBoardReadsPrincipal{}, uuid.New())
	service := &deployBoardReadsService{repository: &deployBoardReadsRepositoryStub{collection: json.RawMessage(`{`)}}
	body := readRawResponse(t, captureRawResponse(service.Collection(ctx, &api.CollectionPayload{ProjectID: uuid.New().String()})))
	if body != "{" {
		t.Fatalf("body=%q", body)
	}
}
