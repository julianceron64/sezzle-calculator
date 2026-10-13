#!/usr/bin/env bash
# PostToolUse hook: auto-format the file Claude just edited.
# Never blocks the agent — formatting/lint failures are surfaced by `make check`.
command -v jq >/dev/null 2>&1 || exit 0

file=$(jq -r '.tool_input.file_path // empty')
[[ -z "$file" || ! -f "$file" ]] && exit 0
root="${CLAUDE_PROJECT_DIR:-$(pwd)}"

case "$file" in
  *.go)
    command -v gofmt >/dev/null 2>&1 && gofmt -w "$file" 2>/dev/null
    ;;
  *.ts|*.tsx|*.css|*.json)
    prettier="$root/frontend/node_modules/.bin/prettier"
    if [[ "$file" == "$root/frontend/"* && -x "$prettier" ]]; then
      "$prettier" --write --log-level warn "$file" 2>/dev/null
    fi
    ;;
esac
exit 0
