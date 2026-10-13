#!/usr/bin/env bash
# Usage: scripts/check-go-coverage.sh <profile relative to repo root> <min percent>
# Fails if total statement coverage is below the threshold (ADR-012).
set -euo pipefail

profile="${1:?coverage profile required}"
min="${2:?minimum percent required}"
root="$(cd "$(dirname "$0")/.." && pwd)"

total=$(cd "$root/backend" && go tool cover -func="$root/$profile" | awk '/^total:/ {gsub("%","",$3); print $3}')

echo "Backend coverage: ${total}% (gate: ${min}%)"
awk -v t="$total" -v m="$min" 'BEGIN { exit !(t+0 >= m+0) }' || {
  echo "FAIL: backend coverage ${total}% is below ${min}%" >&2
  exit 1
}
