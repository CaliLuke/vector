package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/notifications"
	api "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/require"
)

type notificationCleanupBody struct {
	*strings.Reader
	closed int
}

func (body *notificationCleanupBody) Close() error {
	body.closed++
	return errors.New("private input cleanup details")
}

func TestNotificationWriteOutcomeSurvivesBodyCleanupFailure(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		authorization error
		writeError    error
		publicError   string
		writes        int
	}{
		{name: "accepted", writes: 1},
		{name: "forbidden", authorization: store.Failure(403, "error", "denied"), publicError: "status_403"},
		{name: "write failed", writeError: errors.New("write failed"), publicError: "status_500", writes: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{
				authorizeErr: scenario.authorization, record: notificationBody(),
			}, writeError: scenario.writeError}
			service, ctx := actionService(t, repository)
			var output bytes.Buffer
			ctx = log.Context(ctx, log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }))
			body := &notificationCleanupBody{Reader: strings.NewReader("")}

			result, err := service.Archive(ctx, &api.ArchivePayload{Slug: "demo", Pk: "id"}, body)

			require.Equal(t, 1, body.closed)
			require.Equal(t, scenario.writes, repository.writes)
			if scenario.publicError == "" {
				require.NoError(t, err)
				require.NotNil(t, result)
			} else {
				requireActionError(t, err, scenario.publicError)
				require.Nil(t, result)
			}
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.NotEmpty(t, event["error_type"])
			require.Equal(t, false, event["context_canceled"])
			require.Equal(t, false, event["deadline_exceeded"])
			require.NotContains(t, output.String(), "private")
		})
	}
}
