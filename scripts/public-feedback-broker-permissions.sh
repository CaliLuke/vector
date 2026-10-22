#!/usr/bin/env bash
set -euo pipefail

container=plane-api-state-test-mq-1
vhost=/public_feedback
user=plane

current_permissions() {
  docker exec "$container" rabbitmqctl list_permissions -p "$vhost" |
    awk -v user="$user" '$1 == user { print $2, $3, $4; found = 1 } END { if (!found) exit 1 }'
}

mode=${1:-}
case "$mode" in
  deny)
    current=$(current_permissions)
    if [[ "$current" != ".* .* .*" ]]; then
      printf 'refusing broker ACL change; expected baseline permissions .* .* .*, got %s\n' "$current" >&2
      exit 1
    fi
    docker exec "$container" rabbitmqctl set_permissions -p "$vhost" "$user" '.*' 'a^' '.*'
    expected='.* a^ .*'
    ;;
  restore)
    current=$(current_permissions)
    if [[ "$current" == ".* .* .*" ]]; then
      printf 'public feedback broker permissions are already restored\n'
      exit 0
    fi
    if [[ "$current" != ".* a^ .*" ]]; then
      printf 'refusing broker ACL restore; expected injected permissions .* a^ .*, got %s\n' "$current" >&2
      exit 1
    fi
    docker exec "$container" rabbitmqctl set_permissions -p "$vhost" "$user" '.*' '.*' '.*'
    expected='.* .* .*'
    ;;
  *)
    printf 'usage: %s deny|restore\n' "$0" >&2
    exit 2
    ;;
esac

current=$(current_permissions)
if [[ "$current" != "$expected" ]]; then
  printf 'broker ACL verification failed: expected %s, got %s\n' "$expected" "$current" >&2
  exit 1
fi
printf 'public feedback broker permissions %s: %s\n' "$mode" "$current"
