#!/usr/bin/env bash
# Use the existing Colima profile on macOS; preserve Docker configuration elsewhere.
if [[ "$(uname -s)" == "Darwin" && -d "$HOME/.colima/default" ]]; then
    export DOCKER_HOST="${DOCKER_HOST:-unix://${HOME}/.colima/default/docker.sock}"
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
    # Keep the resource reaper enabled so interrupted tests cannot leak containers.
    # A caller may opt out only when its runner provides equivalent cleanup.
    export TESTCONTAINERS_RYUK_DISABLED="${TESTCONTAINERS_RYUK_DISABLED:-false}"
fi
# Keep Ryuk alive across idle gaps between package test binaries. Its stopped-container
# reuse race is tracked at https://github.com/testcontainers/testcontainers-go/issues/3867.
# Remove this default once the pinned version includes the reaper reuse fix in PR #3868.
# Fixture cleanup remains explicit; this is only the fallback reconnection grace.
export TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT="${TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT:-30m}"
export GOTOOLCHAIN=go1.27.3
