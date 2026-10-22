package assets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestEnsureBucket(t *testing.T) {
	for _, test := range []struct {
		name      string
		statuses  []int
		methods   []string
		wantError bool
	}{
		{"existing", []int{200}, []string{"HEAD"}, false},
		{"missing", []int{404, 200, 200}, []string{"HEAD", "PUT", "HEAD"}, false},
		{"forbidden", []int{403}, []string{"HEAD"}, true},
		{"create denied", []int{404, 403}, []string{"HEAD", "PUT"}, true},
		{"verification denied", []int{404, 200, 403}, []string{"HEAD", "PUT", "HEAD"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.URL.Path != "/plane-bootstrap-test" {
					t.Errorf("bucket path = %q", r.URL.Path)
				}
				if len(methods) > len(test.statuses) {
					w.WriteHeader(403)
					return
				}
				w.WriteHeader(test.statuses[len(methods)-1])
			}))
			defer server.Close()
			storage, err := NewStorage(StorageConfig{Bucket: "plane-bootstrap-test", Region: "us-east-1", Endpoint: server.URL, PathStyle: true, AccessKeyID: "test", SecretAccessKey: "test"})
			if err != nil {
				t.Fatal(err)
			}
			err = storage.EnsureBucket(context.Background())
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError = %t", err, test.wantError)
			}
			if !slices.Equal(methods, test.methods) {
				t.Fatalf("methods = %v, want %v", methods, test.methods)
			}
		})
	}
}
