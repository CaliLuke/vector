package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	api "github.com/CaliLuke/go-plane/gen/public_feedback"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	metadata "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/go-plane/internal/workitems"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/jackc/pgx/v5/pgconn"
)

type publicFeedbackMetadataRepository interface {
	ResolvePublicBoard(context.Context, string) (metadata.PublicBoard, error)
	IssueDetail(context.Context, metadata.PublicBoard, string) (json.RawMessage, error)
	ProjectBoards(context.Context, string) (json.RawMessage, error)
}

type publicFeedbackWorkItemsRepository interface {
	PublicIssueList(context.Context, workitems.PublicFeedbackScope, string, map[string]string, map[string]bool) (workitems.Result, error)
	PublicVoteList(context.Context, workitems.PublicFeedbackScope, string) (workitems.Result, error)
	PublicVoteCreate(context.Context, workitems.PublicFeedbackScope, string, string, io.Reader, string) (workitems.Result, error)
	PublicVoteDelete(context.Context, workitems.PublicFeedbackScope, string, string) (workitems.Result, error)
	PublicIntakeList(context.Context, workitems.PublicFeedbackScope, string, string, map[string]string, map[string]bool) (workitems.Result, error)
	PublicIntakeCreate(context.Context, workitems.PublicFeedbackScope, string, string, io.Reader, string) (workitems.Result, error)
	PublicIntakeDetail(context.Context, workitems.PublicFeedbackScope, string, string) (workitems.Result, error)
	PublicIntakeUpdate(context.Context, workitems.PublicFeedbackScope, string, string, string, io.Reader, string) (workitems.Result, error)
	PublicIntakeDelete(context.Context, workitems.PublicFeedbackScope, string, string, string) (workitems.Result, error)
}

type publicFeedbackService struct {
	metadata publicFeedbackMetadataRepository
	items    publicFeedbackWorkItemsRepository
}

func NewPublicFeedback(metadataRepository publicFeedbackMetadataRepository, itemsRepository publicFeedbackWorkItemsRepository) (api.Service, error) {
	if metadataRepository == nil || itemsRepository == nil {
		return nil, errors.New("public feedback metadata and work-item repositories are required")
	}
	return &publicFeedbackService{metadata: metadataRepository, items: itemsRepository}, nil
}

// APIKeyAuth delegates session inspection to the shared public-route middleware.
func (*publicFeedbackService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (service *publicFeedbackService) IssueList(ctx context.Context, payload *api.IssueListPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	result, err := service.items.PublicIssueList(ctx, publicFeedbackScope(payload.Anchor, board), publicFeedbackUserID(ctx), publicFeedbackParams(ctx, payload), publicFeedbackPresent(ctx))
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) IssueDetail(ctx context.Context, payload *api.IssueDetailPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	data, err := service.metadata.IssueDetail(ctx, board, payload.IssueID)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return &api.PublicFeedbackResponseMetadata{}, io.NopCloser(bytes.NewReader(nil)), nil
	}
	return publicFeedbackRaw(data)
}

func (service *publicFeedbackService) VoteList(ctx context.Context, payload *api.VoteListPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	if _, err := publicFeedbackAuthenticatedUser(ctx); err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, payload.Anchor)
	if errors.Is(err, metadata.ErrProjectNotPublished) {
		return publicFeedbackRaw([]any{})
	}
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	result, err := service.items.PublicVoteList(ctx, publicFeedbackScope(payload.Anchor, board), payload.IssueID)
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) VoteCreate(ctx context.Context, payload *api.VoteCreatePayload, body io.ReadCloser) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	defer closePublicFeedbackBody(body)
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return nil, nil, publicFeedbackVoteCreateError(err)
	}
	result, err := service.items.PublicVoteCreate(ctx, publicFeedbackScope(payload.Anchor, board), payload.IssueID, actorID, body, requestContentType(ctx))
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) VoteDelete(ctx context.Context, payload *api.VoteDeletePayload) error {
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return publicFeedbackError(err)
	}
	_, err = service.items.PublicVoteDelete(ctx, publicFeedbackScope(payload.Anchor, board), payload.IssueID, actorID)
	return publicFeedbackError(err)
}

