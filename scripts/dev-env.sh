#!/usr/bin/env bash
# Use Docker Desktop on macOS; preserve explicitly selected engines and Linux CI.
if [[ "$(uname -s)" == "Darwin" && -z "${DOCKER_HOST:-}" && -z "${DOCKER_CONTEXT:-}" ]]; then
    export DOCKER_HOST="unix://${HOME}/.docker/run/docker.sock"
fi
if [[ "${DOCKER_HOST:-}" == "unix://${HOME}/.docker/run/docker.sock" || "${DOCKER_CONTEXT:-}" == "desktop-linux" ]]; then
    # Desktop installs its CLI and credential helper here without admin access.
    case ":$PATH:" in
        *":${HOME}/.docker/bin:"*) ;;
        *) export PATH="${HOME}/.docker/bin:$PATH" ;;
    esac
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="${TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE:-/var/run/docker.sock}"
    export TESTCONTAINERS_HOST_OVERRIDE="${TESTCONTAINERS_HOST_OVERRIDE:-127.0.0.1}"
    export LIVE_TEST_REDIS_BIND_HOST="${LIVE_TEST_REDIS_BIND_HOST:-127.0.0.1}"
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
