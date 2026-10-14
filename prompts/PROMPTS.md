# Prompts used

Workflow: specs + ADRs written first, then Claude Code implemented one spec per session.
`session-log.md` is filled automatically by a `UserPromptSubmit` hook (`.claude/hooks/log-prompt.sh`)
and contains every prompt verbatim. This file is the curated version: the prompts that mattered, and the
corrections that changed the outcome. Prompts are quoted exactly (typos included). `[log HH:MM]` points to the entry
in [`session-log.md`](session-log.md) with that UTC time: entries up to 23:53 are dated 2026-10-13, the rest 2026-10-14.

The three commands every spec ran through are defined in the repo, not typed each time:

| Command | Defined in | What it does |
|---|---|---|
| `/implement-spec 00X` | [`.claude/commands/implement-spec.md`](../.claude/commands/implement-spec.md) | Read the docs, check dependencies, produce a plan with an AC-to-test table, **stop and wait**; after approval, TDD in small commits, `make check`, final report |
| `/review 00X` | [`.claude/commands/review.md`](../.claude/commands/review.md) | Run the read-only `spec-reviewer` subagent ([`.claude/agents/spec-reviewer.md`](../.claude/agents/spec-reviewer.md)) and show its findings by severity without fixing anything |
| `/check` | [`.claude/commands/check.md`](../.claude/commands/check.md) | Run the full quality gate and summarise failures |

## 00 — Planning (specs, ADRs, CLAUDE.md)

The specs, the ADRs and `CLAUDE.md` were already in the repository before the first Claude Code session (I wrote the
specs and the ADRs), so there is no drafting prompt to show. Claude Code's first job was to audit them (see 01), and the
amendments that came out of that audit are recorded in `docs/DECISIONS.md` under each ADR's "Amendment" entries.

## 01 — Kickoff

The first session was read-only: audit my specs and ADRs before any code existed. `[log 23:31, 2026-10-13]`

```text
You are the implementation agent for a Sezzle take-home assignment: a full-stack calculator
(React + TypeScript frontend, Go REST microservice). I own the architecture. You implement
the specs faithfully; you do not redesign them, and you do not make architectural decisions, only suggestions approved by me.

This session is READ-ONLY and with the intent of planning everything(Claude Planning Mode). Do not create or modify any file and do not run /implement-spec yet.

1. Read, in this order: CLAUDE.md, specs/README.md, docs/DECISIONS.md, specs/001 through 005.
2. Inspect the repo (git log, tree). Current state: the Spect Driven Development kit is already in the repository; backend/ has go.mod and
   empty folders; frontend/ was generated but No application code nor dependences exists yet .
3. Verify the repository and the already defined enviroment, Report anything missing; do not install anything.
4. Audit the specs and ADRs for contradictions, ambiguities, missing details, or ACs that cannot be
   tested as written. For each finding give: where (file + section), why it matters, at most two
   options with trade-offs, and your recommendation. Do NOT resolve any of them yourself.
5. Restate the working agreement in at most 8 bullets (spec order, TDD, stdlib-only backend, no new
   dependencies, frontend never computes results, never push, plan-then-wait, report format),
   so I can confirm we share the same understanding.

Then stop and wait for my answer.
```

Two follow-up prompts turned the audit into decisions. They are long, so they are summarised here:

- **My decisions on the audit** `[log 23:53, 2026-10-13]`: I answered the numbered findings (F2–F22) one by one and
  authorised edits to the specs, `docs/DECISIONS.md` and `CLAUDE.md` for that session only. Examples: keep the
  `map[Op]operation` registry and add an ordered slice (F3); compute `percentage` literally as `a*b/100` (F4); fix the
  validation order inside `Calculate` (F5); every router error is problem+json (F6); request-error precedence (F7);
  evidence types and a no-sleeps check (F10, F17); the `prompts/session-log.md` commit rule (F22). I asked for two
  proposals to be presented but **not written**: a "rule 12" for undefined key sequences, and a `formatNumber` table.
- **Proposals approved, with one change** `[log 00:10, 2026-10-14]`: I approved rule 12 and the `formatNumber`
  algorithm, but changed the exponential threshold from `1e15` to `1e12` (keeping `1e-9`), and asked for explicit
  rounding-crossing cases and for the spec to say, and test, whether the threshold uses the raw or the rounded value
  (the answer written into the spec: the **rounded** value). The result is in spec 003 and ADR-006/ADR-010.

## 02 — Per-spec

Each spec followed the same loop: `/implement-spec` → plan → my approval (and answers to any ambiguity) → TDD and
commits → `/review` → I pick which findings to fix → fix round → status set to Done. Apart from the slash commands,
these are the prompts I typed:

| Spec | Plan approval | Fix round after `/review` |
|---|---|---|
| 001 calculator domain `[log 00:13]` | `[log 00:16]` (below) | `[log 00:38]`: fix the CALC-11 exact-equality gap, add godoc, a slice in the `-0` test, an underflow test; skip the rest |
| 002 HTTP API `[log 00:41]` | `[log 00:47]`: `answers: 1. variadic options 2. option A 3. option A` + `approve plan` | `[log 01:01]`: wrapper so non-canonical `/api` paths return problem+json; document the `%zz` limit |
| 003 frontend `[log 01:03]` | `[log 01:06]`: `aprove, all recommended` (the lint-stack question was answered in an interactive choice, not a prompt) | `[log 01:30]`: make the stale-response test discriminating, replace test casts, drop unreachable branches |
| 004 quality `[log 01:36]` | `[log 01:38]`: `approve, all recommended` | `[log 01:55]`: close the unset `VITE_API_BASE_URL` coverage gap; I run `/review 004` myself |
| 005 delivery `[log 01:57]` | `[log 02:03]`: `approve, all recommended` | (pending `/review 005`) |

