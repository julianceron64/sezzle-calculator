#!/usr/bin/env bash
# UserPromptSubmit hook: append every human prompt to prompts/session-log.md.
# The assignment asks to "share any prompts that you used" — this makes the log automatic and verbatim.
command -v jq >/dev/null 2>&1 || exit 0

root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
log="$root/prompts/session-log.md"
prompt=$(jq -r '.prompt // empty')
[[ -z "$prompt" ]] && exit 0

{
  printf '\n### %s\n\n~~~~text\n%s\n~~~~\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$prompt"
} >>"$log"
exit 0
