#!/usr/bin/env bash
# Usage: scripts/check-no-sleeps.sh [backend|frontend|all] [repo root]
# Fails if a test uses a fixed sleep (spec 004, Q-08): `time.Sleep` in
# backend/**/*_test.go or `setTimeout` in frontend/src/**/*.test.*.
# Event-driven waiting (tickers with a deadline, waitFor, findBy*) is fine.
set -euo pipefail

scope="${1:-all}"
root="${2:-$(cd "$(dirname "$0")/.." && pwd)}"
status=0

# scan <label> <dir> <pattern> <include glob>...
scan() {
  local label="$1" dir="$2" pattern="$3"
  shift 3
  local includes=()
  for glob in "$@"; do includes+=(--include="$glob"); done
  local hits
  hits=$(grep -rnE "${includes[@]}" "$pattern" "$dir" || true)
  if [ -n "$hits" ]; then
    echo "FAIL: fixed sleep in $label tests (use event-driven waiting):" >&2
    echo "$hits" >&2
    status=1
  fi
}

case "$scope" in
  backend) ;;
  frontend) ;;
  all) ;;
  *) echo "usage: $0 [backend|frontend|all] [repo root]" >&2; exit 2 ;;
esac

if [ "$scope" = backend ] || [ "$scope" = all ]; then
  scan backend "$root/backend" 'time\.Sleep' '*_test.go'
fi
if [ "$scope" = frontend ] || [ "$scope" = all ]; then
  scan frontend "$root/frontend/src" 'setTimeout' '*.test.*'
fi

[ "$status" -eq 0 ] && echo "No fixed sleeps in tests ($scope)."
exit "$status"
