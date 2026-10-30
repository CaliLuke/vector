package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/project_state_reads"
	store "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/stretchr/testify/require"
)

func TestProjectStateDeletionRetainsPublicFailureAfterRollbackError(t *testing.T) {
	primary := &store.WriteFailure{Status: 404, Body: map[string]string{"detail": "No Project matches the given query."}}
	for _, failure := range []struct {
		name string
		err  error
	}{
		{"original", primary},
		{"cleanup also failed", errors.Join(primary, errors.New("private rollback details"))},
	} {
		t.Run(failure.name, func(t *testing.T) {
			repository := &stateWriteBoundaryRepository{allowed: true, failure: failure.err}
			service := &projectStateReadsService{repository: repository}
			ctx := context.WithValue(t.Context(), projectStateReadsPrincipal{}, uuid.New())

			err := service.Delete(ctx, &api.DeletePayload{Slug: "demo", ProjectID: uuid.New().String(), Pk: uuid.New().String()})

			require.Len(t, repository.writes, 1)
			var response *api.ProjectStateWritesAppError
			require.ErrorAs(t, err, &response)
			require.Equal(t, "not_found", response.Kind)
			body, marshalErr := json.Marshal(response.Body)
			require.NoError(t, marshalErr)
			require.JSONEq(t, `{"detail":"No Project matches the given query."}`, string(body))
			require.False(t, strings.Contains(err.Error(), "private"))
		})
	}
}
