package design_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/gen/http/instance_management/client"
	api "github.com/CaliLuke/go-plane/gen/instance_management"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestInstancePublicGeneratedClientsRecognizeAnonymousThrottle(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error_code":5900,"error_message":"RATE_LIMIT_EXCEEDED"}`))
	}))
	defer remote.Close()
	transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), remote.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for name, endpoint := range map[string]loom.Endpoint{"root": transport.API.RootGet(), "signup": transport.API.SignupVisited()} {
		t.Run(name, func(t *testing.T) {
			_, err := endpoint(t.Context(), nil)
			var throttled *api.InstanceManagementError
			if !errors.As(err, &throttled) || throttled.Kind != "throttled" {
				t.Fatalf("expected declared throttled error, got %T: %v", err, err)
			}
			if throttled.RetryAfter == nil || *throttled.RetryAfter != 60 {
				t.Fatalf("expected Retry-After 60, got %v", throttled.RetryAfter)
			}
		})
	}
}
