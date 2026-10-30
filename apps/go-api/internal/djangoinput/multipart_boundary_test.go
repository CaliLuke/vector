package djangoinput

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
)

func TestParseMultipartPreservesBoundarySemantics(t *testing.T) {
	for _, scenario := range testsupport.MultipartBoundaryCases() {
		t.Run(scenario.Name, func(t *testing.T) {
			input, err := Parse(strings.NewReader(scenario.Body), "multipart/form-data; boundary=sample")
			if (err != nil) != scenario.WantError {
				t.Fatalf("Parse error=%v, want error=%v", err, scenario.WantError)
			}
			if !scenario.WantError && scenario.WantTitle != "" && input.Fields["title"] != scenario.WantTitle {
				t.Fatalf("title=%v, want %q", input.Fields["title"], scenario.WantTitle)
			}
		})
	}
}
