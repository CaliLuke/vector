package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type installationEffects struct {
	bucketInitializations int
	metricsAdmissions     int
}

func recordingInstallation(effects *installationEffects) Installation {
	return Installation{
		// A preflight failure must be returned before any connection is attempted.
		DatabaseURL:      "postgres://127.0.0.1:1/bootstrap-preflight",
		MachineSignature: "preflight-test", SecretKey: "bootstrap-test-secret",
		LookupEnv:     func(string) (string, bool) { return "", false },
		EnsureBucket:  func(context.Context) error { effects.bucketInitializations++; return nil },
		AdmitMetrics:  func(context.Context) error { effects.metricsAdmissions++; return nil },
		LatestVersion: func(context.Context, string) string { return "test" },
	}
}

func assertNoInstallationEffects(t *testing.T, effects installationEffects) {
	t.Helper()
	require.Zero(t, effects.bucketInitializations)
	require.Zero(t, effects.metricsAdmissions)
}

func TestInstallRejectsIncompleteConfigurationBeforeEffects(t *testing.T) {
	for _, invalid := range []struct {
		name      string
		configure func(*Installation)
		message   string
	}{
		{"unsupported database", func(i *Installation) { i.DatabaseURL = "https://127.0.0.1/database" }, "PostgreSQL URL"},
		{"blank machine signature", func(i *Installation) { i.MachineSignature = " \t " }, "machine signature"},
		{"missing secret", func(i *Installation) { i.SecretKey = "" }, "SECRET_KEY"},
		{"missing environment lookup", func(i *Installation) { i.LookupEnv = nil }, "environment lookup"},
		{"missing bucket initializer", func(i *Installation) { i.EnsureBucket = nil }, "bucket initializer"},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			var effects installationEffects
			installation := recordingInstallation(&effects)
			invalid.configure(&installation)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			t.Cleanup(cancel)

			err := Install(ctx, installation)

			require.ErrorContains(t, err, invalid.message)
			assertNoInstallationEffects(t, effects)
		})
	}
}

func TestInstallRequiresMetricsAdmissionBeforeEffects(t *testing.T) {
	var effects installationEffects
	installation := recordingInstallation(&effects)
	installation.AdmitMetrics = nil
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)

	err := Install(ctx, installation)

	require.ErrorIs(t, err, ErrMetricsAdmissionUnavailable)
	assertNoInstallationEffects(t, effects)
}
