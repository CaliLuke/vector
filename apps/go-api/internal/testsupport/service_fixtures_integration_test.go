//go:build integration

package testsupport_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const readinessFailure = "intentional fixture readiness failure"

// The child must fail its testing.T so the parent can inspect the result of
// cleanup after Fatal, without turning the expected failure into a suite failure.
func TestServiceContainerCleanupAfterReadinessFailure(t *testing.T) {
	if marker := os.Getenv("VECTOR_FIXTURE_FAILURE_MARKER"); marker != "" {
		testsupport.StartContainer(t, testcontainers.ContainerRequest{
			Name:       os.Getenv("VECTOR_FIXTURE_FAILURE_NAME"),
			Image:      "valkey/valkey:9.1.2-alpine",
			WaitingFor: failingReadiness{marker: marker},
		})
		t.Fatal("fixture unexpectedly accepted failed readiness")
	}

	sessionID := uuid.NewString()
	name := "vector-fixture-cleanup-" + sessionID
	marker := filepath.Join(t.TempDir(), "created-container")
	api, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.Close(t, api) })
	// The parent owns fallback removal even if the child crashes or regresses.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, cleanupErr := api.ContainerRemove(ctx, name, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
		if !errdefs.IsNotFound(cleanupErr) {
			require.NoError(t, cleanupErr)
		}
	})

	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, executable,
		"-test.run=^TestServiceContainerCleanupAfterReadinessFailure$", "-test.count=1", "-test.timeout=90s")
	command.Env = append(os.Environ(),
		"VECTOR_FIXTURE_FAILURE_MARKER="+marker,
		"VECTOR_FIXTURE_FAILURE_NAME="+name,
		// Keep fallback reaping for interrupted processes, but give this proof
		// its own reaper whose grace exceeds our two-minute deadline. It cannot
		// hide a missing explicit cleanup while the assertion is still valid.
		"TESTCONTAINERS_SESSION_ID="+sessionID,
		"TESTCONTAINERS_RYUK_DISABLED=false",
		"TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT=30m")
	output, runErr := command.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, runErr, &exitErr, "%s", output)
	require.Equal(t, 1, exitErr.ExitCode(), "%s", output)
	require.Contains(t, string(output), readinessFailure)
	id, err := os.ReadFile(marker)
	require.NoError(t, err, "%s", output)
	require.NotEmpty(t, id, "readiness failure must occur after container creation")
	_, err = api.ContainerInspect(ctx, string(id), client.ContainerInspectOptions{})
	require.NoError(t, ctx.Err(), "cleanup proof exceeded its deadline")
	require.True(t, errdefs.IsNotFound(err), "created container survived fixture failure: %v; child output: %s", err, output)
}

type failingReadiness struct{ marker string }

func (strategy failingReadiness) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	inspect, err := target.Inspect(ctx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(strategy.marker, []byte(inspect.ID), 0o600); err != nil {
		return err
	}
	return errors.New(readinessFailure)
}
