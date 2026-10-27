package devtools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHasLocalLoggingMarkerRequiresExactCurrentServiceRecord(t *testing.T) {
	const marker = "local-logging-check-current"
	cutoff := time.Date(2026, time.October, 26, 23, 0, 0, 123456789, time.UTC)
	rows := func(receivedAt, service, body string) string {
		return `{"rows":[{"received_at":"` + receivedAt + `","service_name":"` + service + `","body":{"string":"` + body + `"}}]}`
	}
	tests := []struct {
		name       string
		response   string
		wantResult bool
	}{
		{
			name:       "exact current marker",
			response:   rows(cutoff.Format(time.RFC3339Nano), "plane-logging-check", marker),
			wantResult: true,
		},
		{
			name:     "stale marker",
			response: rows(cutoff.Add(-time.Nanosecond).Format(time.RFC3339Nano), "plane-logging-check", marker),
		},
		{
			name:     "marker is only a substring",
			response: rows(cutoff.Format(time.RFC3339Nano), "plane-logging-check", "prefix-"+marker),
		},
		{
			name:     "other service",
			response: rows(cutoff.Format(time.RFC3339Nano), "plane-go-api", marker),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			scriptDir := filepath.Join(root, "scripts")
			if err := os.MkdirAll(scriptDir, 0700); err != nil {
				t.Fatal(err)
			}
			callsPath := filepath.Join(root, "calls")
			script := `#!/bin/sh
printf '%s\n' "$*" >> "$LOGAL_TEST_CALLS"
printf '%s\n' "$LOGAL_TEST_RESPONSE"
`
			if err := os.WriteFile(filepath.Join(scriptDir, "logal-query.sh"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("LOGAL_TEST_CALLS", callsPath)
			t.Setenv("LOGAL_TEST_RESPONSE", test.response)

			got, err := hasLocalLoggingMarker(context.Background(), root, marker, cutoff)
			if err != nil {
				t.Fatalf("hasLocalLoggingMarker() error = %v", err)
			}
			if got != test.wantResult {
				t.Fatalf("hasLocalLoggingMarker() = %t, want %t", got, test.wantResult)
			}

			calls, err := os.ReadFile(callsPath)
			if err != nil {
				t.Fatalf("read query arguments: %v", err)
			}
			for _, want := range []string{
				"logs --service plane-logging-check --since " + cutoff.Format(time.RFC3339Nano),
				"--time received --json --details --timeout 15s",
			} {
				if !strings.Contains(string(calls), want) {
					t.Errorf("query arguments missing %q: %s", want, calls)
				}
			}
		})
	}
}

func TestHasLocalGoErrorTraceRequiresErrorLogAndMatchingErrorSpan(t *testing.T) {
	const traceID = "0123456789abcdef0123456789abcdef"
	const spanID = "0123456789abcdef"
	cutoff := time.Date(2026, time.October, 26, 23, 0, 0, 123456789, time.UTC)

	tests := []struct {
		name       string
		logs       string
		spans      string
		wantResult bool
	}{
		{
			name:       "matching error log and span",
			logs:       `{"rows":[{"span_id":"` + spanID + `"}]}`,
			spans:      `{"rows":[{"span_id":"` + spanID + `","status_code":2}]}`,
			wantResult: true,
		},
		{
			name:       "missing error log",
			logs:       `{"rows":[]}`,
			spans:      `{"rows":[{"span_id":"` + spanID + `","status_code":2}]}`,
			wantResult: false,
		},
		{
			name:       "different span ID",
			logs:       `{"rows":[{"span_id":"` + spanID + `"}]}`,
			spans:      `{"rows":[{"span_id":"fedcba9876543210","status_code":2}]}`,
			wantResult: false,
		},
		{
			name:       "span has no error status",
			logs:       `{"rows":[{"span_id":"` + spanID + `"}]}`,
			spans:      `{"rows":[{"span_id":"` + spanID + `","status_code":0}]}`,
			wantResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			scriptDir := filepath.Join(root, "scripts")
			if err := os.MkdirAll(scriptDir, 0700); err != nil {
				t.Fatal(err)
			}
			callsPath := filepath.Join(root, "calls")
			script := `#!/bin/sh
signal="$1"
shift
printf '%s %s\n' "$signal" "$*" >> "$LOGAL_TEST_CALLS"
case "$signal" in
  logs) printf '%s\n' "$LOGAL_TEST_LOGS" ;;
  spans) printf '%s\n' "$LOGAL_TEST_SPANS" ;;
  *) exit 2 ;;
esac
`
			if err := os.WriteFile(filepath.Join(scriptDir, "logal-query.sh"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("LOGAL_TEST_CALLS", callsPath)
			t.Setenv("LOGAL_TEST_LOGS", test.logs)
			t.Setenv("LOGAL_TEST_SPANS", test.spans)

			got, err := hasLocalGoErrorTrace(context.Background(), root, traceID, cutoff)
			if err != nil {
				t.Fatalf("hasLocalGoErrorTrace() error = %v", err)
			}
			if got != test.wantResult {
				t.Fatalf("hasLocalGoErrorTrace() = %t, want %t", got, test.wantResult)
			}

			calls, err := os.ReadFile(callsPath)
			if err != nil {
				t.Fatalf("read query arguments: %v", err)
			}
			text := string(calls)
			for _, want := range []string{
				"logs --service plane-go-api --trace-id " + traceID + " --json --since " + cutoff.Format(time.RFC3339Nano) + " --time received --timeout 15s --level error",
				"spans --service plane-go-api --trace-id " + traceID + " --json --since " + cutoff.Format(time.RFC3339Nano) + " --time received --timeout 15s",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("query arguments missing %q: %s", want, text)
				}
			}
		})
	}
}
