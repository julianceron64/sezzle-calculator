# Prompts used

Workflow: specs + ADRs written first, then Claude Code implemented one spec per session.
`session-log.md` is filled automatically by a `UserPromptSubmit` hook (`.claude/hooks/log-prompt.sh`)
and contains every prompt verbatim. This file is the curated version.

## 00 — Planning (specs, ADRs, CLAUDE.md)

<!-- Paste here the prompt(s) you used to draft the specs/ADRs, if you used an assistant for that. -->

## 01 — Kickoff

<!-- Paste the kickoff prompt here -->

## 02 — Per-spec

```text
/implement-spec 001
/implement-spec 002
/implement-spec 003
/implement-spec 004
/implement-spec 005
```

## 03 — Notable corrections

<!-- Plan corrections, review findings you chose to fix, and why. -->
