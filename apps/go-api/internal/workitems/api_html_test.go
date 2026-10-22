package workitems

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed testdata/api_html.json
var apiHTMLSourceFixtures []byte

// TestAPIHTMLSourceParity compares the stored value after the API
// IssueSerializer's lxml parse/serialization and nh3 sanitization. The source
// expressions and dependency versions are recorded in testdata/api_html.json.
func TestAPIHTMLSourceParity(t *testing.T) {
	var fixture struct {
		Cases []struct {
			Name       string `json:"name"`
			Input      string `json:"input"`
			Stored     string `json:"stored"`
			ParseError bool   `json:"parse_error"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(apiHTMLSourceFixtures, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) < 20 {
		t.Fatalf("only %d API HTML source vectors", len(fixture.Cases))
	}
	for _, testCase := range fixture.Cases {
		t.Run(testCase.Name, func(t *testing.T) {
			got, err := normalizeAPIHTML(testCase.Input)
			if testCase.ParseError {
				if err == nil {
					t.Fatalf("normalizeAPIHTML(%q) = %q, want source parse error", testCase.Input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeAPIHTML(%q): %v", testCase.Input, err)
			}
			if got != testCase.Stored {
				t.Errorf("stored HTML = %q, want %q", got, testCase.Stored)
			}
		})
	}
}
