package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestProjectEstimatePointOptionsMetadataMatchesCorrectedSource(t *testing.T) {
	const pattern = "/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/"
	protocol := newProjectConfigurationSessionProtocol(pattern, "POST", "", "estimate points", false)
	var actual any
	if err := json.Unmarshal(protocol.optionsMetadata, &actual); err != nil {
		t.Fatalf("decode generated OPTIONS metadata: %v", err)
	}

	fixture, err := os.ReadFile("../../internal/parity/testdata/fixtures/project_configuration_corrected/options_estimate_points_valid_metadata.json")
	if err != nil {
		t.Fatalf("read corrected source fixture: %v", err)
	}
	var record struct {
		Response struct {
			Body any `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(fixture, &record); err != nil {
		t.Fatalf("decode corrected source fixture: %v", err)
	}
	if !reflect.DeepEqual(actual, record.Response.Body) {
		t.Fatalf("OPTIONS metadata differs from corrected source fixture:\nactual: %#v\nsource: %#v", actual, record.Response.Body)
	}
}
