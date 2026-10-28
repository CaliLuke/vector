#!/usr/bin/env bash
# Use the existing Colima profile on macOS; preserve Docker configuration elsewhere.
if [[ "$(uname -s)" == "Darwin" && -d "$HOME/.colima/default" ]]; then
    export DOCKER_HOST="${DOCKER_HOST:-unix://${HOME}/.colima/default/docker.sock}"
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
    # Keep the resource reaper enabled so interrupted tests cannot leak containers.
    # A caller may opt out only when its runner provides equivalent cleanup.
    export TESTCONTAINERS_RYUK_DISABLED="${TESTCONTAINERS_RYUK_DISABLED:-false}"
    # Native host DNS avoids the SSH sidecar wait defect tracked in testcontainers-go#4054.
    # Remove this default once the released wait fix and Colima SSH forwarding are verified.
    export VECTOR_TEST_HOST_GATEWAY="${VECTOR_TEST_HOST_GATEWAY-host.docker.internal}"
fi
export GOTOOLCHAIN=go1.27.3
