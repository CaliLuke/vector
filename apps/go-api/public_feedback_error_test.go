package systemapi

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"uuid"

	publicfeedback "github.com/CaliLuke/go-plane/gen/public_feedback"
	metadata "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

func TestPublicFeedbackMissingBoardMapsToDjango404(t *testing.T) {
	err := publicFeedbackError(metadata.ErrProjectNotPublished)
	require.IsType(t, (*publicfeedback.PublicFeedbackError)(nil), err)
	var failure *publicfeedback.PublicFeedbackError
	require.ErrorAs(t, err, &failure)
	encoded, err := failure.Body.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if got := body["error"]; got != "Project is not published" {
		t.Fatalf("missing board error=%v, want Project is not published", got)
	}
	if failure.Kind != "status_404" {
		t.Fatalf("missing board status kind=%q, want status_404", failure.Kind)
	}
}

func TestPublicFeedbackMissingBoardErrorsFollowOperationContract(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "vote create", err: publicFeedbackVoteCreateError(metadata.ErrProjectNotPublished), want: "The required object does not exist."},
		{name: "intake", err: publicFeedbackIntakeError(metadata.ErrProjectNotPublished), want: "Not found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.IsType(t, (*publicfeedback.PublicFeedbackError)(nil), test.err)
			var failure *publicfeedback.PublicFeedbackError
			require.ErrorAs(t, test.err, &failure)
			if failure.Kind != "status_404" {
				t.Fatalf("error=%T(%v), want status_404", test.err, test.err)
			}
			encoded, err := failure.Body.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(encoded, &body); err != nil {
				t.Fatal(err)
			}
			if body["error"] != test.want {
				t.Fatalf("error body=%v, want %q", body, test.want)
			}
		})
	}
}

type unresolvedPublicFeedbackMetadata struct {
	publicFeedbackMetadataRepository
}

func (unresolvedPublicFeedbackMetadata) ResolvePublicBoard(context.Context, string) (metadata.PublicBoard, error) {
	return metadata.PublicBoard{}, metadata.ErrProjectNotPublished
}

type unusedPublicFeedbackItems struct {
	publicFeedbackWorkItemsRepository
	voteListCalls int
}

func (items *unusedPublicFeedbackItems) PublicVoteList(context.Context, workitems.PublicFeedbackScope, string) (workitems.Result, error) {
	items.voteListCalls++
	return workitems.Result{Status: 200, Body: []any{}}, nil
}

func TestPublicVoteListMissingBoardReturnsEmptyAfterSessionAuth(t *testing.T) {
	items := &unusedPublicFeedbackItems{}
	service, err := NewPublicFeedback(unresolvedPublicFeedbackMetadata{}, items)
	if err != nil {
		t.Fatal(err)
	}
	ctx := metadata.WithUserID(context.Background(), "00000000-0000-4000-8000-000000000001")
	_, body, err := service.VoteList(ctx, &publicfeedback.VoteListPayload{Anchor: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "[]" || items.voteListCalls != 0 {
		t.Fatalf("body=%s vote list calls=%d, want [] with no repository query", encoded, items.voteListCalls)
	}
}

func (unresolvedPublicFeedbackMetadata) IssueDetail(context.Context, metadata.PublicBoard, string) (json.RawMessage, error) {
	return json.RawMessage("null"), nil
}

type nullIssuePublicFeedbackMetadata struct {
	publicFeedbackMetadataRepository
}

func (nullIssuePublicFeedbackMetadata) ResolvePublicBoard(context.Context, string) (metadata.PublicBoard, error) {
	return metadata.PublicBoard{
		WorkspaceID: uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		ProjectID:   uuid.MustParse("00000000-0000-4000-8000-000000000002"),
	}, nil
}

func (nullIssuePublicFeedbackMetadata) IssueDetail(context.Context, metadata.PublicBoard, string) (json.RawMessage, error) {
	return json.RawMessage("null"), nil
}

func TestPublicIssueDetailMissingBodyHasNoJSONContentType(t *testing.T) {
	service, err := NewPublicFeedback(nullIssuePublicFeedbackMetadata{}, &unusedPublicFeedbackItems{})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.IssueDetail(metadata.WithUserID(context.Background(), "user"), &publicfeedback.IssueDetailPayload{Anchor: "anchor", IssueID: "issue"})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 0 {
		t.Fatalf("missing issue response body=%q, want empty", encoded)
	}
}
