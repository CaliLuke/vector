package main

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	"github.com/stretchr/testify/require"
)

type numericStickyRepository struct {
	*apiV1StickyMiddlewareRepository
	patches int
}

func (r *numericStickyRepository) PatchOwned(ctx context.Context, id stickystore.StickyID, scope stickystore.Scope, input stickystore.PatchStickyInput) (stickystore.Sticky, error) {
	r.patches++
	return r.apiV1StickyMiddlewareRepository.PatchOwned(ctx, id, scope, input)
}

func TestStickyNonfiniteOrderReturns400BeforePersistence(t *testing.T) {
	for _, method := range []string{"POST", "PATCH", "PUT"} {
		for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
			for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e999"} {
				t.Run(method+"/"+contentType+"/"+value, func(t *testing.T) {
					repository := &numericStickyRepository{apiV1StickyMiddlewareRepository: &apiV1StickyMiddlewareRepository{}}
					service, err := systemapi.NewAPIV1Stickies(repository, &apiV1StickyMiddlewareKeys{}, &apiV1StickyMiddlewareThrottle{allowed: true}, apiV1StickyMiddlewareEffects{})
					require.NoError(t, err)
					handler := newAPIV1StickyMiddlewareHandler(t, service, &activityHTTPService{}, &apiV1ActivityLoggerStub{}, defaultFileSizeLimit)
					path := "/api/v1/workspaces/demo/stickies/"
					if method != "POST" {
						path += "0198a28d-d227-7752-82ac-bc4c1c290703/"
					}
					body := `{"name":"Rejected","sort_order":"` + value + `"}`
					if contentType == "application/x-www-form-urlencoded" {
						body = url.Values{"name": {"Rejected"}, "sort_order": {value}}.Encode()
					}
					request := httptest.NewRequest(method, path, strings.NewReader(body))
					request.Header.Set("Content-Type", contentType)
					request.Header.Set("X-Api-Key", "sticky-key")
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					require.Equal(t, 400, response.Code, response.Body.String())
					require.JSONEq(t, `{"sort_order":["A valid number is required."]}`, response.Body.String())
					require.Zero(t, repository.createCalls)
					require.Zero(t, repository.patches)
				})
			}
		}
	}
}
