#!/bin/bash

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT

mkdir -p "$test_root/bin"
for service in web go-api space admin live; do
    mkdir -p "$test_root/apps/$service"
    cp "$repo_root/apps/$service/.env.example" "$test_root/apps/$service/.env.example"
done
cp "$repo_root/.env.example" "$test_root/.env.example"
cp "$repo_root/setup.sh" "$test_root/setup.sh"

cat >"$test_root/bin/corepack" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >>"$COREPACK_CALL_LOG"
test "$#" -eq 2
test "$1" = "pnpm"
test "$2" = "install"
EOF
chmod +x "$test_root/bin/corepack"

cat >"$test_root/bin/pnpm" <<'EOF'
#!/bin/sh
echo "setup.sh called pnpm directly" >&2
exit 99
EOF
chmod +x "$test_root/bin/pnpm"

export COREPACK_CALL_LOG="$test_root/corepack-calls.log"
(
    cd "$test_root"
    PATH="$test_root/bin:/usr/bin:/bin" ./setup.sh >/dev/null
)

test "$(cat "$COREPACK_CALL_LOG")" = "pnpm install"
test -s "$test_root/apps/go-api/.env"
grep -Eq '^SECRET_KEY="[a-z0-9]{50}"$' "$test_root/apps/go-api/.env"

cp "$test_root/apps/go-api/.env" "$test_root/original-api.env"
echo 'EXISTING_LOCAL_SETTING=keep-me' >>"$test_root/apps/web/.env"
(
    cd "$test_root"
    PATH="$test_root/bin:/usr/bin:/bin" ./setup.sh >/dev/null
)
cmp "$test_root/original-api.env" "$test_root/apps/go-api/.env"
grep -q '^EXISTING_LOCAL_SETTING=keep-me$' "$test_root/apps/web/.env"

echo "setup rehearsal passed without Corepack activation"