func (service *publicFeedbackService) InboxList(ctx context.Context, payload *api.InboxListPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	return service.publicIntakeList(ctx, payload.Anchor, payload.IntakeID, payload)
}

func (service *publicFeedbackService) InboxCreate(ctx context.Context, payload *api.InboxCreatePayload, body io.ReadCloser) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	return service.publicIntakeCreate(ctx, payload.Anchor, payload.IntakeID, body)
}

func (service *publicFeedbackService) IntakeList(ctx context.Context, payload *api.IntakeListPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	return service.publicIntakeList(ctx, payload.Anchor, payload.IntakeID, payload)
}

func (service *publicFeedbackService) IntakeCreate(ctx context.Context, payload *api.IntakeCreatePayload, body io.ReadCloser) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	return service.publicIntakeCreate(ctx, payload.Anchor, payload.IntakeID, body)
}

func (service *publicFeedbackService) IntakeDetail(ctx context.Context, payload *api.IntakeDetailPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	_ = actorID
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return nil, nil, publicFeedbackIntakeError(err)
	}
	result, err := service.items.PublicIntakeDetail(ctx, publicFeedbackScope(payload.Anchor, board), payload.IntakeID, payload.Pk)
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) IntakeUpdate(ctx context.Context, payload *api.IntakeUpdatePayload, body io.ReadCloser) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	defer closePublicFeedbackBody(body)
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return nil, nil, publicFeedbackIntakeError(err)
	}
	result, err := service.items.PublicIntakeUpdate(ctx, publicFeedbackScope(payload.Anchor, board), payload.IntakeID, payload.Pk, actorID, body, requestContentType(ctx))
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) IntakeDelete(ctx context.Context, payload *api.IntakeDeletePayload) error {
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, payload.Anchor)
	if err != nil {
		return publicFeedbackIntakeError(err)
	}
	_, err = service.items.PublicIntakeDelete(ctx, publicFeedbackScope(payload.Anchor, board), payload.IntakeID, payload.Pk, actorID)
	return publicFeedbackError(err)
}

func (service *publicFeedbackService) WorkspaceProjectBoards(ctx context.Context, payload *api.WorkspaceProjectBoardsPayload) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	data, err := service.metadata.ProjectBoards(ctx, payload.Slug)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	return publicFeedbackRaw(data)
}

func (service *publicFeedbackService) publicIntakeList(ctx context.Context, anchor, intakeID string, payload any) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, anchor)
	if err != nil {
		return nil, nil, publicFeedbackIntakeError(err)
	}
	result, err := service.items.PublicIntakeList(ctx, publicFeedbackScope(anchor, board), intakeID, actorID, publicFeedbackParams(ctx, payload), publicFeedbackPresent(ctx))
	return publicFeedbackResponse(result, err)
}

func (service *publicFeedbackService) publicIntakeCreate(ctx context.Context, anchor, intakeID string, body io.ReadCloser) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	defer closePublicFeedbackBody(body)
	actorID, err := publicFeedbackAuthenticatedUser(ctx)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	board, err := service.resolve(ctx, anchor)
	if err != nil {
		return nil, nil, publicFeedbackIntakeError(err)
	}
	result, err := service.items.PublicIntakeCreate(ctx, publicFeedbackScope(anchor, board), intakeID, actorID, body, requestContentType(ctx))
	return publicFeedbackResponse(result, err)
}

func publicFeedbackIntakeError(err error) error {
	if errors.Is(err, metadata.ErrProjectNotPublished) {
		err = publicFeedbackIntakeMissingBoardError()
	}
	return publicFeedbackError(err)
}

