# CLAUDE.md — Sezzle Calculator

You are working on a take-home assignment: a full-stack calculator with a **React + TypeScript**
frontend and a **Go** REST microservice. Your job is to
**implement the specs faithfully**, not to redesign them.

## Source of truth (read in this order)

1. `specs/README.md` — spec index, workflow and status
2. `docs/DECISIONS.md` — architecture decisions (ADRs). These are **final** unless you recieve an instruction to do otherwise.
3. `specs/00X-*.md` — the spec you are implementing. Acceptance criteria (ACs) are the definition of done.

If code, specs and decisions disagree, **specs + ADRs win**. If a spec is ambiguous or seems wrong,
**stop and ask** — list the ambiguity, propose at most two options with trade-offs, and wait.
Never silently pick an option that changes the API contract, the error model, or the folder layout.

## Golden rules

- Implement **one spec at a time** via `/implement-spec 00X`. Plan first, wait for approval.
- **Test first** for domain logic (calculator ops, reducer). Failing test > minimal code > refactor.
- No new dependencies without asking. Backend is **stdlib only** (tests included — no testify).
  Frontend runtime deps: `react` and `react-dom` only.
- Layers stay clean: `internal/calculator` never imports `net/http` or `encoding/json`;
  React components never call `fetch` (only `src/api/` does).
- The frontend **never computes arithmetic results**. The backend is the single source of truth.
- Small commits, Conventional Commits (`feat(backend): ...`, `test(frontend): ...`). **Never `git push`.**
- Do not edit `specs/` or `docs/DECISIONS.md` unless asked. Only exception: the Status column in `specs/README.md`.
- Never lower coverage thresholds, skip tests, or add `//nolint` / `eslint-disable` to get green.

## Repository layout

```
backend/                     Go module (stdlib net/http)
  cmd/server/                composition root: config, wiring, graceful shutdown
  internal/calculator/       pure domain: operations registry, sentinel errors
  internal/httpapi/          handlers, DTOs, problem+json mapping, middleware
  internal/config/           env-based config
frontend/                    Vite + React + TypeScript (strict)
  src/api/                   typed HTTP client + API types (only place that calls fetch)
  src/features/calculator/   model/ (pure reducer), useCalculator hook, components/
  src/lib/                   number formatting helpers
  src/test/                  test setup + MSW handlers
specs/                       numbered specs with acceptance criteria
docs/DECISIONS.md            ADRs
prompts/                     prompts used (assignment deliverable)
scripts/                     coverage gate, smoke test
```

## Commands

| Task | Command |
|---|---|
| Backend dev server | `make dev-backend` (port 8080) |
| Frontend dev server | `make dev-frontend` (port 5173, proxies `/api` > 8080) |
| All tests | `make test` |
| Coverage reports + gates | `make coverage` |
| Lint + typecheck + tests + coverage | `make check` |
| Docker | `make docker-build && make docker-run`, then `make smoke` |

## Go conventions

- `gofmt`/`go vet` clean. Errors wrapped with `%w`, compared with `errors.Is`. No panics for control flow.
- Interfaces are defined **by the consumer** (`httpapi` declares the `Calculator` interface it needs).
- Table-driven tests with `t.Run` + `t.Parallel()`; HTTP tests with `httptest`. Always `-race`.
- `log/slog` structured logging. No package-level mutable state. Dependencies passed explicitly.

## TypeScript / React conventions

- `strict: true` + `noUncheckedIndexedAccess`. No `any`, no non-null `!`; use `unknown` + type guards at boundaries.
- State lives in a **pure reducer** (discriminated union on `status`); side effects only in `useCalculator`.
- Exhaustive `switch` with an `assertNever` helper.
- Tests: Vitest + React Testing Library + `user-event`; HTTP mocked with MSW.
  Query by role/label/text — never by CSS class or test-id unless there is no accessible alternative.
- CSS Modules + CSS custom properties. No UI kit.

## Definition of done (every spec)

- Every AC has one primary **evidence type** (see `specs/004`): `test` (an automated test that fails if the behavior
  breaks — the default), `command` (a command whose result is checked) or `review` (checked by `/review` or the human).
- `make check` passes; coverage gates from `specs/004` are met.
- README sections listed in the spec's **Docs** section are updated.
- When a spec is closed, commit `prompts/session-log.md` as `chore(prompts): ...`.
- Final report: AC > evidence-type > evidence table (✅/❌), coverage numbers, deviations (expected: none), open questions.

## Skills policy

- Specs, ADRs and this file outrank any skill. If a skill's workflow conflicts with them, follow the specs and tell me.
- Do NOT run brainstorming or design-exploration workflows and do not propose alternative architectures. Design is decided in `docs/DECISIONS.md`.
- Encouraged: `test-driven-development`, `systematic-debugging`, `verification-before-completion`, `requesting-code-review`.
- Plans are delivered inline in the chat using the `/implement-spec` format. Do not write plan files into the repo.
- Do not create git worktrees or branches, and never push. I manage branches.
- Simplicity (ponytail, level `lite`): prefer the standard library and the smallest change that satisfies the ACs,
  but never by removing a structure mandated by an ADR or spec (operation registry, pure reducer, effects-as-data,
  problem+json, consumer-defined interfaces).