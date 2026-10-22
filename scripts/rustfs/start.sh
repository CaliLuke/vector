#!/bin/sh

set -eu

: "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID is required for RustFS}"
: "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY is required for RustFS}"

export RUSTFS_ACCESS_KEY="$AWS_ACCESS_KEY_ID"
export RUSTFS_SECRET_KEY="$AWS_SECRET_ACCESS_KEY"

exec /entrypoint.sh "$@"