func publicFeedbackVoteCreateError(err error) error {
	if errors.Is(err, metadata.ErrProjectNotPublished) {
		err = &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	}
	return publicFeedbackError(err)
}

func publicFeedbackIntakeMissingBoardError() error {
	return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "Not found"}}
}

func (service *publicFeedbackService) resolve(ctx context.Context, anchor string) (metadata.PublicBoard, error) {
	return service.metadata.ResolvePublicBoard(ctx, anchor)
}

func publicFeedbackScope(anchor string, board metadata.PublicBoard) workitems.PublicFeedbackScope {
	var intakeID *string
	if board.IntakeID != nil {
		value := board.IntakeID.String()
		intakeID = &value
	}
	return workitems.PublicFeedbackScope{
		Anchor: anchor, WorkspaceSlug: board.WorkspaceSlug, WorkspaceID: board.WorkspaceID.String(),
		ProjectID: board.ProjectID.String(), IntakeID: intakeID, VotesEnabled: board.VotesEnabled,
	}
}

func publicFeedbackUserID(ctx context.Context) string { return metadata.UserID(ctx) }

func publicFeedbackAuthenticatedUser(ctx context.Context) (string, error) {
	userID := publicFeedbackUserID(ctx)
	if userID == "" {
		return "", &djangoapi.HTTPError{Status: 401, Body: map[string]any{"detail": "Authentication credentials were not provided."}}
	}
	return userID, nil
}

func publicFeedbackResponse(result workitems.Result, err error) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	return publicFeedbackRaw(result.Body)
}

func publicFeedbackRaw(value any) (*api.PublicFeedbackResponseMetadata, io.ReadCloser, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, nil, publicFeedbackError(err)
	}
	contentType := "application/json"
	return &api.PublicFeedbackResponseMetadata{ContentType: &contentType}, io.NopCloser(bytes.NewReader(data)), nil
}

func publicFeedbackError(err error) error {
	if err == nil {
		return nil
	}
	status, body := 500, map[string]any{"error": "Something went wrong please try again later"}
	if errors.Is(err, metadata.ErrProjectNotPublished) {
		status, body = 404, map[string]any{"error": "Project is not published"}
	}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		status, body = failure.Status, failure.Body
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		if postgres.Code == "22P02" {
			status, body = 400, map[string]any{"error": "Please provide valid detail"}
		}
		if strings.HasPrefix(postgres.Code, "23") {
			status, body = 400, map[string]any{"error": "The payload is not valid"}
		}
	}
	encoded, marshalErr := loom.JSONValueFrom(body)
	if marshalErr != nil {
		panic(marshalErr)
	}
	return &api.PublicFeedbackError{Kind: fmt.Sprintf("status_%d", status), Body: encoded}
}

func publicFeedbackParams(ctx context.Context, payload any) map[string]string {
	values := map[string]string{}
	for key, items := range ReadServiceQueryValues(ctx) {
		if len(items) > 0 {
			values[key] = items[len(items)-1]
		}
	}
	reflected := reflect.ValueOf(payload)
	if reflected.Kind() == reflect.Pointer {
		reflected = reflected.Elem()
	}
	for index := 0; index < reflected.NumField(); index++ {
		field := reflected.Type().Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" || name == "session_id" {
			continue
		}
		value := reflected.Field(index)
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}
		if value.Kind() == reflect.String {
			values[name] = value.String()
		}
	}
	for _, pathKey := range []string{"anchor", "intake_id", "issue_id", "pk", "slug"} {
		if value, exists := values[pathKey]; exists {
			values[pathKey] = value
		}
	}
	return values
}

func publicFeedbackPresent(ctx context.Context) map[string]bool {
	present := map[string]bool{}
	for key := range ReadServiceQueryValues(ctx) {
		present[key] = true
	}
	return present
}

func requestContentType(ctx context.Context) string {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok {
		return ""
	}
	return metadata.Headers().Get("Content-Type")
}

func closePublicFeedbackBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}
