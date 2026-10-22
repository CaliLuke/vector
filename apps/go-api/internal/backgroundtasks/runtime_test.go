package backgroundtasks

import (
	"net/url"
	"testing"
	"time"
)

func TestNextDailyUTCScheduleUsesNextStrictBoundary(t *testing.T) {
	testCases := []struct {
		name string
		now  time.Time
		hour int
		min  int
		want time.Time
	}{
		{name: "before scheduled minute", now: time.Date(2026, 10, 22, 1, 29, 59, 0, time.FixedZone("PDT", -7*60*60)), hour: 8, min: 30, want: time.Date(2026, 10, 22, 8, 30, 0, 0, time.UTC)},
		{name: "at scheduled minute rolls to next day", now: time.Date(2026, 10, 22, 1, 30, 0, 0, time.UTC), hour: 1, min: 30, want: time.Date(2026, 10, 23, 1, 30, 0, 0, time.UTC)},
		{name: "after scheduled minute rolls to next day", now: time.Date(2026, 10, 22, 3, 46, 0, 0, time.UTC), hour: 3, min: 45, want: time.Date(2026, 10, 23, 3, 45, 0, 0, time.UTC)},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if got := nextDailyUTC(test.now, test.hour, test.min); !got.Equal(test.want) {
				t.Fatalf("next daily UTC boundary = %s, want %s", got, test.want)
			}
		})
	}
}

func TestWithExecutionSearchPath(t *testing.T) {
	got, err := withExecutionSearchPath("postgres://plane:secret@db:5432/plane?sslmode=require&application_name=api")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("options") != "-c search_path=plane_execution,public" {
		t.Fatalf("PostgreSQL options = %q", parsed.Query().Get("options"))
	}
	if parsed.Query().Get("sslmode") != "require" || parsed.Query().Get("application_name") != "api" {
		t.Fatalf("unrelated PostgreSQL settings were lost: %s", parsed.RawQuery)
	}
}

func TestWithExecutionSearchPathRejectsUnownedOptions(t *testing.T) {
	if _, err := withExecutionSearchPath("postgres://plane:secret@db:5432/plane?options=-c%20statement_timeout%3D5000"); err == nil {
		t.Fatal("expected an existing PostgreSQL options setting to be rejected")
	}
}

func TestNullableTextMatchesDjangoTextFieldInputs(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		want  any
	}{
		{name: "null", value: nil, want: nil},
		{name: "empty", value: "", want: ""},
		{name: "text", value: "payload", want: "payload"},
		{name: "bytes", value: []byte("payload"), want: "payload"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := nullableText(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("nullableText(%#v) = %#v, want %#v", test.value, got, test.want)
			}
		})
	}
	if _, err := nullableText(map[string]string{"invalid": "Django TextField input"}); err == nil {
		t.Fatal("expected non-text payload to be rejected")
	}
}

func TestCleanupRetentionDaysMatchPlaneSettings(t *testing.T) {
	for _, test := range []struct {
		task     CleanupTask
		variable string
		fallback int
	}{
		{task: CleanupAPIActivityLogs, variable: "API_ACTIVITY_LOG_RETENTION_DAYS", fallback: 14},
		{task: CleanupEmailNotificationLogs, variable: "EMAIL_LOG_RETENTION_DAYS", fallback: 7},
		{task: CleanupWebhookLogs, variable: "WEBHOOK_LOG_RETENTION_DAYS", fallback: 14},
	} {
		t.Run(string(test.task), func(t *testing.T) {
			t.Setenv(test.variable, "3")
			if got, ok := cleanupRetention(test.task); !ok || got != 3 {
				t.Fatalf("configured retention = %d, %t; want 3, true", got, ok)
			}
			t.Setenv(test.variable, "-1")
			if got, ok := cleanupRetention(test.task); !ok || got != test.fallback {
				t.Fatalf("negative retention = %d, %t; want default %d, true", got, ok, test.fallback)
			}
			t.Setenv(test.variable, "invalid")
			if got, ok := cleanupRetention(test.task); !ok || got != test.fallback {
				t.Fatalf("invalid retention = %d, %t; want default %d, true", got, ok, test.fallback)
			}
		})
	}
	if _, ok := cleanupRetention(CleanupPageVersions); ok {
		t.Fatal("version cleanup must not use a time-based retention window")
	}
}

func TestUnuploadedAssetRetentionMatchesCeleryConfiguration(t *testing.T) {
	t.Setenv("UNUPLOADED_ASSET_DELETE_DAYS", "7")
	if got, err := unuploadedAssetRetentionDays(); err != nil || got != 7 {
		t.Fatalf("configured unuploaded asset retention = %d, %v; want 7 days", got, err)
	}
	t.Setenv("UNUPLOADED_ASSET_DELETE_DAYS", "invalid")
	if _, err := unuploadedAssetRetentionDays(); err == nil {
		t.Fatal("invalid unuploaded asset retention must be surfaced")
	}
	t.Setenv("UNUPLOADED_ASSET_DELETE_DAYS", "-1")
	if _, err := unuploadedAssetRetentionDays(); err == nil {
		t.Fatal("negative unuploaded asset retention must not trigger premature deletion")
	}
}
