package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Compare the shipping metadata with the immutable Django observations, not a
// second list of expected values copied from the implementation.
func TestAssetsProtocolMetadataMatchesDjangoEvidence(t *testing.T) {
	var protocols map[string]struct {
		Allow         string `json:"allow"`
		Status        int    `json:"status"`
		Metadata      any    `json:"metadata"`
		SourceFixture string `json:"source_fixture"`
	}
	if err := json.Unmarshal(assetsProtocolMetadataJSON, &protocols); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs/go-backend-replacement/decisions/ASSET-001-invalid-methods.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decisions struct {
		Routes []struct {
			Path        string `json:"path"`
			SourceAllow string `json:"source_allow"`
			TargetAllow string `json:"target_allow"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(data, &decisions); err != nil {
		t.Fatal(err)
	}
	for path, protocol := range protocols {
		if protocol.SourceFixture == "" {
			continue
		}
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, protocol.SourceFixture))
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Response struct {
					Body    any               `json:"body"`
					Status  int               `json:"status"`
					Headers map[string]string `json:"headers"`
				} `json:"response"`
			}
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(protocol.Metadata, fixture.Response.Body) {
				t.Fatalf("metadata differs from %s", protocol.SourceFixture)
			}
			status := protocol.Status
			if status == 0 {
				status = 200
			}
			if status != fixture.Response.Status {
				t.Fatalf("status %d, source %d", status, fixture.Response.Status)
			}
			allow := fixture.Response.Headers["allow"]
			for _, decision := range decisions.Routes {
				if decision.Path == path {
					if allow != decision.SourceAllow {
						t.Fatalf("decision source Allow %q, fixture %q", decision.SourceAllow, allow)
					}
					allow = decision.TargetAllow
				}
			}
			if allow != protocol.Allow {
				t.Fatalf("Allow %q, expected %q", protocol.Allow, allow)
			}
		})
	}
}
