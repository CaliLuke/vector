package djangoinput

import (
	"encoding/json"
	"os"
	"testing"
)

func TestURLValidatorMatchesDjangoSource(t *testing.T) {
	data, err := os.ReadFile("testdata/django_url_validator.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Metadata struct {
			DjangoVersion string `json:"django_version"`
		} `json:"metadata"`
		Cases []struct {
			Name  string `json:"name"`
			Input string `json:"input"`
			Valid bool   `json:"valid"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if fixtures.Metadata.DjangoVersion != "5.2.15" || len(fixtures.Cases) == 0 {
		t.Fatal("missing pinned Django source characterization")
	}
	for _, tc := range fixtures.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := ValidURL(tc.Input); got != tc.Valid {
				t.Fatalf("ValidURL(%q)=%v; Django source=%v", tc.Input, got, tc.Valid)
			}
		})
	}
}
