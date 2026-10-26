package design_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/gen/http/workspace_favorites/client"
	server "github.com/CaliLuke/go-plane/gen/http/workspace_favorites/server"
	api "github.com/CaliLuke/go-plane/gen/workspace_favorites"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type workspaceFavoritesTransportService struct {
	api.Service
	allValues []*bool
}

func (service *workspaceFavoritesTransportService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}

func (service *workspaceFavoritesTransportService) List(_ context.Context, payload *api.ListPayload) ([]*api.WorkspaceFavorite, error) {
	service.allValues = append(service.allValues, payload.All)
	return []*api.WorkspaceFavorite{}, nil
}

func TestWorkspaceFavoriteListAllQueryParameterRoundTrips(t *testing.T) {
	service := &workspaceFavoritesTransportService{}
	mux := loomhttp.NewMuxer()
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	generated.Mount(mux)
	remote := httptest.NewServer(mux)
	defer remote.Close()
	transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), remote.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	for _, testCase := range []struct {
		name string
		all  *bool
	}{
		{name: "absent"},
		{name: "true", all: boolPointer(true)},
		{name: "false", all: boolPointer(false)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := transport.API.List()(context.Background(), &api.ListPayload{Slug: "acme", All: testCase.all})
			if err != nil {
				t.Fatal(err)
			}
		})
	}

	if len(service.allValues) != 3 || service.allValues[0] != nil || service.allValues[1] == nil || !*service.allValues[1] || service.allValues[2] == nil || *service.allValues[2] {
		t.Fatalf("decoded all query values = %#v, want absent, true, false", service.allValues)
	}
}

func boolPointer(value bool) *bool { return &value }