The approval for spec 001, which set the standard for the rest `[log 00:16]`:

```text
Plan approved. Proceed with TDD as defined in CLAUDE.md: one failing test first, minimal code,
refactor, one Conventional Commit per green increment. If you hit an ambiguity or an AC you cannot
meet as written, stop and ask instead of choosing. Finish with make check and the final report.
```

A typical fix-round prompt (spec 003) `[log 01:30]`. It names exactly which findings to fix and which to skip, and
asks for the failing test to be shown before the fix:

```text
Address Should-fix 1 and 2, plus these nits: drop the two unreachable reducer branches, document
the countDigits behavior (leading 0 of 0.xxx counts, so a decimal holds 14 fractional digits) in
the README, and record the AbortSignal/MSW setup finding as a short note in the README (or
docs/ if that fits better). Skip the other nits, including the selectors move and StrictMode.

For Should-fix 1: make the first MSW handler ignore request.signal and still resolve with a body,
so the late response really reaches the hook. Write the test first and show that it fails if the
id guard is removed from the reducer (temporarily), then restore it.
For Should-fix 2: replace the `as` casts in test code with type guards, per CLAUDE.md.

Run npm run lint, typecheck, vitest, build and prettier --check, and report coverage. Commit in
small groups (test:, refactor:, docs:), then commit prompts/session-log.md as chore(prompts) and
mark spec 003 done in specs/README.md. No push. Give me the commit list and stop.
```

Because these fix prompts arrive as pasted blocks, Claude asked me to confirm before acting on them each time; the
short `go, confirmed` / `go` entries in the log are those confirmations.

## 03 — Notable corrections

What went wrong or was incomplete, and what changed. Each item is traceable in the commit history.

**Planning (before any code)**

- The audit found ACs that could not be tested as written. My decisions turned them into amendments (evidence types,
  an order-independence rule, a no-sleeps check). The `formatNumber` threshold moved from `1e15` to `1e12` at my
  request, and the rule is decided on the rounded value, not the raw one.

**Spec 001**

- Review: the comparison helper applied the `1e-12` tolerance to **every** case, so `add(2,3) = 5.000000000001`
  would have passed. Fixed with an `approx` flag that defaults to exact equality, and a test that rejects near misses.
- My underflow example (`1e-308 / 1e10`) was wrong: it gives the subnormal `1e-318`, not `0`. Claude used
  `1e-308 / 1e20`, which really underflows, and said so.
- Claude committed a failing test once (a pipeline hid the exit status) and amended its own unpushed commit.

**Spec 002**

- Claude's plan accepted "Go's router redirects non-canonical paths" as a documented edge. The review showed it
  contradicted the spec (`GET /api` returned a `307` with an HTML body) and the README even named the wrong status.
  Fixed with a wrapper and tests; the one case that cannot be fixed (`/api/v1/%zz`, rejected by `net/http` itself) is
  documented as a known limit.
- The spec said the request DTO uses `*float64`, but that cannot tell `"b": null` from a missing `b`, which another rule
  requires. Resolved with a small `operand` type that never fails during decoding, so an unknown field is always reported
  first regardless of key order.

**Spec 003**

- The current `create-vite` template no longer ships ESLint (only `oxlint`), which conflicts with ADR-008. Claude
  stopped and asked; I chose the ESLint 9 stack, with the extra dev-only dependencies named up front.
- MSW 3 renamed `onUnhandledRequest` to `onUnhandledFrame`; the old name was silently ignored at runtime and only
  `tsc` caught it.
- Review: the "late response is ignored" integration test passed even with the reducer's id guard deleted, because the
  real client already rejects on abort. The suggested fix (have the MSW handler ignore the signal) could not work for
  the same reason; Claude proved that by removing the guard, then tested the hook with a fake client that ignores the
  abort signal, and showed that test failing without the guard.
- Making an impossible state (`input: null` with no pending operator) unrepresentable in the type removed two dead
  branches instead of hiding them behind fallbacks.

**Spec 004**

- A "gofmt must fail" check passed by accident: a formatter hook rewrote the deliberately unformatted file before
  `make` saw it. Claude noticed the exit code was wrong, redid the proof through the shell, and did not count the first
  run.
- Claude's own coverage notes were checked against the code before being committed: two explanations were wrong and a
  third incomplete. This also found a real untested branch (`VITE_API_BASE_URL` unset), fixed in a follow-up.

**Spec 005**

- A `smoke.sh` staged earlier was absorbed into the Docker commit; Claude noticed the failed second commit, undid its own
  unpushed commit with a soft reset and recommitted the two pieces separately.
- A README claim ("healthy within a few seconds") was not measured and contradicted the 10 s health-check interval, so
  it was corrected before committing.
- Port 8080 was already in use by a `server.exe` that Claude had not started, so it verified the container on another
  host port instead of stopping that process.

**Process**

- Two of Claude's shell commands were denied by my permission prompt (an unnecessary `git rm --cached -r .`, and a
  compound command with an `rm -rf` on a variable path). Claude split them into smaller steps.
- `session-log.md` also records hook-generated entries (subagent hand-backs, background-task notices). Those are not
  prompts I wrote.
