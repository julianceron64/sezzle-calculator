# Sezzle Calculator

Full-stack calculator: **React + TypeScript** frontend consuming a **Go** REST microservice.

> Sections marked `TODO(spec-00X)` are completed as each spec is implemented.

## Quick start (Docker)

```bash
make docker-build && make docker-run   # UI + API on http://localhost:8080
make smoke                             # end-to-end smoke test with curl
```

## Local development

Prerequisites: Go ≥ 1.24, Node ≥ 22, make.

```bash
make dev-backend    # API on :8080
make dev-frontend   # UI on :5173 (proxies /api → :8080)
make check          # lint + typecheck + tests + coverage gates
```

## API

<!-- TODO(spec-002): endpoints, curl examples (success + 3 errors), configuration -->

## Error codes

<!-- TODO(spec-002): table code → status → meaning -->

## Operation semantics

<!-- TODO(spec-001): copy ADR-007 table -->

## Frontend

<!-- TODO(spec-003): UX model, keyboard shortcuts -->

## Testing & coverage

<!-- TODO(spec-004): how to run, gates, link to docs/COVERAGE.md -->

## Design decisions

Full rationale in [`docs/DECISIONS.md`](docs/DECISIONS.md). Highlights:

- **Stdlib-only Go** with 1.22+ routing; pure domain package behind a consumer-defined interface.
- **One endpoint per operation** (`POST /api/v1/{operation}`), errors as **RFC 9457 problem+json** with stable codes.
- **float64 end to end**, non-finite results rejected, display formatting owned by the frontend.
- **Pocket-calculator UX** driven by a pure reducer; effects as data, stale responses ignored, requests abortable.
- **Single container**: Go serves API + SPA, same origin, no CORS.

## Assumptions

<!-- TODO(spec-005) -->

## How this was built (AI usage)

Spec-driven development with Claude Code: I wrote the specs (`specs/`) and ADRs (`docs/DECISIONS.md`);
Claude Code implemented them one spec at a time under the rules in `CLAUDE.md`.
Every prompt is in [`prompts/`](prompts/).

## What I'd do next

<!-- TODO(spec-005) -->
