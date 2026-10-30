package main

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
)

func TestNormalizeStickyMultipartPreservesBoundarySemantics(t *testing.T) {
	for _, scenario := range testsupport.MultipartBoundaryCases() {
		t.Run(scenario.Name, func(t *testing.T) {
			body, err := normalizeStickyRequestBody("multipart/form-data; boundary=sample", []byte(scenario.Body))
			if (err != nil) != scenario.WantError {
				t.Fatalf("normalization error=%v, want error=%v", err, scenario.WantError)
			}
			if scenario.WantError {
				return
			}
			var fields map[string]string
			if err := json.Unmarshal(body, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["title"] != scenario.WantTitle {
				t.Fatalf("title=%q, want %q", fields["title"], scenario.WantTitle)
			}
		})
	}
}
