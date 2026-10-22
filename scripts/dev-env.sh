#!/usr/bin/env bash
# Use the existing Colima profile on macOS; preserve Docker configuration elsewhere.
if [[ "$(uname -s)" == "Darwin" && -d "$HOME/.colima/default" ]]; then
    export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
    export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
    export TESTCONTAINERS_RYUK_DISABLED=true
fi
export GOTOOLCHAIN=go1.27.3
