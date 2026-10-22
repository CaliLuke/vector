package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	store "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"uuid"
)

type publicProjectMetadataReadsRepositoryStub struct {
	cycles   json.RawMessage
	labels   json.RawMessage
	members  json.RawMessage
	meta     json.RawMessage
	modules  json.RawMessage
	settings json.RawMessage
	states   json.RawMessage
	err      error
}

func (stub *publicProjectMetadataReadsRepositoryStub) Cycles(context.Context, string) (json.RawMessage, error) {
	return stub.cycles, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) Labels(context.Context, string) (json.RawMessage, error) {
	return stub.labels, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) Members(context.Context, string) (json.RawMessage, error) {
	return stub.members, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) Meta(context.Context, string) (json.RawMessage, error) {
	return stub.meta, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) Modules(context.Context, string) (json.RawMessage, error) {
	return stub.modules, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) ProjectAnchor(context.Context, string, uuid.UUID) (json.RawMessage, error) {
	return stub.settings, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) Settings(context.Context, string) (json.RawMessage, error) {
	return stub.settings, stub.err
}

func (stub *publicProjectMetadataReadsRepositoryStub) States(context.Context, string) (json.RawMessage, error) {
	return stub.states, stub.err
}

func TestNewPublicProjectMetadataReadsRequiresRepository(t *testing.T) {
	if _, err := NewPublicProjectMetadataReads(nil); err == nil {
		t.Fatal("missing repository returned nil error")
	}
}

func TestPublicProjectMetadataReadsDecodeAndCapture(t *testing.T) {
	repository := &publicProjectMetadataReadsRepositoryStub{
		cycles:   json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000003","name":"Sprint 1"}]`),
		labels:   json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000001","name":"Bug","color":"#FF0000","parent":null}]`),
		members:  json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000005","member":null,"member__display_name":null,"member__avatar":null}]`),
		meta:     json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000006","identifier":"PMETA","name":"Public Project","cover_image":null,"icon_prop":null,"emoji":null,"description":""}`),
		modules:  json.RawMessage(`[{"id":"00000000-0000-4000-8000-000000000004","name":"Launch"}]`),
		settings: json.RawMessage(`{"id":"00000000-0000-4000-8000-000000000007","project_details":null,"workspace_detail":{"name":"Demo","slug":"demo","id":"00000000-0000-4000-8000-000000000001","logo_url":null},"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","deleted_at":null,"entity_identifier":null,"entity_name":"project","anchor":"anchor","is_comments_enabled":false,"is_reactions_enabled":false,"is_votes_enabled":false,"view_props":{},"is_activity_enabled":false,"is_disabled":false,"created_by":null,"updated_by":null,"workspace":"00000000-0000-4000-8000-000000000001","project":null,"intake":null}`),
		states:   json.RawMessage(`[{"name":"Backlog","group":"backlog","color":"#60646C","id":"00000000-0000-4000-8000-000000000002","sequence":100.0}]`),
	}
	serviceValue, _ := NewPublicProjectMetadataReads(repository)
	service := serviceValue.(*publicProjectMetadataReadsService)
	capture := &PublicProjectMetadataReadsResponseCapture{}
	ctx := WithPublicProjectMetadataReadsResponseCapture(context.Background(), capture)
	cycles, err := service.Cycles(ctx, &api.CyclesPayload{Anchor: "anchor"})
	if err != nil || len(cycles) != 1 || cycles[0].Name != "Sprint 1" {
		t.Fatalf("cycles=%+v err=%v", cycles, err)
	}
	labels, err := service.Labels(ctx, &api.LabelsPayload{Anchor: "anchor"})
	if err != nil || len(labels) != 1 || labels[0].Name != "Bug" {
		t.Fatalf("labels=%+v err=%v", labels, err)
	}
	modules, err := service.Modules(ctx, &api.ModulesPayload{Anchor: "anchor"})
	if err != nil || len(modules) != 1 || modules[0].Name != "Launch" {
		t.Fatalf("modules=%+v err=%v", modules, err)
	}
	members, err := service.Members(ctx, &api.MembersPayload{Anchor: "anchor"})
	if err != nil || len(members) != 1 {
		t.Fatalf("members=%+v err=%v", members, err)
	}
	meta, err := service.Meta(ctx, &api.MetaPayload{Anchor: "anchor"})
	if err != nil || meta.Identifier != "PMETA" {
		t.Fatalf("meta=%+v err=%v", meta, err)
	}
	settings, err := service.Settings(ctx, &api.SettingsPayload{Anchor: "anchor"})
	if err != nil || settings.WorkspaceDetail.Slug != "demo" || !settings.ProjectDetails.IsNull() {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	anchor, err := service.ProjectAnchor(ctx, &api.ProjectAnchorPayload{Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000007"})
	if err != nil || anchor.WorkspaceDetail.Slug != "demo" {
		t.Fatalf("anchor=%+v err=%v", anchor, err)
	}
	states, err := service.States(ctx, &api.StatesPayload{Anchor: "anchor"})
	if err != nil || len(states) != 1 || states[0].Sequence != 100 {
		t.Fatalf("states=%+v err=%v", states, err)
	}
	if string(capture.Data) != string(repository.states) {
		t.Fatalf("captured=%s", capture.Data)
	}
}

func TestPublicProjectMetadataReadsMapsMissingSettingsObject(t *testing.T) {
	service := &publicProjectMetadataReadsService{
		repository: &publicProjectMetadataReadsRepositoryStub{err: store.ErrRequiredObject},
	}
	_, err := service.Settings(context.Background(), &api.SettingsPayload{Anchor: "missing"})
	var notFound *api.PublicProjectMetadataNotFound
	if !errors.As(err, &notFound) || notFound.Message != "The required object does not exist." {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPublicProjectMetadataReadsMapsInvalidAnchor(t *testing.T) {
	service := &publicProjectMetadataReadsService{
		repository: &publicProjectMetadataReadsRepositoryStub{err: store.ErrInvalidAnchor},
	}
	_, err := service.Labels(context.Background(), &api.LabelsPayload{Anchor: "missing"})
	var notFound *api.PublicProjectMetadataNotFound
	if !errors.As(err, &notFound) || notFound.Message != "Invalid anchor" {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPublicProjectMetadataReadsMapsUnpublishedProject(t *testing.T) {
	service := &publicProjectMetadataReadsService{
		repository: &publicProjectMetadataReadsRepositoryStub{err: store.ErrProjectNotPublished},
	}
	_, err := service.Meta(context.Background(), &api.MetaPayload{Anchor: "missing"})
	var notFound *api.PublicProjectMetadataNotFound
	if !errors.As(err, &notFound) || notFound.Message != "Project is not published" {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestPublicProjectMetadataReadsMarksRenderPhaseFailures(t *testing.T) {
	service := &publicProjectMetadataReadsService{
		repository: &publicProjectMetadataReadsRepositoryStub{err: fmt.Errorf("%w: database unavailable", store.ErrRender)},
	}
	capture := &PublicProjectMetadataReadsResponseCapture{}
	ctx := WithPublicProjectMetadataReadsResponseCapture(context.Background(), capture)
	_, err := service.Labels(ctx, &api.LabelsPayload{Anchor: "anchor"})
	var internal *api.PublicProjectMetadataInternalError
	if !errors.As(err, &internal) || !capture.RenderError {
		t.Fatalf("err=%T %v capture=%#v", err, err, capture)
	}
}
