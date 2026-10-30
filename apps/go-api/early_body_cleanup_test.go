package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	customapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	"github.com/CaliLuke/go-plane/internal/customviewcreation"
	"github.com/CaliLuke/go-plane/internal/projectpublication"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	publicationapi "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type earlyCleanupBody struct{ reads, closes int }

func (body *earlyCleanupBody) Read([]byte) (int, error) { body.reads++; return 0, io.EOF }
func (body *earlyCleanupBody) Close() error {
	body.closes++
	return errors.New("private cleanup contents")
}

type missingPublicationRepository struct {
	projectPublicationRepositoryStub
}

func (*missingPublicationRepository) Exists(context.Context, string, string, string) error {
	return projectpublication.ErrNotFound
}

func TestEarlyRequestRejectionReportsCleanupFailure(t *testing.T) {
	custom := &customViewCreationService{repo: &customViewCreationRepositoryStub{targetErr: customviewcreation.ErrNotFound}}
	publication := &projectPublicationService{repo: &missingPublicationRepository{}}
	for _, test := range []struct {
		name          string
		authenticated bool
		want          string
		invoke        func(*testing.T, context.Context, io.ReadCloser) error
	}{
		{"project view unauthenticated", false, "status_401", func(_ *testing.T, ctx context.Context, body io.ReadCloser) error {
			return custom.CreateProjectView(ctx, &customapi.CreateProjectViewPayload{Slug: "demo"}, body)
		}},
		{"workspace view unauthenticated", false, "status_401", func(_ *testing.T, ctx context.Context, body io.ReadCloser) error {
			return custom.CreateWorkspaceView(ctx, &customapi.CreateWorkspaceViewPayload{Slug: "demo"}, body)
		}},
		{"project view target missing", true, "status_404", func(_ *testing.T, ctx context.Context, body io.ReadCloser) error {
			return custom.CreateProjectView(ctx, &customapi.CreateProjectViewPayload{Slug: "demo"}, body)
		}},
		{"workspace view target missing", true, "status_404", func(_ *testing.T, ctx context.Context, body io.ReadCloser) error {
			return custom.CreateWorkspaceView(ctx, &customapi.CreateWorkspaceViewPayload{Slug: "demo"}, body)
		}},
		{"publication create unauthenticated", false, "status_401", func(t *testing.T, ctx context.Context, body io.ReadCloser) error {
			_, response, err := publication.Create(ctx, &publicationapi.CreatePayload{Slug: "demo"}, body)
			if response != nil {
				testsupport.Close(t, response)
			}
			return err
		}},
		{"publication update unauthenticated", false, "status_401", func(t *testing.T, ctx context.Context, body io.ReadCloser) error {
			_, response, err := publication.Update(ctx, &publicationapi.UpdatePayload{Slug: "demo"}, body)
			if response != nil {
				testsupport.Close(t, response)
			}
			return err
		}},
		{"publication invalid identifier", true, "status_404", func(t *testing.T, ctx context.Context, body io.ReadCloser) error {
			_, response, err := publication.Update(ctx, &publicationapi.UpdatePayload{Slug: "demo", Pk: "invalid"}, body)
			if response != nil {
				testsupport.Close(t, response)
			}
			return err
		}},
		{"publication target missing", true, "status_404", func(t *testing.T, ctx context.Context, body io.ReadCloser) error {
			_, response, err := publication.Update(ctx, &publicationapi.UpdatePayload{Slug: "demo", Pk: "00000000-0000-4000-8000-000000000001"}, body)
			if response != nil {
				testsupport.Close(t, response)
			}
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			if test.authenticated {
				ctx = context.WithValue(ctx, customViewCreationPrincipalKey{}, customViewCreationPrincipal{userID: "00000000-0000-4000-8000-000000000001", timezone: "UTC"})
				ctx = context.WithValue(ctx, projectPublicationPrincipalKey{}, projectPublicationPrincipal{userID: "00000000-0000-4000-8000-000000000001", timezone: "UTC"})
			}
			var output bytes.Buffer
			ctx = log.Context(ctx, log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			body := &earlyCleanupBody{}
			err := test.invoke(t, ctx, body)
			require.Error(t, err)
			var actual string
			var customFailure *customapi.CustomViewCreationErrorResponse
			var publicationFailure *publicationapi.ProjectPublicationErrorResponse
			switch {
			case errors.As(err, &customFailure):
				actual = customFailure.Name
			case errors.As(err, &publicationFailure):
				actual = publicationFailure.Name
			default:
				t.Fatalf("unexpected error type %T: %v", err, err)
			}
			require.Equal(t, test.want, actual)
			require.Zero(t, body.reads)
			require.Equal(t, 1, body.closes)
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.NotEmpty(t, event["error_type"])
			require.False(t, strings.Contains(output.String(), "private"))
		})
	}
}
