package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"
)

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		requestID  string
		wantHeader string
	}{
		{
			name:       "keeps valid upstream ID",
			requestID:  "01989c80-222e-7d61-8c36-b6d00b74f8dd",
			wantHeader: "01989c80-222e-7d61-8c36-b6d00b74f8dd",
		},
		{
			name:      "replaces invalid upstream ID",
			requestID: "not-a-uuid",
		},
		{
			name: "assigns missing ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var contextID string
			handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var ok bool
				contextID, ok = RequestID(r.Context())
				if !ok {
					t.Fatal("request ID missing from context")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(RequestIDHeader, tt.requestID)
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			responseID := res.Header().Get(RequestIDHeader)
			if responseID != contextID {
				t.Fatalf("response ID = %q, context ID = %q", responseID, contextID)
			}
			if _, err := uuid.Parse(responseID); err != nil {
				t.Fatalf("response ID %q is not a UUID: %v", responseID, err)
			}
			if tt.wantHeader != "" && responseID != tt.wantHeader {
				t.Fatalf("response ID = %q, want %q", responseID, tt.wantHeader)
			}
			if tt.wantHeader == "" && responseID == tt.requestID {
				t.Fatalf("invalid request ID %q was accepted", tt.requestID)
			}
		})
	}
}
