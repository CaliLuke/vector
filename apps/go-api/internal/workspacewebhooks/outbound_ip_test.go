package workspacewebhooks

import (
	"encoding/json"
	"net/netip"
	"os"
	"testing"
)

func TestOutboundIPSourceBoundaries(t *testing.T) {
	data, err := os.ReadFile("testdata/outbound_ip_policy.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Python string `json:"python"`
		Cases  []struct {
			Address string `json:"address"`
			Blocked bool   `json:"blocked"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Python == "" || len(fixture.Cases) < 100 {
		t.Fatal("missing source boundary corpus")
	}
	for _, test := range fixture.Cases {
		t.Run(test.Address, func(t *testing.T) {
			if got := blockedIP(netip.MustParseAddr(test.Address)); got != test.Blocked {
				t.Errorf("blockedIP = %v, source Python %s = %v", got, fixture.Python, test.Blocked)
			}
		})
	}
}
