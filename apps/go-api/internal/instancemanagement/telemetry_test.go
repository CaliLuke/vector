package instancemanagement

import "testing"

func TestTelemetryDestinationNeverExposesCredentials(t *testing.T) {
	for _, tc := range []struct{ endpoint, want string }{
		{"https://user:password@collector.example.test:4318/private-token?key=secret#secret", "https://collector.example.test:4318"},
		{"http://[::1]:4318/v1/logs", "http://[::1]:4318"},
		{"collector.example.test:4317", "https://collector.example.test:4317"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			status := exporterStatus(tc.endpoint, true, true)
			if status.Status != "enabled" || status.Destination == nil || *status.Destination != tc.want {
				t.Fatalf("unexpected sanitized status: %#v", status)
			}
		})
	}
}

func TestTelemetryStatusDistinguishesDisabledMissingAndInvalidDestinations(t *testing.T) {
	for _, tc := range []struct {
		endpoint string
		enabled  bool
		want     string
	}{
		{"", true, "not_configured"},
		{"http://logal:4318", false, "disabled"},
		{"", false, "disabled"},
		{"not-a-url", true, "invalid_configuration"},
		{"https://%invalid", true, "invalid_configuration"},
		{"file:///private/secret", true, "invalid_configuration"},
	} {
		t.Run(tc.want+tc.endpoint, func(t *testing.T) {
			status := exporterStatus(tc.endpoint, tc.enabled, false)
			if status.Status != tc.want {
				t.Fatalf("status=%q want %q", status.Status, tc.want)
			}
			if tc.want == "invalid_configuration" && status.Destination != nil {
				t.Fatal("invalid destination must not be exposed")
			}
		})
	}
}
