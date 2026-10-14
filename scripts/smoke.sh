#!/usr/bin/env bash
# Usage: scripts/smoke.sh [base_url]   (default http://localhost:8080)
# End-to-end smoke test with curl only (no jq): healthz, one success for each of the
# 7 operations, and three error cases. Exits non-zero on the first mismatch (spec 005, D-03).
# Success bodies are compared exactly: the API's compact JSON and key order are part of its contract.
set -euo pipefail

base="${1:-http://localhost:8080}"
base="${base%/}"
json='Content-Type: application/json'

body_file="$(mktemp)"
trap 'rm -f "$body_file"' EXIT

checks=0
status=""
body=""

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

# request <method> <path> [data]: sets $status and $body.
request() {
  local method="$1" path="$2"
  local args=(-sS -X "$method" -o "$body_file" -w '%{http_code}')
  if [ "$#" -ge 3 ]; then args+=(-H "$json" --data "$3"); fi
  if ! status="$(curl "${args[@]}" "$base$path")"; then
    fail "cannot reach $base$path"
  fi
  body="$(cat "$body_file")"
}

# expect_body <name> <method> <path> <data|-> <status> <exact body>
expect_body() {
  local name="$1" method="$2" path="$3" data="$4" want_status="$5" want_body="$6"
  if [ "$data" = "-" ]; then request "$method" "$path"; else request "$method" "$path" "$data"; fi
  [ "$status" = "$want_status" ] || fail "$name: status $status, want $want_status ($body)"
  [ "$body" = "$want_body" ] || fail "$name: body $body, want $want_body"
  checks=$((checks + 1))
  echo "ok   $name"
}

# expect_problem <name> <path> <data> <status> <code>
expect_problem() {
  local name="$1" path="$2" data="$3" want_status="$4" code="$5"
  request POST "$path" "$data"
  [ "$status" = "$want_status" ] || fail "$name: status $status, want $want_status ($body)"
  case "$body" in
    *"\"code\":\"$code\""*) ;;
    *) fail "$name: body $body, want code $code" ;;
  esac
  checks=$((checks + 1))
  echo "ok   $name"
}

expect_body healthz GET /healthz - 200 '{"status":"ok"}'

expect_body add POST /api/v1/add '{"a":12,"b":3}' 200 '{"operation":"add","a":12,"b":3,"result":15}'
expect_body subtract POST /api/v1/subtract '{"a":10,"b":4}' 200 '{"operation":"subtract","a":10,"b":4,"result":6}'
expect_body multiply POST /api/v1/multiply '{"a":6,"b":7}' 200 '{"operation":"multiply","a":6,"b":7,"result":42}'
expect_body divide POST /api/v1/divide '{"a":10,"b":4}' 200 '{"operation":"divide","a":10,"b":4,"result":2.5}'
expect_body power POST /api/v1/power '{"a":2,"b":10}' 200 '{"operation":"power","a":2,"b":10,"result":1024}'
expect_body sqrt POST /api/v1/sqrt '{"a":9}' 200 '{"operation":"sqrt","a":9,"result":3}'
expect_body percentage POST /api/v1/percentage '{"a":20,"b":150}' 200 '{"operation":"percentage","a":20,"b":150,"result":30}'

expect_problem DIVISION_BY_ZERO /api/v1/divide '{"a":1,"b":0}' 422 DIVISION_BY_ZERO
expect_problem UNKNOWN_OPERATION /api/v1/modulo '{"a":1,"b":2}' 404 UNKNOWN_OPERATION
expect_problem INVALID_JSON /api/v1/add '{' 400 INVALID_JSON

echo "smoke: $checks checks passed against $base"
